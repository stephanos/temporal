# WASM Gomad execution backend

Implement the complete approved execution-backend plan at `.turbo/plans/wasm-gomad-execution-backend.md`. The user explicitly requested full implementation on2026-10-09. Preserve existing production defaults/native overlays and qualification ownership, comments and local research; no worktrees or force pushes. Dependency/build PRs12495-12497 remain distinct from Gomad integration. User retains commits unless specifically requested; this scope has no new publication authority.

The user requires hermetically sealed guest execution. Every guest file, network, clock and entropy operation uses captured immutable inputs or in-memory models. Deny live host resources and native I/O fallbacks. Build preparation, module loading and retained evidence publication remain outside guest semantics.

Current owner direction (2026-10-10): develop tools/gomad_wasm and tools/gomad3 together in this temporal repository. Task fn-151.12 is cancelled; no standalone extraction or deletion of gomad3 is planned. Tasks 1–4 retain historical acceptance, task 5 remains in progress with incomplete qualification, and tasks 6–11 remain queued. The owner requested a stop after in-flight work: this migration does not authorize further feature work, task5 qualification, commits or publication. Historical receipts bind their original source candidate, not this integrated checkout. Native qualification remains deferred under fn-128/fn-149. [MILESTONES.md](../../MILESTONES.md) is the operative guide.

## Acceptance Criteria

- **R1:** Typed backend/engine identity and bounded isolated Wasmtime execution prove the corrected main-based artifact can instantiate without hidden host imports.
- **R2:** Deterministic captured WASI namespace, entropy, clock/poll and capacity behavior have positive/negative seam tests and retained fresh-instance repeatability.
- **R3:** SQLite and unchanged testcore/functional smoke execute in one guest with truthful capability dispositions and existing provider semantics.
- **R4:** Preparation, campaigns, canonical artifacts, input/engine preflight, replay and lifecycle integration preserve native formats/defaults and negative validation.
- **R5:** Seeded cooperative scheduling with stable goroutine/timer identities makes same-seed guest runs identical and recorded tapes replay exactly, with tampered tapes rejected.
- **R6:** Strict virtual time advances only at guest quiescence, skipping long timer waits deterministically without host wall-clock influence.
- **R7:** A simulated in-guest network delivers fake-TCP traffic deterministically and supports delay, partition/heal and reset with replayable Temporal recovery.
- **R8:** Unchanged one-box smoke suites run deterministically with R5–R7 enabled; measured costs and a go/no-go recommendation for fn-156 are retained.

## Repository migration (owner decision, 2026-10-10)

The active checkout is temporal. tools/gomad_wasm remains a nested module beside tools/gomad3, with reviewed public shared owners rather than copied standalone code. fn-151.12 is administratively cancelled and satisfies no active R-ID. Historical source, unrelated dirty records and bulk retained evidence are archived recoverably in sibling temporal_wasm.retired-20261010; see its RETIRED.md. Original task1–4 receipts reference that archive’s .tmp paths. Task5 partial handover and failed logs are also copied into this checkout’s .flow/tmp/fn151-5-current. Original absolute paths and frozen identities remain historical, not destination qualification.

## Prototype split (owner decision, 2026-10-10)

fn-151 is now the **determinism prototype**: does Temporal's unchanged test stack run deterministically under WASM with controlled scheduling, time skipping and a simulated network, at an acceptable cost? Bug-yield is not the prototype's goal. Tasks 5–7 are reduced-gate versions of plan Phases 4, 5 and 7; task 8 is the integrated gate and go/no-go. Everything else — full scheduling/GC/entropy qualification, checkpoints, exploration/resume/minimization, storage faults, full network model, multi-node broker, qualification, optimization and documentation — moved to [fn-156](fn-156-wasm-gomad-productionization.md), which starts only after a recorded "go". Original R5–R11 now live there as R1–R7. The approved plan text moved to fn-156 unchanged and remains the technical reference for both specs (also at `.turbo/plans/wasm-gomad-execution-backend.md`).

