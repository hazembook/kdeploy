# kdeploy Agent Instructions

> **Alpha Phase**: All work branches from and merges to `main` (fast-forward only). The rewrite has reached feature parity with the bash script, but the project is still alpha: commands, flags, config, and output may change.
>
> **Push policy**: The agent **never pushes** — not to `origin`, `lab`, `hub`, or any other remote, and not tags. It commits locally, runs the gates, and stops with a handoff. The human pushes manually so they can drop, reword, reset, or rebase local history before publishing.

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

### Workflow for AI Agents

1. **Check ready work**: `br ready --json` shows unblocked issues
2. **Create the feature branch from `main`**: `git switch main && git switch -c feat/<short-description>` before any state changes
3. **Claim on the branch and commit beads**: `br update <id> --status in_progress --actor "$ACTOR" --json && git add .beads/ && git commit -m "chore(br): claim <id>"`
4. **Investigate using MCP tools before writing code:**
   - `mcp__semble__search` — search the codebase by intent/concept
   - `git log --follow <affected-files>` — check for prior changes
5. **Present an investigation report** (see Reports section). **Wait for explicit human approval before implementing.**
6. **Implement, run quality gates** (`make check`) on the branch
7. **Present a thorough handoff report** (see Reports section) and **stop and wait for explicit human confirmation** — the agent never assumes approval from a "thanks" or "looks good" alone
8. **After explicit confirmation, run the publish steps** (see Session Completion)

Rationale: single-owner project. The agent works on isolated, reproducible steps; the human reviews the diff and explicitly signs off. The "go ahead" is the gate.

### Human-in-the-Loop Split

- **Agent** owns: check br, branch, claim, investigate, present investigation report, implement, test, quality gates, present handoff report
- **Human** owns: review investigation report (approve approach), review diff + handoff (approve publish), **all pushes and tags**
- **After confirmation, agent** owns: `br close`, commit JSONL export, ff-merge to `main`, clean up branch — then **stops**. The human pushes.

