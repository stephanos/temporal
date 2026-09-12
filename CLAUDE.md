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
