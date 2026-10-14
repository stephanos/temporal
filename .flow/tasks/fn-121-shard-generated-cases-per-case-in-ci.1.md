---
satisfies: [R1, R2, R3, R6]
---
# fn-121-shard-generated-cases-per-case-in-ci.1 Run each generated Case as its own shard unit with the Nexus switch below it, placed by the depth-2 prefix

## Description
Reshape `TestTestpilotGeneratedCases` from `/<hsm|chasm>/<file>` to `/<model>-<query>` with `/<hsm|chasm>` under Nexus Cases only, one dedicated cluster per Case (per switch value for Nexus), the agreement check per Nexus Case, and the sharding check hashing the depth-2 prefix so both switch subtests land in their Case's shard. Fix the stale switch comment. Spec Architecture names the shapes; this task builds them.

**Cross-spec entry gate:** runs after fn-112.10 (structural Case freeze) is done; no Case, manifest, IR or fixture byte changes (`make umpire-check-cases` stays green). fn-119.4 and fn-119.6 also edit `tests/testpilot_generated_test.go`; whichever lands second rebases. flowctl cannot record these edges, so the conductor holds this task until fn-112.10 is done.

**Size:** M
**Files:** `tests/testpilot_generated_test.go`; `tests/testcore/test_env.go` (`checkTestShard` :755-776); a shard unit test in `tests/testcore/` (e.g. beside `test_env_test.go`); `tests/testcore/testpilot/model_fixture.go` (exported name derivation over `lower.GeneratedCase`); `tests/testcore/testpilot/switch.go` (comment :38-39); `tools/optimize-test-sharding/main.go` (cross-reference comment at :23 only).
**Touches:** [tests/testpilot_generated_test.go, tests/testcore/test_env.go, tests/testcore/*_test.go, tests/testcore/testpilot/model_fixture.go, tests/testcore/testpilot/switch.go, tools/optimize-test-sharding/main.go]

### Approach
- Name: add one exported function in `tests/testcore/testpilot` that derives a Case's subtest name from its manifest entry as the `File` stem (`strings.TrimSuffix(entry.File, "-case.json")`, which `tools/umpire/lower/generated.go:88` builds as `<model>-<query>`). The functional test uses it; fn-121.2 pins it.
- Shape: iterate lowered manifest entries at the top level; `t.Run(name)` per Case, and the FIRST statement in it is the exported shard check (below), so an out-of-shard Case skips as a whole before any value subtest runs (a Nexus Case constructs no cluster at depth 2, so without this both children would skip and the Case would have nothing to judge). Then decide Nexus by `bindsNexusEndpoint(fixture.Source)` (`tests/testpilot_run_case_test.go:81`). Nexus: `t.Run(value.Name)` per `testpilotcore.NexusImplementationSwitch()` value, each constructing its own `newTestpilotTestEnvironment` with the value's settings plus `EnableChasm` (keep the comment at :154), running the two bindings and two rounds as today, and appending a `SwitchVerdict` (one Verdict per value, e.g. the first binding's round-0 Verdict; all four are already required equal to the expected status). Inside a value subtest a `PreparationUnsupported` rejection FAILS naming the value (replace today's `t.Skipf` at :193 on this branch); after both values, `require.NoError(t, CheckSwitchAgreement(NexusImplementationSwitchName, results))` over whatever was collected, with NO Case-level count of Verdicts (the test runner retries a failed leaf by its anchored name, `tools/testrunner/testrunner.go:417-426`, which filters the sibling value out; a count would fail that retry deterministically). Non-Nexus: one cluster with `activity.Enabled`, `EnableStandaloneActivityOperatorCommands`, `EnableChasm`, no switch subtest, today's skip on `PreparationUnsupported` kept; `binding.DynamicConfig` records only the settings that cluster set.
- Factor the per-cluster body (bind two lives, two rounds, assessment, exploration artifact, run-id uniqueness, durable-evidence check) into one helper called from both branches rather than duplicating :172-228. The exploration artifact name becomes `<name>[-<value>].html`.
- Sharding: in `checkTestShard`, hash the first two `/`-separated segments of `t.Name()` (fewer segments: the whole name), with the salt as today, and export it (e.g. `testcore.CheckTestShard(t)`) for subtests that construct no cluster; `NewEnv` (:271) and `FunctionalTestBase.SetupTest` (`functional_test_base.go:397,422`) keep calling it. Its comment names the optimizer's `defaultLevel = 2` (`tools/optimize-test-sharding/main.go:23`) and a one-line comment there names the check back (comment only; no optimizer behavior change). Add a unit test in package `testcore`: with `TEST_TOTAL_SHARDS`/`TEST_SHARD_INDEX` set via `t.Setenv`, a depth-3 subtest is skipped or run exactly as its depth-2 parent is (compute the parent's index with the same hash; `testing.T.Skipped()` on the child). Keep the `t.Fatal` paths for malformed variables.
- Comment: rewrite `NexusImplementationSwitchName`'s doc (`switch.go:38-39`) to say the functional tests declare the switch and its value names; no Model, IR or Case declares it, and Case bytes do not depend on it.
- Prove locally: unsharded `-run '^TestTestpilotGeneratedCases'` passes; with `TEST_TOTAL_SHARDS=5` for each index 0-4, every `<name>` appears in exactly one shard and both switch subtests of a Nexus Case appear together (grep `--- PASS` / `--- SKIP` lines). Record the unsharded wall-clock before and after for fn-121.3's summary.

### Investigation targets
**Required:**
- `tests/testpilot_generated_test.go:142-233`
- `tests/testcore/test_env.go:266-275,753-776`
- `tests/testcore/testpilot/switch.go`
- `tests/testpilot_nexus_caller_case_test.go:120-175` - per-value clusters and the agreement check as the hand-written suite does it
- `tools/optimize-test-sharding/main.go:218-262` - depth-2 aggregation and the hash the salt is optimized for
**Optional:**
- `tests/testpilot_run_case_test.go:37-88`
- `tests/testcore/test_env_test.go` - package-internal test style

### Quick commands
```bash
go test -count=1 -tags test_dep ./tests/testcore/ -run 'Shard'
go test -count=1 -v -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases'
for i in 0 1 2 3 4; do TEST_TOTAL_SHARDS=5 TEST_SHARD_INDEX=$i go test -count=1 -v -tags 'test_dep integration' ./tests -run '^TestTestpilotGeneratedCases' | grep -E '^\s*--- (PASS|SKIP|FAIL)'; done
make umpire-check-cases
```

### Execution constraints
- No edit under `model/`, `tools/umpire/`, the salt file or any workflow; build tags unchanged (fn-121.3 owns them).
- One cluster alive at a time in the generated test.
## Acceptance
- [ ] Every lowered manifest Case runs as `TestTestpilotGeneratedCases/<model>-<query>` with its own dedicated cluster; non-lowered entries produce no subtest.
- [ ] Nexus Cases (endpoint binding) run `/hsm` and `/chasm` with one cluster each; the Case subtest fails on a Verdict divergence over the Verdicts collected and counts nothing itself; a value rejected at preparation fails that value's subtest naming it; non-Nexus Cases run once with no switch subtest and keep today's skip.
- [ ] `checkTestShard` hashes the depth-2 prefix and is exported; every Case subtest calls it first; a package test proves a depth-3 subtest is placed with its depth-2 parent; malformed shard variables still fail; the check and the optimizer's level comment on each other.
- [ ] With five local shards every shard is green, every Case name runs in exactly one shard and skips as a whole in the others, both switch subtests of a Nexus Case run together, and `-run` on a single value subtest passes on its own; the unsharded run passes.
- [ ] The switch-name comment no longer claims a `repeat:` declaration; `make umpire-check-cases` and lint-code-fast pass; no Case byte changed.
## Done summary
# fn-121.1 done summary

**What changed** (commits 9c65536753, 275efdadbf; base 5e6961b91a):
- `TestTestpilotGeneratedCases` runs each lowered manifest Case as `TestTestpilotGeneratedCases/<model>-<query>` (name from the new `testpilotcore.GeneratedCaseName`, the file stem), with `testcore.CheckTestShard(t)` first. Nexus Cases (`bindsNexusEndpoint`) run `/hsm` and `/chasm`, one dedicated cluster each (value settings + EnableChasm), append one `SwitchVerdict` (first life's round-0 Verdict) and the Case subtest runs `CheckSwitchAgreement` over what was collected, counting nothing. A Nexus value rejected at preparation fails naming the value; non-Nexus Cases run once (activity.Enabled, operator commands, EnableChasm) and keep the skip. One helper `runGeneratedCaseOnCluster` holds the per-cluster body; artifacts are `<name>[-<value>].html`; the Profile records exactly the settings the cluster was built with.
- `checkTestShard` -> exported `CheckTestShard`, hashing the depth-2 prefix (`shardKey`) + salt; Fatal paths kept; comments cross-reference `defaultLevel = 2` in the optimizer (comment only there). `tests/testcore/test_shard_test.go` proves a depth-3 child is skipped/run exactly as its depth-2 parent for all 5 indices.
- `NexusImplementationSwitchName` doc no longer claims a `repeat:` declaration.

**Evidence** (`.flow/tmp/fn121-1/`): `make umpire-check-cases` exit 0, `git diff 5e6961b91a..HEAD -- model/` empty; lint-code-fast exit 0 (72 s); shard unit test pass; smoke (activity-completion + nexus-caller-syncCompletion hsm/chasm) pass; `-run .../nexus-caller-asyncCompletion/chasm` alone passes. Five local shards: every one of the 16 Case names RUN in exactly one shard and SKIP in the other four (2+3+5+2+4), both switch subtests of each Nexus Case in the same shard. Wall-clock unsharded: before (old shape, 2 clusters, 128 Runs) 93 s; after (24 clusters, 96 Runs) 71 s; per shard 2/10/25/3/24 s.

**Pre-existing failure, not fixed (outside this task's boundary):** activity-terminate and activity-pauseResume fail deterministically and activity/nexus-caller-scheduleToStartTimeout intermittently, in BOTH the base shape and the new shape (shards 2 and 4 red). Diagnosis: the Case's first instruction `stop-worker` times out at the default 10 s instruction limit because `worker.Stop()` waits 2x5 s `WorkerStopTimeout`; matching's ShutdownWorker returns early with "Skipping poll cancellation fan-out: root partition not loaded" (`service/matching/matching_engine.go:1311-1324`, upstream 2220cf011a #9424) without recording the shutdown, so polls landing 1 ms later hang. Disabling `frontend.enableMatchingFanOutForPollCancellation` makes all four pass (20/20 Runs SATISFIED). Present since the Cases were introduced (reproduced at 08832883de). Decision: not masked here, since the boundary forbids changing how a Case is run and the finding is a real server race; left for the conductor (server fix, or a recorded setting in the generated test).

**Decisions:** shard check and name derivation as specified; Prepare check stays after cluster construction (reviewer FYI: a rejected Case now builds its own cluster before skipping; no lowered Case is rejected today).

**Review:** `flowctl claude impl-review --spec claude:claude-opus-5-5:high`, round 1 SHIP with one P3 (derive the artifact name in the helper), applied in 275efdadbf. Writer and reviewer are the same family (Opus 5.5).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9c65536753, 275efdadbf
- Tests: make umpire-check-cases (exit 0), go test -count=1 -tags test_dep ./tests/testcore/ -run Shard (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0), tests.test -run '^TestTestpilotGeneratedCases$/^(nexus-caller-syncCompletion|activity-completion)$' (exit 0), tests.test -run '^TestTestpilotGeneratedCases$/^nexus-caller-asyncCompletion$/^chasm$' (exit 0), TEST_TOTAL_SHARDS=5 TEST_SHARD_INDEX=0..4 tests.test -run '^TestTestpilotGeneratedCases$' (shards 0,1,3 exit 0; 2,4 exit 1 on pre-existing stop-worker timeout, same as base), tests.test -run '^TestTestpilotGeneratedCases$' unsharded (exit 1, pre-existing; base also exit 1)
- PRs: