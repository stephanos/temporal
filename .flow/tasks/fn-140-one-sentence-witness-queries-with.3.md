---
satisfies: [R3, R4]
---
# fn-140-one-sentence-witness-queries-with.3 Replace Query expect with live and validate source-only expectation reasons

## Description
**Size:** M
**Files:** Query/expectation authoring, source-only reason admission, Scala callers/fixtures.
**Touches:** [model/framework/Claims.scala, model/framework/realize/Realize.scala, model/framework/*.test.scala, model/temporal/**/*.scala, model/irgen/Claims.scala, model/irgen/Realizations.scala, model/irgen/Capabilities.scala, model/irgen/test/**, model/irgen/testdata/**]

Finish and verify the Scala half of R3/R4 after .2. The integrated source already uses `.live` and carries because text; audit and complete it rather than reintroducing `.expect` or rewriting unrelated implementation. Original Go Case-generation/membership obligations are retained in deferred task .8; source-only proofs do not satisfy that consumer task.

### Approach
- Audit the existing Query modifier and lift fold, all machine/derived/composition/witness modifiers and every caller/fixture. Capability fields named expect keep their binding shape. Update stale explanatory comments about the renamed Query modifier where owned by this task, without renaming unrelated fixture helper assertions called expect.
- Inventory every non-satisfied Run and nested monitor expectation, including generic capability binding/helper paths. Missing, empty and whitespace-only source explanations refuse with the Query or capability binding and authored position. Satisfied convenience constructors and explicitly satisfied full-Run/monitor authoring forms reject a supplied explanation at compile time, as R4 requires. The current unconstrained case-class because field does not prove that protection; close this actual API gap without hiding it behind convenience-only tests or adding a schema field. Generic non-satisfied helpers and all core expectation fields remain expressive.
- Source each migrated reason from its adjacent Model comment or recorded assessment tests; inspect `tools/umpire/conformance/played_test.go:258-261` read-only for current Activity retry and consume actual fn-155/fn-156 mapping/closure. Never fabricate an explanation. Unestablished reasons are reported by binding before conversion.
- Preserve existing incomplete-Run admission. Follow helper calls and nested monitor values through shared expectation admission; test absent/blank/whitespace text and valid helper-sourced text, including capabilities.
- Ensure `model/irgen/Realizations.scala` explicitly omits source-only text from descriptor emission. Two established explanations of one expectation emit identical full expectation IR, without a Case/schema/assessment reason field. Capture source-level expected-run presence and all Query identities for the deferred Case-selection proof; actual Go manifest verification stays in .8.
- Add focused reason/expectation tests that execute a nonempty suite; the current Fixtures setup eagerly prepares unrelated populations, so a dedicated suite in its existing file may isolate iteration, without discarding the canonical Quick check. Update live/total modifier-order and compiler refusal fixtures.

### Investigation targets
**Required:** `model/framework/realize/Realize.scala:571-621`; `model/temporal/realize/Kit.scala:298-322`; `model/irgen/Claims.scala` expectedRun; `model/irgen/Realizations.scala` explained/descriptor emission; `model/irgen/Capabilities.scala` expectedRun binding path; `model/irgen/testdata/lifts/Realizations.scala` monitor expectations.
**Optional:** `tools/umpire/conformance/played_test.go:251-261`, `tools/umpire/conformance/nexus_test.go:221` (read-only explanation evidence).

### Quick commands
```sh
mise exec -- scala-cli test model/irgen --test-only '*Fixtures*' -- --tests '*expect*'
make lint-model-syntax
rg -n '\.expect\(' model/framework model/temporal model/irgen
```

An empty rg search is the desired absence (exit1), not a failed test. Confirm the fixture filter selects real expectation assertions and add a focused suite if it does not. Serialize heavy commands; no Go commands/changes, production managed regeneration, full gates/review/live.
## Acceptance
- [ ] Query `.expect` is removed and all Query callers/fixtures use live, including typed witness modifier chains; retained capability binding fields stay unchanged (R3).
- [ ] Compile refusals cover supplied explanations on satisfied convenience and explicitly satisfied full-Run/monitor authoring forms; lift refusals cover absent/blank/whitespace non-satisfied text, helpers, nested monitors and capability bindings. Existing incomplete-Run refusals remain exercised (R4).
- [ ] Every production reason has a concrete Model/assessment source; unresolved bindings are reported and never fabricated (R4).
- [ ] Source-text differential yields identical full expectation IR and unchanged Query identities/expected-run presence, with a retained handoff to deferred .8 for actual manifest/Case proof.
- [ ] Nonempty focused Scala expectation tests and Quick checks pass; no Go Case membership, schema or shared closure credit is claimed.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
