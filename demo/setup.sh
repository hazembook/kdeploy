#!/usr/bin/env bash
# Create disposable fixtures for the VHS tapes. Everything lives under
# /var/tmp/kdeploy-demo unless KDEPLOY_DEMO_DIR is set. No real VM or user
# config is touched.
set -euo pipefail

DEMO_DIR="${KDEPLOY_DEMO_DIR:-/var/tmp/kdeploy-demo}"
MODE="${1:-base}"

fresh_dirs() {
	rm -rf "$DEMO_DIR"
	mkdir -p \
		"$DEMO_DIR/home/.ssh" \
		"$DEMO_DIR/config" \
		"$DEMO_DIR/images" \
		"$DEMO_DIR/disks" \
		"$DEMO_DIR/bin"
	chmod 700 "$DEMO_DIR/home/.ssh"
}

ensure_key() {
	if [[ ! -f "$DEMO_DIR/home/.ssh/id_ed25519" ]]; then
		ssh-keygen -q -t ed25519 -N "" -f "$DEMO_DIR/home/.ssh/id_ed25519"
	fi
	chmod 600 "$DEMO_DIR/home/.ssh/id_ed25519"
}

write_config() {
	local path=$1
	cat >"$path" <<EOF
# kdeploy demo configuration
IMAGE_PATH=$DEMO_DIR/images
STORAGE_PATH=$DEMO_DIR/disks
DEFAULT_VM_SIZE=20G
DEFAULT_RAM=2048
DEFAULT_CPUS=2
DEFAULT_PASSWORD=demo-only
EOF
	chmod 600 "$path"
}

write_fake_id() {
	cat >"$DEMO_DIR/bin/id" <<'EOF'
#!/bin/sh
echo "demo libvirt"
EOF
	chmod 755 "$DEMO_DIR/bin/id"
}

write_fake_runtime_tools() {
	for tool in qemu-img cloud-localds virt-install; do
		cat >"$DEMO_DIR/bin/$tool" <<EOF
#!/bin/sh
echo "$tool \$*" >> "$DEMO_DIR/runtime-operations.log"
exit 1
EOF
		chmod 755 "$DEMO_DIR/bin/$tool"
	done
}

write_read_only_virsh() {
	cat >"$DEMO_DIR/bin/virsh" <<'EOF'
#!/bin/sh
case "$*" in
  "list --all")
    cat <<'TABLE'
 Id   Name   State
------------------------
 1    demo   running
TABLE
    ;;
  "domifaddr demo --source agent"|"domifaddr demo")
    cat <<'TABLE'
 Name       MAC address         Protocol   Address
---------------------------------------------------------------
 vnet0      52:54:00:16:01:13   ipv4       192.0.2.42/24
TABLE
    ;;
  *)
    exit 1
    ;;
esac
EOF
	chmod 755 "$DEMO_DIR/bin/virsh"
}

write_deploy_virsh() {
	cat >"$DEMO_DIR/bin/virsh" <<'EOF'
#!/bin/sh
case "$*" in
  "list --all")
    cat <<'TABLE'
 Id   Name   State
------------------------
TABLE
    ;;
  "net-info default")
    cat <<'TABLE'
Name:           default
UUID:           00000000-0000-0000-0000-000000000000
Active:         yes
Persistent:     yes
Autostart:      yes
Bridge:         virbr0
TABLE
    ;;
  "dominfo demo")
    exit 1
    ;;
  *)
    exit 0
    ;;
esac
EOF
	chmod 755 "$DEMO_DIR/bin/virsh"
}

write_destroy_virsh() {
	cat >"$DEMO_DIR/bin/virsh" <<EOF
#!/bin/sh
case "\$*" in
  "dominfo demo")
    exit 0
    ;;
  *)
    echo "\$*" >> "$DEMO_DIR/virsh-operations.log"
    exit 0
    ;;
esac
EOF
	chmod 755 "$DEMO_DIR/bin/virsh"
}

case "$MODE" in
base)
	fresh_dirs
	ensure_key
	;;
deploy)
	fresh_dirs
	ensure_key
	write_fake_id
	write_deploy_virsh
	write_fake_runtime_tools
	: >"$DEMO_DIR/runtime-operations.log"
	# The deploy tape cancels at review, so this only needs to be scannable.
	truncate -s 1M "$DEMO_DIR/images/Rocky-10-GenericCloud-Base.qcow2"
	write_config "$DEMO_DIR/config/demo.yaml"
	;;
first-run)
	fresh_dirs
	ensure_key
	write_fake_id
	write_deploy_virsh
	write_fake_runtime_tools
	: >"$DEMO_DIR/runtime-operations.log"
	truncate -s 1M "$DEMO_DIR/images/Rocky-10-GenericCloud-Base.qcow2"
	;;
read-only)
	fresh_dirs
	ensure_key
	write_fake_id
	write_read_only_virsh
	write_config "$DEMO_DIR/config/read-only.yaml"
	;;
destroy)
	fresh_dirs
	ensure_key
	write_fake_id
	write_destroy_virsh
	: >"$DEMO_DIR/virsh-operations.log"
	mkdir -p "$DEMO_DIR/home/.ssh/kdeploy.d"
	truncate -s 1M "$DEMO_DIR/disks/demo.qcow2"
	truncate -s 1K "$DEMO_DIR/disks/demo-seed.iso"
	cat >"$DEMO_DIR/home/.ssh/kdeploy.d/demo.conf" <<'EOF'
Host demo
    HostName 192.0.2.42
EOF
	write_config "$DEMO_DIR/config/destroy.yaml"
	;;
clean)
	rm -rf "$DEMO_DIR"
	;;
*)
	echo "usage: $0 [base|first-run|deploy|read-only|destroy|clean]" >&2
	exit 2
	;;
esac
