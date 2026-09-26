@AGENTS.md

## Gomad v3

- `tools/gomad3` is a nested Go module pinned to go1.26.4. The patched runtime toolchain is
  qualified on `darwin/arm64` only, so `make gomad3`, `make -C tools/gomad3 test`, and the
  qualification targets need a Mac. `make -C tools/gomad3 validate` (generated outputs, patches,
  scripts, compatibility packs) and the `host-tools-linux` CI packages run on Linux.
- `.plans/GOMAD_MILESTONES.md` is the operative delivery order; `.plans/GOMAD3_NEXT.md` is the
  capability roadmap it draws from.
