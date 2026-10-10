---
satisfies: [R3, R4, R6]
---
# fn-153-checked-property-examples-pilot.3 Classify illustrations through existing Property reads

## Description
Bind each illustration's Property directly and return the checked classification or a located error. Prove the vertical path using .2's fixtures and the existing Go predicate implementation.

**Size:** M
**Files:** `tools/umpire/check/claims.go`, new `tools/umpire/check/illustrations.go`, new `tools/umpire/check/illustrations_test.go`, existing checker integration tests as needed
**Touches:** [tools/umpire/check/claims.go, tools/umpire/check/illustrations.go, tools/umpire/check/*_test.go]

### Approach
- Reuse `propertyReads`, `boundProperty`, `BoundProperty.About/Holds`, `keyedStep` and the existing error-preserving decision path at `claims.go:393,445,481,518,634`. Add direct resolved-Property access without fabricating Queries or requiring a reachable interpreter row.
- Apply owning-domain admission before evaluating any supplied values. Honor sequential binding restrictions. Call About before Holds; return not applicable without reading a predicate that would error. Distinguish predicate holes and evaluation errors from a false predicate.
- Compare only authored satisfied/violated expectations. Emit Property, label, source Position, expected and actual result or nested error. Expose structured checked inputs/results to .4's renderer; keep presentation independent of binding.
- Test a well-typed forbidden result that cannot be reached, an unrelated-action probe and an unqueried Property. Keep illustrative violations in their own result collection. Jointly exercise a genuine model-check failure and successful negative illustration so neither suppresses the other.
- Use local always-true/always-false predicate substitutions on immutable admitted fixtures to test mismatch detection. Mutation controls must call the production classifier and fail for expected-versus-actual classification, with predicates restored and no altered production definitions.

### Investigation targets
**Required:**
- `tools/umpire/check/claims.go:393` - BoundProperty methods and sequential-use contract
- `tools/umpire/check/claims.go:445` - existing binder and evaluation errors
- `tools/umpire/check/claims.go:634` - Property reads independent of Queries
- `tools/umpire/check/activity_properties_test.go:397` - reachable-row harness, not a reachability oracle for supplied steps
- `tools/umpire/check/activity_properties_test.go:1782` - controlled predicate mutation precedent
**Optional:**
- `tools/umpire/check/choices_test.go:97` - immutable fixture/check patterns

### Quick commands
```bash
go test -tags test_dep ./tools/umpire/check -run 'Property|Illustration'
```

## Acceptance
- [ ] The fixture vertical slice uses the production Property selector/predicate without a Query or second evaluator and returns all three classifications.
- [ ] An unrelated action never invokes Holds, including a predicate that would error; errors and holes never pass a negative expectation.
- [ ] Unqueried and individually refining-machine Properties work; valid unreachable inputs remain illustrations with no reachable-witness claim.
- [ ] Mismatch/error diagnostics retain exact owner, label and nested source location; expected illustrative violations cannot mask a genuine model failure.
- [ ] Always-true and always-false controls fail the appropriate admitted fixtures for classification mismatch, and focused tests pass with the original predicates restored.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
