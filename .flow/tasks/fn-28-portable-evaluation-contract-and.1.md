---
satisfies: [R1]
---

# fn-28-portable-evaluation-contract-and.1 Define the protobuf evaluation-contract vocabulary
## Description

Add the smallest versioned protobuf vocabulary that can carry one closed per-test evaluation contract and its detailed/local result, then reconcile the normative Umpire terms and rules with that portable interpreter seam.

**Size:** L
**Files:** `proto/internal/temporal/server/api/umpire/v1/message.proto`, `api/umpire/v1/**`, `proto/image.bin`, `.plans/UMPIRE4_SPEC.md`
**Touches:** [`proto/internal/temporal/server/api/umpire/v1/message.proto`, `api/umpire/v1/**`, `proto/image.bin`, `.plans/UMPIRE4_SPEC.md`]

### Approach
- Encode the parent spec's exact version-one operator table, tagged operand/result types, diagnostic mapping, canonical evaluation order, and per-operator work accounting without embedding arbitrary code or environment selectors.
- Follow the repository's internal proto package/path convention and generate ordinary Go bindings with `make proto`.
- Add stable Umpire terms/rules stating that the contract is model-derived portable data, not a second behavioral authority, and that unknown or unsupported semantics cannot pass locally.

### Investigation targets

**Required** (read before coding):
- `.plans/UMPIRE4_SPEC.md` and `.plans/LEAN_GUIDELINES.md`.
- Existing `proto/internal/temporal/server/api/*/v1` packages and generated `api/*/v1` output.
- Existing `artifactv2` bindings, Limits, Known Gaps, Observation, link, verdict, and Result shapes.

## Acceptance
- [ ] The proto covers exactly the approved per-test execution/evaluation vocabulary and operator table and has no arbitrary extension or whole-world claim surface.
- [ ] Unknown fields/enum/operator semantics can be detected and rejected by strict admission.
- [ ] `make proto`, focused proto lint, and Umpire spec checks pass with generated outputs included.

## Done summary
Defined the versioned, closed Umpire protobuf evaluation-contract vocabulary and generated Go surface, then reconciled the normative portable-interpreter, strict-admission, and per-test local-decision rules in the Umpire 4 specification.

Verification: `make proto`, `go test -count=1 -tags test_dep ./api/umpire/v1`, and `make lint-model` passed. `make umpire-check-regression` reproduced the approved inherited pre-edit red at `model/Umpire/SemanticInventory/KnownGaps.lean:296`; `make lint-code` remained inherited-red with no findings under `api/umpire`, and the later-task `Temporal.Tool.PortableEvaluationContractTests` target is DEFERRED(task .3).

stage: impl-review - ran [2026-09-01T17:33:23Z..2026-09-01T17:40:01Z]
## Evidence
- Commits: 4fb4b7658e8a442fd52455ead88897d0f0a3b6e3
- Tests: make proto, go test -count=1 -tags test_dep ./api/umpire/v1, make lint-model, INHERITED_RED: make umpire-check-regression (pre-edit and verify: model/Umpire/SemanticInventory/KnownGaps.lean:296 active vocabulary), INHERITED_RED: make lint-code (1373 pre-existing repository-wide findings; no api/umpire findings), DEFERRED(task .3): cd model && mise exec -- lake build Temporal.Tool.PortableEvaluationContractTests
- PRs: