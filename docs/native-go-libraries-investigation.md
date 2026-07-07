# Native Go Libraries: Feasibility Investigation

> Status: Research phase — no implementation started
> Date: 2026-07-07

---

## 1. Overview

**Current state:** kdeploy shells out 49 `exec.Command` calls across 13 unique external
commands. The Go binary is a thin orchestration layer over CLI tools.

**Goal:** Replace each shell-out category with a pure Go library, eliminating all host
dependencies except `libvirtd`.

| Category | Calls | Dependency | Feasibility |
|----------|-------|-----------|-------------|
| `virsh` (29 subcommands) | 29 | Requires `libvirtd` running | ✅ Replaceable |
| `virt-install` | 2 | Requires `virt-install` package | ✅ Replaceable |
| `qemu-img create / info` | 4 | Requires `qemu-img` package | ✅ Replaceable |
| `cloud-localds` | 2 | Requires `cloud-image-utils` | ✅ Replaceable |
| `wget` / `curl` | 6 | Requires wget or curl | ✅ Replaceable |
| `openssl passwd -6` | 1 | Requires openssl | ✅ Replaceable |
| `ssh-keygen` | 2 | Requires openssh | ✅ Replaceable |
| `id -nG` | 1 | coreutils | ✅ Replaceable |
| `lsmod` | 1 | procps | ✅ Replaceable |
| `ufw` | 3 | UFW firewall | ❌ Keep shell-out |
| `rm` / `chmod` | 2 | coreutils | ❌ Already os.* primary |
| pacman / apt / dnf | 5 | System package manager | ❌ Cannot replace |

---

## 2. Category Deep Dives

---

### 2a. `wget` / `curl` → `net/http`

**Current:** 6 calls in `cmd/download.go` — downloads cloud images with resume support.

**Go library:** `net/http` (stdlib)

**Feasibility:** ✅ Trivial.

```go
func downloadFile(url, dest string) error {
    resp, err := http.Get(url)
    // write to file, support resume via Range header
}
```

**Resume support:** `net/http` supports `Range` header via
`req.Header.Set("Range", "bytes=N-")`.

**Progress display:** Need `io.Copy` with a progress wrapper.

**Progress bar libraries available:**
- `github.com/cheggaaa/pb/v3` — 4.9k stars, mature
- `github.com/schollz/progressbar/v3` — 4k stars, simpler API
- `github.com/vbauerster/mpb/v8` — 2.4k stars, multi-bar

**Risk:** None. HTTP download is the most well-understood problem in Go.

**Market fit:** Every Go project downloads files. No differentiation needed.

**Effort:** ~2 hours basic, ~4 hours with resume + progress bar.

---

### 2b. `id -nG` → native group lookup

**Current:** 1 call in `pkg/hypervisor/priv.go` — checks libvirt group membership.

**Go approach:** Use `os/user` package.

```go
func NeedsSudo() bool {
    u, _ := user.Current()
    gids, _ := u.GroupIds()
    for _, gid := range gids {
        g, err := user.LookupGroupId(gid)
        if err == nil && g.Name == "libvirt" {
            return false
        }
    }
    return true
}
```

**Risk:** None. `os/user` is cross-platform but we target Linux only.

**Effort:** ~30 minutes.

---

### 2c. `lsmod` → `/proc/modules`

**Current:** 1 call in `pkg/check/check.go` — checks KVM module availability.

**Go approach:** Read `/proc/modules` directly.

```go
func KVM() Result {
    data, _ := os.ReadFile("/proc/modules")
    for _, line := range strings.Split(string(data), "\n") {
        if strings.HasPrefix(line, "kvm ") {
            return Result{Name: "KVM module", Passed: true}
        }
    }
    // also check /dev/kvm
}
```

**Risk:** None. `/proc/modules` is Linux-specific but we target Linux.

**Effort:** ~15 minutes.

---

### 2d. `openssl passwd -6` → native SHA-512 crypt

**Current:** 1 call in `pkg/iso/passwd.go` — generates `$6$...` password hash for cloud-init.

