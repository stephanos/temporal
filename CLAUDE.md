@AGENTS.md

## Gomad v3

- `tools/gomad3` is a nested Go module pinned to go1.27.1. The patched runtime toolchain is
  qualified on `darwin/arm64` and `linux/amd64`; `make gomad3` and `make -C tools/gomad3 test`
  run on both. The macOS sandbox test and the DTrace clock audit are darwin-only, and the
  modernc libc adapter is still darwin-only. The toolchain builder downloads the Go source
  archive from go.dev, which cloud sessions cannot reach; GitHub CI can.
- `.plans/GOMAD_MILESTONES.md` is the operative delivery order; `.plans/GOMAD3_NEXT.md` is the
  capability roadmap it draws from.
