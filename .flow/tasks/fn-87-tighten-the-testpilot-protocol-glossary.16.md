---
satisfies: [R15]
---
# fn-87-tighten-the-testpilot-protocol-glossary.16 A comparison with an absent operand is false; Producers drop presence checks

## Description
The semantic half of R15: in every expression context, a comparison with an absent operand evaluates to false, preparation stops demanding a presence guard for comparison operands, and Producers stop emitting `present(p)` beside a comparison on the same path. Per the spec's Edge Cases, the new rule is checked against every conformance class and live test before fixtures are regenerated; any Verdict that moves is a finding explained in the done summary.

**Size:** M
**Files:** `common/testing/testpilot/internal/ir/{expression.go,evaluate.go,expression_test.go,evaluate_test.go}`, `common/testing/testpilot/internal/verification/{correlated.go,correlated_prepare.go,evaluator.go}`, `common/testing/testpilot/internal/execution/{dataflow.go,projection.go}`, `common/testing/testpilot/preparation_error_test.go`, `model/Testpilot/Correlated.lean` (if Lean evaluates correlated comparisons), `model/Temporal/Testpilot/CaseSupport.lean`, `model/Umpire/Case/Projection/Lowering.lean` (presence atoms), `model/Umpire/Case/Correlated.lean`, `model/Temporal/Case/Evidence.lean`, typed Producers and tests, fixtures, mapping, `common/testing/testpilot/internal/execution/README.md:47-51`
**Touches:** [common/testing/testpilot/**, model/Testpilot/**, model/Temporal/**, model/Umpire/Case/**, tests/testcore/testpilot/**]

### Approach
- Semantics (record in the done summary and the spec decision note): every `compare` operator, `NOT_EQUAL` and the ordering operators included, is false when either operand is absent; `not(compare EQUAL ...)` over an absent operand is therefore true, so `NOT_EQUAL` stops being defined as `not(EQUAL)` exactly at absence. `present` is unchanged. A bare absent boolean used as a predicate (not inside a comparison) and an absent value used as an instruction input still reject at preparation ("requires an explicit presence guard"); `preparation_error_test.go:71-74` keeps pinning that.
- Go: preparation no longer requires presence facts for comparison operands (`ir/expression.go:220,228-230`, facts at `:334-337,435-469`); runtime `ir/evaluate.go:124-126` returns false instead of "absent comparison operand". Correlated: a missing evidence field or capture inside a comparison is false instead of Malformed (`verification/correlated.go:337-347`); a missing reference outside a comparison keeps its error. Evaluation stays deterministic and fail-closed otherwise (EVD-12, EVD-04: absence still never establishes success on its own; it only makes a comparison false).
- Verdict check first: before touching Producers, run the conformance corpus, `correlated.json` and the live suite with the new evaluator over the current fixtures (which still carry presence checks, so results must be identical); then remove presence checks in Producers, regenerate, and run everything again. Record both runs' results. If a Verdict moves, explain why in the summary; the recommended resolution is to keep the explicit presence check at that one Producer site (the comparison was relying on "absent is an error") and record it, rather than changing the rule.
- Producers: drop exactly a `present(p)` conjunct that sits in the same `all` as a `compare` whose operand is the same path `p` (and the same reference); an `all` left with one operand collapses to that operand. `present(observation)` alone, and presence checks guarding non-comparison uses, stay. The typed field-lowering presence atoms (`Umpire/Case/Projection/Lowering.lean`, fn-84 .5 "presence atoms are consumed") stop producing Contract presence conjuncts beside comparisons. Evidence-lift guards `all[present(path), compare(EQUAL, path, literal)]` become the comparison.
- Mapping: a validated step that matches exactly the pattern above and removes the conjunct (and collapses singleton `all`); any other `present` is left, so an unexpected removal fails the comparison.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/ir/expression.go:200-240,320-470` — presence facts
- `common/testing/testpilot/internal/ir/evaluate.go:110-140`
- `common/testing/testpilot/internal/verification/correlated.go:330-370`
- `model/Umpire/Case/Projection/Lowering.lean` — presence atoms
- `common/testing/testpilot/preparation_error_test.go:60-90`

**Optional:**
- `.plans/UMPIRE4_SPEC.md` EVD-04, EVD-12
- `common/testing/testpilot/internal/execution/README.md:47-51`

### Key context
- This is the one sanctioned Verdict-computation change in the spec (Boundaries); anything else that moves is a bug.

## Acceptance
- [ ] a comparison with an absent operand is false in Program, Contract, correlated and evidence-lift contexts (unit tests per context and per operator, including `NOT_EQUAL`); bare absent predicates and absent inputs still reject at preparation
- [ ] the new rule was run over the unchanged fixtures and after Producer changes; both runs' conformance, correlated and live results are in the done summary, and any moved Verdict is explained with its resolution
- [ ] no Producer emits `present(p)` beside a comparison on `p`; other presence checks remain
- [ ] equivalence test passes with the validated presence-removal step; `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
A comparison with an absent operand is now false in every expression context, and Producers no longer emit `present(p)` beside an equality over `p`. No Verdict moved: the regression gate passed over the unchanged fixtures and again after regeneration.

**Semantics**
- Every `compare` operator, `NOT_EQUAL` and the ordering operators included, is false when either operand is absent. So `not(EQUAL)` is true there while `NOT_EQUAL` is false.
- `ir`: `pair` binds comparison operands with `allowAbsent`, and `binary` returns false instead of "absent comparison operand".
  - A bare absent boolean and an absent instruction input still reject with "requires an explicit presence guard" (`preparation_error_test.go` unchanged).
  - A true comparison supplies no presence fact, so the default success guard keeps `present(status)`. Its comment now says why.
- Correlated, Go and `Testpilot.Correlated`: a missing evidence field or retained occurrence compares false. The Lean correlation is now a `Bool`.
  - A step that failed as "missing correlation field operand" or "missing retained capture occurrence" now fails as "correlation rejected this operation's step".
  - An `any` group can still admit the step through another operand.

**Producers**
- `Umpire/Case/Projection/Lowering.lean` (new `compared`) drops the presence check beside an equality.
  - It keeps `present(observation)`.
  - It keeps the presence check beside a negated comparison: the reject transition of an EQUAL Property and the match transition of a NOT_EQUAL one. The negation is true on an absent read, so dropping that check would turn "never established" into a violation. This is the task's recommended resolution.
- The typed Nexus lift guard `readsText` is now the comparison alone.
- Regenerated fixtures (bytes): typed-nexus 47,519 → 43,215; typed-unary 11,001 → 10,471. No other fixture changed.

**Oracle**
- The last R15 step, `dropComparedPresence`, removes exactly `present(p)` beside a `compare` whose left or right operand is `p`, and collapses a singleton `all`. Presence checks beside a negation, a different path, or inside `any` stay.
- With the step disabled, the oracle goes red on typed-nexus and typed-unary.
- Two composed-step tests changed to the collapsed shape, and the README documents the step.

**Tests**
- ir: `TestComparisonsWithAnAbsentOperandAreFalse` covers the Program and Contract contexts and all six operators, operands in either position or both, negation, the bare predicate and the absent input. It was confirmed red before the fix. The presence-fact test now observes facts through a guarded input.
- verification:
  - capture tests: two new cases for comparisons with an unassigned capture, which admit; a boolean-capture flag keeps the definite-assignment rejection covered;
  - payload tests: an arm some kind may lack is admitted in a comparison, and an evaluation case covers an event without the arm;
  - correlated tests: new cases for NOT_EQUAL over a missing operand and for a disjunction.
- execution: `TestEvidenceLiftGuardComparisonsWithAnAbsentPathAreFalse` covers every operator in the evidence-lift context (review P3). The comparison-only "unguarded absent read" lift rejection is gone.
- activation: three unit tests evaluated `finish` before `await`'s outcome existed. Its guard `status == SUCCEEDED` has no presence check, so they now assert a skipped instruction (enabled false, nil input) instead of an error. This is a hand-built Go test Case, not a Producer, and a presence check would also give a skip. The worker never evaluates out of order, so no Run changes.
- Lean `Testpilot/Tests/Fields.lean` guards were updated, with new NOT_EQUAL and disjunction guards.

**Verdict runs**
- Stage 1, new evaluator over unchanged fixtures (3560959b): `make umpire-check-regression` exit 0, 9 live identities passed. Conformance and `correlated.json` regenerated byte-identical, and the oracle passed.
- Stage 2, after Producer changes (ac348e10): exit 0, 9 live identities.
- HEAD (4941c94b): run 1 was red on known flake (c), `TestTestpilotAsyncNexusCase` SATISFIED → INCONCLUSIVE. Run 2 was exit 0 with 9 identities, and a green receipt was written.
- Flake comparison: `-count=5` of the two async-nexus tests failed 1 of 10 at base and 2 of 10 at HEAD, with the identical signature. Baseline at base was also red only on that flake. The async-nexus Case has no comparison over an operand this change affects (only step conditions).

**Gates**
- baseline: red (pre-edit `make umpire-check-regression` failed only on known live flake (c)).
- `lint-code` 161 (baseline); `lint-model` 163 (baseline). The package lint after the follow-up test found 0 issues.

**Other**
- Planning decision "Absent operands (decided in .16)" was recorded in the fn-87 spec via `spec set-plan`.
- Follow-up, not built: if a Producer ever needs to drop a presence check that an instruction input reads through, a true comparison would have to supply presence facts.

stage: impl-review - ran (claude backend, SHIP on the first round; P3 applied in 4941c94b: evidence-lift per-operator test)
## Evidence
- Commits: 3560959b9272a8bad58aa8a08728a397495ef718, ac348e109f0aa58b18737f25939eda5d71d1d576, d514322eb7bbc4fc42429ee8b00567f17377f83f, 4941c94b0fafd25bcc5a80cdda7c6a469a99ce87
- Tests: baseline: red (make umpire-check-regression failed pre-edit only on known live flake (c): TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone plain async-nexus Run INCONCLUSIVE), CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (stage 1, new evaluator over unchanged fixtures, at 3560959b: exit 0, 9 live identities), CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (stage 2, after Producer changes, at ac348e10: exit 0, 9 live identities), CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (HEAD 4941c94b: run 1 red on known flake (c) TestTestpilotAsyncNexusCase; run 2 exit 0, 9 live identities, green receipt), go test -count=5 -tags 'test_dep integration' ./tests -run '^TestTestpilot(WorkerOutageCaseLeavesAnotherQueueAlone|AsyncNexusCase)$' (base: 1 of 10 fail; HEAD: 2 of 10 fail; same SATISFIED->INCONCLUSIVE signature), go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161, baseline), make lint-model (163, baseline), golangci-lint run ./common/testing/testpilot/internal/execution/ (0 issues, after the review follow-up test)
- PRs: