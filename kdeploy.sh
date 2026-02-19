#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer v3.0
# "The Professor's Edition" - Configurable Paths, Image Downloads, Robust Deploy
# ==============================================================================

set -e

# --- CONFIG CONSTANTS ---
CONFIG_FILE="$HOME/.config/kdeploy.conf"
DEFAULT_IMAGE_PATH="$HOME/VM/cloud_images"
DEFAULT_STORAGE_PATH="/var/lib/libvirt/images"
DEFAULT_VM_SIZE="20G"
DEFAULT_PASS="linux"

# --- IMAGE CATALOG FOR DOWNLOADS ---
declare -A IMAGE_URLS=(
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
    ["rocky9"]="Rocky Linux 9 (latest)"
    ["rocky8"]="Rocky Linux 8 (latest)"
    ["ubuntu2404"]="Ubuntu 24.04 LTS (Noble)"
    ["ubuntu2204"]="Ubuntu 22.04 LTS (Jammy)"
    ["ubuntu2004"]="Ubuntu 20.04 LTS (Focal)"
    ["debian12"]="Debian 12 (Bookworm)"
    ["debian13"]="Debian 13 (Trixie)"
    ["fedora41"]="Fedora 41"
)

declare -a IMAGE_KEYS=("rocky9" "rocky8" "ubuntu2404" "ubuntu2204" "ubuntu2004" "debian12" "debian13" "fedora41")

# --- SSH CONSTANTS ---
SSH_DIR="$HOME/.ssh"
SSH_CONFIG="$SSH_DIR/config"
SSH_INCLUDE_DIR="$SSH_DIR/kdeploy.d"

# --- VARIABLES (will be set via config or flags) ---
IMAGE_PATH=""
STORAGE_PATH=""
VM_NAME=""
VM_SIZE=""
SKIP_CONFIRM=false

# ---------------------
# Handle if script is run via sudo/doas
VM_USER="${SUDO_USER:-${DOAS_USER:-$USER}}"
SSH_PUB_KEY="$HOME/.ssh/id_rsa.pub"
SSH_PRIV_KEY="${SSH_PUB_KEY%.pub}"
# ---------------------

# --- HELPER FUNCTIONS ---

show_help() {
    cat <<EOF
KVM Cloud Image Deployer v3.0

Usage: $0 <vm_name> [disk_size] [options]

Arguments:
    vm_name         Name of the virtual machine (required)
    disk_size       Size of VM disk (default: ${DEFAULT_VM_SIZE})

Options:
    -y, --yes              Skip confirmation prompts
    -i, --image-path DIR   Override image library path (one-time)
    -s, --storage-path DIR Override VM storage path (one-time)
    --reconfig             Reconfigure saved paths
    --show-config          Show current configuration
    -h, --help             Show this help message

Examples:
    $0 webserver                    Deploy VM with defaults
    $0 webserver 50G                Deploy VM with 50GB disk
    $0 webserver -y                 Deploy without confirmation
    $0 webserver -i /tmp/img        Use custom image path (one-time)
    $0 --reconfig                   Reconfigure saved paths

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
        -y|--yes)
            SKIP_CONFIRM=true
            shift
            ;;
        -i|--image-path)
            IMAGE_PATH="${2/#\~/$HOME}"
            shift 2
            ;;
        -s|--storage-path)
            STORAGE_PATH="${2/#\~/$HOME}"
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
load_config

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

# --- PRIVILEGE DETECTION ---
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

if [[ ! -f "$SSH_PUB_KEY" ]]; then
    echo "❌ SSH key not found at $SSH_PUB_KEY"
    exit 1
fi

# --- PATH CONFIRMATION ---
echo ""
echo "📂 Paths:"
echo "   Images:  $IMAGE_PATH"
echo "   Storage: $STORAGE_PATH"
echo ""

if [[ "$SKIP_CONFIRM" != true ]]; then
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

echo ""
echo "Available Images:"
i=1
for img in "${AVAILABLE_IMAGES[@]}"; do
    echo "  $i) $img"
    ((i++))
done
echo "  $i) 📥 Download new image"
echo ""

PS3="👉 Select Base Image (number): "
select choice in "${AVAILABLE_IMAGES[@]}" "DOWNLOAD"; do
    if [[ "$choice" == "DOWNLOAD" ]]; then
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
                break
            else
                echo "❌ Invalid selection."
            fi
        done
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
cat > "$CLOUD_INIT_DIR/meta-data" <<EOF
instance-id: ${VM_NAME}-$(date +%s)
local-hostname: ${VM_NAME}
EOF

cat > "$CLOUD_INIT_DIR/user-data" <<EOF
#cloud-config
disable_root: false
users:
  - name: root
    ssh_authorized_keys:
      - $(cat "$SSH_PUB_KEY")
  - name: $VM_USER
    groups: [ wheel ]
    sudo: [ "ALL=(ALL) NOPASSWD:ALL" ]
    lock_passwd: false
    passwd: $PASSWORD_HASH
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$SSH_PUB_KEY")
ssh_pwauth: false
preserve_hostname: false
hostname: ${VM_NAME}
packages:
  - qemu-guest-agent
  - git
  - vim
  - htop
runcmd:
  - systemctl enable --now qemu-guest-agent
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
  --memory 2048 \
  --vcpus 2 \
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
