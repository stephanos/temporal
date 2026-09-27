---
satisfies: [R8, R10]
---
# fn-89-one-contract-rule-per-entity.5 Instance-value conformance sub-entry and umpire-assess rule set

## Description
Add the `static-preparation-rejection/instance-value` conformance sub-entry (R10's corpus half) and derive `umpire-assess`'s expected rule IDs from Rule instances (R8). Both are small Go/Lean consumers of the landed surface; combined into one task. It follows task 4 because the sub-entry is written by the same generator that regenerates the functional fixtures, so running it beside task 4 would conflict on generated files.

**Size:** S/M
**Files:** `model/Temporal/Testpilot/Conformance.lean`, `model/Temporal/Tool/Testpilot.lean`, `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go`, `common/testing/testpilot/conformance_test.go` (if sub-entries are listed there), `common/testing/testpilot/testdata/case-runtime-conformance/static-preparation-rejection/instance-value/**` (generated), `tools/umpire/evaluation/admission.go`, `tools/umpire/evaluation/admission_test.go`
**Touches:** [model/Temporal/Testpilot/Conformance.lean, model/Temporal/Tool/Testpilot.lean, tools/umpire/cmd/umpire-gen-case-runtime-conformance/**, common/testing/testpilot/conformance_test.go, common/testing/testpilot/testdata/case-runtime-conformance/**, tools/umpire/evaluation/**]

## Approach
- Sub-entry: author a Case in `Conformance.lean` beside `conformanceDuplicateEvidenceRejectionCase`, with a Rule whose instance assigns a value of the wrong type (the most representative R2 rejection for a non-Lean Producer), built with task 1's Authoring constructors. Wire it in `Tool/Testpilot.lean:73-88` as `conformance-static-preparation-rejection-instance-value` and add `typedRejectionEntry("instance-value", <category>, <path>)` to `productionManifest` (`generate.go:584-613`) with the category and by-ID location task 2's preparation produces (`contract.rules[<rule_id>].instances[<instance rule id>].assignments[<id>]`).
- Regenerate through `make umpire-gen-case-runtime-conformance`; only the new sub-entry appears in `git status`.
- `ruleSetCrossed` (`tools/umpire/evaluation/admission.go:211`): expected IDs are, per Contract Rule, its instances' rule IDs when it has instances, else its own ID; correlated rules as today. Tests in `admission_test.go`: a Verdict naming the Rule's own ID instead of its instances, and one missing an instance, are both reported; the regenerated pair Case's Verdict is accepted.

## Investigation targets
**Required**:
- `tools/umpire/cmd/umpire-gen-case-runtime-conformance/generate.go:584-660`
- `model/Temporal/Testpilot/Conformance.lean` — existing rejection Cases
- `model/Temporal/Tool/Testpilot.lean:60-95`
- `tools/umpire/evaluation/admission.go:180-252`

## Key context
- Conformance bytes come from the Lean encoder, never from Go `protojson.Marshal`.
- `.flow/memory/bug/integration/moved-conformance-tests-must-not-import-2026-09-06.md`: conformance tests must not import functional adapters.

### Carried from fn-89.3 (2026-09-27)
- R3 gap at Case level: `execution.Prepare` checks the Case's surface size as written, not as expanded, so a very large instanced Case could pass where its expansion is rejected. Charge it as expanded (like `ir.CheckExpandedSurface` for the Contract) and cover it in the corpus sub-entry.
- The production expansion helpers in `verification/prepare.go` nearly duplicate the test's `expand`/`inline`; share one so they cannot drift. Add `t.Helper()` to `nexusWorld` and `nexusRun`.

- `make lint-code-fast` reports one finding at `common/testing/testpilot/internal/ir/expression_test.go:370` (from fn-89.1/.3); fix it here.

## Acceptance
- [ ] the `instance-value` sub-entry exists, generated only by `make umpire-gen-case-runtime-conformance`, and rejects at the pinned category and location
- [ ] every other conformance entry and fixture is byte-identical; `make umpire-check-case-runtime-conformance` passes
- [ ] `umpire-assess` expects instance rule IDs; both negative cases are tested
- [ ] `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/...` passes; `make lint-code-fast` clean on changed packages

## Done summary
Added the `static-preparation-rejection/instance-value` conformance sub-entry (R10). Its Rule's second instance, `result-2`, assigns an integer to the text instance value `instruction`. Preparation rejects it as `type_mismatch` at `contract.rules[result].instances[result-2].assignments[instruction]`; a throwaway probe confirmed the Case is otherwise admitted. `umpire-assess`'s `ruleSetCrossed` now expects each Rule instance's rule ID instead of the Rule's own ID (R8). `TestRuleSetExpectsRuleInstances` accepts the pair Case's rule set and reports both negative cases: the own ID named instead of the instances, and an instance missing.

Carried items:
- `execution.Prepare` now charges the Case surface as expanded; see `TestPrepareBoundsTheCaseSurfaceAsExpanded`, which was confirmed red first.
- The expansion helper now lives once, as `ir.ExpandRule` / `ir.ExpandRuleInstances`, and the test's `expand` uses it.
- `nexusWorld` and `nexusRun` now call `t.Helper()`.
- The lint finding in `expression_test.go` is fixed by renaming the loop variable.

Deviation: the Case-level expanded-size check is covered by a Go unit test, not by a corpus fixture. A fixture over 16 MiB would break the corpus's small-fixture rule.

The manifest-length pin in `generate_test.go` moved from 16 to 17 by declared intent. The Driver catalog is unchanged, so no pinned Runs were re-recorded.

`make umpire-check-regression` is red, and the cause is inherited. The concurrent fn-88.5 commits `ebccb6940a` and `e187d2ccfa` make `make -n umpire-check-regression` fail in `umpire-check-veil-pin`, which breaks `TestUmpireCIWorkflowRunsSeparatedUnitAndLiveProofs`. Every other sub-gate of the regression target passes.

Follow-ups (reviewer P3s):
1. Share an `ir.HasRuleInstances` gate between `execution` and `verification`.
2. Record that the corpus does not pin expanded-size charging.

stage: impl-review - ran (claude, first-pass SHIP, base f559ed9566)
## Evidence
- Commits: e33c6be0529fb6b6668cd1d1d563834d7218c21d
- Tests: baseline: green (go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/...; make umpire-check-case-runtime-conformance canary-check-case), go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/evaluation/... ./tools/umpire/cmd/umpire-gen-case-runtime-conformance/, make umpire-gen-case-runtime-conformance (git status: only the new instance-value sub-entry), make umpire-check-case-runtime-conformance canary-check-case, make lint-code-fast GOLANGCI_LINT_FIX=false (0 issues), LEAN_NUM_THREADS=1 make lint-model (rc=0), make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance canary-check-case umpire-check-inventory umpire-check-retired-vocabulary umpire-check-live-tests (rc=0), TMPDIR=... make umpire-check-regression: red, inherited from concurrent fn-88.5 commits ebccb6940a/e187d2ccfa (umpire-check-veil-pin under make -n fails TestUmpireCIWorkflowRunsSeparatedUnitAndLiveProofs); every other regression sub-gate green
- PRs: