# kdeploy Agent Instructions

## Environment
- Host: CachyOS (Arch-based) — uses **UFW** (not firewalld)
- libvirt group membership checked at runtime; file ops under `/var/*` use sudo separately
- SSH configs auto-added to `~/.ssh/kdeploy.d/` with `<name>` and `<name>-root` hosts
- Static MAC→IP convention: `52:54:00:16:01:13` → `172.16.1.13`

## Build & Quality
```bash
make build        # Build into bin/kdeploy
make format       # gofmt + goimports
make vet          # go vet ./...
make test         # go test ./... -v
make lint         # format + vet + golangci-lint
make clean        # rm -rf bin/
```

## br Workflow
1. **Check** — `br ready --json | jq -r '.[0] | .id + ": " + .title'`
2. **Branch** — `git switch -c feat/<short-description>`
3. **Claim** — `ACTOR="${BR_ACTOR:-assistant}" br update <id> --status in_progress --actor "$ACTOR"`
4. **Implement** — write code, run `make build && make vet && make test`
5. **Handoff** — `br close <id> --reason "Handoff: <summary>" --actor "$ACTOR"`
6. **Publish** — `git add -A && git commit -m "type(scope): message"`
