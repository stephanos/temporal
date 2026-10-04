# fn-114-state-every-scala-model-declaration-once.10 Reduce copied Model text in the lifter fixtures to minimal fixture-local Models

## Description
Owner request (2026-10-04): the lifter fixtures under `model/lifter/testdata/lifts/` copy live Model code that drifts whenever a Model changes, and they re-test what the model gate already covers by lifting every live Model against `model/ir/*.json`. Measured on 2026-10-04: `Realizations.scala` (1,084 lines, 86 substantive lines shared with live realizations), `Admission.scala` (294 lines, a copy of the activity specimen's reviewed admission block, 75 shared), `CloseReset.scala` (314 lines, 61 shared with the close policy) and `Typed.scala` (447 lines, 25 shared).

**Entry gate:** after fn-114.7 (fixtures restated in fn-114.6, `Spelled.scala` retired by fn-114.7), before fn-114.9 (folder rename), so each fixture is rewritten once.

**Size:** M
**Files:** `model/lifter/testdata/lifts/**` and `expected/**`, `model/lifter/test/**`, `.flow/tmp/fn114-10/**`.
**Touches:** [model/lifter/testdata/**, model/lifter/test/**, .flow/tmp/fn114-10/**]

### Approach
- For each copied fixture, list the lifter constructs its tests assert (by expected-IR fields and test names). Reduce the fixture to a minimal Model that exercises exactly those constructs, with fixture-local names, instead of copied Model text. Where a construct is already covered by the gate's lift of a live Model, record that coverage and drop the fixture case.
- Keep the negative fixtures (`Rejects.scala`, `werror/`, `*Invalid/`, `nonfinite/`, `unsupported/`, `crossed/`) and any spelling-equivalence pair still needed after fn-114.7.
- Regenerate `expected/*.json` only for the reduced fixtures; the change to each is reviewed as a removal of copied text, not a lifter behavior change.
- Add a check that fails when a lifter fixture shares more than a small, stated number of substantive lines with `model/temporal/**` (the measurement script from this task, kept as a test).

### Investigation targets
**Required:** `model/lifter/testdata/lifts/{Realizations,Admission,CloseReset,Typed}.scala`, `model/lifter/testdata/lifts/expected/`, `model/lifter/test/Fixtures.test.scala`.

## Acceptance
- [ ] Each lifter fixture contains only fixture-local Model text; the overlap check reports at most the stated threshold of substantive lines shared with `model/temporal/**`, and the check runs in the lifter tests.
- [ ] Every lifter construct the removed text exercised is still asserted, either by a reduced fixture or by the gate's lift of a live Model; a mapping in `.flow/tmp/fn114-10/` lists each.
- [ ] Negative fixtures are unchanged; lifter tests, `make umpire-gen-model` (no `model/ir/**` or `model/cases/**` change), the original-baseline check and the model gate pass.
- [ ] Fixture line counts before and after are recorded.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
