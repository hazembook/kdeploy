# kdeploy — Project Instructions for AI Agents

## Project Overview

KVM Cloud Image Deployer v3.4. Bash production core + Go rewrite on `golang-rewrite-experiment` branch.

## Key Files

| File | Purpose |
|---|---|
| `kdeploy` | Bash deployer script (production) |
| `README.md` | Documentation |
| `cmd/` | Go Cobra CLI commands |
| `pkg/config/` | Config parser (kdeploy.conf) |
| `pkg/iso/` | Cloud-init ISO generation |
| `pkg/hypervisor/` | Libvirt/virt-install wrappers |
| `pkg/storage/` | Disk checks, overlay creation |
| `pkg/ssh/` | SSH config management |

## Environment Quirks

- **CachyOS uses UFW**, not firewalld → `firewall-cmd` will error; use `ufw allow in on virbr-xxx` + route rules
- Custom bridges block DHCP by default → `sudo ufw allow in on virbr-xxx && sudo ufw route allow in/out on virbr-xxx`
- Static MAC→IP for RHCSA lab: `52:54:00:16:01:13` → `172.16.1.13` (server), `52:54:00:16:01:12` → `172.16.1.12` (client)
- Use `-n "netname,mac=XX:XX"` syntax for static DHCP bindings with custom networks
- `kdeploy` auto-appends SSH configs to `~/.ssh/kdeploy.d/` → enables `ssh server-root` / `ssh client-root` directly
- libvirt group membership needed for passwordless operation; falls back to sudo/doas

## Issue Tracking with `br` (beads_rust)

```bash
# Check ready work
br ready --json

# Create new task
br create "Title" --description "Details" -t task|bug|feature|epic -p 0-4

# Claim and work
br update <id> --claim
git switch -c <br-id>
# ... work ...
br close <id> --reason "Summary"

# Sync to git
br sync --flush-only
git add .beads/ && git commit -m "chore(br): <summary>"
```

### Workflow

1. **Check ready** → `br ready`
2. **Branch** → `git switch -c <br-id>` from `main`
3. **Claim** → `br update <id> --claim`
4. **Investigate** → find relevant code, present report, **wait for approval**
5. **Implement** → code + test, commit with `Refs: <br-id>` footer
6. **Handoff** → present summary, **wait for explicit confirmation**
7. **Publish** → close task, commit beads, ff main, push, delete branch

## Architecture

**Bash** (`main`): single script, cloud-init via `cloud-localds`, `virt-install`, `virsh`, DHCP discovery.

**Go** (`golang-rewrite-experiment`): Cobra CLI, shell-out stubs (cloud-localds, virt-install, virsh) → eventually native go-diskfs/go-libvirt bindings.

## Quality Gates

```bash
bash -n kdeploy
go build ./... && go vet ./...
bash -n ../ops/spinup-lab.sh ../ops/rebuild-lab.sh 2>/dev/null
```
