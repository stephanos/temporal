---
satisfies: [R1, R2, R3, R4]
---
# fn-66-remove-unused-umpire-tooling-after.3 Trim orphaned internal codecs and verify the retained surface

## Description
Apply the second slice only after recomputing consumers, then consolidate final evidence and gates.

**Size:** M
**Files:** tools/umpire/internal/artifactv2/{runtime,evidence,result,clone}.go and clone_test.go; inventory and active compatibility/component docs
**Touches:** [tools/umpire/internal/artifactv2/**, tools/umpire/CLEANUP_INVENTORY.md, model/Umpire/Property/COMPATIBILITY.md, .plans/UMPIRE4_SPEC_COMPS.md, .plans/UMPIRE4_COMPONENTS.md]

### Approach
- Recompute symbol/import/reference closure after .2. Remove only orphan runtime/evidence/result/clone support and its five clone tests. Preserve every symbol used by the Experiment reader, generator, regression consumers and their tests, including checksum/sealing helpers.
- Keep retained decoder behavior, bytes, identities, diagnostics and comments exact. Do not introduce validation, runtime behavior, wrapper APIs or generated-output changes merely to make cleanup easier.
- Reconcile every package/command and individual deletion row against the frozen baseline. Finish active documentation: distinguish retained Lean artifact semantics and narrow Experiment decoding from retired Go codecs. Keep historical .flow records immutable.
- Freeze source before the complete serial gates. Check real retained command outcomes and existing generated-output comparisons; the complete regression target owns its existing generation/live checks. Capture actual exit codes and exact inherited live identities and lint subtraction.
- Record final evidence, including an unchanged-runtime/10x-cost explanation. Whole-spec completion review remains required after all tasks are done.

### Investigation targets
**Required:**
- `tools/umpire/CLEANUP_INVENTORY.md` — frozen and realized first-slice evidence.
- `tools/umpire/internal/artifactv2` — post-removal symbol closure and five clone tests.
- `tools/umpire/cmd/umpire-gen-regression-views/generated_view.go:12` — required generator symbols.
- `tools/umpire/regression/generated_view.go:17` — required regression reader symbols.
- `tools/umpire/regression/ci_workflow_test.go:378` — immutable ledger and promotion retention.
- `Makefile:1108` — complete live selector and final regression graph.
- `.plans/UMPIRE4_COMPONENTS.md:199` — final active codec description.

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression
```
Final serial commands: `go test -count=1 -tags test_dep ./tools/umpire/...`, `make umpire-build-model`, physical-canonical-TMPDIR `make umpire-check-regression`, `make lint-model`, `make lint-code GOLANGCI_LINT_FIX=false`. Do not duplicate generation checks already executed by the aggregate. Use integration tags only for integration tests.

## Acceptance
- [ ] Post-.2 closure proves every removed internal symbol orphaned; all Experiment consumer symbols and exact retained reader/process behavior survive, with no new wrapper or hardening.
- [ ] All 97 Test names, one Fuzz name and 27 fixture paths reconcile individually; final inventory has no unclassified ownership, unexecuted removal decision or unexplained reference. Active docs describe the surviving surface.
- [ ] Focused and complete tagged/model/regression/generation/import/lint gates pass or match explicitly verified inherited failures; full live selector and exact failure-identity set are unchanged, and retained generated output is byte-identical.
- [ ] Lint equals the original baseline minus only approved deleted-file headers. Current final expectation removes 44 headers: 1272 remain, SHA-256 `06c1dfdf2e88baf387145e49dd835c044a0e7f33bf75d75aea566f17bfc6cce3`; any new or unexplained difference fails.
- [ ] Final ledger records terminal command evidence, unchanged runtime/10x behavior, preserved comments/contracts and exact removal accounting before implementation and whole-spec reviews.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
