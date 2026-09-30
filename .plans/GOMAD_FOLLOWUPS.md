# Gomad follow-ups: deferred scope

## Goal & Context
<!-- scope: business -->

On 2026-09-29 the open Gomad milestones were reviewed for scope. Work that delivers no capability
the functional-test goal or the downstream-cell goal needs right now moved here, so the active
specs stay small: F7 keeps only its CI smoke gate, F8 keeps only the generated bootstrap decoder,
the virtual-clock tick spec keeps only the `forward` and `strict` policies, and F9 targets linked
mode on darwin/arm64 only. Each item below is a self-contained follow-up with the rationale that
deferred it and the trigger that should revive it. Nothing here blocks another spec.

## Architecture & Data Models
<!-- scope: technical -->

Items keep the requirement text of the spec they came from, so reviving one means copying its
acceptance back or working it here.

**Architecture consolidation (from F8, `fn-102` R2–R6).**
- **D1** Shared completed-execution assessment owner across seed, choice, and simulation
  exploration (was `fn-102.2`, R2).
- **D2** Shared retention policy and artifact-input composition without merging strategy
  transactions (was `fn-102.3`, R3).
- **D3** Move public `Executor`/`ReplayExecutor` injection behind private dependencies (was
  `fn-102.4`, R4). An explicit Go source API change.
- **D4** Architecture fitness checks: package coverage, host-effect purity, public signature
  visibility, with negative fixtures (was `fn-102.5`, R5).
- **D5** Reconcile architecture, platform, and determinism documentation with current evidence
  (was `fn-102.6`, R6).
Deferred because it is maintenance with no behavior change and no consumer waiting on it.
Revive when a second consumer (F9, or a new exploration strategy) hits the duplication.
The [architecture assessment](../.flow/artifacts/fn-102-gomad-architecture-consolidate/architecture-assessment.md)
still holds the evidence.

**Virtual-clock tick exploration (from `fn-103`).**
- **D6** `seeded` (probabilistic mixture with ties) and `fixed=<d>` tick policies, their
  per-workload manifest setting, and a runtime fixture and core workload per policy. Deferred
  because `forward` alone removes the known tie failures; `seeded` is an exploration feature
  without a demonstrated bug that needs it. Revive when a bug class that needs deliberate ties or
  constant quanta is found.

**CI (from F7, `fn-101.4`).**
- **D7** A macOS job for the functional-test smoke gate. Deferred because the linux job is the
  required gate and the existing macOS `temporal-integration` job already exercises darwin.
  Revive if a darwin-only regression escapes to main.

**Downstream cell (from F9, `fn-104`).**
- **D8** Closure-mode support for downstream targets: the exact adapter for the signal-handling
  metrics library the linker otherwise removes. Deferred because F9 qualifies in linked mode.
  Revive if a downstream module needs "any test prepares without building" or closure-mode
  manifests.
- **D9** linux/amd64 downstream compatibility packs and qualification. Deferred because F9 is
  measured on darwin/arm64 only. Revive when a downstream gate must run in linux CI.
- **D10** A downstream-seam guide (was `fn-104` R4). Deferred because the analysis output already
  names each site. Revive when a second downstream module adopts Gomad.

**Clock audit (from F7, `fn-101.3`).**
- **D11** A dynamic linux/amd64 host-clock audit: the fixture disables the runtime's vDSO symbols
  and a seccomp filter kills the process on clock syscalls after activation, with a positive
  control. Deferred because the static inventory pins every host-clock reference on both
  platforms and the darwin DTrace audit exercises the platform-neutral interception. Revive if a
  linux-only clock escape is ever observed.

## Edge Cases & Constraints
<!-- scope: technical -->

- The constraints of `.plans/GOMAD_MILESTONES.md` apply to every item.
- D1–D5 keep F8's constraints: preserve CLI behavior, schemas, canonical bytes, failure
  classification and precedence, and replay compatibility.
- D6 changes identities when a new default is chosen; it carries the COMPAT-5 evidence set.
- D8 and D9 bind exact module versions; a downstream dependency bump reopens them.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** Each deferred item has a task here that carries its origin, the reason it was deferred,
  and its revival trigger; the origin specs no longer list it as open work.
- **R2:** An item is implemented only after its revival trigger is recorded in the milestone
  status; its acceptance is the origin spec's requirement text for that item.

## Boundaries
<!-- scope: business -->

- Not a work queue to drain. Items may stay open indefinitely or be closed as won't-do.
- No new capability beyond what the origin specs described.

## Decision Context
<!-- scope: both -->

2026-09-29, user: review the open specs for cuttable scope, then move the cut candidates to a new,
separate follow-up milestone.