**Algorithm:** SHA-512 crypt as specified by Ulrich Drepper:
https://www.akkadia.org/drepper/SHA-crypt.txt

**Approach options:**

1. **Manual implementation** (~100 lines using `crypto/sha512`)
   - Well-documented, deterministic algorithm
   - No dependency added
   - Must test against known `openssl passwd -6` outputs

2. **Third-party library**
   - `github.com/sergeymakinen/go-crypt` — 90 stars, MIT, last updated 2024
   - `github.com/ammario/mksh` — smaller scope, implements crypt(3)

**Risk:** Low. The algorithm is deterministic and well-tested.

**Market fit:** Niche need (cloud-init password hashing). No market consideration.

**Effort:** ~2 hours manual, ~30 minutes with library.

---

### 2e. `cloud-localds` → native ISO 9660

**Current:** 2 calls in `pkg/iso/cloudinit.go` — creates seed ISO with cloud-init files
(user-data, meta-data, network-config).

**Go library option 1:** `github.com/diskfs/go-diskfs` (v1.7.0, ~1.2k stars)

- Full ISO 9660 support with Rock Ridge extensions
- Create ISO from scratch with arbitrary files
- Used by `docker-machine-driver-xenserver` in production
- Supports volume labels, Joliet, Rock Ridge
- v1.6.0 released 2026-05-13 — actively maintained

```go
import "github.com/diskfs/go-diskfs"
import "github.com/diskfs/go-diskfs/disk"
import "github.com/diskfs/go-diskfs/filesystem"

mydisk, _ := diskfs.Create(outputPath, 4*1024*1024, diskfs.Raw)
mydisk.LogicalBlocksize = 2048
fs, _ := mydisk.CreateFilesystem(disk.FilesystemSpec{
    Partition: 0,
    FSType:    filesystem.TypeISO9660,
    VolumeLabel: "cidata",
})
// write user-data, meta-data
iso := fs.(*iso9660.FileSystem)
iso.Finalize(iso9660.FinalizeOptions{
    RockRidge: true,
    VolumeIdentifier: "cidata",
})
```

**Go library option 2:** `github.com/kdomanski/iso9660` (~80 stars)

- Simpler, lighter, dedicated to ISO 9660
- Fewer dependencies
- Supports Rock Ridge

**Key requirement:** Cloud-init requires volume label `cidata` and files at root level.
Both libraries support custom volume labels.

**Risk:** Low. Need to verify result matches `cloud-localds` output (compare hex dumps).

**Market fit:** Both libraries exist specifically because cloud-init ISO generation is
common in Go-based VM tools (Terraform providers, machine drivers).

**Effort:** ~1-2 days including testing.

---

### 2f. `qemu-img create / info` → native qcow2

**Current:** 4 calls in `pkg/storage/disk.go` — overlay creation + format detection.

**Go library:** `github.com/dypflying/go-qcow2lib` (v1.0.1, ~50 stars)

**Capabilities:**
- Create qcow2 files with backing files (external snapshots) ✅
- Read qcow2 headers (backing file path, format, cluster size) ✅
- L2/refcount block caches ✅
- Subcluster support ✅
- NOT supported: compression, internal snapshots, lazy refcounts

**Backing file support:** Creates qcow2 with `backing_file` and `backing_fmt` in header.
Exactly what `qemu-img create -f qcow2 -b base.qcow2 -F qcow2 overlay.qcow2` does.

**Alternative:** `github.com/zchee/go-qcow2` — similar scope, also supports backing files.

**Risk:** Medium. qcow2 is complex with edge cases:
1. Cluster size compatibility between base and overlay
2. Backing file path encoding (relative vs absolute)
3. Must verify resulting overlay works with real VMs
4. May not handle all qemu-img features (e.g., `-o compat=1.1`)

**Market fit:** qcow2 is the primary KVM/QEMU disk format. Native Go libs exist for:
- `container-image` tools needing qcow2 manipulation
- `machine-driver` projects (xenserver, virtualbox) needing disk creation
- Terraform libvirt provider

**Effort:** ~3-5 days including thorough testing with real VM boot.

---

### 2g. `virsh` + `virt-install` → `go-libvirt`

