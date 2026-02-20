#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer v3.3
# "The Professor's Edition" - Configurable Paths, Image Downloads, Robust Deploy
# ==============================================================================

set -e

# --- CONFIG CONSTANTS ---
CONFIG_FILE="$HOME/.config/kdeploy.conf"
DEFAULT_IMAGE_PATH="$HOME/VM/cloud_images"
DEFAULT_STORAGE_PATH="/var/lib/libvirt/images"
DEFAULT_VM_SIZE="20G"
DEFAULT_RAM="2048"
DEFAULT_CPUS="2"
DEFAULT_PASS="linux"

# --- IMAGE CATALOG FOR DOWNLOADS ---
declare -A IMAGE_URLS=(
    ["rocky10"]="https://dl.rockylinux.org/pub/rocky/10/images/x86_64/Rocky-10-GenericCloud-Base.latest.x86_64.qcow2"
    ["rocky9"]="https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
    ["rocky8"]="https://dl.rockylinux.org/pub/rocky/8/images/x86_64/Rocky-8-GenericCloud-Base.latest.x86_64.qcow2"
    ["ubuntu2404"]="https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img"
    ["ubuntu2204"]="https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img"
    ["ubuntu2004"]="https://cloud-images.ubuntu.com/focal/current/focal-server-cloudimg-amd64.img"
    ["debian12"]="https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2"
    ["debian13"]="https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2"
    ["fedora41"]="https://download.fedoraproject.org/pub/fedora/linux/releases/41/Cloud/x86_64/images/Fedora-Cloud-Base-Generic-41-1.4.x86_64.qcow2"
)

declare -A IMAGE_NAMES=(
    ["rocky10"]="Rocky Linux 10 (latest)"
    ["rocky9"]="Rocky Linux 9 (latest)"
    ["rocky8"]="Rocky Linux 8 (latest)"
    ["ubuntu2404"]="Ubuntu 24.04 LTS (Noble)"
    ["ubuntu2204"]="Ubuntu 22.04 LTS (Jammy)"
    ["ubuntu2004"]="Ubuntu 20.04 LTS (Focal)"
    ["debian12"]="Debian 12 (Bookworm)"
    ["debian13"]="Debian 13 (Trixie)"
    ["fedora41"]="Fedora 41"
)

declare -a IMAGE_KEYS=("rocky10" "rocky9" "rocky8" "ubuntu2404" "ubuntu2204" "ubuntu2004" "debian12" "debian13" "fedora41")

# --- PACKAGE CATALOG ---
declare -A DISTRO_PACKAGES=(
    ["arch"]="libvirt virt-install qemu-img cloud-utils openssl wget curl"
    ["debian"]="libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl"
    ["ubuntu"]="libvirt-daemon-system virtinst qemu-utils cloud-image-utils openssl wget curl"
    ["fedora"]="libvirt virt-install qemu-img cloud-utils openssl wget curl"
    ["rocky"]="libvirt virt-install qemu-img cloud-utils openssl wget curl"
    ["almalinux"]="libvirt virt-install qemu-img cloud-utils openssl wget curl"
    ["opensuse"]="libvirt virt-install qemu-tools cloud-utils openssl wget curl"
)

# --- SSH CONSTANTS ---
SSH_DIR="$HOME/.ssh"
SSH_CONFIG="$SSH_DIR/config"
SSH_INCLUDE_DIR="$SSH_DIR/kdeploy.d"

# --- VARIABLES (will be set via config or flags) ---
IMAGE_PATH=""
STORAGE_PATH=""
VM_NAME=""
VM_SIZE=""
VM_RAM=""
VM_CPUS=""
OVERRIDE_IMAGE_PATH=""
OVERRIDE_STORAGE_PATH=""
OVERRIDE_RAM=""
OVERRIDE_CPUS=""

# ---------------------
# Handle if script is run via sudo/doas
VM_USER="${SUDO_USER:-${DOAS_USER:-$USER}}"
SSH_PUB_KEY="$HOME/.ssh/id_rsa.pub"
SSH_PRIV_KEY="${SSH_PUB_KEY%.pub}"
# ---------------------

# --- HELPER FUNCTIONS ---

show_help() {
    cat <<EOF
KVM Cloud Image Deployer v3.3

Usage: $0 <vm_name> [disk_size] [options]

Arguments:
    vm_name         Name of the virtual machine (required)
    disk_size       Size of VM disk (default: ${DEFAULT_VM_SIZE})

Options:
    -r, --ram MB           RAM in MB (default: ${DEFAULT_RAM})
    -c, --cpu COUNT        Number of vCPUs (default: ${DEFAULT_CPUS})
    -i, --image-path DIR   Override image library path (one-time)
    -s, --storage-path DIR Override VM storage path (one-time)
    --reconfig             Reconfigure saved paths
    --show-config          Show current configuration
    -h, --help             Show this help message

Pre-flight Checks (automatic):
    - KVM kernel module verification
    - Libvirt daemon status
    - Default network setup
    - SSH key verification/generation
    - Required package dependencies

Interactive Prompts:
    - First run: Configure image & storage paths
    - Each run: Select base image (or download new)
    - Each run: Set VM password
    - Each run: Confirm RAM/vCPU/disk (or use defaults)

Examples:
    $0 webserver                    Deploy VM with defaults
    $0 webserver 50G                Deploy VM with 50GB disk
    $0 webserver -r 4096 -c 4       Deploy with 4GB RAM, 4 vCPUs
    $0 webserver -i /tmp/img -s /tmp/vms  Use custom paths
    $0 --reconfig                   Reconfigure default paths
    $0 --show-config                Show current configuration

Configuration file: $CONFIG_FILE
EOF
}

