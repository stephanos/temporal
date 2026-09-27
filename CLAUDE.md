@AGENTS.md

## Gomad v3

- `tools/gomad3` is a nested Go module pinned to go1.27.1. The patched runtime toolchain is
  qualified on `darwin/arm64` and `linux/amd64`; `make gomad3` and `make -C tools/gomad3 test`
  run on both. The macOS sandbox test and the DTrace clock audit are darwin-only; the modernc
  libc adapter and the core SQLite workloads qualify on both. The toolchain builder downloads the Go source
  archive from go.dev, which cloud sessions cannot reach; GitHub CI can.
- `.plans/GOMAD_MILESTONES.md` is the operative delivery order; `.plans/GOMAD3_NEXT.md` is the
  capability roadmap it draws from. The remaining work of each milestone is a flow-next spec
  (`fn-95` F1 through `fn-101` F7, chained by spec dependencies); those specs are the
  authoritative work list.
