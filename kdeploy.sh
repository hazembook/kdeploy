#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer v2.0
# Author: Hazem Shaban (Refactored by Scott Champine)
# Usage: ./kdeploy.sh <vm-name>
# ==============================================================================

set -e

VM_NAME="$1"

# --- CONFIGURATION ---
DEFAULT_PASS="linux"
# Rocky 9.3 Cloud Image URL (Update as needed)
IMG_URL="https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
BASE_IMG_NAME=$(basename "$IMG_URL")
LIBVIRT_PATH="/var/lib/libvirt/images"
VM_USER="${USER}" # Use current user, not sudo_user, assuming group membership
SSH_KEY_PATH="$HOME/.ssh/id_rsa.pub"
TIMEOUT_SECONDS=60
# ---------------------

BASE_IMG="$LIBVIRT_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$LIBVIRT_PATH/${VM_NAME}.qcow2"

# --- 1. Pre-Flight Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

# Check for required tools
for cmd in virsh virt-install cloud-localds qemu-img openssl curl; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ Critical dependency missing: '$cmd'"
        exit 1
    fi
done

# Check SSH Key
if [[ ! -f "$SSH_KEY_PATH" ]]; then
    echo "❌ SSH key not found at $SSH_KEY_PATH. Generate one with 'ssh-keygen'."
    exit 1
fi

# Check Libvirt Group Membership (To avoid sudo overuse)
if ! groups "$USER" | grep -q "libvirt"; then
    echo "⚠️  Warning: User '$USER' is not in the 'libvirt' group."
    echo "   You may need to run: sudo usermod -aG libvirt $USER"
    echo "   (Log out and back in for this to take effect)."
    # We continue, but expect password prompts
fi

# --- 2. Image Management ---
if [[ ! -f "$BASE_IMG" ]]; then
    echo "⬇️  Base image not found. Downloading Rocky Linux 9..."
    # We use sudo here because /var/lib/libvirt/images is usually root-owned
    sudo curl -L "$IMG_URL" -o "$BASE_IMG"
    sudo chown root:kvm "$BASE_IMG"
    sudo chmod 644 "$BASE_IMG" # Readable by KVM
    echo "✅ Download complete."
else
    echo "✅ Base image found."
fi

# --- 3. Secure Workspace ---
WORK_DIR=$(mktemp -d -t kdeploy-XXXXXXXX)
CLOUD_INIT_DIR="$WORK_DIR/config"
SEED_ISO="$WORK_DIR/seed.iso"
mkdir -p "$CLOUD_INIT_DIR"

trap 'rm -rf "$WORK_DIR"' EXIT

# --- 4. Credentials ---
if [[ -z "$VM_PASSWORD" ]]; then 
    # Only prompt if variable not already set (allows for automation)
    echo "🔑 Setup VM Password (Enter for default: '$DEFAULT_PASS')"
    read -s -p "   Password: " INPUT_PASS
    echo ""
    VM_PASSWORD="${INPUT_PASS:-$DEFAULT_PASS}"
fi
PASSWORD_HASH=$(openssl passwd -6 "$VM_PASSWORD")

# --- 5. Cleanup & Preparation ---
echo "🧹 Pre-cleaning..."
# Redirect stderr to devnull to keep it clean if VM doesn't exist
virsh destroy "$VM_NAME" &>/dev/null || true
virsh undefine "$VM_NAME" --remove-all-storage &>/dev/null || true
sudo rm -f "$OVERLAY_IMG" "$LIBVIRT_PATH/${VM_NAME}-seed.iso"

# --- 6. Cloud-Init Gen ---
echo "🔧 Generating Cloud-Init..."

cat > "$CLOUD_INIT_DIR/meta-data" <<EOF
instance-id: ${VM_NAME}-$(date +%s)
local-hostname: ${VM_NAME}
EOF

cat > "$CLOUD_INIT_DIR/user-data" <<EOF
#cloud-config
disable_root: false
timezone: Africa/Tripoli

users:
  - name: $VM_USER
    groups: [ wheel ]
    sudo: [ "ALL=(ALL) NOPASSWD:ALL" ]
    lock_passwd: false
    passwd: $PASSWORD_HASH
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$SSH_KEY_PATH")

packages:
  - qemu-guest-agent
  - bash-completion
  - vim
  - git

runcmd:
  - systemctl enable --now qemu-guest-agent
  - sed -i 's/^PermitRootLogin.*/PermitRootLogin prohibit-password/g' /etc/ssh/sshd_config
  - systemctl restart sshd

ssh_pwauth: false
hostname: ${VM_NAME}
EOF

# --- 7. Disk & ISO Creation ---
echo "💿 Building disks..."
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data"

FINAL_ISO="$LIBVIRT_PATH/${VM_NAME}-seed.iso"
sudo mv "$SEED_ISO" "$FINAL_ISO"

# Create Overlay (CoW) - Keeps base image clean
sudo qemu-img create -f qcow2 -b "$BASE_IMG" -F qcow2 "$OVERLAY_IMG" 20G

# Permissions Fix (Critical for non-root KVM/QEMU)
sudo chown libvirt-qemu:kvm "$OVERLAY_IMG" "$FINAL_ISO" 2>/dev/null || \
    sudo chown nobody:kvm "$OVERLAY_IMG" "$FINAL_ISO"
sudo chmod 660 "$OVERLAY_IMG"
sudo chmod 644 "$FINAL_ISO"

# --- 8. Deployment ---
echo "🚀 Launching $VM_NAME..."
virt-install \
  --name "$VM_NAME" \
  --memory 2048 \
  --vcpus 2 \
  --disk path="$OVERLAY_IMG",format=qcow2,bus=virtio \
  --disk path="$FINAL_ISO",device=cdrom \
  --os-variant rocky9 \
  --graphics none \
  --network network=default \
  --import \
  --noautoconsole

# --- 9. Connection Wait Loop ---
echo "⏳ Waiting for IP (Timeout: ${TIMEOUT_SECONDS}s)..."
START_TIME=$(date +%s)
IP=""

while [ -z "$IP" ]; do
    CURRENT_TIME=$(date +%s)
    ELAPSED=$((CURRENT_TIME - START_TIME))
    
    if [ $ELAPSED -gt $TIMEOUT_SECONDS ]; then
        echo -e "\n⚠️  Timed out. Check console via: virsh console $VM_NAME"
        exit 1
    fi
    
    IP=$(virsh domifaddr "$VM_NAME" 2>/dev/null | awk '/ipv4/ {print $4}' | cut -d/ -f1 || true)
    
    if [ -z "$IP" ]; then
        sleep 2
        printf "."
    fi
done

echo -e "\n\n✅ VM Ready!"
echo "   ssh $VM_USER@$IP"
