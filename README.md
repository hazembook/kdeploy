# kdeploy - KVM Cloud Image Deployer

A fast, interactive CLI tool for deploying KVM virtual machines from cloud images.

> **Status: alpha.** Usable for everyday VM provisioning, but commands, flags,
> config, and output may still change. Use external backups for irreplaceable
> VM disks.

## Overview

**kdeploy** automates the creation of KVM virtual machines using cloud-init enabled cloud images (qcow2 format). What used to take 15-30 minutes of GUI installation now takes under 2 minutes.

### Why kdeploy?

- **Speed**: Deploy a VM in under 2 minutes
- **No GUI Needed**: Fully automated, command-line driven
- **Cloud-Init Integration**: Automatic SSH key injection, user setup, and package installation
- **Multiple Distros**: Support for Rocky Linux (10, 9, 8), Ubuntu (24.04, 22.04, 20.04), and Debian (13, 12, 11)
- **Interactive**: Guided setup with sensible defaults
- **Open Source**: Released under the Waqf Public License

## The Story Behind kdeploy

This tool was born out of frustration with slow VM installations. After 5 years of using traditional GUI-based VM installations, I discovered:

1. **Cloud Images**: Pre-configured, minimal OS images in qcow2 format
2. **Overlay Images**: Copy-on-write snapshots that don't modify the base image
3. **Cloud-Init**: Automatic VM configuration (users, SSH keys, packages) at first boot

The combination of these three technologies completely changed how I work with VMs. No more clicking through installation wizards, no more waiting for packages to download, no more manual SSH configuration.

### How It Works

kdeploy creates VMs in three simple steps:

1. **Download a Cloud Image**: Fetch a pre-configured minimal OS image in qcow2 format (~1GB) from upstream providers like Rocky Linux, Ubuntu, or Debian.

2. **Create an Overlay**: Generate a copy-on-write snapshot that references the base image. This is fast to create and leaves the original image untouched.

3. **Cloud-Init Configuration**: On first boot, cloud-init automatically injects your SSH key, creates your user account, and installs any packages you specify.

**Total time**: ~30 seconds from start to SSH access.

This is essentially how AWS, DigitalOcean, and other cloud providers create VMs instantly. Now you can do the same locally.

## Features

- **Speed**: Deploy a VM in seconds with `--yes`
- **Guided interaction**: Charmbracelet forms for first-run setup, images, resources, and review
- **Review before create**: Confirm the image, resources, storage, and SSH aliases before `virt-install`
- **Safe defaults**: Destructive confirmations default to No; non-TTY without `--yes` fails early
- **Scriptable output**: Final results on stdout, prompts/progress on stderr; plain output when piped
- **Smart path handling**: User-owned or system-managed storage paths
- **Image download**: Built-in catalog, custom URLs, and paused safe demos
- **Automatic OS detection**: Detects Rocky, Ubuntu, Debian, and Alpine from the filename
- **SSH config management**: Automatic host aliases and config cleanup
- **Fast IP discovery**: Uses DHCP leases (no qemu-guest-agent needed)
- **Dependency management**: `check` reports details, exits `1` on failure, and can install missing packages
- **Pre-flight checks**: KVM, libvirt, network, tools, SSH key, and disk space
- **UTC timezone**: Enforces UTC for cloud-native standardization

## Requirements

### System Packages

| Distribution | Packages |
|-------------|----------|
| Arch/CachyOS | `libvirt virt-install qemu-img cloud-utils openssl wget curl` |
| Debian/Ubuntu | `libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl` |
| Fedora/Rocky | `libvirt virt-install qemu-img cloud-utils openssl wget curl` |

### Prerequisites

1. **KVM virtualization** enabled in BIOS/UEFI
2. **SSH public key** at `~/.ssh/id_ed25519.pub` (auto-generated if missing)
3. **libvirt group membership** (recommended for passwordless operation):

   ```bash
   sudo usermod -aG libvirt $USER
   # Log out and back in for this to take effect
   ```

   Without this, `kdeploy` falls back to `sudo`/`doas` for privileged operations.
4. **Write access** to chosen image and storage paths

## Installation

### Go Rewrite (Recommended)

