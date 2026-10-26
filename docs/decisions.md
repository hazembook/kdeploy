# kdeploy decisions

Durable decisions for the Go rewrite and the Tier B UI refresh. Last updated
2026-10-26. The approved implementation plan remains the source for the
phase-by-phase rollout; this file records the choices that should survive it.

## Stack and scope

| ID | Decision | Reason |
|---|---|---|
| D1 | Use the Charmbracelet **Go libraries** `fang`, `huh`, and `lipgloss`, not the `gum` executable | Keep one static binary; no runtime dependency on an external UI executable |
| D2 | Adopt the Tier B interaction-layer refresh | Maximum usability gain without adding commands or rewriting VM operations |
| D3 | Do not add a `create` command | `deploy` already means create; a second name would duplicate the surface |
| D4 | Keep Bubble Tea indirect | `huh` is built on it; direct `tea.NewProgram` is unnecessary for this tier |
| D5 | Replace plain Cobra execution with `fang` | Styled grouped help, errors, `--version`, and consistent failure handling |
| D6 | Preserve all commands, flags, config keys, catalog URLs, VM operations, network/firewall behavior, sudo model, and `virt-install` arguments | The refresh is UI-only |
| D7 | Prompts, form UI, spinners, progress, warnings, errors, and section headings go to **stderr**; final results go to **stdout** | Clean pipes and scriptability |
| D8 | Plain mode is a first-class renderer: no ANSI, no spinner, ASCII symbols, stable tab-separated tables | Automation, accessibility, and deterministic tests |
| D9 | `--yes` is fully non-interactive | Predictable automation |
| D10 | Non-TTY without `--yes` fails before a destructive action | Never accept a default deletion or overwrite silently |
| D11 | First interactive `deploy` without config runs a setup form | Restores the useful part of the legacy first-run experience |
| D12 | Resolution order is explicit flag > config value > built-in default | The config keys existed but were previously unused by deploy |
| D13 | Destructive confirmations default to **No** | Replacing or destroying a VM must never be accidental |
| D14 | Review/edit/change-image loop before creation | create-vite-like confidence without a silent final configuration |
| D15 | Mask the password in `config show` and interactive summaries | Stop printing secrets to the terminal |
| D16 | No `--json`, `--quiet`, no-arg home menu, or native download progress in this refresh | Each is a separate product surface |
| D17 | Do not add `internal/version`; inject the version from `main` | Avoid a package for build metadata |
| D18 | Add exactly one package, `internal/ui` | Keep `cmd` orchestration and `pkg` operations UI-free |
| D19 | `check` exits `1` when any dependency fails and `0` only when all pass | A diagnostic must be usable as a gate; deploy already uses the same dependency checks |
| D20 | Exit codes are `0`, `1`, and `130`; usage-error `2` is deferred | Simple contract with cancellation distinct from failure |
| D21 | VHS `v0.12.1` is a development-only demo/review tool, not a Go dependency | Reproducible terminal recordings without runtime or CI dependency |

## Behavior details

| ID | Decision | Reason |
|---|---|---|
| D22 | `ip` prints exactly one plain line to stdout, even on a TTY | It is commonly used in scripts |
| D23 | The download picker includes a Custom URL option and validates a plain final filename | Keep the catalog friendly without letting URL path traversal choose the destination |
| D24 | Download overwrite defaults to No; `--yes` overwrites; non-TTY without `--yes` fails | Safe interactive default with an explicit automation escape hatch |
| D25 | Native `wget -c` / `curl -L -C -` behavior is retained | Downloads are resumable; replacing the downloader is separate work |
| D26 | Deploy `--yes` uses the first image and form defaults; no image is an error | No silent network fetch during automation |
| D27 | Cancel at review and Ctrl+C in a form both exit `130` | Users can distinguish cancellation from failure |
| D28 | Destroy verifies `VMExists` after undefine and propagates cleanup errors | Never report a successful removal while the VM remains |
| D29 | `check` re-checks dependencies after `--install`; installer output is on stderr | Exit `0` means the install actually completed, and stdout stays report-only |
| D30 | VHS fixtures live under `/var/tmp/kdeploy-demo` by default; `KDEPLOY_DEMO_DIR` can override the setup script | `/tmp` may be a small tmpfs, while deploy preflight requires 20 GB free; `/var/tmp` is still disposable |

## Rejected libraries and approaches

| Rejected | Why |
|---|---|
| `gum` | Standalone executable, not a Go library; adds a runtime dependency and shell-out layer |
| `bubbles` imported directly | `huh` already provides the needed inputs, selects, and spinner |
| `progressbar` | Download progress stays with `wget`/`curl`; a native downloader is deferred |
| Cobra-only help/errors | Does not deliver the grouped, styled interaction refresh |
| Custom Bubble Tea dashboard | Overkill for Tier B; `huh` spinner covers silent work; would make Bubble Tea direct |
| Extra UI packages | One `internal/ui` boundary is enough; avoid fragmentation |
| Home menu as default invocation | Changes the command surface; `kdeploy` continues to show help |
| JSON/quiet modes | Requires a complete output contract across every command |
| Usage-error exit `2` now | Would change expectations mid-refresh; keep `1` until a typed error contract exists |

## Demo and verification decisions

| ID | Decision | Reason |
|---|---|---|
| D31 | Safe tapes use fake `virsh`, `id`, `qemu-img`, `cloud-localds`, and `virt-install`; deploy cancels at review and destroy keeps the default No | Record the real UI without touching a VM, disk, SSH config, or libvirt state |
| D32 | `make demo` renders only safe tapes and is not part of `make check` | VHS/ttyd/ffmpeg are developer tools, not build requirements |
| D33 | The first-run setup gets its own tape; deploy recordings start from a disposable config | Keeps each tape focused and avoids fragile form-to-form input timing |
| D34 | Styled terminal output is reviewed through VHS/PNG artifacts; Go golden tests remain the authority for plain output | Avoid brittle ANSI golden tests while keeping deterministic gates |

## Deferred work

| ID | Deferred item | Why |
|---|---|---|
| D35 | Custom Bubble Tea progress dashboard | `huh` spinner is sufficient; direct dashboard can come with a dedicated design |
| D36 | No-arg home menu | A new default surface needs separate approval |
| D37 | `--json` and `--quiet` | Requires an output contract for every command |
| D38 | Native Go downloader with a styled progress bar | Separate downloader investigation already exists |
| D39 | `create` alias | Redundant with `deploy` |
| D40 | `completion` and `man` commands | Disabled intentionally to preserve the command set |
| D41 | Usage-error exit code `2` | Needs a typed usage-error path and a deliberate compatibility change |

## References

- Approved plan: `~/.opencode/plan/kdeploy-charm-ui.md`
- UX contract and behavior matrix: `docs/ui-refresh.md`
- Safe demo instructions: `demo/README.md`
- Plain-output golden test: `internal/ui/testdata/table.golden`
