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
- Preserve raw lint headers and canonicalize only the two overlapping Revive identities at `catalog.go:33` and `catalog.go:39`; require identical row count and byte-exact equality everywhere else.
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
- [ ] Lint equals the original baseline minus only approved deleted-file headers. Current final expectation removes 44 headers: 1272 remain; raw expected SHA-256 is `06c1dfdf2e88baf387145e49dd835c044a0e7f33bf75d75aea566f17bfc6cce3`, raw actual may differ only where Revive emits its overlapping `unnecessary-format` or `use-errors-new` identity for the same `fmt.Errorf` call and replacement, and the narrowly canonicalized headers must equal SHA-256 `6af8054b95719a1cfbaeab137e49dfdc0b654272867910850f5f7c7f65f6b136`. Any other new or unexplained difference fails.
- [ ] Final ledger records terminal command evidence, unchanged runtime/10x behavior, preserved comments/contracts and exact removal accounting before implementation and whole-spec reviews.

## Done summary
Removed the five ledger-approved orphaned internal artifactv2 files and their five clone tests after confirming zero live consumers of the removed symbols/files. Preserved the complete Experiment reader closure unchanged, reconciled the final 56-file/97-Test/1-Fuzz/27-fixture removal, and updated the three active ownership documents to distinguish retained Lean artifact semantics from retired Go codecs.

Baseline and focused post-trim tagged tests passed. The model build, physical-TMPDIR aggregate regression, generated-view and conformance comparisons, exact inherited live identities, aggregate complete Go selector, and final model lint passed. The direct host `go test` selector encountered an inherited C header lookup failure, while the same selector passed under the repository's `mise` aggregate. Final-tree repository lint reported the expected 1,272 findings with raw SHA-256 `f53910724a8830c9fc2a58e67a3d3569e5710b8bd4242ecfc54307d9b09e439f`; only the two proven overlapping Revive identities at `catalog.go:33` and `catalog.go:39` differed. Narrow canonicalization of those aliases produced byte-identical 1,272-row expected and actual files with SHA-256 `6af8054b95719a1cfbaeab137e49dfdc0b654272867910850f5f7c7f65f6b136`. A global-rule diagnostic experiment changed unrelated identities and was discarded, leaving source and lint configuration unchanged.

Completion review found two stale prospective claims in `.plans/UMPIRE4_SPEC_COMPS.md`: an intended exact Artifact/set check and a recommended public `tools/umpire/artifact/` tree entry. Both were removed to match the retired surface. This documentation-only correction passed `git diff --check`; executable gates were not repeated.

No commit was created because the user retains commit control.

stage: impl-review - ran [2026-09-06T16:23:14Z..2026-09-06T16:31:19Z] (SHIP; continuity re-review after completion-review correction)
## Evidence
- Commits:
- Tests: baseline: TMPDIR=<physical> go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression (rc=0), post-trim: TMPDIR=<physical> go test -count=1 -tags test_dep ./tools/umpire/internal/artifactv2 ./tools/umpire/cmd/umpire-gen-regression-views ./tools/umpire/regression (rc=0), go list -tags test_dep ./tools/umpire/... (19 retained packages, 6 commands), go test -count=1 -tags test_dep ./tools/umpire/... (rc=1: inherited runtime/cgo stddef.h host-toolchain lookup failure), make umpire-build-model (rc=0, 329 jobs), TMPDIR=<physical canonical> make umpire-check-regression (rc=0: generated views, Case conformance, semantic inventory, vocabulary, exact inherited live identities, complete tagged selector and model replay passed), make lint-model (initial rc=2 from concurrent external draft; final-tree rerun rc=0), make lint-code GOLANGCI_LINT_FIX=false (final-tree rc=2: expected 1272 inherited findings; raw SHA-256 f53910724a8830c9fc2a58e67a3d3569e5710b8bd4242ecfc54307d9b09e439f), narrow Revive alias normalization: expected and actual 1272 rows, SHA-256 6af8054b95719a1cfbaeab137e49dfdc0b654272867910850f5f7c7f65f6b136, cmp=0; only catalog.go:33/39 use-errors-new vs unnecessary-format aliases canonicalized, discarded diagnostic experiment: globally disabling use-errors-new produced 1285 rows/SHA-256 0ba0abc42f34f5e92878adc2b5a2dfe0075a3f322b89ee4d3e9b8223b7a69c0f and was reverted because it changed unrelated identities, identity checks: Switch source 55f0961e02761ed6ec3718ef6d22fa4284e70e729dba7f21fabb0a3e8798bac0; Go view 8a23cdc22e53a2a9d2860522d3f1353898336f5758830ffb32d42948125300a5; Markdown view 33608f42fccedfe34309a429506778f0272eeec8acd2a136b8310c8936e46747; conformance manifest 7809b6829822c097dccb76a07e4abdba233f3a552cb553c5ab019ca1888dddb9; API manifest 613a888529b061c095c9da5ae9301154297ecd1a6459108d8c5f5b0c06551d4e; dynamic-config manifest 9f114f949e39464e61fe5993e5fe7599a9e96b78f0701c37a4cc02f05b11e970; semantic inventory e534439582339a330d32562f5c796a2a9736a7db200786653902d0c97c204d25, empty-directory check: no empty directories remain under tools/umpire, completion-review correction: removed stale exact Artifact/set-check command bullet and tools/umpire/artifact tree entry; git diff --check passed; executable gates not repeated for documentation-only change
- PRs: