# kdeploy - KVM Cloud Image Deployer

A fast, interactive CLI tool for deploying KVM virtual machines from cloud images.

## Overview

**kdeploy** automates the creation of KVM virtual machines using cloud-init enabled cloud images (qcow2 format). What used to take 15-30 minutes of GUI installation now takes under 2 minutes.

### Why kdeploy?

- **Speed**: Deploy a VM in under 2 minutes
- **No GUI Needed**: Fully automated, command-line driven
- **Cloud-Init Integration**: Automatic SSH key injection, user setup, and package installation
- **Multiple Distros**: Support for Rocky Linux (10, 9, 8), Ubuntu (24.04, 22.04, 20.04), Debian (13, 12, 11), and more
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

1. **Download a Cloud Image**: Fetch a pre-configured minimal OS image in qcow2 format (~1GB) from upstream providers like Rocky Linux, Ubuntu, Debian, or Fedora.

2. **Create an Overlay**: Generate a copy-on-write snapshot that references the base image. This is fast to create and leaves the original image untouched.

3. **Cloud-Init Configuration**: On first boot, cloud-init automatically injects your SSH key, creates your user account, and installs any packages you specify.

**Total time**: ~30 seconds from start to SSH access.

This is essentially how AWS, DigitalOcean, and other cloud providers create VMs instantly. Now you can do the same locally.

## Features

- **Speed**: Deploy a VM in 14 seconds (with `-y` flag)
- **Interactive Setup**: Guided configuration with sensible defaults
- **Smart Path Handling**: User-owned or system-managed storage paths
- **Image Download**: Built-in catalog of popular cloud images
- **Automatic OS Detection**: Detects Rocky, Ubuntu, Debian, Alpine Linux from filename
- **Resource Selection**: Choose RAM, vCPUs, and disk size
- **SSH Config Management**: Automatic SSH host configuration
- **Fast IP Discovery**: Uses DHCP leases (no qemu-guest-agent needed)
- **Dependency Management**: Auto-detects and offers to install missing packages
- **Pre-flight Checks**: KVM, libvirt, network, storage pool, disk space
- **Permission Handling**: Auto-detects and offers to add user to libvirt group
- **UTC Timezone**: Enforces UTC timezone for cloud-native standardization

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
# Clone
git clone https://github.com/hazembook/kdeploy.git
cd kdeploy

# Build (requires Go 1.21+)
make build

# Install globally
cp bin/kdeploy ~/.local/bin/
# or: make install  (not yet implemented)
```

### Bash Script (Legacy)

The original bash script `kdeploy` also ships in the repo root. It requires no build step.<br>
*Note: all development is now focused on the Go version.*

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

# Force-teardown (remove all storage, no file cleanup)
kdeploy nuke myvm
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

```
🎯 First-time setup - configuring paths...

📂 Image Library Path
   [1] ~/VM/cloud_images (user-owned, recommended)
   [2] /var/lib/libvirt/images (system-managed)
   [3] Custom path

   Choice [1]: 1

💾 VM Storage Path
   [1] /var/lib/libvirt/images (libvirt standard)
   [2] ~/VM/disks (user-owned)
   [3] Custom path

   Choice [1]: 2

✅ Configuration saved to ~/.config/kdeploy.conf
```

## Configuration

Configuration is stored in `~/.config/kdeploy.conf`:

```bash
IMAGE_PATH=/home/user/VM/cloud_images
STORAGE_PATH=/home/user/VM/disks
DEFAULT_VM_SIZE=20G
DEFAULT_RAM=2048
DEFAULT_CPUS=2
DEFAULT_PASSWORD=linux
```

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

## Built-in Virsh Replacement

The Go version has all these commands **built in** — no aliases needed:

| Old alias | `kdeploy` command | What it does |
|-----------|-------------------|--------------|
| `vv` | `kdeploy list` | List all VMs |
| `vs` | `kdeploy start` | Start a VM |
| `vq` | `kdeploy shutdown` | ACPI graceful shutdown |
| `vc` | `kdeploy console` | Serial console |
| `vd` | `kdeploy stop` | Force power-off |
| `vr` | `kdeploy reboot` | Reboot |
| `ve` | `kdeploy edit` | Edit XML config |
| `va` | `kdeploy ip` | Show VM IP |
| `vxx` | `kdeploy nuke` | Destroy + remove all storage |

---

## Roadmap

- ✅ **Go rewrite**: Full feature parity with the bash script
- ✅ **Built-in virsh commands**: No more shell aliases needed
- ✅ **Dependency management**: Auto-detect and install missing packages
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