**Current:** 31 calls total — the biggest category.

**Go library:** `github.com/digitalocean/go-libvirt` (~1.2k stars, Apache 2.0)

**What it is:** Pure Go (no CGo) implementation of libvirt's RPC protocol.
Communicates with `libvirtd` via Unix socket
(`/var/run/libvirt/libvirt-sock` or `/var/run/libvirt/virtqemud-sock`) or
TCP/TLS/SSH.

**Capabilities vs shell-out:**

| Virsh command | go-libvirt equivalent | Status |
|--------------|----------------------|--------|
| `list --all` | `ConnectListAllDomains()` | ✅ |
| `start` | `DomainCreate()` | ✅ |
| `shutdown` | `DomainShutdown()` | ✅ |
| `destroy` | `DomainDestroy()` | ✅ |
| `reboot` | `DomainReboot()` | ✅ |
| `domifaddr` | `DomainInterfaceAddresses()` | ✅ |
| `dumpxml` | `DomainGetXMLDesc()` | ✅ |
| `dominfo` | `DomainGetInfo()` | ✅ |
| `edit` | `DomainGetXMLDesc()` + `DomainDefineXML()` | ✅ |
| `net-info` | `NetworkGetXMLDesc()` | ✅ |
| `net-dhcp-leases` | `NetworkGetDHCPLeases()` | ✅ |
| `net-start` | `NetworkCreate()` | ✅ |
| `net-define` | `NetworkDefineXML()` | ✅ |
| `net-autostart` | `NetworkSetAutostart()` | ✅ |
| `virt-install` | `DomainDefineXML()` (craft XML manually) | ✅ |

**Key difference:** `virt-install` generates domain XML internally. With `go-libvirt`,
you craft the XML yourself — more work but full control.

```go
xml := fmt.Sprintf(`<domain type='kvm'>
  <name>%s</name>
  <memory unit='MiB'>%s</memory>
  <vcpu>%s</vcpu>
  ...
</domain>`, name, ram, cpus, diskPath)
dom, err := l.DomainDefineXML(xml)
dom.Create()
```

**Risk:** Medium-High.
1. XML generation error-prone — every libvirt version slightly different schemas
2. `virt-install` handles many details automatically (boot order, firmware selection)
3. Console access uses libvirt stream protocol — more complex than stdin/stdout piping
4. Error handling: shell-out returns stderr; go-libvirt returns RPC error codes
5. Breaking changes in `go-libvirt` between versions

**Market fit:** DigitalOcean built `go-libvirt` for managing hundreds of thousands of VMs
with Go tooling. Battle-tested at scale. Other users:
- `kubevirt` (Kubernetes VM orchestration)
- `cluster-api-provider-libvirt`
- Various Go-based VM managers

**Effort:** ~2-3 weeks full replacement:
- Week 1: Basic domain lifecycle (define, start, stop, destroy, list, reboot, shutdown)
- Week 2: Advanced ops (console, IP discovery, network management, edit)
- Week 3: Testing, edge cases, XML template refinement

---

## 3. Commands That Should Stay Shell-Out

| Command | Reason |
|---------|--------|
| `ufw` | UFW is a high-level wrapper over iptables. Replacing means managing iptables rules directly — fragile, distro-specific, error-prone. Only 3 low-frequency calls. |
| `apt update` / `apt install` / `pacman -S` / `dnf install` | System package managers. No sane Go API exists. |
| `rm`, `chmod` | Already `os.Remove()` / `os.Chmod()` as primary. sudo fallback only for privileged paths. |

---

## 4. Recommended Migration Strategy

**Phase 1 — Quick wins** (no new dependencies, ~1 week):
1. `wget`/`curl` → `net/http` + progress bar library
2. `id -nG` → `os/user`
3. `lsmod` → `/proc/modules`
4. `openssl passwd -6` → native SHA-512 crypt
5. `ssh-keygen` → `crypto/ed25519` + `golang.org/x/crypto/ssh`

**Phase 2 — ISO + qcow2** (2 new deps, ~1 week):
6. `cloud-localds` → `go-diskfs` or `kdomanski/iso9660`
7. `qemu-img create/info` → `go-qcow2lib`

**Phase 3 — libvirt native** (1 major dep, ~2-3 weeks):
8. All `virsh` + `virt-install` → `go-libvirt`

**Total estimate:** 4-5 weeks. Work targets `main` directly.

---

## 5. Open Questions for Debate

1. **Is eliminating `openssl` worth it?** It's installed on virtually every Linux system.
   The payoff is zero-dependency binary, but actual value is small.

2. **`go-diskfs` vs `kdomanski/iso9660`?** First is heavier but more established (1.2k
   stars). Second is lighter but less proven.

3. **`go-qcow2lib` vs keeping `qemu-img`?** `qemu-img` is always present when libvirt is
   installed (it's a dependency). The library has only ~50 stars — relatively niche.

4. **When to stop shelling out?** The current approach works. `go-libvirt` pros:
   - No sudo needed (Unix socket + libvirt group)
   - XML-level control
   - Better error handling
   - No process overhead
   - Remote connections possible
   Cons: XML generation, console stream protocol, learning curve, API churn.

5. **100% native or "good enough"?** A pragmatic approach: replace the easy ones
   (HTTP, password hash, group check) and keep complex ones (virsh, qemu-img,
   cloud-localds) as shell-out until the value is clear.

---

## 6. Full Call Map (for reference)

All 49 `exec.Command` calls across the codebase, grouped by command:

### `wget` — 2 calls
- `cmd/download.go:158,160` — download with resume, conditional sudo

### `curl` — 2 calls
- `cmd/download.go:165,167` — download with resume, conditional sudo

### `chmod` — 1 call
- `cmd/download.go:180` — set permissions after download, only with sudo

### `ssh-keygen` — 2 calls
- `cmd/deploy.go:291` — clear known_hosts entry
- `pkg/check/check.go:89` — generate ed25519 key

### `rm` — 1 call
- `cmd/destroy.go:21` — force-remove file, sudo fallback

### `qemu-img` — 4 calls
- `pkg/storage/disk.go:31,33` — create overlay, conditional sudo
- `pkg/storage/disk.go:44,46` — detect format, conditional sudo

### `cloud-localds` — 2 calls
- `pkg/iso/cloudinit.go:68,70` — generate seed ISO, conditional sudo

### `openssl` — 1 call
- `pkg/iso/passwd.go:10` — SHA-512 password hash

### `lsmod` — 1 call
- `pkg/check/check.go:23` — KVM module check

### `ufw` — 3 calls
- `pkg/check/check.go:168,173,178` — firewall rule management, always sudo

### `virsh` — 29 calls
- `pkg/check/check.go:31,35` — list (libvirt accessibility check)
- `pkg/check/check.go:117,119` — net-info
- `pkg/check/check.go:127,129` — net-start / net-autostart (closure)
- `pkg/check/check.go:141,143` — net-define
- `pkg/hypervisor/libvirt.go:76` — list --all
- `pkg/hypervisor/libvirt.go:107` — start
- `pkg/hypervisor/libvirt.go:119` — shutdown
- `pkg/hypervisor/libvirt.go:131` — destroy (stop)
- `pkg/hypervisor/libvirt.go:143` — reboot
- `pkg/hypervisor/libvirt.go:155` — console
- `pkg/hypervisor/libvirt.go:174` — destroy + undefine (nuke)
- `pkg/hypervisor/libvirt.go:187` — edit
- `pkg/hypervisor/libvirt.go:196,198` — dominfo
- `pkg/hypervisor/libvirt.go:208` — dumpxml
- `pkg/hypervisor/libvirt.go:220` — net-dhcp-leases
- `pkg/hypervisor/ip.go:34,46` — domifaddr (agent + ARP fallback)

### `virt-install` — 2 calls
- `pkg/hypervisor/libvirt.go:36,38` — deploy VM

### `id` — 1 call
- `pkg/hypervisor/priv.go:11` — group membership

### Package managers — 5 calls
- `pkg/check/check.go:357,360,366,369` — sudo pacman/apt/dnf install