needs_privilege() {
    local path="$1"
    [[ "$path" == /var/* || "$path" == /srv/* || "$path" == /opt/* ]]
}

load_config() {
    if [[ -f "$CONFIG_FILE" ]]; then
        source "$CONFIG_FILE"
    else
        IMAGE_PATH="$DEFAULT_IMAGE_PATH"
        STORAGE_PATH="$DEFAULT_STORAGE_PATH"
    fi
    VM_SIZE="${VM_SIZE:-$DEFAULT_VM_SIZE}"
    VM_RAM="${VM_RAM:-$DEFAULT_RAM}"
    VM_CPUS="${VM_CPUS:-$DEFAULT_CPUS}"
}

save_config() {
    local config_dir
    config_dir="$(dirname "$CONFIG_FILE")"
    mkdir -p "$config_dir"
    cat > "$CONFIG_FILE" <<EOF
# kdeploy configuration
# Generated: $(date '+%Y-%m-%d %H:%M:%S')

IMAGE_PATH=$IMAGE_PATH
STORAGE_PATH=$STORAGE_PATH
DEFAULT_VM_SIZE=$DEFAULT_VM_SIZE
DEFAULT_RAM=$DEFAULT_RAM
DEFAULT_CPUS=$DEFAULT_CPUS
DEFAULT_PASSWORD=$DEFAULT_PASS
EOF
    chmod 600 "$CONFIG_FILE"
    echo "✅ Configuration saved to $CONFIG_FILE"
}

configure_paths() {
    echo ""
    echo "🎯 First-time setup - configuring paths..."
    echo ""
    
    echo "📂 Image Library Path"
    echo "   [1] ~/VM/cloud_images (user-owned, recommended)"
    echo "   [2] /var/lib/libvirt/images (system-managed)"
    echo "   [3] Custom path"
    echo ""
    read -p "   Choice [1]: " img_choice
    img_choice="${img_choice:-1}"
    
    case "$img_choice" in
        1) IMAGE_PATH="$HOME/VM/cloud_images" ;;
        2) IMAGE_PATH="/var/lib/libvirt/images" ;;
        3) 
            read -p "   Enter path: " IMAGE_PATH
            IMAGE_PATH="${IMAGE_PATH/#\~/$HOME}"
            ;;
        *) 
            echo "   Invalid choice, using default"
            IMAGE_PATH="$HOME/VM/cloud_images"
            ;;
    esac
    
    echo ""
    echo "💾 VM Storage Path"
    echo "   [1] /var/lib/libvirt/images (libvirt standard)"
    echo "   [2] ~/VM/disks (user-owned)"
    echo "   [3] Custom path"
    echo ""
    read -p "   Choice [1]: " storage_choice
    storage_choice="${storage_choice:-1}"
    
    case "$storage_choice" in
        1) STORAGE_PATH="/var/lib/libvirt/images" ;;
        2) STORAGE_PATH="$HOME/VM/disks" ;;
        3)
            read -p "   Enter path: " STORAGE_PATH
            STORAGE_PATH="${STORAGE_PATH/#\~/$HOME}"
            ;;
        *)
            echo "   Invalid choice, using default"
            STORAGE_PATH="/var/lib/libvirt/images"
            ;;
    esac
    
    echo ""
    save_config
}

detect_distro() {
    if [[ -f /etc/arch-release ]]; then
        echo "arch"
    elif [[ -f /etc/debian_version ]]; then
        if grep -qi ubuntu /etc/os-release 2>/dev/null; then
            echo "ubuntu"
        else
            echo "debian"
        fi
    elif [[ -f /etc/fedora-release ]]; then
        echo "fedora"
    elif [[ -f /etc/rocky-release ]]; then
        echo "rocky"
    elif [[ -f /etc/almalinux-release ]]; then
        echo "almalinux"
    elif [[ -f /etc/os-release ]]; then
        source /etc/os-release
        case "$ID" in
            arch|manjaro|endeavouros|cachyos) echo "arch" ;;
            debian|ubuntu|linuxmint|pop) echo "debian" ;;
            fedora|rhel|centos) echo "fedora" ;;
            rocky|almalinux) echo "rocky" ;;
            opensuse*) echo "opensuse" ;;
            *) echo "$ID" ;;
        esac
    else
        echo "unknown"
    fi
}

get_package_manager() {
    if command -v pacman &>/dev/null; then
        echo "pacman -S"
    elif command -v apt &>/dev/null; then
        echo "apt install"
    elif command -v dnf &>/dev/null; then
        echo "dnf install"
    elif command -v yum &>/dev/null; then
        echo "yum install"
    elif command -v zypper &>/dev/null; then
        echo "zypper install"
    else
        echo "unknown"
    fi
}

check_dependencies() {
    local missing=()
    local critical_cmds=("virsh" "virt-install" "cloud-localds" "qemu-img" "openssl")
    
    for cmd in "${critical_cmds[@]}"; do
        if ! command -v "$cmd" &>/dev/null; then
            missing+=("$cmd")
        fi
    done
    
    # wget or curl (need at least one)
    if ! command -v wget &>/dev/null && ! command -v curl &>/dev/null; then
        missing+=("wget/curl")
    fi
    
    if [[ ${#missing[@]} -eq 0 ]]; then
        return 0
    fi
    
    echo ""
    echo "❌ Missing dependencies: ${missing[*]}"
    echo ""
    
    local distro
    distro=$(detect_distro)
    
    local pkg_manager
    pkg_manager=$(get_package_manager)
    
    local packages="${DISTRO_PACKAGES[$distro]}"
    
    if [[ -z "$packages" ]]; then
        echo "Could not detect required packages for your distribution ($distro)."
        echo "Please install the missing commands manually."
        exit 1
    fi

    echo "Detected system: $distro"
    echo "Package manager: $pkg_manager"
    echo "Required packages: $packages"
    echo ""
    
    read -p "Install now? [Y/n] " install_choice
    install_choice="${install_choice:-Y}"
    
    if [[ "${install_choice,,}" == "y" ]]; then
        echo ""
        echo "Installing dependencies..."
        
        # Determine install command structure
        case "$pkg_manager" in
            "pacman -S")
                # Arch/CachyOS specific: sync before install, handle meta-packages
                $PRIV_CMD pacman -S --needed $packages
                ;;
            "apt install")
                $PRIV_CMD apt update
                $PRIV_CMD apt install -y $packages
                ;;
            "dnf install"|"yum install")
                $PRIV_CMD $pkg_manager -y $packages
                ;;
            "zypper install")
                $PRIV_CMD zypper install -y $packages
                ;;
            *)
                echo "❌ Unknown package manager. Install manually."
                exit 1
                ;;
        esac
        
        echo ""
        echo "✅ Dependencies installed."
        echo ""
        
        # Re-check after installation
        for cmd in "${critical_cmds[@]}"; do
            if ! command -v "$cmd" &>/dev/null; then
                echo "❌ $cmd still not found. You may need to log out and back in."
                exit 1
            fi
        done
    else
        echo "Aborted. Install dependencies and try again."
        exit 1
    fi
}

download_image() {
    local url="$1"
    local filename
    filename="$(basename "$url")"
    local dest="$IMAGE_PATH/$filename"
    
    echo ""
    echo "📥 Downloading: $filename"
    echo "   From: $url"
    echo "   To: $dest"
    echo ""
    
    if needs_privilege "$IMAGE_PATH"; then
        if command -v wget &> /dev/null; then
            $PRIV_CMD wget -c --progress=bar:force "$url" -O "$dest"
        elif command -v curl &> /dev/null; then
            $PRIV_CMD curl -L -C - --progress-bar "$url" -o "$dest"
        else
            echo "❌ Need wget or curl to download"
            return 1
        fi
        $PRIV_CMD chmod 644 "$dest"
    else
        if command -v wget &> /dev/null; then
            wget -c --progress=bar:force "$url" -O "$dest"
        elif command -v curl &> /dev/null; then
            curl -L -C - --progress-bar "$url" -o "$dest"
        else
            echo "❌ Need wget or curl to download"
            return 1
        fi
        chmod 644 "$dest"
    fi
    
    echo ""
    echo "✅ Download complete: $filename"
}

# --- PARSE ARGUMENTS ---
RECONFIG=false
POSITIONAL_ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --reconfig)
            RECONFIG=true
            shift
            ;;
        --show-config)
            if [[ -f "$CONFIG_FILE" ]]; then
                cat "$CONFIG_FILE"
            else
                echo "No configuration file found at $CONFIG_FILE"
                echo "Run '$0 --reconfig' to create one."
            fi
            exit 0
            ;;
        -r|--ram)
            VM_RAM="$2"
            OVERRIDE_RAM="yes"
            shift 2
            ;;
        -c|--cpu)
            VM_CPUS="$2"
            OVERRIDE_CPUS="yes"
            shift 2
            ;;
        -i|--image-path)
            IMAGE_PATH="${2/#\~/$HOME}"
            OVERRIDE_IMAGE_PATH="yes"
            shift 2
            ;;
        -s|--storage-path)
            STORAGE_PATH="${2/#\~/$HOME}"
            OVERRIDE_STORAGE_PATH="yes"
            shift 2
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        -*)
            echo "Unknown option: $1"
            echo "Run '$0 --help' for usage."
            exit 1
            ;;
        *)
            POSITIONAL_ARGS+=("$1")
            shift
            ;;
    esac
done

set -- "${POSITIONAL_ARGS[@]}"

VM_NAME="${POSITIONAL_ARGS[0]:-}"
VM_SIZE="${POSITIONAL_ARGS[1]:-$DEFAULT_VM_SIZE}"

# --- INITIALIZE CONFIG ---
# Save flag-provided paths before loading config
FLAG_IMAGE_PATH="$IMAGE_PATH"
FLAG_STORAGE_PATH="$STORAGE_PATH"

load_config

# Apply overrides after loading config (flags take precedence)
if [[ -n "$OVERRIDE_IMAGE_PATH" ]]; then
    IMAGE_PATH="$FLAG_IMAGE_PATH"
fi
if [[ -n "$OVERRIDE_STORAGE_PATH" ]]; then
    STORAGE_PATH="$FLAG_STORAGE_PATH"
fi

# --- FIRST RUN OR RECONFIG ---
if [[ ! -f "$CONFIG_FILE" ]] || [[ "$RECONFIG" == true ]]; then
    configure_paths
fi

# --- VALIDATION ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name> [disk_size] [options]"
    echo "Run '$0 --help' for more information."
    exit 1
fi

# --- CREATE DIRECTORIES ---
if needs_privilege "$IMAGE_PATH"; then
    $PRIV_CMD mkdir -p "$IMAGE_PATH" 2>/dev/null || true
else
    mkdir -p "$IMAGE_PATH" 2>/dev/null || true
fi

if needs_privilege "$STORAGE_PATH"; then
    $PRIV_CMD mkdir -p "$STORAGE_PATH" 2>/dev/null || true
else
    mkdir -p "$STORAGE_PATH" 2>/dev/null || true
fi

# Disk Space Check
echo "🔍 Checking disk space..."
STORAGE_DIR=$(df "$STORAGE_PATH" | tail -1 | awk '{print $4}')  # Available space in KB
REQUIRED_SPACE=20971520  # 20GB in KB (approx)

if [[ $STORAGE_DIR -lt $REQUIRED_SPACE ]]; then
    echo "   ⚠️  Low disk space in $STORAGE_PATH"
    echo "   Available: $((STORAGE_DIR / 1024 / 1024))GB"
    echo "   Recommended: 20GB+ for comfortable use"
else
    echo "   ✅ Sufficient disk space: $((STORAGE_DIR / 1024 / 1024))GB available"
fi

# --- PRIVILEGE DETECTION ---
check_permissions() {
    local missing_groups=()
    local needed_groups=("libvirt")
    
    # Check for libvirt group
    if ! id -nG "$USER" | grep -qw "libvirt"; then
        missing_groups+=("libvirt")
    fi
    
    if [[ ${#missing_groups[@]} -gt 0 ]]; then
        echo ""
        echo "⚠️  Missing group membership: ${missing_groups[*]}"
        echo ""
        echo "   This is needed for full libvirt access without sudo."
        echo ""
        read -p "   Add user '$USER' to libvirt group now? [Y/n] " add_group
        add_group="${add_group:-Y}"
        
        if [[ "${add_group,,}" == "y" ]]; then
            echo "   Adding user to libvirt group..."
            if command -v doas &>/dev/null; then
                doas usermod -aG libvirt "$USER" 2>/dev/null && echo "   ✅ Added to libvirt group" || echo "   ❌ Failed to add group"
            elif command -v sudo &>/dev/null; then
                sudo usermod -aG libvirt "$USER" 2>/dev/null && echo "   ✅ Added to libvirt group" || echo "   ❌ Failed to add group"
            else
                echo "   ❌ No privilege escalation tool found"
                echo "   Run manually: sudo usermod -aG libvirt $USER"
            fi
        else
            echo "   Skipping. Using privilege escalation (sudo/doas) instead."
        fi
        echo ""
        echo "   Note: Group changes require logging out and back in to take effect."
        echo "   For this session, the script will use sudo/doas."
    fi
    
    # Check libvirt socket
    if [[ -S /var/run/libvirt/libvirt-sock ]]; then
        if [[ -r /var/run/libvirt/libvirt-sock ]]; then
            return 0
        else
            echo ""
            echo "⚠️  Cannot access libvirt socket."
            echo "   Your user needs libvirt group membership."
            return 1
        fi
    fi
}

check_permissions

if id -nG "$USER" | grep -qw "libvirt"; then
    PRIV_CMD=""
else
    if command -v doas &> /dev/null; then
        PRIV_CMD="doas"
    elif command -v sudo &> /dev/null; then
        PRIV_CMD="sudo"
    else
        echo "❌ Critical: No privilege escalation tool found."
        exit 1
    fi
fi

# --- PRE-FLIGHT CHECKS ---

# 1. KVM Hardware Acceleration Check
echo "🔍 Checking KVM support..."
if lsmod | grep -q "^kvm"; then
    echo "   ✅ KVM kernel module loaded"
elif [[ -e /dev/kvm ]]; then
    echo "   ✅ KVM device available (/dev/kvm)"
else
    echo "   ⚠️  KVM not detected. Virtualization may be disabled in BIOS/UEFI."
    echo "   Enable virtualization in BIOS/UEFI settings."
fi

# 1b. Nested Virtualization Check (optional)
echo "🔍 Checking nested virtualization..."
if [[ -e /sys/module/kvm_intel/parameters/nested ]] 2>/dev/null; then
    NESTED=$(cat /sys/module/kvm_intel/parameters/nested 2>/dev/null)
    if [[ "$NESTED" == "Y" || "$NESTED" == "1" ]]; then
        echo "   ✅ Nested virtualization enabled"
    else
        echo "   ℹ️  Nested virtualization disabled (VMs inside VMs won't work)"
    fi
elif [[ -e /sys/module/kvm_amd/parameters/nested ]] 2>/dev/null; then
    NESTED=$(cat /sys/module/kvm_amd/parameters/nested 2>/dev/null)
    if [[ "$NESTED" == "Y" || "$NESTED" == "1" ]]; then
        echo "   ✅ Nested virtualization enabled"
    else
        echo "   ℹ️  Nested virtualization disabled (VMs inside VMs won't work)"
    fi
else
    echo "   ℹ️  Nested virtualization status unknown"
fi

# 2. Libvirt Service Check
echo "🔍 Checking libvirt service..."
if $PRIV_CMD virsh list &>/dev/null; then
    echo "   ✅ Libvirt daemon is running"
else
    echo "   ⚠️  Libvirt daemon not running or no permission"
    echo "   Trying to start libvirtd..."
    if $PRIV_CMD systemctl start libvirtd 2>/dev/null; then
        echo "   ✅ Started libvirtd"
    else
        echo "   ⚠️  Could not start libvirtd. Try: $PRIV_CMD systemctl start libvirtd"
    fi
fi

# 3. Default Network Check
echo "🔍 Checking default network..."
if $PRIV_CMD virsh net-info default &>/dev/null; then
    NET_STATE=$($PRIV_CMD virsh net-info default 2>/dev/null | grep "Active:" | awk '{print $2}')
    if [[ "${NET_STATE,,}" == "yes" ]]; then
        echo "   ✅ Default network is active"
    else
        echo "   ⚠️  Default network exists but not active"
        echo "   Attempting to start..."
        if $PRIV_CMD virsh net-start default 2>&1; then
            echo "   ✅ Started default network"
        else
            echo "   ⚠️  Could not start default network"
            echo "   Error: $($PRIV_CMD virsh net-start default 2>&1 || true)"
            echo "   Try: sudo virsh net-start default"
        fi
    fi
else
    echo "   ⚠️  Default network not defined"
    echo "   Creating default network..."
    
    # Create temporary XML file for network definition
    NET_XML=$(mktemp)
    cat > "$NET_XML" <<'EOF'
<network>
  <name>default</name>
  <forward mode='nat'/>
  <bridge name='virbr0' stp='on' delay='0'/>
  <ip address='192.168.122.1' netmask='255.255.255.0'>
    <dhcp>
      <range start='192.168.122.2' end='192.168.122.254'/>
    </dhcp>
  </ip>
</network>
EOF
    
    if $PRIV_CMD virsh net-define "$NET_XML" 2>/dev/null; then
        rm -f "$NET_XML"
        if $PRIV_CMD virsh net-start default 2>/dev/null; then
            $PRIV_CMD virsh net-autostart default 2>/dev/null
            echo "   ✅ Created, started, and enabled autostart for default network"
        else
            echo "   ⚠️  Created network but could not start it"
        fi
    else
        rm -f "$NET_XML"
        echo "   ❌ Could not create default network"
        echo "   Try manually: sudo virsh net-define <xml-file> && sudo virsh net-start default"
    fi
fi

# 4. Storage Pool Check
echo "🔍 Checking storage pool..."
if $PRIV_CMD virsh pool-info default &>/dev/null; then
    POOL_STATE=$($PRIV_CMD virsh pool-info default 2>/dev/null | grep "State:" | awk '{print $2}')
    if [[ "${POOL_STATE,,}" == "running" ]]; then
        echo "   ✅ Default storage pool is active"
    else
        echo "   ⚠️  Storage pool exists but not running"
        read -p "   Start storage pool now? [Y/n] " start_pool
        start_pool="${start_pool:-Y}"
        if [[ "${start_pool,,}" == "y" ]]; then
            if $PRIV_CMD virsh pool-start default 2>/dev/null; then
                echo "   ✅ Started storage pool"
            else
                echo "   ⚠️  Could not start storage pool"
            fi
        fi
    fi
else
    echo "   ℹ️  Default storage pool not configured (using custom path)"
fi

# 5. SSH Key Check & Generation
echo "🔍 Checking SSH key..."
if [[ ! -f "$SSH_PUB_KEY" ]]; then
    echo "   ⚠️  SSH key not found at $SSH_PUB_KEY"
    echo ""
    read -p "   Generate SSH key now? [Y/n] " gen_key
    gen_key="${gen_key:-Y}"
    if [[ "${gen_key,,}" == "y" ]]; then
        echo "   Generating SSH key..."
        ssh-keygen -t ed25519 -f "$SSH_PRIV_KEY" -N "" -C "$USER@$(hostname)"
        chmod 600 "$SSH_PRIV_KEY"
        chmod 644 "$SSH_PUB_KEY"
        echo "   ✅ SSH key generated at $SSH_PUB_KEY"
    else
        echo "   ❌ Cannot proceed without SSH key"
        exit 1
    fi
else
    echo "   ✅ SSH key found at $SSH_PUB_KEY"
fi

# --- DEPENDENCY CHECK ---
check_dependencies

# --- SMART CONFIRMATION ---
CONFIRM_NEEDED=false

if [[ -n "$OVERRIDE_IMAGE_PATH" || -n "$OVERRIDE_STORAGE_PATH" ]]; then
    CONFIRM_NEEDED=true
    echo ""
    echo "⚠️  Using override paths:"
    [[ -n "$OVERRIDE_IMAGE_PATH" ]] && echo "   Images:  $IMAGE_PATH (override)"
    [[ -n "$OVERRIDE_STORAGE_PATH" ]] && echo "   Storage: $STORAGE_PATH (override)"
fi

# Check if VM already exists (will be destroyed)
if $PRIV_CMD virsh dominfo "$VM_NAME" &>/dev/null; then
    CONFIRM_NEEDED=true
    echo ""
    echo "⚠️  VM '$VM_NAME' already exists - will be DESTROYED and recreated!"
fi

if [[ "$CONFIRM_NEEDED" == true ]]; then
    echo ""
    read -p "   Continue? [Y/n] " confirm
    if [[ "${confirm,,}" == "n" ]]; then
        echo "Aborted."
        exit 0
    fi
fi

# --- DYNAMIC IMAGE SELECTOR ---
echo ""
echo "📂 Scanning $IMAGE_PATH for base images..."

scan_cmd="find"
scan_args=("$IMAGE_PATH" -maxdepth 1 -type f \( -name "*.qcow2" -o -name "*.img" \) -printf "%f\n")

if needs_privilege "$IMAGE_PATH"; then
    mapfile -t AVAILABLE_IMAGES < <($PRIV_CMD find "${scan_args[@]}" 2>/dev/null | sort)
else
    mapfile -t AVAILABLE_IMAGES < <(find "${scan_args[@]}" 2>/dev/null | sort)
fi


IMAGE_SELECTED=false

PS3="👉 Select Base Image (number): "
select choice in "${AVAILABLE_IMAGES[@]}" "📥 Download new image"; do
    if [[ "$choice" == "📥 Download new image" ]]; then
        echo ""
        echo "📥 Download Cloud Image"
        echo ""
        echo "Popular Images:"
        local_i=1
        for key in "${IMAGE_KEYS[@]}"; do
            echo "  $local_i) ${IMAGE_NAMES[$key]}"
            ((local_i++))
        done
        echo "  $local_i) Enter custom URL"
        echo ""
        
        read -p "👉 Select image to download: " dl_choice
        
        if [[ "$dl_choice" -eq "${#IMAGE_KEYS[@]}+1" ]] 2>/dev/null; then
            read -p "   Enter cloud image URL: " custom_url
            if [[ -n "$custom_url" ]]; then
                download_image "$custom_url"
            else
                echo "❌ No URL provided"
                exit 1
            fi
        elif [[ "$dl_choice" -ge 1 && "$dl_choice" -le "${#IMAGE_KEYS[@]}" ]] 2>/dev/null; then
            key="${IMAGE_KEYS[$((dl_choice-1))]}"
            download_image "${IMAGE_URLS[$key]}"
        else
            echo "❌ Invalid selection"
            exit 1
        fi
        
        if needs_privilege "$IMAGE_PATH"; then
            mapfile -t AVAILABLE_IMAGES < <($PRIV_CMD find "${scan_args[@]}" 2>/dev/null | sort)
        else
            mapfile -t AVAILABLE_IMAGES < <(find "${scan_args[@]}" 2>/dev/null | sort)
        fi
        
        if [[ ${#AVAILABLE_IMAGES[@]} -eq 0 ]]; then
            echo "❌ No images found after download"
            exit 1
        fi
        
        echo ""
        echo "Available Images:"
        i=1
        for img in "${AVAILABLE_IMAGES[@]}"; do
            echo "  $i) $img"
            ((i++))
        done
        echo ""
        
        PS3="👉 Select Base Image (number): "
        select img in "${AVAILABLE_IMAGES[@]}"; do
            if [[ -n "$img" ]]; then
                BASE_IMG_NAME="$img"
                IMAGE_SELECTED=true
                break
            else
                echo "❌ Invalid selection."
            fi
        done
        
        if [[ "$IMAGE_SELECTED" == true ]]; then
            break
        fi
    elif [[ -n "$choice" ]]; then
        BASE_IMG_NAME="$choice"
        break
    else
        echo "❌ Invalid selection."
    fi
done

BASE_IMG="$IMAGE_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$STORAGE_PATH/${VM_NAME}.qcow2"

# --- SMART DETECTION ---
echo ""
echo "🔍 Inspecting image..."
if needs_privilege "$IMAGE_PATH"; then
    BACKING_FMT=$($PRIV_CMD qemu-img info "$BASE_IMG" | grep "file format" | cut -d: -f2 | xargs)
else
    BACKING_FMT=$(qemu-img info "$BASE_IMG" | grep "file format" | cut -d: -f2 | xargs)
fi
LOWER_NAME="${BASE_IMG_NAME,,}"

case "$LOWER_NAME" in
    *rocky*10*)   OS_VARIANT="rocky10" ;;
    *rocky*9*)   OS_VARIANT="rocky9" ;;
    *rocky*8*)   OS_VARIANT="rocky8" ;;
    *noble*)     OS_VARIANT="ubuntunoble" ;;
    *jammy*)     OS_VARIANT="ubuntujammy" ;;
    *focal*)     OS_VARIANT="ubuntufocal" ;;
    *debian*12*) OS_VARIANT="debian12" ;;
    *debian*13*) OS_VARIANT="debian13" ;;
    *fedora*)    OS_VARIANT="fedora-unknown" ;;
    *)           OS_VARIANT="generic" ;;
esac

echo "   Format: $BACKING_FMT"
echo "   OS:     $OS_VARIANT"

# --- SECURE WORKSPACE ---
WORK_DIR=$(mktemp -d -t kdeploy-XXXXXXXX)
CLOUD_INIT_DIR="$WORK_DIR/config"
SEED_ISO="$WORK_DIR/seed.iso"
mkdir -p "$CLOUD_INIT_DIR"
trap 'rm -rf "$WORK_DIR"' EXIT

# --- PASSWORD SETUP ---
echo ""
echo "🔑 Setup VM Password (Enter for default: '$DEFAULT_PASS')"
read -s -p "   Password: " INPUT_PASS
echo ""
VM_PASSWORD="${INPUT_PASS:-$DEFAULT_PASS}"
PASSWORD_HASH=$(openssl passwd -6 "$VM_PASSWORD")

# --- VM RESOURCES ---
echo ""
echo "⚙️  VM Resources (Enter for defaults):"
echo "   Disk:      $VM_SIZE"

if [[ -z "$OVERRIDE_RAM" ]]; then
    read -p "   RAM (MB) [${VM_RAM}]: " input_ram
    VM_RAM="${input_ram:-$VM_RAM}"
else
    echo "   RAM:       $VM_RAM (flag)"
fi

if [[ -z "$OVERRIDE_CPUS" ]]; then
    read -p "   vCPUs    [${VM_CPUS}]: " input_cpu
    VM_CPUS="${input_cpu:-$VM_CPUS}"
else
    echo "   vCPUs:     $VM_CPUS (flag)"
fi

# --- PACKAGE SELECTION ---
echo ""
echo "📦 Package Selection"
echo "   [1] None (default - fastest, cloud images usually have qemu-guest-agent)"
echo "   [2] Custom packages"
echo ""
read -p "   Choice [1]: " pkg_choice
pkg_choice="${pkg_choice:-1}"

case "$pkg_choice" in
    1)  # None (default, fastest)
        PACKAGES=()
        ;;
    2)  # Custom
        echo ""
        read -p "   Enter packages (space-separated): " custom_pkgs
        PACKAGES=($custom_pkgs)
        ;;
    *)
        echo "   Invalid choice, using default (none)"
        PACKAGES=()
        ;;
esac

# --- CLEANUP OLD VM ---
echo ""
echo "🧹 Cleaning up old resources..."
$PRIV_CMD virsh destroy "$VM_NAME" 2>/dev/null || true
$PRIV_CMD virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true

if needs_privilege "$STORAGE_PATH"; then
    $PRIV_CMD rm -f "$OVERLAY_IMG" "$STORAGE_PATH/${VM_NAME}-seed.iso"
else
    rm -f "$OVERLAY_IMG" "$STORAGE_PATH/${VM_NAME}-seed.iso"
fi

# --- CLOUD-INIT ---
echo "🔧 Generating cloud-init..."

# Build cloud-init content dynamically
CLOUDINIT_CONTENT="#cloud-config
disable_root: false
users:
  - name: root
    ssh_authorized_keys:
      - $(cat "$SSH_PUB_KEY")
  - name: $VM_USER
    groups: [ wheel ]
    sudo: [ \"ALL=(ALL) NOPASSWD:ALL\" ]
    lock_passwd: false
    passwd: $PASSWORD_HASH
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$SSH_PUB_KEY")
ssh_pwauth: false
preserve_hostname: false
hostname: ${VM_NAME}
"

# Add packages if any
if [[ ${#PACKAGES[@]} -gt 0 ]]; then
    CLOUDINIT_CONTENT+="packages:\n"
    for pkg in "${PACKAGES[@]}"; do
        CLOUDINIT_CONTENT+="  - $pkg\n"
    done
fi

# Add qemu-guest-agent service if installed
if [[ " ${PACKAGES[*]} " =~ "qemu-guest-agent" ]]; then
    CLOUDINIT_CONTENT+="runcmd:\n"
    CLOUDINIT_CONTENT+="  - systemctl enable --now qemu-guest-agent\n"
fi

echo -e "$CLOUDINIT_CONTENT" > "$CLOUD_INIT_DIR/user-data"

cat > "$CLOUD_INIT_DIR/meta-data" <<EOF
instance-id: ${VM_NAME}-$(date +%s)
local-hostname: ${VM_NAME}
EOF

# --- SEED ISO & OVERLAY ---
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data" > /dev/null
FINAL_ISO="$STORAGE_PATH/${VM_NAME}-seed.iso"

if needs_privilege "$STORAGE_PATH"; then
    $PRIV_CMD mv "$SEED_ISO" "$FINAL_ISO"
    $PRIV_CMD chmod 644 "$FINAL_ISO"
else
    mv "$SEED_ISO" "$FINAL_ISO"
    chmod 644 "$FINAL_ISO"
fi

echo ""
echo "💾 Creating overlay image ($VM_SIZE)..."
if needs_privilege "$STORAGE_PATH"; then
    $PRIV_CMD qemu-img create -f qcow2 -b "$BASE_IMG" -F "$BACKING_FMT" "$OVERLAY_IMG" "$VM_SIZE" > /dev/null
    $PRIV_CMD chmod 644 "$OVERLAY_IMG"
else
    qemu-img create -f qcow2 -b "$BASE_IMG" -F "$BACKING_FMT" "$OVERLAY_IMG" "$VM_SIZE" > /dev/null
    chmod 644 "$OVERLAY_IMG"
fi

# --- DEPLOY ---
echo ""
echo "🚀 Booting VM: $VM_NAME ($OS_VARIANT)"
$PRIV_CMD virt-install \
  --name "$VM_NAME" \
  --memory "$VM_RAM" \
  --vcpus "$VM_CPUS" \
  --disk path="$OVERLAY_IMG",format=qcow2,bus=virtio \
  --disk path="$FINAL_ISO",device=cdrom \
  --os-variant "$OS_VARIANT" \
  --graphics vnc,listen=127.0.0.1 \
  --console pty,target_type=serial \
  --network network=default \
  --import \
  --noautoconsole

# --- ROBUST IP DISCOVERY ---
echo ""
echo "⏳ Waiting for IP address..."
IP=""
count=0
MAC=$($PRIV_CMD virsh dumpxml "$VM_NAME" | grep "mac address" | head -1 | cut -d"'" -f2)

while [ -z "$IP" ]; do
    if [ $count -gt 45 ]; then
        echo -e "\n⚠️  Timed out. Check console: virsh console $VM_NAME"
        exit 1
    fi
    
    IP=$($PRIV_CMD virsh domifaddr "$VM_NAME" --source agent 2>/dev/null \
        | grep "ipv4" \
        | grep -v "127.0.0.1" \
        | awk '{print $4}' \
        | cut -d/ -f1 \
        | head -n1)
    
    if [ -z "$IP" ]; then
        IP=$($PRIV_CMD virsh net-dhcp-leases default \
            | grep "$MAC" \
            | awk '{print $5}' \
            | cut -d/ -f1 \
            | head -n1)
    fi

    if [ -z "$IP" ]; then sleep 2; count=$((count + 1)); printf "."; fi
done

ssh-keygen -R "$IP" 2>/dev/null

# --- SAFER SSH CONFIG ---
echo -e "\n📝 Managing SSH Config..."

mkdir -p "$SSH_INCLUDE_DIR"
chmod 700 "$SSH_INCLUDE_DIR"

if [ ! -f "$SSH_CONFIG" ]; then
    touch "$SSH_CONFIG"
    echo "Include $SSH_INCLUDE_DIR/*" > "$SSH_CONFIG"
elif ! grep -q "Include $SSH_INCLUDE_DIR/\*" "$SSH_CONFIG"; then
    echo "Include $SSH_INCLUDE_DIR/*" | cat - "$SSH_CONFIG" > temp && mv temp "$SSH_CONFIG"
    echo "   -> Added Include directive to $SSH_CONFIG"
fi

cat > "$SSH_INCLUDE_DIR/${VM_NAME}.conf" <<EOF
Host $VM_NAME $VM_NAME-$VM_USER
    HostName $IP
    User $VM_USER
    IdentityFile $SSH_PRIV_KEY
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null

Host ${VM_NAME}-root
    HostName $IP
    User root
    IdentityFile $SSH_PRIV_KEY
    StrictHostKeyChecking no
    UserKnownHostsFile /dev/null
EOF

chmod 600 "$SSH_INCLUDE_DIR/${VM_NAME}.conf"

echo ""
echo "✅ VM Ready!"
echo "-------------------------------------"
echo "🌐 IP:      $IP"
echo "👤 User:    ssh $VM_NAME"
echo "👑 Root:    ssh $VM_NAME-root"
echo "-------------------------------------"
