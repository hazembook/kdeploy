# kdeploy UI refresh (Tier B)

Status: implemented on `feat/charm-ui-refresh` for the Tier B scope approved on
2026-10-25. This document is the user-facing UX contract and the maintenance
guide for the interaction layer.

The refresh changes presentation and interaction only. Command names, flags,
config keys, catalog URLs, VM operations, network/firewall behavior, the sudo
model, and `virt-install` arguments are unchanged.

## 1. What changed

- `fang` renders grouped help, errors, and `--version`.
- `huh` drives first-run setup, image selection, resource/access forms, review,
  confirmations, and password input.
- `lipgloss` renders tables, panels, key/value blocks, and the shared theme.
- Bubble Tea remains an indirect dependency through `huh`; no custom Bubble Tea
  program is imported by `kdeploy`.
- `check`, `download`, `destroy`, and the lifecycle commands use one shared
  UI object for consistent status lines and plain output.
- A Huh form-capture spike was validated with VHS. Safe recordings live in
  `docs/media/`; see `demo/README.md`.

## 2. Package boundaries

```
main.go
  └── cmd/              command orchestration and policy
        ├── internal/ui presentation and interaction only
        └── pkg/        VM, config, check, storage, SSH operations
```

Rules:

1. `pkg/*` does not import Charm libraries.
2. `internal/ui` does not import `cmd` or `pkg`.
3. `cmd` owns behavior: it resolves options, decides when to prompt, and calls
   `pkg`.
4. New prompts are composed in `cmd`; the shared theme, width, TTY gate, and
   cancellation mapping live in `internal/ui`.
5. Only one UI package exists. New concerns go into focused files inside
   `internal/ui`, not new packages.

## 3. Stream contract

| Stream | Content |
|---|---|
| **stdout** | Final results: VM lists, info, IP, masked config values, check results, download catalog/completion, deploy summary |
| **stderr** | Prompts, form UI, spinners, progress, section headings, warnings, errors, review panels |
| **Plain mode** | No ANSI, no spinner, ASCII symbols, stable tab-separated tables |

`kdeploy ip` always prints one plain line to stdout, including on a TTY, so it
remains script-friendly. Package-manager output from `check --install` is
streamed to stderr; stdout contains the report, not installer noise.

## 4. TTY and `--yes`

Prompts require a usable terminal on **both** stdin and stderr, and require
`TERM` not to be `dumb`. When a prompt is needed but no terminal is available,
the command fails before doing anything destructive:

```
interactive terminal required: <action>; use --yes for non-interactive mode
```

- `--yes` / `-y` is fully non-interactive.
- `--yes` uses built-in/config defaults where a form would otherwise appear.
- Non-TTY without `--yes` fails before deletion, overwrite, or a partial deploy.
- A TTY without `--yes` still asks for destructive confirmations.
- Ctrl+C inside a Huh form prints `Cancelled.` and exits `130`.
- Choosing Cancel at deploy review is treated as cancellation, not success.

Exit codes:

| Code | Meaning |
|---|---|
| `0` | Success; for `check`, every dependency passed |
| `1` | Error, failed or incomplete check, declined install, or declined destroy |
| `130` | User cancelled a prompt |

Usage errors remain `1` in this refresh. A typed usage-error exit `2` is
deferred.

## 5. Flows

### Root, help, and version

`kdeploy` with no arguments continues to show help. Help is grouped into VM
Lifecycle, Inspect, Images, and Configuration. `completion` and `man` are
disabled to keep the command surface unchanged. `--version` and `-v` print the
version injected at build time, for example:

```text
kdeploy version v0.1.0
```

`--help` and `--version` do not load config or touch libvirt or a VM.

### First run and configuration

The first interactive `deploy` without a config file opens a Huh setup form:

- Image library: `~/VM/cloud_images` (recommended), `/var/lib/libvirt/images`,
  or a custom path.
- VM storage: `/var/lib/libvirt/images`, `~/VM/disks`, or a custom path.
- Paths beginning with `~` are expanded.
- System paths explain that sudo may be requested.
- The config is written atomically with mode `0600`.
- `--yes` uses defaults and does **not** write a config file.

`kdeploy --reconfig` runs the same form and writes the selected path. An
existing config is preserved unless `--reconfig` or
`kdeploy config init --force` is used.

### Deploy

1. Parse the VM name, optional disk size, and flags.
2. Load config and resolve **flag > config > built-in default**.
3. First interactive run without config: setup form.
4. Non-TTY without `--yes`: fail with an actionable error before changes.
5. Preflight: KVM, libvirt, disk space, required tools, SSH key, default
   network. Every result is shown, including the detail on failure.
6. Select an image with a Huh select. Options show filename and detected OS;
   the last option is “Download a new image…”.
7. Fill Memory, vCPUs, Disk size, Network, Root password, and Packages.
8. Review Name, Image, OS, Resources, Network, Storage, and SSH aliases.
9. Choose Create VM, Edit settings, Change image, or Cancel.
10. Only after Create proceeds: optionally stop/undefine the old VM, generate
    the seed ISO, create the overlay, run `virt-install`, wait for IP/SSH, and
    write the SSH config.

Safety properties:

- An existing VM is never destroyed at review time.
- Cancel at review leaves all existing resources untouched.
- The password is masked in `config show` and in review output.
- The disk size positional argument still overrides `--size` and config.
- `--yes` uses the first image and form defaults; if no image exists, it fails
  with a clear message instead of downloading blindly.

### Download

`download --list` prints the catalog. With no key or URL, an interactive TTY
gets the catalog picker and a “Custom URL…” option. The destination path is
printed to stderr before a transfer starts. If the destination already exists:

- interactive: confirmation defaults to **No**;
- `--yes`: overwrite without prompting;
- non-TTY without `--yes`: fail with a message to use `--yes`.

The native resumable behavior remains: `wget -c` when available, otherwise
`curl -L -C -`. URL filenames must be plain names; path traversal attempts,
including percent-encoded slashes/backslashes, are rejected.

### Check

`check` remains a dependency diagnostic, not a deploy preflight replacement.
It reports each required tool and the final `Dependencies` row.

- Exit `0` only when every check passes.
- Exit `1` when any check fails.
- `--install` shows the detected distro, package manager, and package list.
- Install requires a TTY or `--yes`.
- Package-manager output goes to stderr.
- After install, the dependency set is checked again; an exit `0` is printed
  only when the result is complete.
- Declining or an incomplete install exits `1`.

### Destroy

`destroy <name>` builds a removal plan before asking:

```text
Removal plan
  VM          demo
  Definition  will be undefined
  Disk        /path/to/demo.qcow2
  Seed        /path/to/demo-seed.iso
  SSH config  /path/to/kdeploy.d/demo.conf
```

The confirmation defaults to **No**. If the VM does not exist and no leftovers
exist, the command reports that accurately and leaves the filesystem alone.
After an undefine, `VMExists` is checked again; if the VM still exists the
command reports failure. Cleanup failures are propagated, not swallowed.

### Lifecycle commands

`start`, `shutdown`, `stop`, `reboot`, `console`, and `edit` use consistent
stdout/stderr behavior. Status and success lines are on stderr; data results
(for example `list`, `info`, and `ip`) are on stdout. `console` connects
stdin so the serial console is usable.

## 6. Colors and plain output

- `NO_COLOR` disables color.
- `TERM=dumb` selects plain mode.
- Piping stdout forces plain rendering even when stdin/stderr is a TTY.
- Plain tables are tab-separated and deterministic.
- Styled output is still legible with common terminals; the theme uses
  adaptive light/dark colors.

There is no runtime `gum` dependency. VHS is a development-only tool and is not
in `go.mod`.

## 7. Adding a prompt or output path

1. Decide the domain value and validation in `cmd`.
2. Build the Huh field/group in `cmd`, or use the small `UI` helpers.
3. Run it through `u.RunForm(ctx, form)` so the TTY gate, width, theme, and
   Ctrl+C mapping are shared.
4. Never write prompts to stdout.
5. Add a deterministic plain result path for piped stdout.
6. Add a unit test for pure decisions and a manual/TTY check for the form.
7. If the output is a new table, add a plain golden test under
   `internal/ui/testdata`.

Example form:

```go
form := huh.NewForm(
    huh.NewGroup(
        huh.NewInput().
            Title("Memory (MB)").
            Value(&memory).
            Validate(validatePositiveInt),
    ),
)
if err := u.RunForm(ctx, form); err != nil {
    return err
}
```

`UI.RunForm` returns `ui.ErrCancelled` for Ctrl+C/abort. The root error handler
prints `Cancelled.` once and exits `130`.

## 8. Behavior matrix

| Rule | Automated test | Implementation |
|---|---|---|
| Piped stdout is plain and has no ANSI | `internal/ui.TestTablePlainNoANSI`, `TestInteractivePromptWithPipedOutputIsPlain` | `internal/ui/table.go`, `ui.go` |
| Plain tables are stable | `internal/ui.TestTablePlainGolden` | `internal/ui/testdata/table.golden` |
| Review panels go to stderr | `internal/ui.TestPanelPlainGoesToStderr` | `internal/ui/output.go` |
| Forms require a TTY | `internal/ui.TestRunFormRequiresTTY` | `internal/ui/prompt.go` |
| Password display is masked | `internal/ui.TestMaskSecret`, `cmd.TestShowConfigMasksPassword` | `internal/ui/ui.go`, `cmd/config.go` |
| Flag > config > default | `cmd.TestResolveDeployOptions*` | `cmd/options.go` |
| Invalid names/sizes/resources/networks are rejected | `cmd.TestValidate*` | `cmd/validate.go` |
| Deploy form validation is shared | `cmd.TestReviewRows`, `cmd.TestParsePackages`, `cmd.TestImageOptions` | `cmd/deploy_prompt.go` |
| Download overwrite defaults to No and honors `--yes` | `cmd.TestConfirmDownloadOverwrite` | `cmd/download.go` |
| Download URL traversal is rejected | `cmd.TestImageFilenameFromURL` | `cmd/download.go` |
| Check exit decision is testable | `cmd.TestCheckExit` | `cmd/check.go` |
| Destroy resource paths/cleanup are deterministic | `cmd.TestVMResourcePaths`, `TestRemoveVMResources`, `TestExistingResources` | `cmd/destroy.go` |
| Destroy reports a VM that still exists | `pkg/hypervisor.TestDestroyReportsFailureWhenVMStillExists` | `pkg/hypervisor/libvirt.go` |
| Errors are printed once, cancellation is distinct | `cmd.TestHandleErrorCancelled`, `TestHandleErrorPlainWhenPiped` | `cmd/root.go` |
| Help groups stay stable | `cmd.TestConfigureCommandGroups` | `cmd/root.go` |
| Existing command behavior is unchanged | full `go test ./...` plus manual matrix | all packages |

Golden tests use plain output. Styled output is reviewed manually and by the VHS
recordings; Go tests remain the authoritative gate.

## 9. Manual verification matrix

| Scenario | Expected |
|---|---|
| TTY at 80 columns | Forms and review fit/scroll readably |
| TTY at 120 columns | Tables and panels do not stretch without bound |
| Pipe `kdeploy list` | Plain table, no ANSI |
| `NO_COLOR=1` | No color; readable symbols |
| `TERM=dumb` | Plain mode; no spinner |
| `deploy` first run | Setup form writes `0600` config |
| `deploy --yes` first run | Defaults, no config write, no prompts |
| `deploy` non-TTY without `--yes` | Clear failure before changes |
| Existing VM, cancel at review | Old VM and files untouched |
| Existing VM, create | Replaced only after final confirmation |
| No images | Download flow, then selection |
| `download --yes` overwrite | Overwrites without a prompt |
| `download` non-TTY overwrite without `--yes` | Clear failure, existing file intact |
| `destroy` without `--yes` | Confirmation, default No |
| `destroy --yes` | No prompt |
| Missing dependency | `check` detail, exit `1`; deploy points to `check --install` |
| `check` all pass | Exit `0` |
| `check --install` declined | Exit `1`, missing dependencies still reported |
| `check --install` still incomplete | Exit `1`, post-install re-check reported |
| Ctrl+C in a form | `Cancelled.`, exit `130`, no partial VM changes |
| `kdeploy ip` | One plain line |
| `kdeploy --version` | Version line, exit `0` |

Safe VHS tapes render the interactive flows without creating or deleting a VM:

```bash
make demo   # requires vhs, ttyd, ffmpeg
```

## 10. Deferred work

- No-arg home menu.
- `--json` and `--quiet`.
- Direct Bubble Tea progress dashboard.
- Native Go downloader with progress.
- Usage-error exit code `2`.
- `create` alias; `completion`/`man` commands.

These are product surfaces, not gaps in the Tier B contract. See
`docs/decisions.md` for the rationale.
