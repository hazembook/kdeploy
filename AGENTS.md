# kdeploy Agent Instructions

> **Go Rewrite Phase**: All work branches from and merges to `golang-rewrite-experiment`. Nothing is pushed to remotes. Once the rewrite is stable with thorough and comprehensive tests, this changes to target `main`.

<!-- BEGIN BR INTEGRATION v:1 profile:full -->

## Issue Tracking with br (beads_rust)

**IMPORTANT**: This project uses **br (beads_rust)** for ALL issue tracking. Do NOT use markdown TODOs, task lists, or other tracking methods.

### Why br?

- Dependency-aware: Track blockers and relationships between issues
- Git-friendly: SQLite + JSONL export for git collaboration
- Agent-optimized: JSON output, ready work detection, actor tracking
- Prevents duplicate tracking systems and confusion
- Rust port (faster than the original Go `bd`)

### Quick Start

**Check for ready work:**
```bash
br ready --json
```

**Create new issues:**
```bash
br create "Title" --description="Context" -t task|bug|feature|epic -p 0-4 --json
br create "Title" -p 1 --deps discovered-from:br-abc --json
```

**Claim and update:**
```bash
ACTOR="${BR_ACTOR:-assistant}"
br update <id> --status in_progress --actor "$ACTOR" --json
```

**Sync before git commit:**
```bash
br sync --flush-only
git add .beads/
git commit -m "chore(br): <summary>"
```

**Complete work:**
```bash
br close <id> --reason "Substantive summary of what was done" --actor "$ACTOR" --json
```

### Issue Types

- `bug` — Something broken
- `feature` — New functionality
- `task` — Work item (tests, docs, refactoring)
- `epic` — Large feature with subtasks

### Priorities

- `0` — Critical (broken builds, security)
- `1` — High (major features, important bugs)
- `2` — Medium (default)
- `3` — Low (polish, optimization)
- `4` — Backlog (future ideas)

### Workflow for AI Agents (Go Rewrite Phase)

1. **Check ready work**: `br ready --json` shows unblocked issues
2. **Create the feature branch from the rewrite branch**: `git switch golang-rewrite-experiment && git switch -c feat/<short-description>` before any state changes
3. **Claim on the branch and commit beads**: `br update <id> --status in_progress --actor "$ACTOR" --json && git add .beads/ && git commit -m "chore(br): claim <id>"`
4. **Investigate using MCP tools before writing code:**
   - `mcp__semble__search` — search the codebase by intent/concept
   - `git log --follow <affected-files>` — check for prior changes
5. **Present an investigation report** (see Reports section). **Wait for explicit human approval before implementing.**
6. **Implement, run quality gates** (`make build && make vet && make test`) on the branch
7. **Present a thorough handoff report** (see Reports section) and **stop and wait for explicit human confirmation** — the agent never assumes approval from a "thanks" or "looks good" alone
8. **After explicit confirmation, run the publish steps** (see Session Completion)

Rationale: single-owner project, feature branches stay local. The agent works on isolated, reproducible steps; the human reviews the diff and explicitly signs off. The "go ahead" is the gate.

### Human-in-the-Loop Split

- **Agent** owns: check br, branch, claim, investigate, present investigation report, implement, test, quality gates, present handoff report
- **Human** owns: review investigation report (approve approach), review diff + handoff (approve publish)
- **After confirmation, agent** owns: `br close`, commit JSONL export, ff-merge to `golang-rewrite-experiment`, clean up branch
- **When rewrite is stable with thorough tests**, the target branch changes to `main`

### Pre-change Workflow (Direct Requests)

For **direct requests** from the user or **work that isn't tracked**:

1. **Check br for related work**: `br ready --json` and `br list --json`. If an existing issue covers the request, claim it. Otherwise create a new task.
2. **Branch from `golang-rewrite-experiment` first, then claim on the branch:**
   ```bash
   git switch golang-rewrite-experiment
   git switch -c feat/<short-description>
   br update <id> --status in_progress --actor "$ACTOR" --json
   git add .beads/ && git commit -m "chore(br): claim <id>"
   ```
3. **Investigate using MCP tools**
4. **Present investigation report** — wait for approval
5. **Implement, run quality gates, commit** with `Refs: <id>` footer
6. **Present handoff report** — wait for explicit confirmation
7. **After "go ahead", execute publish steps**

**Pausing a task (work not finished):**
```bash
git add -A && git commit -m "wip: <id> partial checkpoint"
git switch -c feat/<next-id>
# When returning:
git switch <paused-branch>
git rebase golang-rewrite-experiment
# continue
```

### Important Rules

- Use br for substantive work; trivial doc-only changes may skip br
- Always use `--json` flag for programmatic use
- Link discovered work with `discovered-from` dependencies
- Check `br ready` before asking "what should I work on?"
- Branch per task — create a feature branch for each change
- Commit between tasks — close and commit before claiming the next one
- Do NOT create markdown TODO lists
- Do NOT use external issue trackers
- Do NOT duplicate tracking systems

## Reports

### Investigation Report

Presented after MCP investigation, before implementation. Wait for explicit human approval.

- **Related code found**: what semble/code search revealed
- **Prior art**: `git log --follow` findings
- **Current behavior**: how the code works today
- **Proposed approach**: specific plan
- **Alternatives considered**: rejected approaches and why
- **Risks**: edge cases, breaking changes, dependencies
- **Affected files**: list of files expected to change

### Handoff Report

Presented after quality gates pass, before publishing. Wait for explicit confirmation.

