---
satisfies: [R2, R3]
---
# fn-89-one-contract-rule-per-entity.2 Go preparation: bind a Rule with instances once, admission equal to the expansion

## Description
Admit Rule instances in Go preparation (R2) and make admission equal to the expansion's (R3's admission half). Binding and capture analysis run once per Rule; each instance's assignments are checked against the declared types; every ceiling is charged per instance. The prepared form carries, per Rule, its bound machine and its ordered instances (rule ID plus assigned values), which the Evaluator task consumes. Split from the Evaluator so each diff stays reviewable; the Evaluator task owns the per-instance runtime and the differential proof.

**Size:** M
**Files:** `common/testing/testpilot/internal/ir/expression.go`, `common/testing/testpilot/internal/ir/expression_test.go`, `common/testing/testpilot/internal/verification/prepare.go`, `common/testing/testpilot/internal/verification/captures.go`, `common/testing/testpilot/internal/verification/correlated_prepare.go` (shared name set only), `common/testing/testpilot/internal/verification/prepare_test.go`, `common/testing/testpilot/internal/execution/prepare_test.go` and `common/testing/testpilot/internal/verification/correlated_test.go` (their out-of-context reference lists gain the new arm)
**Touches:** [common/testing/testpilot/internal/ir/**, common/testing/testpilot/internal/verification/prepare.go, common/testing/testpilot/internal/verification/captures.go, common/testing/testpilot/internal/verification/correlated_prepare.go, common/testing/testpilot/internal/verification/prepare_test.go, common/testing/testpilot/internal/execution/prepare_test.go, common/testing/testpilot/internal/verification/correlated_test.go]

## Approach
- `ir`: add a `ReferenceKind` for instance values (`expression.go:14-28`), admit it only in `ContractContext` in `admittedReferences` (`:58-63`), compile it in `compiler.reference` (`:364+`). It resolves through the scope like a capture but is always available (`Available: true` in `scopeFor`, `captures.go:44`), so `equal(path, instanceValue)` needs no presence guard, exactly like a literal. It carries its declared type; unlike a literal (which `pair()` binds with `expected = a.typ`) it does not adapt, so reject a use whose expected type differs from the declared type (spec R2). Extend the Program and correlated context tests (`execution/prepare_test.go:76`, `verification/correlated_test.go:225`) with the new out-of-context arm. Extend `TestExpressionContextsRejectReferencesOutsideThem` (`expression_test.go:349`).
- `prepare.go`: validate the declarations and every instance before binding (all R2 error cases, in the spec's by-ID location grammar: `contract.rules[<rule_id>].instance_values[<id>]`, `contract.rules[<rule_id>].instances[<instance rule id>].assignments[<id>]`). Enum assignment values go through the catalog's literal check (undefined enum value, `catalog.go:163`). Text, integer or enum only; boolean is rejected (capture analysis prunes on boolean literals in `refine`, `captures.go:202`, which a once-analyzed Rule cannot do per instance). Reuse the singular-type classification the capture types use.
- Locations of reference errors: `bind` reports an empty or undeclared scope reference at the unlocated path `expression` (`ir/expression.go:301`). Do not change that shared behavior; instead walk each transition predicate in `prepare.go` before binding and reject an empty or undeclared `instance_value_id` at the predicate's location.
- Names: put the Rule's own ID and every instance ID into the one `seen` set (`prepare.go:145-156`) that `bindCorrelated` already shares (`correlated_prepare.go:87,257`), so correlated IDs are checked against instance IDs too.
- Ceilings (spec Architecture, Preparation): `MaxRules` against the total instance count (a Rule with no instances counts one) before any per-instance allocation (`prepare.go:127`); states/transitions (`bindMachine` `:168-175`), captures (`bindCaptures`), binding work (`a.charge`, `prepare.go:210`, `captures.go:86`) and per-event/total work (`boundWork`, `captures.go:237`) multiplied per instance. Charge each instance value reference what the inlined literal would cost: in binding, the literal check (`type.go:198`) and the value surface; in `expressionWork` (`captures.go:267-270`) the proto size of that instance's assigned value. Since that differs per instance, compute the per-event bound per instance and charge each. The target is that an instanced Contract and its expansion reject on the same ceiling or are both admitted.
- Error wrapping: `bindMachine` failures keep the `rule %s:` prefix (`prepare.go:152-153`) naming the Rule; an assignment failure names the instance.
- Tests (`prepare_test.go`): one table case per R2 error with its expected location (boolean type and declared/expected type mismatch included); admission of a one-instance Rule and of an unread declared value; a ceiling case where the instanced Contract and a hand-built expansion both reject naming the same ceiling, and one where both admit.

## Investigation targets
**Required**:
- `common/testing/testpilot/internal/verification/prepare.go:114-215,243,298-387`
- `common/testing/testpilot/internal/verification/captures.go:13-100,237-270`
- `common/testing/testpilot/internal/ir/expression.go:14-63,364-420,642-660`
- `common/testing/testpilot/internal/verification/correlated_prepare.go:87,257`

**Optional**:
- `.flow/memory/bug/integration/contract-work-bounds-must-follow-typed-2026-09-04.md` — work bounds must follow typed values

## Acceptance
- [ ] every R2 error case rejects before Driver I/O at the stated location, each with a test
- [ ] a one-instance Rule and an unread declared value are admitted
- [ ] instance IDs, Rule IDs and correlated IDs share one uniqueness check
- [ ] ceilings are charged per instance (binding and literal-size work included) and a test shows instanced and expanded Contracts reject on the same ceiling
- [ ] `go test -count=1 -tags test_dep ./common/testing/testpilot/internal/...` passes; `make lint-code-fast` clean on changed packages

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