```bash
# Clone (Codeberg is primary; GitLab and GitHub are mirrors)
git clone https://codeberg.org/hazembook/kdeploy.git
cd kdeploy

# Build (requires Go 1.26+; produces one static binary)
make build

# Install globally
cp bin/kdeploy ~/.local/bin/
# or: make install  (not yet implemented)
```

Mirrors: [GitLab](https://gitlab.com/hazembook/kdeploy) ·
[GitHub](https://github.com/hazembook/kdeploy).

`kdeploy` is one static Go binary. The interactive UI uses Charmbracelet Go
libraries; `gum`, VHS, `ttyd`, and `ffmpeg` are **not** runtime dependencies.

Quality gate for development:

```bash
make check   # gofmt check, go vet, and race tests
```

### Bash Script (Legacy)

The original bash script `kdeploy` also ships in the repo root and is tagged
`v3.4`. It requires no build step. All development is now focused on the Go
version.

## Commands

### VM Lifecycle

```bash
# Deploy a new VM (interactive)
kdeploy deploy myvm

# Fast deploy (skip all prompts)
kdeploy deploy myvm -y

# Deploy with custom resources
kdeploy deploy myvm 50G -r 4096 -c 4 -i /path/to/images -s /path/to/storage

# Custom network with static MAC→IP mapping
kdeploy deploy myvm -n "my-custom-network,mac=52:54:00:ab:cd:01"

# Destroy a VM (clean up files + SSH config)
kdeploy destroy myvm

```

### VM Management (built-in virsh wrappers)

```bash
kdeploy list              # List all VMs
kdeploy start myvm        # Start a stopped VM
kdeploy shutdown myvm     # ACPI graceful shutdown
kdeploy stop myvm         # Force power-off (virsh destroy)
kdeploy reboot myvm       # Reboot
kdeploy console myvm      # Interactive serial console
kdeploy edit myvm         # Edit VM XML ($EDITOR)
kdeploy ip myvm           # Show VM IP address
kdeploy info [vm]         # Show VM info (name, state, IP)
```

### Image Management

```bash
kdeploy download              # Interactive image picker
kdeploy download --list       # List available images
kdeploy download rocky10      # Download by key
kdeploy download --url <url>  # Custom URL
```

### Configuration

```bash
kdeploy config init           # Create default config
kdeploy config init --force   # Overwrite existing config
kdeploy config show           # Display current config
kdeploy --reconfig            # Shortcut for config init --force
kdeploy --show-config         # Shortcut for config show
```

### System

```bash
kdeploy check              # Check dependencies
kdeploy check --install    # Check + auto-install missing packages
```

## First Run Experience

The first interactive `deploy` without a config file opens a form instead of
asking for numbered choices:

- **Image library**: `~/VM/cloud_images`, `/var/lib/libvirt/images`, or Custom.
- **VM storage**: `/var/lib/libvirt/images`, `~/VM/disks`, or Custom.
- Paths beginning with `~` are expanded.
- System paths explain that sudo may be requested.
- The config is saved atomically with mode `0600`.

![First-run setup form](docs/media/first-run.gif)

After setup, deploy scans the image library, collects resources and access
settings, and shows a review panel. Nothing is created until **Create VM** is
selected. **Cancel** leaves existing VMs and files untouched.

## Interactive and scripted use

### Streams

| Stream | Content |
|---|---|
| stdout | Final results: lists, info, IP, check report, download catalog, deploy summary |
| stderr | Prompts, forms, spinners, progress, warnings, errors, review panels |

Piping stdout switches to plain output: no ANSI, no spinner, stable
tab-separated tables. `kdeploy ip` always prints one plain line to stdout.

### TTY and `--yes`

- Interactive prompts need a terminal on stdin and stderr (`TERM=dumb` disables them).
- `--yes` / `-y` is fully non-interactive and uses config/built-in defaults.
- Non-TTY without `--yes` fails before destructive action.
- `deploy --yes` uses the first available image; it will not download blindly.
- `download --yes` overwrites an existing image without prompting.
- `destroy` and `download` overwrite confirmations default to **No**.
- Ctrl+C in a form prints `Cancelled.` and exits `130`.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | Success; for `check`, every dependency passed |
| `1` | Error, failed/incomplete check, declined install, or declined destroy |
| `130` | Cancelled with Ctrl+C |

`kdeploy check` exits `0` only when all checks pass. `kdeploy --version` prints
the injected build version and does not touch config or VM state.

### Colors

Set `NO_COLOR=1` to disable color. `TERM=dumb` and piped stdout use plain mode.

## Demos

Safe terminal recordings are committed under [`docs/media/`](docs/media/) and
are regenerated by developers with [VHS](https://github.com/charmbracelet/vhs):

```bash
make demo   # requires vhs, ttyd, and ffmpeg; not part of make check
```

The safe tapes use disposable fixtures and fake VM tools. Deploy cancels at
review, destroy presses the default No, and download cancels before a transfer.
The recordings never create, stop, or delete a real VM. See
[`demo/README.md`](demo/README.md) for the tape list and safety rules.

| Flow | 120 columns | 80 columns |
|---|---|---|
| Deploy review | ![Deploy at 120 columns](docs/media/deploy-120.gif) | ![Deploy at 80 columns](docs/media/deploy-80.gif) |
| Download picker | ![Download at 120 columns](docs/media/download-120.gif) | ![Download at 80 columns](docs/media/download-80.gif) |
| Destroy confirmation | ![Destroy at 120 columns](docs/media/destroy-120.gif) | ![Destroy at 80 columns](docs/media/destroy-80.gif) |

More stills: [first-run setup](docs/media/first-run-setup.png),
[deploy review at 80 columns](docs/media/deploy-80-review.png),
[deploy review at 120 columns](docs/media/deploy-120-review.png).

## Configuration

Configuration is stored in `~/.config/kdeploy.conf`, atomically written with
mode `0600`:

```bash
IMAGE_PATH=/home/user/VM/cloud_images
STORAGE_PATH=/home/user/VM/disks
DEFAULT_VM_SIZE=20G
DEFAULT_RAM=2048
DEFAULT_CPUS=2
DEFAULT_PASSWORD=linux
```

`DEFAULT_PASSWORD` is stored in plaintext for compatibility with the existing
config contract. `config show` and the interactive review mask it; do not use a
real secret unless you accept that file-permission model. Override the path with
`--config`, or re-run the wizard with `kdeploy --reconfig`.

Resolution order for deploy values is **explicit flag > config file > built-in
default**.

See [`docs/ui-refresh.md`](docs/ui-refresh.md) for the full UX contract and
[`docs/decisions.md`](docs/decisions.md) for the curated decision log.

## Use Cases

### Learning Linux Administration

Quickly spin up VMs to practice:
- Systemd and services
- Network configuration
- User and group management
- LVM and storage
- Container orchestration

### Development Environments

Create isolated dev environments in seconds:
- Database servers
- Web application stacks
- Kubernetes nodes

## How It Compares

| Method | Time to First Boot | Manual Config |
|--------|-------------------|---------------|
| GUI (Virt-Manager) | 15-30 min | Yes |
| Text Installer | 10-20 min | Yes |
| **kdeploy** (interactive) | ~30 sec | **No** |
| **kdeploy** (`-y` flag) | **~14 sec** | **No** |

## Troubleshooting

### Timeout Waiting for IP

If deployment times out while waiting for DHCP/IP:

1. **Check the VM console**: `virsh console <vm-name>` (login: cloud-user/rocky/ubuntu/debian)
2. **Check network leases**: `sudo virsh net-dhcp-leases <network-name>` (use your custom network name, not `default`)
3. **Verify your firewall allows traffic on the virtual bridge** (see below):

#### Firewall Blocking DHCP on Custom Networks

If you deployed with `-n my-custom-network`, your host firewall may block DHCP. Fix depends on your distro:

| Distribution | Firewall | Fix |
|---|---|---|
| **Arch/CachyOS** | UFW | `sudo ufw allow in on virbr-xxx` + route rules (see below) |
| **Ubuntu/Debian** | UFW | `sudo ufw allow in on virbr-xxx` + route rules (see below) |
| **Fedora/Rocky/RHEL** | firewalld | `sudo firewall-cmd --zone=libvirt --add-interface=virbr-xxx --permanent` |

**UFW-specific fix** (find your bridge name with `sudo virsh net-info <net-name>`):
```bash
sudo ufw allow in on virbr-xxx
sudo ufw route allow in on virbr-xxx
sudo ufw route allow out on virbr-xxx
```

After applying, restart the VM to trigger a fresh DHCP lease:
```bash
virsh destroy <vm-name> && virsh start <vm-name>
```

### Permission Denied

If you get permission errors:
```bash
sudo usermod -aG libvirt $USER
# Log out and back in
```

### KVM Not Available

Ensure virtualization is enabled in BIOS/UEFI, then load module:
```bash
sudo modprobe kvm
sudo modprobe kvm_intel  # for Intel CPUs
sudo modprobe kvm_amd    # for AMD CPUs
```

## Hardware & Performance

### Test Hardware
- **Machine**: Lenovo ideapad 320-15IKB Touch
- **CPU**: Intel Core i7-8550U (4 cores, 8 threads) @ 4.00 GHz
- **RAM**: 20 GB DDR4
- **Storage**: 477GB SSD (512MB boot + 128GB root + 348GB home)

### Real-World Times

| Mode | Time to First Boot | Notes |
|------|-------------------|-------|
| Interactive | 18-30 sec | With prompts |
| Non-interactive (`-y`) | 14-18 sec | Fully automated |
| GUI (Virt-Manager) | 15-30 min | Manual install |

## Aliases Recommendation

If you prefer the original `virsh` shortcuts, or if you're using the legacy bash script, add these to your shell config.

### Bash / Zsh

Add these to your `~/.bashrc` or `~/.zshrc`:

```bash
## Virsh (Left-Handed V-Set)
alias vv='virsh list --all'
alias vs='virsh start'
alias vq='virsh shutdown'
alias vc='virsh console'
alias vd='virsh destroy'
alias vr='virsh reboot'
alias ve='virsh edit'

function va() {
    virsh domifaddr "$1" | awk '/ipv4/ {print $4}' | cut -d/ -f1
}

function vxx() {
    echo "Destroying and Undefining: $1"
    virsh destroy "$1" 2>/dev/null
    virsh undefine "$1" --remove-all-storage
}
```

### Fish

Add these to your `~/.config/fish/config.fish`:

```fish
## Virsh (Left-Handed V-Set)
alias vv 'virsh list --all'
alias vs 'virsh start'
alias vq 'virsh shutdown'
alias vc 'virsh console'
alias vd 'virsh destroy'
alias vr 'virsh reboot'
alias ve 'virsh edit'
function va; virsh domifaddr $argv[1] | awk '/ipv4/ {print $4}' | cut -d/ -f1; end
function vxx 
    echo "Destroying and Undefining: $argv[1]"
    virsh destroy $argv[1] 2>/dev/null
    virsh undefine $argv[1] --remove-all-storage
end 
```

## Versioning

kdeploy follows SemVer on a `0.x` line while it is alpha:

| Tag | Meaning |
|---|---|
| `v3.4` | Legacy bash script release |
| `v0.0.1` | Go rewrite scaffold |
| `v0.1.0` | First usable Go release: feature parity + Charm UI refresh |

Version bumps belong to program releases, not docs-only commits. Patch releases
(`0.1.x`) carry fixes; minor releases (`0.x.0`) carry new features or UX
changes. `v1.0.0` will mark a frozen CLI, config, and output contract.

---

## Roadmap

- ✅ **Go rewrite**: Full feature parity with the bash script
- ✅ **Built-in virsh commands**: No more shell aliases needed
- ✅ **Dependency management**: Auto-detect and install missing packages
- ✅ **Tier B UI refresh**: Charmbracelet forms, review flow, safe defaults, plain output
- ⬜ **Backup and restore**: Durable VM backups (XML + flattened disk + manifest)
- ⬜ **Snapshots**: Quick rollback via `virsh snapshot-create-as`
- ⬜ **Native Go downloader**: Styled progress without `wget`/`curl`
- ⬜ **Pure-Go ISO 9660**: Replace `cloud-localds` dependency
- ⬜ **Native libvirt**: Direct Unix socket (`libvirt.so`) binding
- ⬜ **Concurrent deploys**: Multi-VM lab orchestration via goroutines

---

This project is licensed under the **Waqf Public License 2.0**.

The Waqf license is an Islamic-inspired, business-friendly, permissive license that:
- Allows free use, modification, and redistribution
- Requires attribution
- Does not restrict commercial use
- Has no time limit on the license

See the [LICENSE](LICENSE) file for full text.
