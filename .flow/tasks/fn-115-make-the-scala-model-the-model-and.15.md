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
Dropped the `Scala` qualifier from live Go test names, files and helpers: four test files (`testpilot_generated_test.go`, `testpilot_exploration_test.go`, `testpilot_activity_control_test.go`, `testpilot_activity_canary_test.go`), `tests/testcore/testpilot/model_fixture{,_test}.go`, four tests (e.g. `TestTestpilotGeneratedCases`, `TestModelCasesLowerForExistingConsumers`) and their helpers (`ModelCase`, `LoadModelCase`, `LoadGeneratedCase`, `GeneratedCases`). Names-only; references in `tools/canary/preflight/harness_test.go` and `model/README.md` follow, and the Makefile and CI select by prefix, so they needed no change.

Kept: the two tests where Scala is the subject (`TestValidateReportsEveryProblemAtItsScalaPosition`, `TestALoweredCaseCarriesWhatTheScalaDeclares`), `Scalar*` identifiers, golden path keys, and runtime values the tests send (`Identity: "scala"`, the `scala-` namespaces, and the `scala-discovery` campaign Profile identity), which are test inputs and recorded data, not names.

`make umpire-check-live-tests` passes with the same 81 identities; the canary-harness test passes separately; focused tests, the functional compile and `lint-code-fast` pass. Independent review (Claude Fable, fresh context) returned SHIP in round 1. Handover: .flow/tmp/fn115-15-summary.md; evidence: .flow/tmp/fn115-15-evidence.json; review: .flow/tmp/fn115-15-review/round1-review.md. No agent commits.
## Evidence
- Commits:
- Tests: CC=/usr/bin/clang mise exec -- go vet -tags 'test_dep integration canary_harness' ./tests/testcore/testpilot ./tools/canary/preflight, CC=/usr/bin/clang mise exec -- go test -tags "test_dep integration canary_harness" -run '^$' ./tests, CC=/usr/bin/clang mise exec -- go test -count=1 -tags 'test_dep canary_harness' ./tests/testcore/testpilot ./tools/canary/preflight, CC=/usr/bin/clang make umpire-check-live-tests (81 passing identities, before 81), cd tests && go test -count=1 -tags 'test_dep integration canary_harness' -run '^TestTestpilotActivitySharedWithCanary$' . (PASS), GOLANGCI_LINT_FIX=false CC=/usr/bin/clang mise exec -- make lint-code-fast (0 issues), Independent review round 1 SHIP (claude-fable-5-1); .flow/tmp/fn115-15-review/round1-review.md
- PRs: