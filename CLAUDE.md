@AGENTS.md

## Lean Toolchain

Claude Code cloud sessions have no Lean toolchain pre-installed. The cloud environment's
setup script installs Lean (pinned by `model/lean-toolchain`) and protoc (pinned by
`mise.toml`) under `/opt/temporal-toolchain` and adds them to `PATH` via `/etc/profile.d`.

- If `lake` is missing, the setup script did not run. Report that; do not hand-install a
  per-session toolchain.
- `mise` in that environment is a `mise exec -- CMD` passthrough shim and enforces no
  version pins; `go` is the session image's Go, which already matches the `mise.toml` pin.
- The toolchain is cached across sessions but `model/.lake` is not, so a cold
  `make umpire-build-model` costs ~12 minutes.
- `make lint-model` and `lake lint` need `LEAN_NUM_THREADS=1` on a 16 GB session, or the
  lint driver is OOM-killed.

## Flow-Next

`flowctl` owns task tracking (see AGENTS.md) but is not in a cloud session's image.
`.claude/settings.json` declares the marketplace and enables the plugin, so a session that
honours project settings gets it and puts `flowctl` on `PATH`. To install it by hand:

```bash
claude plugin marketplace add gmickel/flow-next
claude plugin install flow-next@flow-next
# scripts/flowctl lands under ~/.claude/plugins/cache/flow-next/flow-next/<version>/
```

- The published plugin carries the same store `SCHEMA_VERSION` as this repository's `.flow`,
  so it reads and writes the store without migrating it. Check that before installing a
  version that has moved on.
- A fresh clone cannot change task status: runtime state lives in the clone's `.git`
  common-dir, so every task reads `todo` from the committed snapshot and `start`, `done` and
  `spec close` refuse. Reviews, dependencies, spec status and `validate` all work, and the
  one-line "runtime state absent" advisory on read commands is expected, not a fault.
