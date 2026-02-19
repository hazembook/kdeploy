#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer (Home Directory Edition)
# Usage: ./kdeploy.sh <vm-name>
# Source: ~/VM/qcow2_images
# Dest:   ~/VM/disks
# ==============================================================================

set -e

VM_NAME="$1"

# --- CONFIGURATION ---
DEFAULT_PASS="linux"

# Your custom directory structure
ISO_LIB_PATH="$HOME/VM/cloud_images"
VM_STORAGE_PATH="$HOME/VM/disks"

# ---------------------

VM_USER="${SUDO_USER:-$USER}"
SSH_KEY_PATH="$HOME/.ssh/id_rsa.pub"

# --- 1. Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

# Ensure directories exist
if [[ ! -d "$ISO_LIB_PATH" ]]; then
    echo "❌ Base Image Directory not found: $ISO_LIB_PATH"
    echo "   Please move your base images there."
    exit 1
fi
mkdir -p "$VM_STORAGE_PATH"

for cmd in virsh virt-install cloud-localds qemu-img openssl; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ Critical dependency missing: '$cmd'"
        exit 1
    fi
done

if [[ ! -f "$SSH_KEY_PATH" ]]; then
    echo "❌ SSH key not found at $SSH_KEY_PATH"
    exit 1
fi

# --- 2. Dynamic Image Selector ---
echo "📂 Scanning $ISO_LIB_PATH for base images..."

# Find .qcow2 and .img files
mapfile -t AVAILABLE_IMAGES < <(find "$ISO_LIB_PATH" -maxdepth 1 -type f \( -name "*.qcow2" -o -name "*.img" \) -printf "%f\n" | sort)

if [ ${#AVAILABLE_IMAGES[@]} -eq 0 ]; then
    echo "❌ No images found in $ISO_LIB_PATH"
    exit 1
fi

PS3="👉 Select Base Image (number): "
BASE_IMG_NAME=""
select img in "${AVAILABLE_IMAGES[@]}"; do
    if [[ -n "$img" ]]; then
        BASE_IMG_NAME="$img"
        break
    else
        echo "❌ Invalid selection."
    fi
done

BASE_IMG="$ISO_LIB_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$VM_STORAGE_PATH/${VM_NAME}.qcow2"

# --- 3. Smart Detection (Format & OS) ---
echo "🔍 Inspecting image..."

# Detect Backing File Format
BACKING_FMT=$(qemu-img info "$BASE_IMG" | grep "file format" | cut -d: -f2 | xargs)
echo "   -> Backing format: $BACKING_FMT"

# Detect OS Variant
LOWER_NAME="${BASE_IMG_NAME,,}"
case "$LOWER_NAME" in
    *rocky*10*)  OS_VARIANT="rocky10" ;;
    *rocky*9*)   OS_VARIANT="rocky9" ;;
    *rocky*8*)   OS_VARIANT="rocky8" ;;
    *plucky*)    OS_VARIANT="ubuntuplucky" ;;
    *oracular*)  OS_VARIANT="ubuntuoracular" ;;
    *noble*)     OS_VARIANT="ubuntunoble" ;;
    *mantic*)    OS_VARIANT="ubuntumantic" ;;
    *jammy*)     OS_VARIANT="ubuntujammy" ;;
    *focal*)     OS_VARIANT="ubuntufocal" ;;
    *debian*13*) OS_VARIANT="debian13" ;;
    *debian*12*) OS_VARIANT="debian12" ;;
    *debian*11*) OS_VARIANT="debian11" ;;
    *ubuntu*)    OS_VARIANT="ubuntu-lts-latest" ;;
    *debian*)    OS_VARIANT="debian12" ;;
    *fedora*)    OS_VARIANT="fedora-latest" ;;
    *centos*)    OS_VARIANT="centos-stream9" ;;
    *arch*)      OS_VARIANT="archlinux" ;;
    *)           OS_VARIANT="generic" ;;
esac
echo "   -> OS Variant:     $OS_VARIANT"

# --- 4. Secure Workspace ---
WORK_DIR=$(mktemp -d -t kdeploy-XXXXXXXX)
CLOUD_INIT_DIR="$WORK_DIR/config"
SEED_ISO="$WORK_DIR/seed.iso"
mkdir -p "$CLOUD_INIT_DIR"

trap 'rm -rf "$WORK_DIR"' EXIT

# --- 5. Password Setup ---
echo "🔑 Setup VM Password"
echo "   (Leave empty for default: '$DEFAULT_PASS')"
read -s -p "   Enter Password: " INPUT_PASS
echo ""
VM_PASSWORD="${INPUT_PASS:-$DEFAULT_PASS}"
PASSWORD_HASH=$(openssl passwd -6 "$VM_PASSWORD")

# --- 6. Cleanup Old VM Resources ---
echo "🧹 Cleaning up old resources..."
virsh destroy "$VM_NAME" 2>/dev/null || true
virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
rm -f "$OVERLAY_IMG" "$VM_STORAGE_PATH/${VM_NAME}-seed.iso"

# --- 7. Cloud-Init Generation ---
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
      - $(cat "$SSH_KEY_PATH")
  - name: $VM_USER
    groups: [ wheel ]
    sudo: [ "ALL=(ALL) NOPASSWD:ALL" ]
    lock_passwd: false
    passwd: $PASSWORD_HASH
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$SSH_KEY_PATH")
ssh_pwauth: false
preserve_hostname: false
hostname: ${VM_NAME}
EOF

# --- 8. Create Seed ISO ---
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data" > /dev/null
FINAL_ISO="$VM_STORAGE_PATH/${VM_NAME}-seed.iso"
mv "$SEED_ISO" "$FINAL_ISO"

# --- 9. Create Overlay Disk ---
echo "💾 Creating overlay image..."
qemu-img create -f qcow2 -b "$BASE_IMG" -F "$BACKING_FMT" "$OVERLAY_IMG" 20G > /dev/null

# --- 10. CRITICAL: Permissions ---
# Since this is in HOME, we must ensure 'others' (QEMU user) can read the specific files
# Your folders (disks/qcow2_images) are already o+rx, so we just need o+r on files.
chmod 644 "$OVERLAY_IMG" "$FINAL_ISO"

# --- 11. Deploy VM ---
echo "🚀 Booting VM: $VM_NAME ($OS_VARIANT)"
sudo virt-install \
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

# --- 12. IP Discovery ---
echo "⏳ Waiting for IP address..."
sleep 5
IP=""
count=0
while [ -z "$IP" ]; do
    if [ $count -gt 45 ]; then
        echo -e "\n⚠️  Timed out waiting for IP."
        echo "   Check console: virsh console $VM_NAME"
        exit 1
    fi
    
    IP=$(virsh domifaddr "$VM_NAME" 2>/dev/null | awk '/ipv4/ {print $4}' | cut -d/ -f1 || true)
    
    if [ -z "$IP" ]; then
        sleep 2
        count=$((count + 1))
        printf "."
    fi
done

echo -e "\n✅ VM Ready!"
echo "🌐 IP:      $IP"
echo "👤 User:    $VM_USER"
echo "🔐 SSH:     ssh $VM_USER@$IP"
echo "💻 Console: virsh console $VM_NAME"
