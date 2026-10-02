---
satisfies: [R13]
---
# fn-105-gomad-follow-ups-deferred-scope.13 D13: make choice tracing opt-in for routine qualification

## Description
Origin: fn-106.3 (2026-09-30). TestTaskQueueStats_Pri_Suite, TestVersioning3FunctionalSuite, TestVersioning3QueryFunctionalSuite, and TestWorkerDeploymentSuite exceed the 64 MiB choice-trace maximum and currently run without a choice trace. Their recorded same-seed repeatability evidence remains valid; it does not establish choice-tape replay.

Decision on 2026-09-30: make runtime choice tracing opt-in for routine full-suite functional qualification. The CLI already defaults seed runs to tracing off; the full-suite generator currently enables it by default. Change that generator policy and its generated manifest, preserve explicitly traced replay/conformance gates and choice-based exploration, and distinguish seed repeatability from verified choice-tape replay in reports and documentation. D12 and D14 remain required fixes with tracing-enabled verification. Larger-trace support is separately deferred under R15 and is not required for completion of this policy task.

## Acceptance
- Default routine full-suite generation to choice_bytes 0 and replay_successes false, with consistent zero success-retention limits. Individual workloads may explicitly opt into bounded tracing and replay.
- Preserve the CLI's existing opt-in controls and tracing requirements for choice-based exploration. Preserve explicitly traced representative replay/conformance gates.
- Reports and documentation distinguish same-seed repeatability, tape availability, and verified choice-tape replay. Untraced runs make no choice-tape replay claim and continue to reject same-seed evidence mismatches and target/infrastructure failures.
- D12 and D14 retain explicitly tracing-enabled qualification and exact-replay verification; disabling tracing or weakening their existing traced gates cannot close either fix task.
- Verify both generator defaults and explicit tracing overrides, and regenerate the full-suite manifest. Closing this task claims only the opt-in policy change; the larger-trace extension remains recorded and deferred under R15.


## Done summary
Routine `./tests` qualification is now untraced by default, and a test opts into tracing and success replay by name. Nothing is committed: every change is in the working tree, as instructed.

- **Generator policy.** `tests.generator.json` defaults to `choice_bytes` 0, `replay_successes` false, and zero success limits. Regenerated `tests.json` has 146 of 147 workloads untraced.
- **Opt-in path.** `TestOverride` gained `success_artifact_limit` and `success_bytes_limit`. The generator refuses replay without a trace or limits and limits without replay, naming the spec entry. `TestSignalWorkflowTestSuiteChasm` opts in (64 MiB, replay, 1 artifact, 1 GiB) because its D14 finding is a replay divergence.
- **Unchanged.** CLI flags and defaults, choice exploration, `temporal.json`, `smoke.json`, `core.json`, every D12/D14 expectation, and the report schema. No report field was added: an untraced `qualified` seed already reports `replayed` and `choice.available` false with no `choice_replay_exact`, which `TestProjectSeedReportClaimsNoReplayForAnUntracedWorkload` pins.
- **Tests.** `TestUntracedDefaultsTraceOnlyTheTestsThatOptIn` covers defaults and the override; six new refusal cases cover the enumerated spec errors; `TestCheckedInTestsManifestTracesOnlyByOverride` failed before the spec change and passes after.
- **Docs.** `tools/gomad3/README.md`, `CLI.md`, `tools/gomad3integration/README.md`, and the D13 bullet and row of `MILESTONES.md` separate same-seed repeatability, tape availability, and verified choice-tape replay.
- **Real run (darwin/arm64).** One untraced run of the generated `TestUserTimersTestSuite` entry on seed 11 qualified in 38 s by repeatability: two executions with equal evidence digests, no `replay` entry, `replayed: false`. The full untraced set and linux/amd64 were not run.

Open, not changed here:
- **D12:** generated `tests.json` still expects the F5/F6 suites `qualified` on linux/amd64 while `temporal.json` and `smoke.json` expect `intermittent`, and `tools/gomad3integration/README.md` still calls the F6 slice qualified on both platforms. The linux channel can hit any tier-3 suite, so the generator input is not the place for a twelve-suite patch; the milestones' "Intermittent suites" bullet already records it.
- **fn-111 audit:** `verify-guides.py` expects the milestones to name every `tests.json` suite with `choice_bytes` 0 under "No exact replay for N suites". With 146 untraced suites that check now fails by design; the script is an fn-111 artifact and was not edited or rerun.
- **Milestones Status paragraph:** "Implementation has not started." is stale for D13 and D21-D25; it is outside the D13 wording this task may edit.
- **Set runner:** a workload with a choice trace and no success replay passes manifest validation but cannot qualify in a set, because `projectSeedReport` reads choice coverage from retained success Artifacts. The guides now say a traced set workload must replay its successes; the validator was not changed.
- **Not captured to memory:** the review fixes were documentation wording.

baseline: none (the spec defines no Quick commands)
stage: impl-review - ran (raw codex bridge on working-tree diff; commits forbidden) (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: none (spec defines no Quick commands); platform darwin/arm64, toolchain go1.27.1 build c0661e38b4e0..., runner sha256:3062e6f84a85..., red before spec change (exit 1): go test -count=1 ./qualification/set/manifestgen/ (TestCheckedInTestsManifestTracesOnlyByOverride: workload defaults retain choiceBytes 67108864), make -C tools/gomad3 tests-qualification-generate (exit 0; 146 of 147 workloads untraced), make -C tools/gomad3 validate validate-qualification (exit 0; .flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d13-validate.log), GOWORK=off go test -count=1 -tags test_dep ./qualification/... ./cmd/gomadtool/... ./cmd/gomad/... in tools/gomad3 (exit 0; fn105-d13-unit.log), gofmt -l qualification cmd; go vet ./qualification/set/... ./cmd/gomadtool/... in tools/gomad3 (clean), go test -tags test_dep,gomad3_integration -count=1 -run 'TestQualificationManifestsUsePortableV3|TestSmokeSuitesMatchRepresentativeSuites' ./tools/gomad3integration (exit 0), gomad qualify-set --check on temporal.json (28 workloads), smoke.json (4), tests.json (147) (exit 0 each), gomad qualify-set on the generated tests-user-timers-test-suite entry, seed 11, untraced (exit 0, 38 s; expectations-met supported=1; seed qualified, replayed false, choice.available false, no choice_replay_exact; fn105-d13-untraced-set-report.json, fn105-d13-untraced-qualification-seed11.json), flowctl gate classify --base d4d800fb47: FULL (exit 1); full gates above were run, no gate receipts apply, impl-review: codex exec gpt-5.6-sol high, round 1 NEEDS_WORK, round 2 SHIP (.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/task13-review.md), not run: the full untraced ./tests set; linux/amd64
- PRs: