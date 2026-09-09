---
satisfies: [R2, R5, R8, R9]
---
# fn-77-typed-operations-parameterized-actions.11 Verify complete typed-operation compatibility and bounded qualification

## Description
Verify complete typed-operation compatibility and bounded qualification for the referenced parent requirements.

**Size:** M
**Files:** model/README.md; model/ARCHITECTURE.md; model/Umpire/ARCHITECTURE.md; model/Umpire/**/Tests*; model/Temporal/Feature/Nexus3/**; tools/umpire/cmd/umpire-gen-lean-api/**; common/testing/testpilot/**/*_test.go; tests/testpilot_async_nexus_case_test.go
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, model/Umpire/**/Tests*, model/Temporal/Feature/Nexus3/**, tools/umpire/cmd/umpire-gen-lean-api/**, common/testing/testpilot/**/*_test.go, tests/testpilot_async_nexus_case_test.go]

### Approach
- Compare final source/fixture bytes and complete raw transitive axiom sets to task1 original captures using explicit renamed/generated declaration mapping; missing/truncated evidence fails qualification.
- Run schema mutation and deterministic complete regeneration/staleness controls, preserving old canonical fixtures when semantics are unchanged; verify newly versioned identities change only on declared semantic edits.
- Exercise tenfold finite request variation, payload/collection sizes and live captures plus independent sequential/concurrent Runs; assert bounded atomic rejection and unchanged finite-versus-runtime claim scope.
- Run both real examples and all affected existing regression gates with named outcomes; retain environmental/inherited failures explicitly instead of treating skipped checks as passing.
- Update checked authoring walkthroughs and supported-form documentation, including float/recursive/enum policies, independent clauses/coverage, symbolic binding and deferred cancellation. Keep raw schema metadata distinct from exact executable values.

### Investigation targets
**Required:**
- model/README.md:90 — authoring walkthrough owner.
- model/Umpire/ARCHITECTURE.md:183 — lowering ownership.
- Makefile:1019 — complete API generation; line1035 explicit fixture rewrite.
- Makefile:1164 — current complete regression recipe.
- common/testing/testpilot/scoped_facade_test.go:189 — isolation/load pattern.

### Quick commands
`go test -count=1 -tags test_dep ./tools/umpire/cmd/umpire-gen-lean-api ./common/testing/testpilot/... ./tests/testcore/testpilot/...`
`go test -race -tags test_dep ./common/testing/testpilot/...`
`go test -json -count=1 -tags 'test_dep integration' ./tests -run '^(TestTestpilotTypedUnaryCase|TestTestpilotTypedNexusOperationsCase|TestTestpilotAsyncNexusCase|TestTestpilotAsyncNexusCaseMissingRemoteEndpoint)$' > /tmp/fn77-task11-live.jsonl && python3 -c 'import json,sys; e=[json.loads(x) for x in open(sys.argv[1])]; names=sys.argv[2:]; assert all(any(v.get("Test")==n and v.get("Action")=="run" for v in e) and any(v.get("Test")==n and v.get("Action")=="pass" for v in e) for n in names); assert not any(v.get("Action") in ("skip","fail") for v in e)' /tmp/fn77-task11-live.jsonl TestTestpilotTypedUnaryCase TestTestpilotTypedNexusOperationsCase TestTestpilotAsyncNexusCase TestTestpilotAsyncNexusCaseMissingRemoteEndpoint`
`make umpire-build-model`
`make lint-model`
`make umpire-check-regression`
`make lint-code GOLANGCI_LINT_FIX=false`

### Execution constraints
Read the full parent spec. Preserve existing comments and unrelated dirty source. New paths listed here are proposed owners; reuse an established equivalent before creating one. Run Lean jobs serially; fn76 lint recovery is complete. Preserve fn75 semantic import isolation and fn78 obligation/evidence authority. No operation cancellation, new dependency/toolchain, or implicit fixture promotion. Do not stage, commit, or push; the user owns commits. Capture task-local before/after trust for changed load-bearing declarations; task11 additionally compares to the original task1 substrate.

### Environment note
`go test` in this checkout needs `CC=/usr/bin/cc`: mise's lean4 clang shadows the toolchain and cgo
fails with `stddef.h not found`. Every Go gate on this branch needs it.

### Follow-up inherited from task .9
The runtime rule carries only `pending` and `satisfied`, so a started event recording a different
workflow type leaves the rule inconclusive where the model Property distinguishes violated from
inconclusive. Adding a `violated` state plus a tampered-fixture live assertion is a real R6
improvement; task .9's review raised it as a non-blocking P3 and assigned it here or to .10.

### Known findings introduced by this spec
`make lint-model` is red at 171 findings: 169 in the generated `Temporal/API/{Types,Proto}.lean`,
which are pre-existing and not this spec's, plus the two below. An earlier version of this note said
"exactly 2" and undercounted by omitting the generated-API findings; treat 171 as the baseline. Only
the two below were introduced by this spec, so only they may not be recorded as inherited:
- `unusedArguments` on `Umpire.instReprPropertyFieldProjection` (`model/Umpire/Property/Evaluation.lean:470`)
  — from task .5's `deriving Repr`; no consumer of that instance exists.
- `simpNF` on `Umpire.Operation.CheckedRpc.mk.injEq` (`model/Umpire/Operation.lean:58`) — from task .1;
  the structure's only non-proof field makes the generated lemma simp-provable.

## Acceptance
- [ ] Original before/after raw trust inventories have no unapproved expansion or missing entries; unchanged canonical/fixture/identity bytes are preserved and new schema/parameter changes follow checked versioning.
- [ ] Tenfold variation/payload/collection/capture tests reject within declared bounds without cross-Run leaks, resets or broader exhaustive claims.
- [ ] Both real Driver examples and affected generator/semantic/codec/evaluator/functional/staleness gates have exact terminal receipts; any baseline failure is identified and compared rather than called clean.
- [ ] Documentation and executable walkthroughs report actual exact/unsupported forms, both examples, source clause coverage and finite/runtime scope; Known Gaps waive no requested requirement.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
