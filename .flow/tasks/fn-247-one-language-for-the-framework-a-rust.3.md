---
satisfies: [R3, R4, R5]
---
# fn-247-one-language-for-the-framework-a-rust.3 Complete Rust-only gate wiring, documentation and integrated parity evidence

## Description

Complete current-tool documentation, aggregate no-Python proof and final integrated validation (R3, R4, R5). Preserve existing .3 ID; it follows every port and retains its original .2 prerequisite.

**Size:** M
**Files:** `Makefile`, `model/{README.md,GUEST_BUILDS.md,DESIGN.md}`, `.plans/UMPIRE_MODULES.md`, completed Rust test harness/gate wiring.
**Touches:** [Makefile, model/README.md, model/GUEST_BUILDS.md, model/DESIGN.md, .plans/UMPIRE_MODULES.md, model/dev/xtask/**, model/umpire/backends/process/**]

## Approach

- Reconcile the complete14-file and embedded-Python inventory against the Rust counterpart ledger. No missing check can be classified as obsolete merely because its implementation was Python. Preserve historical evidence commands outside the active toolchain and all frozen content.
- Replace the Makefile unittest gate with the process/xtask Cargo controls, retaining every existing gate responsibility. Ensure ordinary defaults need no Python, while Rust tools remain host-only and wasm checks still cover only their intended crates.
- Update current prerequisite/command/alias/wrapper/identity/platform/ownership documentation. Record libc and any separately approved R18 decision accurately; do not describe unapproved dependency work or simulated platform tests as completed.
- Consume the delivered `model-validate` target and run one `UMPIRE_APALACHE=require make model-validate` pipeline using conductor-owned caches/leases. This includes one complete TLC/Apalache-required gate and one existing edit/test benchmark, preserving the previously required Apalache controls without reactivating deferred fn211 consumer work. Do not add separate default/full/timing gate repetitions. Preserve default/skip semantics through the new task's focused mode controls. Compare the exact retained baseline hashes/bytes, ordinary deterministic goldens and guest before declaring equivalence; do not accept snapshots or refresh guest bytes.
- Record actual command/results, tools/platform, old/new control mapping, guest hashes and measured timing once. Identify required-mode timing separately from historical default-mode timing. Report soft-target overruns and absent native Darwin evidence. MILESTONES remains root-owned and frozen during validation.

## Investigation targets

**Required**:
- `Makefile:1277` — complete gate ordering and tool calls.
- `model/GUEST_BUILDS.md:6` — current prerequisites and wrapper details.
- `model/GUEST_BUILDS.md:94` — control commands/preflight.
- `model/README.md:17` — active crate map and prerequisites.
- `model/DESIGN.md:245` — retained reproduction evidence.
**Optional**:
- `.plans/UMPIRE_MODULES.md:25` — active module ownership.

## Quick commands

```bash
UMPIRE_APALACHE=require make model-validate
git ls-files 'model/*.py' 'model/**/*.py'
```

The last command must produce no paths after deletion is staged. Also inspect tracked active Model executable source for Python invocations/heredocs; historical prose is not an executable dependency. Use fresh actual output as evidence; no commands were run during planning.

## Acceptance

- [ ] Complete14-file/embedded-source inventory maps every old control to a passing named Rust counterpart; staged tracked model Python inventory is empty and ordinary gate execution invokes no Python.
- [ ] Current docs/maps/preflight match Rust ownership, entrypoints, host-only build choreography, platform policy and approved dependency decisions; historical evidence remains accurate.
- [ ] One canonical TLC/Apalache-required pipeline and one edit/test benchmark pass with exact recorded command/mode/stages/output/platform/timing; existing focused controls retain legacy default/skip behavior. Any failure remains a concrete open obligation, not a relabelled pass. Required timings and soft overruns are reported honestly.
- [ ] Guest/golden/original hashes and bytes match the exact post-predecessor baseline; no fixture acceptance, pin refresh, frozen modification or feature-semantic change occurs.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