**Pausing a task (work not finished):**
```bash
git add -A && git commit -m "wip: <id> partial checkpoint"
git switch -c feat/<next-id>
# When returning:
git switch <paused-branch>
git rebase main
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
- **Quality gates**: `make check` (and `git diff --check`) results
- **What was NOT done**: deferred work, known limitations, follow-up issues
- **Suggested merge message**: proposed commit message for `main`

## Session Completion

### Agent handoff (end of session)

1. **File issues for remaining work** — anything noticed but not fixed
2. **Run quality gates** on the branch — `make check`
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

8. **Fast-forward `main` locally (no push):**
   ```bash
   git rebase main
   git switch main
   git merge <feat-branch>   # fast-forward
   ```
   The agent stops here. The human reviews `git log` / `git diff` and pushes
   manually (`git push <remote> main` and any tags) when ready.

9. **Delete the local feature branch:**
   ```bash
   git branch -d <feat-branch>
   ```

**CRITICAL RULES:**
- The agent **never** pushes (`git push`), force-pushes, or creates/deletes remote refs or tags — all publishing is human-only
- A "thanks" or "looks good" is **not** approval
- If the agent hits a git state it can't recover from, it commits a `wip:` checkpoint and hands off

<!-- END BR INTEGRATION -->

## Build & Test

```bash
make build        # Build the version-injected binary into bin/
make check        # gofmt check + go vet + go test -race (required gate)
make format       # gofmt + goimports (writes)
make vet          # go vet ./...
make test         # go test ./... -v
make lint         # format + vet + golangci-lint
make demo         # optional dev-only VHS render; never part of make check
make clean        # rm -rf bin/
```

Every implementation phase must end with `make check`. `make demo` requires
VHS, `ttyd`, and `ffmpeg`; those are development tools, not Go dependencies.

## Architecture Overview

- **Entry point**: `main.go` → `cmd.Execute()` (Cobra + `fang`)
- **UI layer**: `internal/ui/` owns the Huh form theme, Lip Gloss output,
  width/TTY detection, plain renderer, spinner, and cancellation mapping.
  It must not import `cmd` or `pkg`.
- **Commands**: `cmd/root.go` (flags), `cmd/deploy.go`, `cmd/destroy.go`, `cmd/download.go`, `cmd/check.go`, `cmd/config.go`, `cmd/list.go`, `cmd/info.go`, `cmd/start.go`, `cmd/shutdown.go`, `cmd/stop.go`, `cmd/reboot.go`, `cmd/console.go`, `cmd/edit.go`, `cmd/ip.go`
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

- **Go style**: stdlib first, minimal dependencies. Cobra for CLI. Charmbracelet Go libraries (`fang`, `huh`, `lipgloss`) are the approved interaction exception; do not add `gum` or another UI framework.
- **Charm boundaries**: `cmd` composes commands and forms; `internal/ui` owns theme/rendering/TTY/cancellation; `pkg/*` must not import Charm or `internal/ui`. Bubble Tea stays indirect through `huh` unless direct use is explicitly approved.
- **Stream contract**: final data results go to stdout; prompts, forms, spinners, progress, headings, warnings, and errors go to stderr. Piped stdout is plain: no ANSI, ASCII symbols, stable tab-separated tables. `kdeploy ip` is always one plain stdout line.
- **Interaction contract**: prompts require a TTY on stdin and stderr (and `TERM != dumb`). `--yes` is fully non-interactive. Non-TTY without `--yes` fails before destructive action. Destructive confirmations default to No. Ctrl+C in a form maps to exit `130`; `check` exits `0` only when all checks pass, otherwise `1`.
- **Error handling**: wrap errors with `%w` using `fmt.Errorf("context: %w", err)`. Return early on failure in deploy pipeline.
- **Testing**: std `testing` package, table-driven tests for pure functions, integration tests for hypervisor/ssh (gated on virsh availability). UI policy tests belong in `internal/ui`; command policy tests in `cmd`.
- **Commit messages**: Conventional Commits (`type(scope): message`), imperative mood, ≤ 72 chars. Use footer `Refs: br-xxx` for issue linkage.
- **Branches**: `feat/<short-description>` off `main`. One branch per task. Merge fast-forward into `main` only after explicit approval. **Never push**; the human publishes manually.
- **Config format**: `~/.config/kdeploy.conf` — `KEY=VAL` lines, `#` comments. Keys: `IMAGE_PATH`, `STORAGE_PATH`, `DEFAULT_VM_SIZE`, `DEFAULT_RAM`, `DEFAULT_CPUS`, `DEFAULT_PASSWORD`.
- **Code comments**: keep code self-documenting. No TODO/FIXME/HACK markers — use br. No commented-out code.
- **New output**: use the shared `internal/ui` helpers. Never write a prompt to stdout. Add a plain golden test when adding a table and a manual TTY check for a new form.

## Demos

- `demo/` contains developer-only VHS tapes and `setup.sh`; `docs/media/` contains the committed captures.
- `make demo` is the only supported render command and is **not** part of `make check`.
- Tapes must use fake `virsh`/runtime tools and disposable paths under `/var/tmp/kdeploy-demo`. Deploy cancels at review; destroy keeps the default No; download cancels before transfer.
- A recording must never create, stop, or delete a real VM, or touch real config/images/disks/SSH files.
- There is no checked-in full-deploy tape. A real deploy recording is opt-in and manual, never part of CI or `make demo`.
- VHS, `ttyd`, and `ffmpeg` are development tools; they must not enter `go.mod` or the runtime binary.

## Environment

- Host: CachyOS (Arch-based) — uses **UFW** (not firewalld)
- libvirt group membership gives passwordless `virsh`/`virt-install`
- File ops under `/var/*`, `/srv/*`, `/opt/*` need sudo regardless of libvirt group
- SSH configs auto-added to `~/.ssh/kdeploy.d/` with `<name>` and `<name>-root` host aliases
- Static MAC→IP convention: `52:54:00:16:01:13` → `172.16.1.13`
- Images in `~/VM/cloud_images/`, VMs in `~/VM/disks/` (both user-owned)
