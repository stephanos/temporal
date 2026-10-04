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
Reduced the lifter fixtures that copied or imported live Model text to fixture-local Models (owner request 2026-10-04). Commits: 487ad43c8e (reduction), c0bddb361d (lint fixes), fcea872750 (comment). Merges first: 83181324fc and d68960a50b (umpire with fn-122.1-.4, fn-114.12 and fn-122.8; conflicts were fn-114.11's dropped annotations, resolved by taking umpire's structure without the annotations inference reproduces; gen-model byte-identical; merge gates green at d68960a50b: Go suite, lint-model, lint-code-fast).

**Fixtures** (lines before -> after; mapping of every dropped construct in `.flow/tmp/fn114-10/mapping.md`)
- Realizations.scala 1104 -> 1009; expected/realizations.json 14255 -> 3932. learnedRun and pauseRace run over fixture-local machines; nothing is imported from nexuscaller, standaloneactivity or the Admission fixture; the live root syncCompletion left the roots (and its source_root_moves entry left the golden config). heldByValue/heldByName watch a fixture-local door. Door, errand and tally are unchanged.
- Admission.scala 308 -> 370; admission.json 6405 -> 5167. It declares a 77-line fixture-local product instead of importing the live one (the drift source); every design, verdict, row and witness ~45 Go tests pin is unchanged, only product IDs, family and line pins moved.
- CloseReset.scala (325) retired with closereset.json: the live close policy carries all three designs, nexus_close_test.go pins N1-N5. Dropped without replacement: the sketch's free-search explored counts (61/76/71), oracles of a specimen document that no longer exists.
- Typed.scala 449 -> 186 (keeps the run_id bind, the mapped scalar read, a Long through a helper, an Option.map evidence field, the two refusal roots); Members.scala 106 -> 93 and Inputs.scala 119 -> 126 drop their comparisons with live Properties and the live protocol state (Inputs declares Stage/Deadline; totals 720/360).
- Negative fixtures unchanged.

**Overlap check**: `model/lifter/test/Overlap.test.scala` fails when a fixture shares more than 6 substantive lines, in runs of 4 or more consecutive substantive lines that stand in one model/temporal file. After: every fixture 0 except Scripts.scala 6 (its core records spell the Temporal kit's six correlation defaults, the oracle of the helper) and Capabilities.scala 4. Shorter runs are shared vocabulary (`enum Outcome derives Finite:` ...).

**Golden harnesses**: R1's frozen golden set held admission/closereset/realizations. original.json gains the closed `reduced_fixtures` list, read by both harnesses (original baseline and migration goldens): a listed archived lifter fixture is compared with no frozen original and may be retired; validation refuses ir/, cases/, rejects, unknown and duplicate keys; tests cover both directions. Archive and goldens untouched; model/ir and model/cases compared exactly. The spec's R1 allowed list and Decision Context record the amendment.

**Decisions taken autonomously**
- Merged umpire early (before the conductor's go) to rewrite the fixtures on fn-122's versions; then merged again on the conductor's request.
- Retired CloseReset rather than reduce it: nothing it asserted lacked live or fixture coverage.
- Admission keeps its design names and structure and grows a local product rather than shrinking: the Go semantic tests need the frozen system, not the live record.
- Threshold 6 at run length 4, so Scripts' kit-default oracle stays.
- Out of the task's Files list but required: tools/umpire/internal/golden/**, tools/umpire/{model,lower} tests that pin the fixtures (line pins, IDs, counts).

**Gates** (after commit): make umpire-gen-model (model/ir, model/cases, expected byte-identical), OriginalBaseline, full Go tooling suite, lint-model, lint-code-fast, golden/migration tests — all exit 0 (`.flow/tmp/fn114-10/gates.txt`).

**Review**: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the same family (Opus). Round 1 SHIP. Deferred P3: runMachine/raceMachine restate doorMachine's body (as tallyMachine already did). FYIs: the Overlap comment now says runs are counted across live files (fixed, fcea872750); config.json keeps the now-unused CloseReset path/label substitutions, which describe the frozen archive; Admission keeps cases no step takes so totals stay.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 83181324fc, d68960a50b, 487ad43c8e, c0bddb361d, fcea872750
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 557s; model/ir, model/cases, lifts/expected unchanged), mise exec -- go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 121s), mise exec -- go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 264s, 46 packages ok), mise exec -- make lint-model (exit 0, 262s), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0, 336s, 0 issues), mise exec -- go test -tags test_dep -count=1 -p 2 -run 'Original|Migration|Golden|Inventory|Reduced' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 315s), mise exec -- scala-cli test model/lifter --test-only umpire.lift.Overlap (exit 0), merge d68960a50b in a clean worktree: Go suite (exit 0, 1486s), lint-model (exit 0), lint-code-fast (exit 0)
- PRs: