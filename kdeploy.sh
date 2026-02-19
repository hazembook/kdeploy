#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer with cloud-utils
# Usage: ./kdeploy.sh <vm-name>
# ==============================================================================

set -e

VM_NAME="$1"
BASE_IMG_NAME="Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
LIBVIRT_PATH="/var/lib/libvirt/images"
BASE_IMG="$LIBVIRT_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$LIBVIRT_PATH/${VM_NAME}.qcow2"
SEED_ISO="/tmp/${VM_NAME}-seed.iso"  # Create in /tmp first
CLOUD_INIT_DIR="/tmp/cloud-init-$VM_NAME"

# --- 1. Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

if [[ ! -f "$BASE_IMG" ]]; then
    echo "❌ Base image not found at: $BASE_IMG"
    exit 1
fi

if [[ ! -f "$HOME/.ssh/id_rsa.pub" ]]; then
    echo "❌ SSH key not found. Generate with: ssh-keygen -t rsa -b 4096"
    exit 1
fi

# --- 2. Cleanup ---
echo "🧹 Cleaning up old resources..."
virsh destroy "$VM_NAME" 2>/dev/null || true
virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
sudo rm -f "$OVERLAY_IMG" "$LIBVIRT_PATH/${VM_NAME}-seed.iso"

# --- 3. Cloud-Init Config ---
echo "🔧 Creating cloud-init configuration..."
rm -rf "$CLOUD_INIT_DIR"
mkdir -p "$CLOUD_INIT_DIR"

# Create meta-data
cat > "$CLOUD_INIT_DIR/meta-data" <<EOF
instance-id: ${VM_NAME}-$(date +%s)
local-hostname: ${VM_NAME}
EOF

# Create user-data
cat > "$CLOUD_INIT_DIR/user-data" <<EOF
#cloud-config
users:
  - name: hazem
    groups: [ wheel ]
    sudo: [ "ALL=(ALL) NOPASSWD:ALL" ]
    shell: /bin/bash
    ssh_authorized_keys:
      - $(cat "$HOME/.ssh/id_rsa.pub")

ssh_pwauth: false
preserve_hostname: false
hostname: ${VM_NAME}
EOF

# --- 4. Create Seed ISO with cloud-localds ---
echo "💿 Creating seed ISO with cloud-localds..."
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data"

# Move to libvirt directory with correct permissions
sudo mv "$SEED_ISO" "$LIBVIRT_PATH/${VM_NAME}-seed.iso"
sudo chown libvirt-qemu:kvm "$LIBVIRT_PATH/${VM_NAME}-seed.iso" 2>/dev/null || \
    sudo chown nobody:kvm "$LIBVIRT_PATH/${VM_NAME}-seed.iso"
sudo chmod 644 "$LIBVIRT_PATH/${VM_NAME}-seed.iso"

# Cleanup temp files
rm -rf "$CLOUD_INIT_DIR"
SEED_ISO="$LIBVIRT_PATH/${VM_NAME}-seed.iso"  # Update path for virt-install

# --- 5. Create Overlay Disk ---
echo "💾 Creating overlay image..."
sudo qemu-img create -f qcow2 -b "$BASE_IMG" -F qcow2 "$OVERLAY_IMG" 20G
sudo chown libvirt-qemu:kvm "$OVERLAY_IMG" 2>/dev/null || \
    sudo chown nobody:kvm "$OVERLAY_IMG"
sudo chmod 660 "$OVERLAY_IMG"

# --- 6. Deploy ---
echo "🚀 Booting VM: $VM_NAME"
sudo virt-install \
  --name "$VM_NAME" \
  --memory 2048 \
  --vcpus 2 \
  --disk path="$OVERLAY_IMG",format=qcow2,bus=virtio \
  --disk path="$SEED_ISO",device=cdrom \
  --os-variant rocky9 \
  --graphics vnc,listen=127.0.0.1 \
  --network network=default \
  --import \
  --noautoconsole

# --- 7. Wait and Show Info ---
echo "⏳ Waiting for IP..."
sleep 15

IP=$(virsh domifaddr "$VM_NAME" | awk '/ipv4/ {print $4}' | cut -d/ -f1)
if [[ -n "$IP" ]]; then
    echo -e "\n✅ VM ready!"
    echo "🌐 IP: $IP"
    echo "🔐 SSH: ssh hazem@$IP"
else
    echo "⚠️  Could not get IP. Check: virsh domifaddr $VM_NAME"
fi

