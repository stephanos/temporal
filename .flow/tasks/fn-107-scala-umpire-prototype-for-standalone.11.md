---
satisfies: [R1, R7, R8, R10]
---
# fn-107-scala-umpire-prototype-for-standalone.11 Close the exploration, regression, and trace-inspection loop

Touches: [model/scalav2/scala/temporal/**, model/scalav2/explore/**, model/scalav2/README.md, model/scalav2/SEMANTICS.md, tools/umpire/replay/**, tests/testpilot_nexus_control_case_test.go]

## Description
Connect Scala variation/reduction declarations to existing exploration/replay and produce the final bounded demonstration artifacts.

**Size:** M
**Files:** proposed IR exploration/replay adapter and trace renderer; existing replay bridge seam/tests; prototype README/SEMANTICS and fixtures.

### Approach
- Reuse generic exploration, rerun, minimization, and proposal algorithms. Scala selects parameter classes, priorities, and legal reductions.
- Discover an unpinned bounded execution. Use the existing forged-completion runtime control to demonstrate reproduction/minimization if the real activity implementation conforms.
- Preserve DAG dependencies, learned values, scripts, and failure identity while reducing. Emit byte-stable check-in-ready proposals with physical resources bound only at execution.
- Render one local trace linking product/system steps, monitors, fault/evidence/holes and source definitions. Combine documentation and final validation here.
- Run the Go-developer authoring exercise and record steps, feature source size, and diagnostic latency.

### Investigation targets
**Required:** tools/umpire/replay/core.go; tools/umpire/replay/minimize.go; tools/umpire/replay/bridge.go; tools/umpire/replay/proposal.go; tests/testpilot_nexus_control_case_test.go:30.
**Optional:** tests/testpilot_nexus_control_case_test.go:116; model/scalav2/README.md.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./tools/umpire/replay/...`; final gate includes `make umpire-check-scala`, `make lint-scala`, `make lint-code-fast`, and the selected live demonstrations.

## Acceptance
- [ ] Declared variation priorities discover an execution absent from pinned regressions with exact finite/sample coverage.
- [ ] A controlled runtime failure reproduces, minimizes without breaking dependencies, and produces a replayable proposal; unreproduced failures do not promote.
- [ ] Repeat generation is byte-identical and the trace artifact exposes both levels, monitor state, evidence, faults, and holes.
- [ ] The feature-only Scala authoring exercise and all targeted/final demonstration commands are recorded with honest support limits.

- [ ] Feature-only edits use model/scalav2/scala/temporal; final generation/native tests/lint still pass with model/scala absent and preserve the guarded legacy baseline.

## Done summary
Implemented Scala-declared finite exploration through the IR, a Go bridge for the existing campaign/replay protocols, checked prefix reduction, deterministic model-recipe proposals, and local traces. The new Scala async-callback negative control deliberately retains the real failed branch beside a forged success branch; it exercises actual runtime reproduction and learned callback authority while redundant inspection commands are removed.

Implementation and conductor verification complete. No commits or staging were performed, as instructed by the user.

stage: impl-review - ran (native independent-context reviewer gpt-6.1-sol at high; SHIP after one corrected finding; same model family)
stage: plan-sync - skipped(config: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)

The task delta is `.flow/tmp/fn107-11/changed-paths.json` and `task-delta.patch`, measured against the conductor's saved current-tree baseline (not HEAD). The authoring exercise's 16-line feature-only Scenario/Query adds two alternatives, then reverses the search priorities and checks the selected candidate changes. Exact sources, commands, compiler diagnostic and timings are under `authoring/`.

The final live artifacts are under `.flow/tmp/fn107-11/artifacts/`; generated Case recordings from the final combined pass are under `generated-runs-final/`. Discovery enumerates exactly three declared alternatives and runs one unpinned deadline execution, reporting one covered and two pending. Runtime coverage is sampled. The control's Contract is violated while its conformance assessment remains inconclusive because the monitor stops the Run; its Property is already violated by every evidence-compatible execution.

Support limits: finite prefix substitutions (4096 combinations maximum), one bounded reverse prefix-deletion sweep, no global minimality or fixed-schedule guarantee. Proposals archive checked IR plus the authored Query/edit recipe and regenerate exact Case bytes; the deliberately forged control demonstrates promotion mechanics and is not a claimed platform bug. Trace rendering preserves recorded events and distinguishes predicted internal steps from established evidence. Visual browser rendering was unavailable to the conductor; HTML/source/evidence checks are recorded separately.

Scala formatting/lint passed. Scalafix emits inherited nonfatal Java 27 reflection traces; the conductor's controlled probes proved DisableSyntax and RemoveUnused still reject bad input, and clean input passes (`lint-probe/results.json`). Go lint passed with zero issues (`lint-code-fast-4.log`). Initial red/diagnostic iterations remain in the evidence directory, including unknown IR declarations, candidate-name collision, JSON-envelope escaping, the initially omitted real failure branch, cleanup ordering, and exact negative-control assessment expectations.

Final worker verification: the required `go test -tags test_dep ./model/scalav2/... ./tools/umpire/replay/...` passed (`quick-final.log`), and the combined selected live suite passed (`live-final-2.log`, 74.172 seconds). Its 128 generated Runs comprise 120 satisfied and 8 deliberately violated controls, all with successful cleanup; the existing pinned control, unpinned discovery, paired reproduction, three paired reductions, exact proposal recovery, and a fresh proposal Run with the same re-derived failure key also passed. There are 34 final HTML traces. `reduction-invariants.json` records identical roles/learned slots and preserved controller/workflow/handler scripts, with controller nodes reduced from 9 to 6 and the other scripts unchanged.

`make umpire-gen-scala` passed fully before the final expectation correction. The correction was then packaged, lifted and generated with the same checked pipeline (`expectation-package.log`, `expectation-lift.log`, `expectation-cases.log`), followed by the green full Quick and live gates. The conductor's isolated check/regeneration is the final source-currentness and byte-repeatability gate. No Testpilot wire-catalog change was made, so no pinned Run rerecording was necessary.

Conductor gates: isolated make umpire-check-scala (279.8s), make lint-scala (19.8s), and make umpire-gen-scala (229.5s) passed with model/scala absent and v2 jars rebuilt. All checked IR, Cases and lifter fixtures retained identical hashes. The exact historical pre-isolation gate failed on the missing model/scala/scala.sh in the same snapshot; the current gate was restored. The live legacy tree hash inventory is unchanged. See isolation/isolation-receipt.json for exact commands, environment, hashes and logs.

Independent implementation review returned SHIP for the exact 34 current paths after fixing the negative control Property expectation; conformance stays inconclusive while supported Property violation is retained. Digest: .flow/tmp/fn107-11-review.md. Current path hashes match review-correction-sha256.json. Final-artifact-audit.json independently verifies 128 distinct successful-cleanup Runs across 16 Cases, 34 trace pages and valid source links, declared response-loss/hold evidence, failure-preserving reductions and exact finite/sample coverage. Browser rendering remains unverified because no browser was connected.
## Evidence
- Commits:
- Tests: baseline: green — .flow/tmp/fn107-11/baseline.log, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/explore ./model/scalav2/goir -run Test(Exploration|Finite|Alternative|CampaignBridge|Proposal|ReplayBridge|Unreproduced|Trace) — unit-final.log, exit 0, CC=/usr/bin/clang make lint-scala — lint-scala.log, exit 0; inherited reflection diagnostics verified by conductor lint-probe/results.json, CC=/usr/bin/clang mise exec -- make lint-code-fast — lint-code-fast-4.log, exit 0, 0 issues, GOFLAGS=-tags=test_dep CC=/usr/bin/clang make umpire-gen-scala — generate-final.log, exit 0; followed by checked control-expectation re-lift/regeneration, python3 .flow/tmp/fn107-11/authoring.py — authoring-2.log, exit 0; expected compiler rejection plus feature-only Scenario/policy exercise, UMPIRE_REPEAT_RUN_DIR=$PWD/.flow/tmp/fn107-11/generated-runs-final UMPIRE_EXPLORATION_DIR=$PWD/.flow/tmp/fn107-11/artifacts CC=/usr/bin/clang TMPDIR=<physical TMPDIR> mise exec -- go test -count=1 -tags 'test_dep integration canary_harness' ./tests -run '^(TestTestpilotScalaGeneratedCases|TestTestpilotScalaExplorationDiscoversUnpinnedExecution|TestTestpilotNexusControlReplaysThroughTheCommand|TestTestpilotNexusControlForgedCompletionIsViolated)$' — live-final-2.log, exit 0, CC=/usr/bin/clang mise exec -- go test -tags test_dep ./model/scalav2/... ./tools/umpire/replay/... — quick-final.log, exit 0, model/scalav2/scala.sh --power package --library model/scalav2/scala/project.scala model/scalav2/scala/umpire model/scalav2/scala/temporal -f -o model/scalav2/gen/model-scala.jar — expectation-package.log, exit 0, mise exec -- scala-cli run --suppress-outdated-dependency-warning model/scalav2/lifter -- model/scalav2/gen/model-scala.jar=model/scalav2/scala/ model/scalav2/gen/model-scala.classpath model/scalav2/ir/nexus-control.json temporal.nexuscaller.Control$.forgedCompletion temporal.nexuscaller.NexusRealization$.forgedCompletion — expectation-lift.log, exit 0, CC=/usr/bin/clang mise exec -- go run -tags test_dep ./tools/umpire/cmd/umpire-gen-cases -update — expectation-cases.log, exit 0, PASS: isolated mise exec -- make umpire-check-scala (exit 0), PASS: isolated mise exec -- make lint-scala (exit 0), PASS: isolated mise exec -- make umpire-gen-scala (exit 0); checked IR/Cases/lifter fixture hashes identical, PASS: exact historical e1d0753f41^ gate fails without model/scala/scala.sh (expected exit 1); current gate restored; live legacy inventory unchanged, PASS: final artifact audit: 128 distinct Runs, 34 traces, source links/fault evidence/reduction/coverage identities, PASS: all 34 current task paths match independently reviewed correction hashes
- PRs: