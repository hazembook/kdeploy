# Safe terminal demos

The tapes in this directory are reproducible terminal recordings for review
and documentation. They are **not** required to build or run `kdeploy`, and
`make demo` is **not** part of `make check`.

## Requirements

- [VHS](https://github.com/charmbracelet/vhs) `v0.12.1`
- `ttyd`
- `ffmpeg`

On Arch/CachyOS:

```bash
sudo pacman -S vhs ttyd ffmpeg
```

On macOS with Homebrew:

```bash
brew install vhs ttyd ffmpeg
```

The tools are development-only. They are not in `go.mod` and are not needed by
users of the `kdeploy` binary.

## Render the safe set

From the repository root:

```bash
make demo
```

This builds `bin/kdeploy`, renders the safe tapes into `docs/media/`, and fails
if a fake runtime tool or fake `virsh` records a real operation. To render one
tape:

```bash
vhs demo/deploy.tape
```

## Tapes

| Tape | Flow | Safety |
|---|---|---|
| `help.tape` | `--version`, grouped help | Read-only |
| `check.tape` | Dependency report | Read-only |
| `read-only.tape` | `list`, `info demo`, `ip demo`, `config show` | Fake `virsh`; disposable config; password masked |
| `first-run.tape` | Setup form with custom paths, then cancel | Never creates a VM; disposable config |
| `deploy.tape` | Image select, resources/access form, review, Cancel | Fake runtime tools; never creates a VM |
| `deploy-80.tape` | Same flow at 80 columns | Same |
| `download.tape` | Catalog list and picker, then Ctrl+C | No transfer starts |
| `download-80.tape` | Same at 80 columns | No transfer starts |
| `destroy.tape` | Removal plan, default No | Fake `virsh`; VM and files remain |
| `destroy-80.tape` | Same at 80 columns | Same |

`deploy.tape` and `deploy-80.tape` start from a disposable config so the
recording focuses on the resource form and review. `first-run.tape` covers the
setup wizard separately.

## Fixtures and safety

`setup.sh` creates everything under:

```text
/var/tmp/kdeploy-demo/
  bin/       fake virsh, id, qemu-img, cloud-localds, virt-install
  config/    disposable config files
  disks/     fake disk and seed files for destroy
  home/      disposable HOME and SSH config
  images/    one scannable fake image
```

`/var/tmp` is used because deploy's preflight requires 20 GB free and `/tmp`
is a small tmpfs on the development host. Set `KDEPLOY_DEMO_DIR` when running
`setup.sh` yourself if you want a different disposable path; the checked-in
tapes use the default above.

Safety rules:

- Fake `virsh` and `id` prevent libvirt access and sudo selection.
- Fake `qemu-img`, `cloud-localds`, and `virt-install` log and fail, so even an
  accidental Create would not produce a VM.
- Deploy tapes choose **Cancel** at review.
- Destroy tapes press the default **No**.
- Download tapes cancel before starting a transfer.
- `make demo` checks the operation logs and fails if a real VM action was
  attempted.

No tape uses the host's real `~/.config/kdeploy.conf`, VM images, VM disks, SSH
config, or libvirt domains.

## Optional full deploy

There is deliberately no checked-in `full-deploy.tape`. It would create a real
VM, and the safe set is intended to be runnable on a normal development host
without that side effect. A disposable full deploy can be recorded manually
with a unique VM name and an explicit destroy afterward, outside `make demo`.
