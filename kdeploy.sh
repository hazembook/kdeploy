#!/bin/bash
# ==============================================================================
# KVM Cloud Image Deployer v2.0
# "The Professor's Edition" - Safer SSH, Robust Networking, Flexible Storage
# ==============================================================================

set -e

# --- INPUTS ---
VM_NAME="$1"
VM_SIZE="${2:-20G}"  # Default to 20G if not specified

# --- CONFIGURATION ---
DEFAULT_PASS="linux"
ISO_LIB_PATH="$HOME/VM/cloud_images"
VM_STORAGE_PATH="$HOME/VM/disks"
SSH_DIR="$HOME/.ssh"
SSH_CONFIG="$SSH_DIR/config"
SSH_INCLUDE_DIR="$SSH_DIR/kdeploy.d" # Dedicated folder for VM configs

# ---------------------

# Handle if script is run via sudo/doas
VM_USER="${SUDO_USER:-${DOAS_USER:-$USER}}"
SSH_PUB_KEY="$HOME/.ssh/id_rsa.pub"
SSH_PRIV_KEY="${SSH_PUB_KEY%.pub}"

# --- 1. Checks ---
if [[ -z "$VM_NAME" ]]; then
    echo "Usage: $0 <vm_name> [disk_size]"
    echo "Example: $0 webserver 50G"
    exit 1
fi

if [[ ! -d "$ISO_LIB_PATH" ]]; then
    echo "❌ Base Image Directory not found: $ISO_LIB_PATH"
    exit 1
fi
mkdir -p "$VM_STORAGE_PATH"

# --- Privilege Detection ---
if id -nG "$USER" | grep -qw "libvirt"; then
    echo "✅ User '$USER' is in libvirt group. Running without sudo."
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

# --- 2. Dynamic Image Selector ---
echo "📂 Scanning $ISO_LIB_PATH for base images..."
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

# --- 3. Smart Detection ---
echo "🔍 Inspecting image..."
BACKING_FMT=$(qemu-img info "$BASE_IMG" | grep "file format" | cut -d: -f2 | xargs)
LOWER_NAME="${BASE_IMG_NAME,,}"

# Simple variant logic (add more as needed)
case "$LOWER_NAME" in
    *rocky*10*)   OS_VARIANT="rocky10" ;;
    *rocky*9*)   OS_VARIANT="rocky9" ;;
    *rocky*8*)   OS_VARIANT="rocky8" ;;
    *noble*)    OS_VARIANT="ubuntunoble" ;;
    *debian*12*)    OS_VARIANT="debian12" ;;
    *debian*13*)    OS_VARIANT="debian13" ;;
    *)           OS_VARIANT="generic" ;;
esac

# --- 4. Secure Workspace ---
WORK_DIR=$(mktemp -d -t kdeploy-XXXXXXXX)
CLOUD_INIT_DIR="$WORK_DIR/config"
SEED_ISO="$WORK_DIR/seed.iso"
mkdir -p "$CLOUD_INIT_DIR"
trap 'rm -rf "$WORK_DIR"' EXIT

# --- 5. Password Setup ---
echo "🔑 Setup VM Password (Enter for default: '$DEFAULT_PASS')"
read -s -p "   Password: " INPUT_PASS
echo ""
VM_PASSWORD="${INPUT_PASS:-$DEFAULT_PASS}"
PASSWORD_HASH=$(openssl passwd -6 "$VM_PASSWORD")

# --- 6. Cleanup Old VM ---
echo "🧹 Cleaning up old resources..."
$PRIV_CMD virsh destroy "$VM_NAME" 2>/dev/null || true
$PRIV_CMD virsh undefine "$VM_NAME" --remove-all-storage 2>/dev/null || true
rm -f "$OVERLAY_IMG" "$VM_STORAGE_PATH/${VM_NAME}-seed.iso"

# --- 7. Cloud-Init ---
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

# --- 8. Seed ISO & Overlay ---
cloud-localds -v "$SEED_ISO" "$CLOUD_INIT_DIR/user-data" "$CLOUD_INIT_DIR/meta-data" > /dev/null
FINAL_ISO="$VM_STORAGE_PATH/${VM_NAME}-seed.iso"
mv "$SEED_ISO" "$FINAL_ISO"

echo "💾 Creating overlay image ($VM_SIZE)..."
qemu-img create -f qcow2 -b "$BASE_IMG" -F "$BACKING_FMT" "$OVERLAY_IMG" "$VM_SIZE" > /dev/null
chmod 644 "$OVERLAY_IMG" "$FINAL_ISO"

# --- 9. Deploy ---
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

# --- 10. Robust IP Discovery ---
echo "⏳ Waiting for IP address..."
IP=""
count=0
MAC=$($PRIV_CMD virsh dumpxml "$VM_NAME" | grep "mac address" | head -1 | cut -d"'" -f2)

while [ -z "$IP" ]; do
    if [ $count -gt 45 ]; then
        echo -e "\n⚠️  Timed out. Check console: virsh console $VM_NAME"
        exit 1
    fi
    
    # Method A: Ask the Guest Agent (Filter out 127.0.0.1)
    # We use grep -v "127.0.0.1" to ensure we don't grab the loopback interface
    IP=$($PRIV_CMD virsh domifaddr "$VM_NAME" --source agent 2>/dev/null \
        | grep "ipv4" \
        | grep -v "127.0.0.1" \
        | awk '{print $4}' \
        | cut -d/ -f1 \
        | head -n1)
    
    # Method B: Ask the DHCP Server (If agent fails or returns nothing yet)
    if [ -z "$IP" ]; then
        # Look for IP in DHCP leases matching our MAC
        IP=$($PRIV_CMD virsh net-dhcp-leases default \
            | grep "$MAC" \
            | awk '{print $5}' \
            | cut -d/ -f1 \
            | head -n1)
    fi

    if [ -z "$IP" ]; then sleep 2; count=$((count + 1)); printf "."; fi
done

# Remove old fingerprint
ssh-keygen -R "$IP" 2>/dev/null

# --- 11. Safer SSH Config ---
echo -e "\n📝 Managing SSH Config..."

# Create the dedicated directory if it doesn't exist
mkdir -p "$SSH_INCLUDE_DIR"
chmod 700 "$SSH_INCLUDE_DIR"

# Ensure the main config includes our directory
# We append this only ONCE to the very top of the config if missing
if [ ! -f "$SSH_CONFIG" ]; then
    touch "$SSH_CONFIG"
    echo "Include $SSH_INCLUDE_DIR/*" > "$SSH_CONFIG"
elif ! grep -q "Include $SSH_INCLUDE_DIR/\*" "$SSH_CONFIG"; then
    # Prepend the Include directive to the top of the file
    echo "Include $SSH_INCLUDE_DIR/*" | cat - "$SSH_CONFIG" > temp && mv temp "$SSH_CONFIG"
    echo "   -> Added Include directive to $SSH_CONFIG"
fi

# Write the specific VM config to its own file
# This completely overwrites just THIS VM's config, leaving others safe
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

echo -e "\n✅ VM Ready!"
echo "-------------------------------------"
echo "🌐 IP:      $IP"
echo "👤 User:    ssh $VM_NAME"
echo "👑 Root:    ssh $VM_NAME-root"
echo "-------------------------------------"
