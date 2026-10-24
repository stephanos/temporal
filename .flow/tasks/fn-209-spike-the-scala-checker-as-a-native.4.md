---
satisfies: [R4]
---
# fn-209-spike-the-scala-checker-as-a-native.4 Go/no-go, recommendation, and the fn-203.3 / fn-204 recut on go

## Description
Record independent live and simulation outcomes from integrated target evidence: native-image .2 is the live-deployment target and a viable synchronous simulation option if R3c restoration and measured latency fit; WASM .5 is preferred for simulation packaging and in-host latency. Include .3's conditional fallback disposition without requiring an unnecessary third build. Simulation correctness requires deterministic prior-point restoration, not mandatory same-store cloning. Assess measured workload costs, not process-boundary assumptions.

**Files:** `model/framework/design/native-checker-spike.md`, the task-owned companion target notes, fn-202/fn-203/fn-204 planning records and MILESTONES.md.
**Touches:** [model/framework/design/native-checker-spike.md, model/framework/design/native-checker-graal-spike.md, model/framework/design/native-checker-native-spike.md, .flow/specs/fn-202-*, .flow/tasks/fn-202-*, .flow/specs/fn-203-*, .flow/tasks/fn-203-*, .flow/specs/fn-204-*, .flow/tasks/fn-204-*, MILESTONES.md]

Use Flow commands for all planning state. On a proven applicable go, recut fn-203.3/fn-204 and fn-202's generic driver plan according to the owner's target boundary: Go acts/observes; the Model decides/judges. Preserve distinct live and simulation support for mixed outcomes; do not remove a descriptor route for an unsupported consumer. Revisit driver naming as the owner requested, record production compile-time replacement as a separate prerequisite before adoption, and keep all later implementation held. On no-go retain the exact toolchain/host blocker and upstream status; leave dependent implementation routes available rather than claiming adoption. Preserve owner edits and historical proof reports.

**Quick commands:** validate the affected Flow specs and cross-check cited target artifacts/committed closures; no production build.

## Acceptance
- [ ] The main note links target evidence and states separate live/simulation go/no-go plus conditional Scala Native disposition, distinguishing build/load/parity, the chosen R3c restore mechanism and latency. Compare actual sidecar round trips and in-host calls where available; an unavailable comparison remains explicit. Recorded blockers name exact versions/commands and upstream status; no missing measurement is success and no store-clone requirement overrides the owner's snapshot revision.
- [ ] Dependent planning and MILESTONES are reconciled through Flow where applicable to proven outcomes, without executing fn-202/fn-203/fn-204 or dropping support for an unproved consumer. Recommendation, naming decision and production-replacement prerequisite are explicit; historical results and owner holds are preserved.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: