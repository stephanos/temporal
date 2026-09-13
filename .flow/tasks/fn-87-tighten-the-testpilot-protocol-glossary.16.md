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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
