---
satisfies: [R2]
---
# fn-222-gomad-qualify-the-one-box-cell-and.2 Define the per-test support list and produce the first one

## Description
Define the candidate-bound per-test support list: candidate identity (helper digest, compiler, runtime overlay identity, module identity), one row per supported test with package, function, gates passed with exits and retained artifact reference, and an explicit statement that unlisted tests are unsupported. Produce it through the existing qualification owner (internal/evidence/qualification wrapped by internal/backends/wasm/qualification), store it next to the checkpoint records, and add a QUALIFICATION.md section describing the shape and pointing at the current list. The first list contains exactly the tests that passed the cell gates, at minimum TestFrontendSystemInfo.

**Size:** S
**Files:** tools/gomad/internal/backends/wasm/qualification/**, tools/gomad/internal/backends/wasm/QUALIFICATION.md, .flow/artifacts/fn222-*.md

## Acceptance
- [ ] Support list shape documented in QUALIFICATION.md with per-test, candidate-bound wording
- [ ] First list for the current candidate contains exactly the gate-passing tests
- [ ] A sweep pass alone cannot add a row (documented and enforced by the producer)

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