- **Branch**: the branch name
- **Task**: br issue ID and title
- **Key commits**: `git log --oneline` of work commits
- **Diff highlights**: what changed and why
- **Quality gates**: `make build`, `make vet`, `make test` results
- **What was NOT done**: deferred work, known limitations, follow-up issues
- **Suggested merge message**: proposed commit message for `golang-rewrite-experiment`

## Session Completion

### Agent handoff (end of session)

1. **File issues for remaining work** — anything noticed but not fixed
2. **Run quality gates** on the branch — `make build && make vet && make test`
3. **Present handoff report** to the human
4. **Stop and wait for explicit confirmation** — "thanks" or "looks good" is NOT approval

### Human review (gate)

5. **Review the diff on the branch** (and the handoff summary)
6. **Give explicit confirmation** — say "go ahead" (or "merge it", "publish", "ship it")

### Agent publish (only after explicit confirmation)

7. **Close the task on the feature branch**, then commit the JSONL export:
   ```bash
   br close <id> --reason "<resolution summary>" --actor "$ACTOR" --json
   git add .beads/ && git commit -m "chore(br): close <id> — <summary>"
   ```
   The `--reason` must be a substantive summary, not git bookkeeping.

8. **Fast-forward `golang-rewrite-experiment`:**
   ```bash
   git rebase golang-rewrite-experiment
   git switch golang-rewrite-experiment
   git merge <feat-branch>   # fast-forward
   ```

9. **Delete the local feature branch:**
   ```bash
   git branch -d <feat-branch>
   ```

**CRITICAL RULES:**
- The agent **never** force-pushes (`--force` / `--force-with-lease`)
- The agent **never** merges to `golang-rewrite-experiment` without explicit human confirmation
- The agent **never** pushes to any remote
- A "thanks" or "looks good" is **not** approval
- If the agent hits a git state it can't recover from, it commits a `wip:` checkpoint and hands off

> **Future**: Once the rewrite is stable with thorough and comprehensive tests, the target branch changes to `main` and pushing to `origin` becomes part of the flow.

<!-- END BR INTEGRATION -->

## Build & Test

```bash
make build        # Build into bin/kdeploy
make format       # gofmt + goimports
make vet          # go vet ./...
make test         # go test ./... -v
make lint         # format + vet + golangci-lint
make clean        # rm -rf bin/
```

## Architecture Overview

- **Entry point**: `main.go` → `cmd.Execute()` (Cobra CLI)
- **Commands**: `cmd/root.go` (flags), `cmd/deploy.go`, `cmd/destroy.go`
- **Packages** (`pkg/`):
  - `pkg/config/` — `~/.config/kdeploy.conf` key=value parser
  - `pkg/check/` — pre-flight checks (KVM, libvirt, disk space, SSH key)
  - `pkg/detect/` — OS variant detection from filename, qcow2/img scanner
  - `pkg/hypervisor/` — `virt-install`/`virsh` wrappers, IP discovery (`domifaddr` + DHCP leases), privilege detection (`NeedsSudo`, `NeedsPrivilegedPath`)
  - `pkg/iso/` — cloud-init `user-data`/`meta-data` generation, SHA-512 crypt password hashing via `openssl passwd -6`
  - `pkg/storage/` — native `syscall.Statfs` disk checks, overlay creation via `qemu-img`
  - `pkg/ssh/` — SSH config include management (`~/.ssh/kdeploy.d/`), `WaitForSSH` (TCP 22 poll)
- **Shell-out pattern**: MVP calls `virt-install`, `virsh`, `qemu-img`, `cloud-localds`, `openssl` via `os/exec`; pure-Go bindings deferred
- **Privilege model**: two separate concerns —
  - `cmdSudo` (libvirt group check) for `virsh`/`virt-install`
  - `fileSudo` (path-based check via `NeedsPrivilegedPath`) for `qemu-img`, `cloud-localds`, `rm`

## Conventions & Patterns

- **Go style**: stdlib first, minimal dependencies. Cobra for CLI, no framework.
- **Error handling**: wrap errors with `%w` using `fmt.Errorf("context: %w", err)`. Return early on failure in deploy pipeline.
- **Testing**: std `testing` package, table-driven tests for pure functions, integration tests for hypervisor/ssh (gated on virsh availability).
- **Commit messages**: Conventional Commits (`type(scope): message`), imperative mood, ≤ 72 chars. Use footer `Refs: br-xxx` for issue linkage.
- **Branches**: `feat/<short-description>` off `golang-rewrite-experiment`. One branch per task. Do not push — local-only. Targets `main` once the rewrite is stable with thorough tests.
- **Config format**: `~/.config/kdeploy.conf` — `KEY=VAL` lines, `#` comments. Keys: `IMAGE_PATH`, `STORAGE_PATH`, `DEFAULT_VM_SIZE`, `DEFAULT_RAM`, `DEFAULT_CPUS`, `DEFAULT_PASSWORD`.
- **Code comments**: keep code self-documenting. No TODO/FIXME/HACK markers — use br. No commented-out code.

## Environment

- Host: CachyOS (Arch-based) — uses **UFW** (not firewalld)
- libvirt group membership gives passwordless `virsh`/`virt-install`
- File ops under `/var/*`, `/srv/*`, `/opt/*` need sudo regardless of libvirt group
- SSH configs auto-added to `~/.ssh/kdeploy.d/` with `<name>` and `<name>-root` host aliases
- Static MAC→IP convention: `52:54:00:16:01:13` → `172.16.1.13`
- Images in `~/VM/cloud_images/`, VMs in `~/VM/disks/` (both user-owned)
