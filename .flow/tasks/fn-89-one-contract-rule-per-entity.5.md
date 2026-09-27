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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
