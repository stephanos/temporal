---
satisfies: [R3, R4]
---
# fn-140-one-sentence-witness-queries-with.3 Replace Query expect with live and validate source-only expectation reasons

Touches: [model/umpire/Claims.scala, model/umpire/realize/Realize.scala, model/temporal/**/*.scala, model/irgen/Claims.scala, model/irgen/Realizations.scala, model/irgen/test/**, model/irgen/testdata/**]

## Description
Implement the R3/R4 expectation contract and perform its mechanical call-site migration together, so the Query rename and non-satisfied constructors leave compilable Models and fixtures. This is one expectation mechanism plus mechanically affected callers, rather than separate tasks per Model.

**Size:** M
**Files:** `model/umpire/Claims.scala`, `model/umpire/realize/Realize.scala`, `model/temporal/realize/Kit.scala`, `model/irgen/{Claims,Realizations}.scala`, existing Query/expectation callers and fixtures.

### Approach
- Rename the existing Query modifier at `Claims.scala:99` and the lift fold at `irgen/Claims.scala:333`. Keep capability fields named expect where the spec retains their binding shape.
- Make source-only text part of non-satisfied expectation values, including explicit full Runs and monitors at `Realize.scala:571-613`. A satisfied constructor offers no because argument. The existing incomplete Run admission still applies.
- Route witness, triple and capability expectations through the common admission at `irgen/Claims.scala:428`; follow helper calls and nested monitor values. Missing, empty and whitespace-only text fail with the Query or capability binding and source position.
- Explicitly omit explanatory text when `Realizations.scala:63` emits the existing descriptor-shaped expectation. Same expectation with two established explanations must emit identical IR; never add a schema field or change judge reason IDs.
- Inventory every non-satisfied source call before editing it. Source each because from the adjacent Model comment or the recorded-assessment tests, including `tools/umpire/conformance/played_test.go:258-261` for the current Activity retry. Re-anchor after the activity batch. If those sources no longer establish a reason, identify that exact binding for the owner and stop that conversion under R4.
- Capture pre-edit IR expectation records and Case standings in task scratch. The rename/reason migration alone must preserve every expectation and Case membership. Update compiler-refusal and live/total modifier-order fixtures in the same change.

### Investigation targets
**Required:**
- `model/umpire/realize/Realize.scala:571-613` - Run and monitor expectation values.
- `model/temporal/realize/Kit.scala:298-318` - convenience expectations.
- `model/irgen/Claims.scala:333-444` - shared expectation admission.
- `model/irgen/Realizations.scala:63` - descriptor emission.
- `model/irgen/testdata/lifts/Realizations.scala:534` - full monitor expectations.
**Optional:**
- `tools/umpire/conformance/played_test.go:251-261` and `tools/umpire/conformance/nexus_test.go:221` - sourced explanations.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*expect*'
rg -n '\.expect\(' model/umpire model/temporal model/irgen
```

## Acceptance
- [ ] Query `.expect` is removed and every Query caller/fixture uses live; capability expectation bindings keep their established field shape (R3).
- [ ] Compile/lift refusals cover satisfied-with-reason, absent/blank non-satisfied text, nested monitors and capability bindings; existing incomplete-Run refusals remain exercised (R4).
- [ ] Each migrated reason has a concrete Model/assessment source; unresolved explanations are reported by binding and never fabricated.
- [ ] A source-only text differential produces identical expectation IR and preserves live Case membership; no Case/assessment/schema reason field is introduced.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
