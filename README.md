# kdeploy - KVM Cloud Image Deployer

A fast, interactive CLI tool for deploying KVM virtual machines from cloud images.

## Overview

**kdeploy** automates the creation of KVM virtual machines using cloud-init enabled cloud images (qcow2 format). What used to take 15-30 minutes of GUI installation now takes under 2 minutes.

### Why kdeploy?

- **Speed**: Deploy a VM in under 2 minutes
- **No GUI Needed**: Fully automated, command-line driven
- **Cloud-Init Integration**: Automatic SSH key injection, user setup, and package installation
- **Multiple Distros**: Support for Rocky Linux, Ubuntu, Debian, Fedora, and more
- **Interactive**: Guided setup with sensible defaults
- **Open Source**: Released under the Waqf Public License

## The Story Behind kdeploy

This tool was born out of frustration with slow VM installations. After 5 years of using traditional GUI-based VM installations, I discovered:

1. **Cloud Images**: Pre-configured, minimal OS images in qcow2 format
2. **Overlay Images**: Copy-on-write snapshots that don't modify the base image
3. **Cloud-Init**: Automatic VM configuration (users, SSH keys, packages) at first boot

The combination of these three technologies completely changed how I work with VMs. No more clicking through installation wizards, no more waiting for packages to download, no more manual SSH configuration.

### How It Works

```
┌─────────────────┐     ┌──────────────────┐     ┌─────────────────┐
│  Cloud Image    │────▶│  Overlay (qcow2) │────▶│   VM Created    │
│  (Base, ~1GB)   │     │  (Copy-on-write) │     │   in ~1 minute  │
└─────────────────┘     └──────────────────┘     └─────────────────┘
                                │
                                ▼
                        ┌──────────────────┐
                        │   Cloud-Init     │
                        │  (user, SSH,     │
                        │   packages)      │
                        └──────────────────┘
```

This is essentially how AWS, DigitalOcean, and other cloud providers create VMs instantly. Now you can do the same locally.

## Features

- **Interactive Setup**: Guided configuration with sensible defaults
- **Smart Path Handling**: User-owned or system-managed storage paths
- **Image Download**: Built-in catalog of popular cloud images
- **Automatic OS Detection**: Detects Rocky, Ubuntu, Debian, Fedora from filename
- **Resource Selection**: Choose RAM, vCPUs, and disk size
- **SSH Config Management**: Automatic SSH host configuration
- **Dual IP Discovery**: Works with qemu-guest-agent or DHCP leases
- **Dependency Management**: Auto-detects and offers to install missing packages

## Requirements

### System Packages

| Distribution | Packages |
|-------------|----------|
| Arch/CachyOS | `libvirt virt-install qemu-img cloud-utils openssl wget curl` |
| Debian/Ubuntu | `libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl` |
| Fedora/Rocky | `libvirt virt-install qemu-img cloud-utils openssl wget curl` |

### Other Requirements

- KVM virtualization enabled in BIOS/UEFI
- SSH public key at `~/.ssh/id_rsa.pub`
- Write access to chosen storage path

## Installation

```bash
# Clone or download the script
git clone https://github.com/hazembook/kdeploy.git
cd kdeploy

# Make executable
chmod +x kdeploy.sh

# Run - first run will guide you through setup
./kdeploy.sh
```

## Usage

```bash
# Basic deployment (follow interactive prompts)
./kdeploy.sh myvm

# Specify disk size
./kdeploy.sh myvm 50G

# Override resources
./kdeploy.sh myvm -r 4096 -c 4

# Use custom paths (one-time override)
./kdeploy.sh myvm -i /path/to/images -s /path/to/storage

# Reconfigure default paths
./kdeploy.sh --reconfig

# Show current configuration
./kdeploy.sh --show-config
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

### RHEL/CentOS Certification Prep

Practice on:
- Rocky Linux 9/10
- Debian 12/13
- Ubuntu 24
- or download your prefered distro cloud image

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
| **kdeploy** | **1-2 min** | **No** |

## License

This project is licensed under the **Waqf Public License 2.0**.

The Waqf license is an Islamic-inspired, business-friendly, permissive license that:
- Allows free use, modification, and redistribution
- Requires attribution
- Does not restrict commercial use
- Has no time limit on the license

See the [LICENSE](LICENSE) file for full text.
