# Fresh Choice Exploration divergence fixture slice

Recommend exactly three call-local attachments of the existing `scriptedPreparationDependencies` helper in `tools/gomad3/runner/choice_exploration_divergence_test.go`. Static tracing finds no additional seam needed for their preparation, root/prefix execution, published campaign inspection or error classification. Preserve every fixture, assertion and shared helper. The next admitted worker must observe the existing assertions after attachment before claiming restoration.

This report is bounded research by the assigned gpt-6-astra high researcher. It authorizes no implementation or task admission. Only this report was written. No Go, build, test, lint, vet, generator, Flow operation, Git mutation, native execution, CI, PR or push ran. Root retains admission, verification, integrated review and lifecycle ownership. Task68's isolated `runner_test.go` work and exclusive Go lane remain separate.

## Current bindings and retained RED

The inspected primary checkout is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, HEAD `7727b062b0c263046f0409e8f9d6cf5e58e7c0ef`. Source citations below use that checkout. The current owner-amended fn-109 spec SHA-256 is `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Its 2026-10-09 amendment removes format/byte-equality acceptance while retaining classifications, error precedence, capabilities and transaction guarantees. This proposal changes no assertions or formats. Dirty Flow, milestone and unrelated user files were left untouched.

The observation is `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-66-67/ordinary-runner.log`, SHA-256 `f97441bc0b8f4ebc4da6673b0e3b097a9420fe6e5bd1c1871de83c6f333eb032`. Its `ordinary-runner.json` records `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner`, exit 1, 13.211667173003661 seconds, unchanged before/after source, and `run-binding.json` SHA-256 `45f31ddfd586858fda9d0b9afb76e421a0f989c659957ba0727fe23f55073bb5`. That binding names joined HEAD `f699252450b8e67f1edb50ed8e4cff4cb6e644c0` and developmental Linux/aarch64. A read-only diff from that revision to primary HEAD found no changes under `runner`, `choice`, `internal/preparation`, `deterministicio/profile.go` or `target/target.go`.

The log contains 27 failed named outcomes for the exact three selected tests, counting three parents plus 3 policy, 14 host-error and 7 mismatch-reason subtests. This count was obtained from their actual terminal JSON events. It is neither 27 independent top-level tests nor a future pass count.

| Selected test | Current call and assertions | Actual log failure |
| --- | --- | --- |
| `TestRunChoiceExplorationDivergencePoliciesAndInspection` | Definition 65, call 69, summary 79, real `Inspect` 82, projection 86/90 | Raw lines 562, 567 and 572 report source line 71's unsupported-host error; parent failure at 576. |
| `TestRunChoiceExplorationKeepsOtherErrorsAsHostErrors` | Definition 97, call 143, typed error/reason 145, committed-state assertion 148 | Raw line 581 reports source line 146's preparation refusal instead of `target_supervision`; remaining table diagnostics continue through raw line 646, parent failure at 650. |
| `TestRunChoiceExplorationRetainsOnlyPrefixMismatchReasons` | Definition 225, call 249, real `campaign.OpenCampaign` 253, retained reason 257 | Raw lines 665, 670, 675, 680, 685, 690 and 695 report source line 251's unsupported-host error; parent failure at 699. |

All selected subtest diagnostics report `deterministic I/O requires one of darwin/arm64, linux/amd64; host is linux/arm64`. The host-error test's asserted classifications were never reached. The adjacent `outcome-comparison.json`, SHA-256 `5e4c48dd7caef408a4f4c5dcffc6be045219e7ffafc6b0533a424c2fb7129a7b`, records 673 named outcomes for that capture, with 389 passes, 272 failures and 12 skips. These remain combined66/67 observations; this research supplies no newer ordinary result or reduction.

The preceding report convention is `post-task66-next-slice.md` in this directory, SHA-256 `b2e54c6b48f714e53e83e4f0f129c025346c6db9d06da7899e56c576e0934033`. Its three proposed consumers are in `runner_test.go` and remain outside this report's proposed Touches.

## Exact proposed Touches

The complete proposed code/test Touches set is `tools/gomad3/runner/choice_exploration_divergence_test.go`. Insert this existing-helper attachment immediately before each of the three `exploreWith` calls at current lines 69, 143 and 249, inside the existing subtest body and after its executor mutation is complete.

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

There are three source additions, executed for each existing table row. Removing those three lines must recover the current test file. No imports, comments, assertions, expected values, table rows, target metadata or helper bodies change. Keep `divergenceCampaignConfig`, `testConfig`, `candidateDivergenceExecutor`, `explorationExecutor`, `newFakePreparer` and `scriptedPreparationDependencies` unchanged. Use the outer `configDependencies.executor`; substituting its `.base` executor would erase the divergence/error behavior under test.

## Preparation and root/prefix execution

`divergenceCampaignConfig` at lines 51-62 constructs a new private fake preparer and a `candidateDivergenceExecutor` wrapping a four-alternative `explorationExecutor`. It sets one base seed 7, parallelism 1 for these consumers, one-record choice-trace capacity, maximum executions 8, depth 4 and exploration bytes 1 MiB. `testConfig` at `runner_test.go:2501-2508` supplies `KindGoRun`, source `.`, no target arguments, `MODE=test`, failure budget 1, one-second execution timeout and ten-second overall timeout. Its returned dependencies contain only the executor. None of the three consumers changes the target or preparer.

`newFakePreparer` at `runner_test.go:1978-1995` supplies an actual 0500 file, its SHA-256 and size, matching kind/source, `Argv = [gomad3-target]`, fixed hexadecimal build key, and platform/Go metadata from `deterministicio.Default().TargetContract()`. `fakePreparer.Prepare` at 2009-2027 performs the real file copy and mode assignment into the requested preparation directory. The existing helper at `preparation_fixture_test.go:17-44` checks the exact preparer, nonempty preparation root, absent adapter replacements, matching kind/source/arguments, nonempty argv and absent adapters. It calls the real preparer and `Prepared.Verify`, which checks compatibility, an executable regular file, digest and size (`target/target.go:305-318,889-908`). The helper returns the explicit empty adapter list and its visibly synthetic bootstrap marker.

Without attachment, `preparation_dependencies.go:17-21` invokes ordinary preparation. Even a public custom preparer still passes through `internal/preparation/preparation.go:97-103` profile validation, which reaches `deterministicio/profile.go:278-284` and rejects the developmental host. With a call-local fixture, the existing preparation hook at `runner_local.go:250` and bootstrap hook at `runner.go:681` receive the completed campaign request. No propagation change is needed. The bootstrap marker is supplied to the scripted executor and is never decoded or executed as a real toolchain frame.

`runner_local.go:98` passes the request to the real choice-strategy controller. `choice_exploration_campaign.go:228-241` constructs a recording root and identity-bound forced-prefix jobs, forwarding the same request into `runSeed`. `explorationExecutor.Run` at `runner_test.go:2151-2177` produces one complete runnable-choice record, substitutes the forced canonical rank for prefix requests, and uses `completeChoiceTrace` at 2556-2569 to construct the actual trace and implementation identity. Root and later non-divergent traces retain `runSeed`'s normal choice validation at `runner.go:716-717,793-809`.

`candidateDivergenceExecutor.Run` at test-file lines 25-48 delegates to that base, leaves the recording root unchanged, and mutates only the first prefix under its mutex. It takes Expected from the requested prefix, derives Observed from the actual scripted trace, changes the alternative-set digest and returns the typed divergence or the row's existing callback result. Subsequent prefixes return the base result. Preserve these exact mutations, result flags and error objects.

Choice identities remain real projections of the prepared target SHA-256, build key, platform and derived choice implementation (`runner.go:812-824`). `choice.ValidateExecutionIdentity` validates completeness and hexadecimal build-key shape and hashes the supplied platform (`choice/tape.go:479-502`). It does not consult the supported-native manifest or read an installed toolchain. The fake metadata therefore needs no platform alteration or identity hook for this path.

## Policy, round and classification controls

The existing policy test at lines 73-91 requires `first` to retain 2 attempts, 1 success and stop `first_failure`; `budget` and `all` require 4 attempts, 3 successes and `exploration_exhausted`. Every row requires 1 failure, 1 replay divergence, 0 distinct failures, the matching execution count, no failure artifacts, and the second execution's runner-domain, ordinal-0 alternative-set evidence with Expected/Observed and no outcome hash.

The source explains these expectations without establishing a runtime pass. Four root alternatives produce three non-selected prefixes. With parallelism 1 each accepted execution commits one round. The first divergent prefix stops PolicyFirst; divergence does not create a distinct-failure signature or consume the PolicyBudget distinct-failure budget (`internal/exploration/choice/engine.go:228-254,260-269`). Thus the policy fixture's asserted execution counts correspond to 2 versus 4 committed single-candidate rounds. These committed-round totals are a static consequence, not additional observed assertions in this test. The runner advances its public summary only after real `CommitRound`, immutable journal commit and execution append (`choice_exploration_campaign.go:126-163`).

The 14-row host-error test at lines 103-138 preserves the following classifications. Every row additionally requires exactly one committed root attempt, no replay divergences and one committed round at lines 145-149. A candidate error must not commit candidate state.

| Existing row | Required `HostError.Reason` | Existing rejection mechanism |
| --- | --- | --- |
| normal error | `target_supervision` | Ordinary executor error. |
| cancelled | `runner_cancelled` | Cancellation flag with nil executor error. |
| typed cancelled | `target_supervision` | Cancellation prevents typed divergence admission; error classification precedes the nil-error cancellation branch. |
| overflow | `choice_trace_overflow` | Existing overflow sentinel. |
| malformed trace | `choice_trace_malformed` | Existing malformed sentinel. |
| unterminated trace | `choice_trace_unterminated` | Existing unterminated sentinel. |
| missing identity | `target_supervision` | Reason is outside the admitted forced-prefix mismatch set. |
| duplicate identity | `target_supervision` | Reason is outside that set. |
| alternative capacity | `target_supervision` | Reason is outside that set. |
| tape exhaustion | `target_supervision` | Ordinal is outside the one-decision prefix and Expected is absent. |
| typed watchdog | `target_supervision` | Watchdog flag prevents typed divergence admission. |
| wrong prefix | `target_supervision` | Expected site differs from the actual forced prefix. |
| malformed observed | `target_supervision` | Observed selected identity is zero and fails record validation. |
| joined failure | `target_supervision` | A two-error join fails the single-error divergence path. |

The operative guards are `choice_exploration_campaign.go:453-470`, `internal/exploration/choice/engine.go:388-404`, and `choice/divergence.go:60-78`. Error conversion occurs at `choice_exploration_campaign.go:255-260`, with trace-sentinel precedence in `runner.go:964-974`. These paths precede processing and committing the candidate round. No error expectation should change to preparation refusal, and no failed table row should be skipped.

The seven retained-reason rows at test-file lines 225-258 cover kind, site, alternatives, selected, alternative set, tape unconsumed and observation. Each row preserves its reason-specific mutation, typed error and real `campaign.OpenCampaign` call, then requires one replay divergence and the same reason at execution index 1. The helper attaches after the callback assignment. `ValidateCandidateDivergence` explicitly admits those seven reasons and validates them against the forced prefix; successful admission produces the real runner-domain divergence record without outcome, trace or artifact evidence (`choice_exploration_campaign.go:291-316`).

## Public Inspect remains real

The policy test calls the public `Inspect(summary.CampaignPath, InspectOptions{})` at line 82. Its published-campaign branch checks filesystem entries and conflicts, calls `campaign.InspectCampaignLifecycle`, opens the campaign and projects its executions (`inspect.go:361-466`). The published lifecycle branch calls `OpenCampaign` before reporting publication (`internal/campaign/lifecycle.go:65-82`). It returns from that branch before the interrupted-campaign `ReadResumePlan` checks at lines 97-107.

`OpenCampaign` retains real directory/mode/root-pinning, bounded record decoding, journal/index validation, count/domain/provenance validation, artifact-capacity checks and published exploration validation (`internal/campaign/open_campaign.go:35-89,125-285`). Divergence records must have forced-prefix provenance and no outcome, failure, success, transcript or choice-trace payload metadata (340-362). Ordinary successful trace/tape summaries remain validated at 383-414.

Published exploration inspection reads the real exploration plan and immutable round files, reconstructs the choice engine, checks the controller and initial-state identities, replays every committed segment, compares provenance/divergence evidence and summary/chain/execution projections, and rejects incomplete round state (`internal/campaign/choice_exploration_journal.go:428-517,729-750`). Segment replay reuses actual engine divergence validation. Identity validation uses recorded supplied identities; it does not load the installed toolchain.

These selected published campaigns retain neither success artifacts nor target-failure artifacts. The divergence record deliberately has no artifact. Artifact-capacity validation therefore takes its no-reference branches (`open_campaign.go:306-320`), and projection copies real executions while leaving artifact lists empty (`inspect.go:770-849`). No selected inspection path invokes `ReadToolchainIdentity`, `VerifyAdapters`, replay preflight or bootstrap decoding. Public `Inspect` accepts no preparation dependencies, and none is proposed. If its real validators reject newly reached evidence, retain that failure and return it to root; replacing public inspection or weakening validation is outside these three attachments.

## Verification boundary and exclusions

The meaningful existing RED is the retained preparation refusal. An admitted worker should retain unchanged-source results for these exact three top-level names on its actual candidate, then attach the fixture and observe their unchanged assertions. No additional test or private hook is required by this static proposal. Keep existing seam controls in the focused selection: `TestPreparationDependenciesForwardRealFixtureInputs`, `TestPreparationDependenciesOperationErrorsRemainUnchanged`, `TestPreparationDependenciesFailuresStopAtOriginalStages`, `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard` and `TestInjectionCharacterizationIsolatedPreparationDependencies`. In particular, the default/bootstrap guard at `preparation_dependencies_test.go:205-239` preserves public, executor-only and prepare-only unsupported-host refusals. New behavioral failures after attachment are evidence, not permission to repair production code or expectations within this scope.

Exclude the shared configuration helper and all its other consumers, especially `TestRunChoiceExplorationResumePreservesDivergenceIdentity` at test-file line 180. Also exclude killed-process tests, `TestProcessExplorationCompletionKeepsRunnerDomainFallback` at line 155, shared test helpers, task68's `runner_test.go` additions, real compiler/bootstrap paths, replay/minimize, guidance, adapters and installation identity validation. A broad outcome comparison must keep those exclusions visible rather than infer success from this selection.

This slice contributes only ordinary host-source coverage under fn-112.10/fn-109's retained acceptance. Applicable lint, preservation, integrated source review, both-source-set static checks, generated validation and other retained source requirements remain root-owned until evidenced. Native execution and soak bounds remain deferred to fn-128/fn-149. Scripted execution plus real filesystem/journal inspection supplies no supported-native full-host pass, replay qualification or determinism bound. No future suite count, task completion or native revival is claimed.

## Source SHA-256 ledger

Paths below are relative to `tools/gomad3/` in the bound primary checkout.

| Source | SHA-256 |
| --- | --- |
| `runner/choice_exploration_divergence_test.go` | `93b138931736026645f0c4f05940a95ef8fe301de36bc3dbab3bea9c4c4b23df` |
| `runner/preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `runner/preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `runner/runner_test.go` | `7045165b88318f57fb147882b051cb7bfd2aac8c3182039a0cca0fce6f5af8e0` |
| `runner/runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner/runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| `runner/choice_exploration_campaign.go` | `5744948588413558345e30b479a906baf74d87e3c33c171d15dffd7001ee88fa` |
| `runner/inspect.go` | `94d69de99dbb7a30c9a307dfd4a2e51b35f24b49aaf0ae0d1da5f77766af7359` |
| `runner/internal/campaign/open_campaign.go` | `1003d279b9b66cc34a51254c3205db602ec69b98af3cc61ecf71c816ebe8fc0f` |
| `runner/internal/campaign/lifecycle.go` | `83920733121a8cb23277e2498468c064572fb0c37d4535062e100a64b252cfc8` |
| `runner/internal/campaign/choice_exploration_journal.go` | `f13b01b463d359c08be7fa16f36d6759cac74921a02e6c2c41a46cbe8511c3c1` |
| `runner/internal/exploration/choice/engine.go` | `2ddd22a25431d20a74a9eb193911e781adea0bbedb53e39c468a084953cb47c0` |
| `choice/tape.go` | `e7c8c7774e25d83a0007ba3c234d7004d9ec49affa062c5b448d993299e0559e` |
| `choice/divergence.go` | `01a0aca396d6965f6b768d63d8dba444203d0268b0035e20acafaf3f48e42f0a` |
| `target/target.go` | `bf5a1c8e193650913220fd3a1de6ae2aa0dbf264f1a77a1c77bb52bb9a848bf7` |
| `internal/preparation/preparation.go` | `52f02f1dc0d1f90910eb9c7e409093e1731e01c91467084c040f4369417247f6` |
| `deterministicio/profile.go` | `d068900a3b76bc91e5b94f0d67e00e67e225c50c0f3d23cbf92ce992ef0c5f21` |
