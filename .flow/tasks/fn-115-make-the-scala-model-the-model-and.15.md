---
satisfies: [R16]
---
# fn-115-make-the-scala-model-the-model-and.15 Drop the Scala qualifier from live Go test names

## Description
Drop the `Scala` qualifier from live Go test names, files and helpers. Implements R16's naming rule ("Names that mark history (`v2`, `0`, `new`, `scala` as a qualifier on a Go test) appear only on the archives"), found open by the task 13 review. Conductor decision under the owner's delegation: rename rather than record a deviation.

**Size:** S
**Files:** tests/testpilot_scala_*_test.go, tests/testcore/testpilot/scala_fixture*.go, tools/umpire/lower tests naming Scala Cases, the live-test identity list of `make umpire-check-live-tests`, citations in model/README.md and tools/canary/README.md
**Touches:** [tests/**, tools/umpire/**, tools/canary/**, model/README.md, Makefile]

### Approach
- Rename tests, files and helpers for what they check, e.g. `TestTestpilotScalaGeneratedCases` → `TestTestpilotGeneratedCases`, `testpilot_scala_generated_test.go` → `testpilot_generated_test.go`, `scala_fixture.go` → `model_fixture.go`, `ScalaManifest` → a name for what it is. Find every live occurrence with a search over `Scala` in Go identifiers and file names outside the archives.
- Persisted identity strings stay (`scala.`, `scala.explore.`, `scala-model`, Case IDs such as `temporal.case.scala.*`): they are compatibility data, not names.
- Names where Scala is the subject rather than a history marker stay, for example `TestValidateReportsEveryProblemAtItsScalaPosition` and `TestALoweredCaseCarriesWhatTheScalaDeclares`; list them in the done summary. `Scalar*` identifiers are not qualifiers.
- Update every reference: the live-test identity list and its nonzero floor, CI selectors, Make `-run` patterns, docs citing the tests.

## Acceptance
- [ ] No live Go test, test file or helper name uses `Scala`/`scala` as a qualifier; persisted identity strings are unchanged.
- [ ] Every reference (live-test identity list, Make/CI selectors, docs) follows, and `make umpire-check-live-tests` runs the same number of tests as before.
- [ ] Focused Go tests, the functional compile and `lint-code-fast` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
