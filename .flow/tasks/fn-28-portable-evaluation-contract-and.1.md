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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
