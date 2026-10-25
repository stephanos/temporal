---
satisfies: [R3, R4, R5]
---
# fn-247-one-language-for-the-framework-a-rust.3 Complete Rust-only gate wiring, documentation and integrated behavioral evidence

## Description

Complete current-tool documentation, aggregate no-Python proof and final integrated validation (R3, R4, R5). Preserve existing .3 ID; it explicitly waits for .5, .6, .7 and .9, retaining its original .2 prerequisite. This is the central integration/documentation/final-validation owner, not a reason to serialize the three source lanes.

**Size:** M
**Files:** `Makefile`, `model/{README.md,GUEST_BUILDS.md,DESIGN.md}`, `.plans/UMPIRE_MODULES.md`, completed Rust test harness/gate wiring.
**Touches:** [Makefile, model/README.md, model/GUEST_BUILDS.md, model/DESIGN.md, .plans/UMPIRE_MODULES.md, model/dev/xtask/**, model/umpire/backends/process/**]

## Approach

- Consume the conductor's .5/.6/.7 integration join: preserve every new dispatcher command and Make caller, combine Cargo/features without guest-closure leakage, and rerun affected focused controls. Confirm .5's final asset publication/two-build reproducibility/freshness, .6's real pinned install and .7's lifecycle/mutation proofs were completed on joined inputs before accepting their completion. No additional full gate is introduced; the single required pipeline below remains the final integrated gate.
- Treat the initial 14-file Python inventory as historical evidence and take a fresh integrated tracked/embedded-Python census. Map useful old responsibilities to native controls, recording pure Python quirks and byte-only verifier assertions retired under owner permission. No safety/semantic control is obsolete merely because its implementation was Python. Preserve historical evidence outside the active toolchain and frozen originals.
- Remove the obsolete Python unittest stage after its retained responsibilities have native Cargo controls. The existing `cargo test --workspace --locked` stage exclusively owns process and xtask unit/integration controls, exactly once per canonical gate; do not add a second package-specific Cargo pass as the replacement stage. Preserve every control responsibility, update stage reporting to describe the ownership transfer, and keep host-only tools outside intended wasm targets.
- Update current prerequisite/command/alias/wrapper/identity/platform/ownership documentation. Record libc and the R18 admissions (`toml`, `sha2`, `tar`, `flate2`, `tempfile`, `rustix`) accurately; document Rust bootstrap prerequisites/native CLI/schema/parse/error changes, remove duplicated Bash semantic validation and the old helper CI skip matrix, and drop Python/`verify.py`/`manifest.json` verification instructions; do not describe unapproved dependency work or simulated platform tests as completed.
- Consume the delivered `model-validate` target and run one `UMPIRE_APALACHE=require make model-validate` pipeline using conductor-owned caches/leases. This includes one complete TLC/Apalache-required gate and one existing edit/test benchmark, preserving the previously required Apalache controls without reactivating deferred fn211 consumer work. Do not add separate default/full/timing gate repetitions. Preserve default/skip semantics through the new task's focused mode controls. Verify semantic expectations and mutation/negative controls, native driver controls, and freshness/reproducibility against current committed assets. Necessary guest/golden/pin regeneration is permitted with documented cause and old/new hashes; no historical byte/diagnostic/API parity proof or redundant full-gate run is required. Never repin or accept snapshots to hide semantic regressions.
- Record actual command/results, tools/platform, useful-control mapping and explicitly retired Python quirks, R18 dependency admissions with before/after inventory, process-crate unsafe counts from .9, guest hashes and measured timing once. Identify required-mode timing separately from historical default-mode timing. Report soft-target overruns and absent native Darwin evidence. MILESTONES remains root-owned and frozen during validation.

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

- [ ] A fresh integrated tracked/embedded-Python census reconciles the historical initial 14-file inventory with useful native controls and documented retired Python quirks; staged tracked Model Python inventory is empty and the ordinary gate invokes no Python.
- [ ] Current docs/maps/preflight match Rust ownership, entrypoints, host-only build choreography, platform policy and approved dependency decisions; historical evidence remains accurate.
- [ ] One canonical TLC/Apalache-required pipeline and one edit/test benchmark pass with exact recorded command/mode/stages/output/platform/timing; existing focused controls retain legacy default/skip behavior. Any failure remains a concrete open obligation, not a relabelled pass. Required timings and soft overruns are reported honestly.
- [ ] Semantic expectations, lifecycle comparison and mutation/negative controls pass, with guest freshness/reproducibility against current committed assets. Necessary guest/golden/pin regenerations document cause and old/new hashes; retained originals and frozen artifacts remain unmodified. Historical byte equality is not required; feature-semantic changes remain outside scope.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:



Conductor validation ownership clarification (2026-10-25): component completion requires its joined native/cold admission and artifact proofs plus primary review. The one complete semantic Go/TLC/Apalache pipeline belongs to final .3 before parent closure; it is not repeated per port, and component tasks do not wait for their own dependent final task. All semantic and negative controls remain mandatory in that pipeline.
