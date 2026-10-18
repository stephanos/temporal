---
satisfies: [R2]
---
# fn-211-apalache-backend-for-umpire-in-rust.2 umpire-apalache: pinned release, bounded check per invariant, deterministic lifecycle

## Description
**Placement (owner 2026-10-18):** the crate lives at `model2/umpire/backends/apalache/`.

New crate next to `umpire-tlc`. Fetch a pinned Apalache release into `model2/build/` (Makefile target, checksum recorded), run it on the mise JDK (21+), one `check --inv=<name> --length=<native depth + 1>` per invariant, parse the verdict into `Bounded`. If the JSON-RPC server is used, the runner owns its start and stop. Nothing written outside the build directory.

**Files:** model2/umpire/backends/apalache/, model2/Cargo.toml (members), Makefile.

**Quick commands:** `UMPIRE_BACKENDS=require mise exec -- cargo test -p umpire-apalache`
## Acceptance
- Kettle's invariants each return a bounded verdict.
- Missing or mis-versioned release fails before tests; a tool failure is NotRun with its message.
- No leftover process after the test.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
