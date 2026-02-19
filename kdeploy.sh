#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer
# Usage: ./kdeploy.sh <vm-name>
# Security: Prompts strictly for password (no CLI args)
# ==============================================================================

set -e

VM_NAME="$1"

# --- CONFIGURATION ---
DEFAULT_PASS="linux"   # <--- The default if you just hit Enter
# ---------------------

VM_USER="${SUDO_USER:-$USER}"
BASE_IMG_NAME="Rocky-9-GenericCloud-Base.latest.x86_64.qcow2"
LIBVIRT_PATH="/var/lib/libvirt/images"
BASE_IMG="$LIBVIRT_PATH/$BASE_IMG_NAME"
OVERLAY_IMG="$LIBVIRT_PATH/${VM_NAME}.qcow2"
SEED_ISO="/tmp/${VM_NAME}-seed.iso"
CLOUD_INIT_DIR="/tmp/cloud-init-$VM_NAME"
SSH_KEY_PATH="/home/$VM_USER/.ssh/id_rsa.pub"

# --- 1. Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name>"
    exit 1
fi

for cmd in virsh virt-install cloud-localds qemu-img openssl; do
    if ! command -v $cmd &> /dev/null; then
        echo "❌ Critical dependency missing: '$cmd'"
        exit 1
    fi
done

if [[ ! -f "$BASE_IMG" ]]; then
    echo "❌ Base image not found at: $BASE_IMG"
    exit 1
fi

if [[ ! -f "$SSH_KEY_PATH" ]]; then
    echo "❌ SSH key not found for user $VM_USER at: $SSH_KEY_PATH"
    exit 1
fi

# --- 2. Secure Password Prompt ---
echo "🔑 Setup VM Password"
echo "   (Leave empty to use default: '$DEFAULT_PASS')"
# -s hides input, -p prints prompt
read -s -p "   Enter Password: " INPUT_PASS
echo "" # Newline

if [[ -z "$INPUT_PASS" ]]; then
    VM_PASSWORD="$DEFAULT_PASS"
    echo "✓ Using default password."
else
    VM_PASSWORD="$INPUT_PASS"
    echo "✓ Using custom password."
fi

# Hash the password (SHA-512)
PASSWORD_HASH=$(openssl passwd -6 "$VM_PASSWORD")

# --- 3. Cleanup ---
echo "🧹 Cleaning up old resources..."
virsh destroy "$VM_NAME" 2>/dev/null || true
virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
sudo rm -f "$OVERLAY_IMG" "$LIBVIRT_PATH/${VM_NAME}-seed.iso"

# --- 4. Cloud-Init Config ---
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

runcmd:
  - sed -i 's/^PermitRootLogin.*/PermitRootLogin prohibit-password/g' /etc/ssh/sshd_config
  - systemctl restart sshd

ssh_pwauth: false
preserve_hostname: false
hostname: ${VM_NAME}
EOF

# --- 5. Create Seed ISO ---
echo "💿 Creating seed ISO..."
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data"

sudo mv "$SEED_ISO" "$LIBVIRT_PATH/${VM_NAME}-seed.iso"
sudo chown libvirt-qemu:kvm "$LIBVIRT_PATH/${VM_NAME}-seed.iso" 2>/dev/null || \
    sudo chown nobody:kvm "$LIBVIRT_PATH/${VM_NAME}-seed.iso"
sudo chmod 644 "$LIBVIRT_PATH/${VM_NAME}-seed.iso"

rm -rf "$CLOUD_INIT_DIR"
SEED_ISO_FINAL="$LIBVIRT_PATH/${VM_NAME}-seed.iso"

# --- 6. Create Overlay Disk ---
echo "💾 Creating overlay image..."
sudo qemu-img create -f qcow2 -b "$BASE_IMG" -F qcow2 "$OVERLAY_IMG" 20G
sudo chown libvirt-qemu:kvm "$OVERLAY_IMG" 2>/dev/null || \
    sudo chown nobody:kvm "$OVERLAY_IMG"
sudo chmod 660 "$OVERLAY_IMG"

# --- 7. Deploy ---
echo "🚀 Booting VM: $VM_NAME"
sudo virt-install \
  --name "$VM_NAME" \
  --memory 2048 \
  --vcpus 2 \
  --disk path="$OVERLAY_IMG",format=qcow2,bus=virtio \
  --disk path="$SEED_ISO_FINAL",device=cdrom \
  --os-variant rocky9 \
  --graphics vnc,listen=127.0.0.1 \
  --console pty,target_type=serial \
  --network network=default \
  --import \
  --noautoconsole

# --- 8. Smart Wait for IP ---
echo "⏳ Waiting for IP address..."
sleep 5
IP=""
count=0
while [ -z "$IP" ]; do
    if [ $count -gt 45 ]; then
        echo -e "\n⚠️  Timed out waiting for IP. Check console:"
        echo "virsh console $VM_NAME"
        exit 1
    fi
    
    IP=$(virsh domifaddr "$VM_NAME" 2>/dev/null | awk '/ipv4/ {print $4}' | cut -d/ -f1 || true)
    
    if [ -z "$IP" ]; then
        sleep 2
        count=$((count + 1))
        printf "."
    fi
done

echo -e "\n✅ VM ready!"
echo "🌐 IP: $IP"
echo "👤 User: $VM_USER"
echo "🔐 SSH: ssh $VM_USER@$IP"
echo "💻 Console: virsh console $VM_NAME (Escape character is ^])"
