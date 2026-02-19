#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer (Universal) - v2
# Usage: ./vm-deploy.sh <vm-name>
# ==============================================================================

set -e  # Exit on error

VM_NAME="$1"
BASE_IMG_NAME="Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
LIBVIRT_PATH="/var/lib/libvirt/images"
BASE_IMG="$LIBVIRT_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$LIBVIRT_PATH/${VM_NAME}.qcow2"
SEED_ISO="$LIBVIRT_PATH/${VM_NAME}-seed.iso"
CLOUD_INIT_DIR="./cloud-init"

# --- 1. Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

# Check for required commands
for cmd in virsh qemu-img virt-install genisoimage; do
    if ! command -v "$cmd" &> /dev/null; then
        echo "❌ Required command not found: $cmd"
        exit 1
    fi
done

# Ensure SSH key exists
if [[ ! -f "$HOME/.ssh/id_rsa.pub" ]]; then
    echo "❌ SSH public key not found at $HOME/.ssh/id_rsa.pub"
    echo "   Generate one with: ssh-keygen -t rsa -b 4096"
    exit 1
fi

# Ensure Base Image exists
if [[ ! -f "$BASE_IMG" ]]; then
    echo "❌ Base image not found at: $BASE_IMG"
    echo "   Download it: sudo curl -L -o $BASE_IMG https://dl.rockylinux.org/pub/rocky/9/images/x86_64/Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
    exit 1
fi

# --- 2. Cleanup Old VM ---
echo "🧹 Cleaning up old resources..."
virsh destroy "$VM_NAME" 2>/dev/null || true
virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
rm -f "$OVERLAY_IMG" "$SEED_ISO"

# --- 3. Create Cloud-Init Config ---
echo "🔧 Creating cloud-init configuration..."
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
disable_root: true
preserve_hostname: false
hostname: ${VM_NAME}
fqdn: ${VM_NAME}.local

# Install packages
packages:
  - vim
  - htop
  - tmux

# Update all packages on first boot
package_update: true
package_upgrade: true

# Final message
final_message: "VM ${VM_NAME} is ready! IP: \$(hostname -I)"
EOF

# --- 4. Create Seed ISO ---
echo "💿 Creating seed ISO..."
genisoimage \
  -output "$SEED_ISO" \
  -volid cidata \
  -joliet \
  -rock \
  "$CLOUD_INIT_DIR/user-data" \
  "$CLOUD_INIT_DIR/meta-data"

sudo chown libvirt-qemu:kvm "$SEED_ISO" 2>/dev/null || sudo chown nobody:kvm "$SEED_ISO"
sudo chmod 644 "$SEED_ISO"

# --- 5. Create Overlay Disk ---
echo "💾 Creating overlay image..."
qemu-img create -f qcow2 -b "$BASE_IMG" -F qcow2 "$OVERLAY_IMG" 20G
sudo chown libvirt-qemu:kvm "$OVERLAY_IMG" 2>/dev/null || sudo chown nobody:kvm "$OVERLAY_IMG"
sudo chmod 660 "$OVERLAY_IMG"

# --- 6. Deploy ---
echo "🚀 Booting VM: $VM_NAME"
virt-install \
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
echo "⏳ Waiting for VM to get IP..."
sleep 15

echo -e "\n📊 VM Information:"
virsh dominfo "$VM_NAME"
echo -e "\n🌐 Network:"
virsh domifaddr "$VM_NAME"

IP=$(virsh domifaddr "$VM_NAME" | grep ipv4 | awk '{print $4}' | cut -d/ -f1)
if [[ -n "$IP" ]]; then
    echo -e "\n✅ Ready to SSH:"
    echo "   ssh hazem@$IP"
fi

