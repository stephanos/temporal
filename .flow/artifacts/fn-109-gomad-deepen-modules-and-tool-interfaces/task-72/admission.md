# Restore explicit scripted seed completion coverage

Root admits fn-109.72 to support R5/R18/R19 and open fn-109.63/fn-112.10 source acceptance. Root confirmed the available ID and created the task through flowctl after reading the retained research and both complete source tables. The task remains TODO; no implementation or execution lane is granted. Root alone selects its worker and candidate, integrates, reviews, commits and controls lifecycle.

The owner has no predecessor completion dependency; still-RED corrective owners remain in progress. This does not remove delivery prerequisites or the root's serial scheduler hold. No worker or shared gate may begin until root explicitly dispatches and grants its lane.

Proposal BASE is PRIMARY HEAD `87d1927a90295dc267c33d6226a643a74c425f51`. The selected `tools/gomad3/runner/seed_completion_characterization_test.go` hashes to `b35b3ec2d623c5746cc58e8996ff033b8e584813e6308f5fe7e0a03fde76e2b6`. Follow [seed-completion-next-slice.md](../../fn-112-gomad-determinism-assurance-and-test/task-10/runner-preparation-design/seed-completion-next-slice.md), SHA-256 `68cd8095264fcf63a8d757724e00f774433570548de2665bd4b9405b890e8fc5`, for the independent source trace, actual fourteen leaves, retained input equality and evidence bindings. This proposal does not enlarge task71's separate three-call completion scope.

## Exact boundary

**Touches:** [tools/gomad3/runner/seed_completion_characterization_test.go, .flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/task-72/**]

Add exactly two syntactic assignments using the existing statement, immediately before each named table's existing final `exploreWith` call:

```go
config.dependencies = scriptedPreparationDependencies(t, config.Preparer, config.dependencies.executor)
```

- `TestSeedCompletionKeepsCampaignStatistics`: after `ctx, config := test.configure(t)` at current line 212, before the call at 213. Every row has already selected its final preparer, executor, policy, bounds and cancellation callback. Preserve the existing executor pointers and shared state, including the first-failure barrier, scripted supervision/drain results, `mutatingExecutor{}` and `blockingExecutor{}`.
- `TestSeedCompletionFaultsKeepCampaignStatistics`: after `injectedCompletionCampaign(...)` at current line 255, before the call at 256. Preserve the final preparer and outer `faultExecutor`; never unwrap its base or substitute another executor.

Removing only these two new lines must reconstruct the entire BASE file byte-for-byte. Keep every original helper, import, comment, table row, assertion, datum, payload, executor, call and cleanup unchanged, including `injectedTestConfig`, `injectedCompletionCampaign`, `testConfig`, `completionCampaign`, `scriptedPreparationDependencies` and all excluded consumers. This selective-removal boundary reinstates no encoded-output compatibility requirement.

The first table retains its complete `seedCompletionObservation` comparison: reason/cause, execution counters, actual opened artifact metadata, journal entries, partial states and controller statistics. Preserve success/failure deduplication, failure-budget and first-failure cancellation behavior, supervision/drained-result precedence, actual copied-target mutation and integrity verification, and cancellation cause/partial recovery states. The second table compares every `CampaignStatistics` field and calls `observeCompletion`, retaining its error-type, artifact-open and journal-read checks; it does not compare the observation's remaining fields with expected values. Describe this narrower assertion accurately.

## Historical RED and required verification

The [combined68 ordinary-runner.log](../combined-68/ordinary-runner.log), SHA-256 `3b75cacdac2bcc7016dc5f7a99f924f6b958f0cbc7d58e59afe77980b7b34b1e`, retains the following actual failed terminal names. [run-binding.json](../combined-68/run-binding.json), SHA-256 `50780dd40849c42621f0f1d707132fb44b022b1aaf7fb9846da2315b79483d28`, binds frozen HEAD `524c092a3f6cbf5834895ae5ba7821d6fa610924` on developmental Linux/aarch64. Its full historical run has 673 named outcomes: 392 PASS, 269 FAIL, 12 SKIP. These are historical observations, not results for an unexecuted future candidate.

| Actual failed terminal name | Raw log line |
| --- | ---: |
| `TestSeedCompletionKeepsCampaignStatistics/successes` | 2768 |
| `TestSeedCompletionKeepsCampaignStatistics/distinct_and_duplicate_failures` | 2773 |
| `TestSeedCompletionKeepsCampaignStatistics/first_failure_cancels_active` | 2778 |
| `TestSeedCompletionKeepsCampaignStatistics/failure_budget` | 2783 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure` | 2788 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_cancelled_attempt` | 2793 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_success` | 2798 |
| `TestSeedCompletionKeepsCampaignStatistics/supervision_failure_drains_a_failure` | 2803 |
| `TestSeedCompletionKeepsCampaignStatistics/prepared_target_integrity` | 2808 |
| `TestSeedCompletionKeepsCampaignStatistics/campaign_cancelled_while_running` | 2813 |
| `TestSeedCompletionKeepsCampaignStatistics` | 2815 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/malformed_World` | 2822 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/supervision_rejected_the_choice_trace` | 2827 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/watchdog` | 2832 |
| `TestSeedCompletionFaultsKeepCampaignStatistics/cancelled_execution` | 2837 |
| `TestSeedCompletionFaultsKeepCampaignStatistics` | 2839 |

These are fourteen leaf executions and two emitted parents, not sixteen independent cases. No intermediate slash name is invented. The preceding diagnostics retain validation-stage preparation errors before the existing comparisons. The independent report establishes source feasibility only; it executed no changed-source assertion and predicts no successor PASS count.

Before either insertion, the admitted worker must retain meaningful unchanged-source preparation RED for both exact tables on its actual bound candidate, plus the unchanged focused controls. Use pinned Go 1.27.1/tools, `test_dep`, `-count=1` and externally bounded commands. Missing tools, compilation failure or timeout establish no meaningful preparation RED. After exactly two insertions, execute every original row and assertion. Any newly reached original assertion or validation failure must be retained and returned to root for separate bounded admission; it grants no authority to change expectations, helpers, fixtures or production behavior.

Keep the existing preparation controls `TestPreparationDependenciesForwardRealFixtureInputs`, `TestPreparationDependenciesOperationErrorsRemainUnchanged`, `TestPreparationDependenciesFailuresStopAtOriginalStages`, `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard`, `TestInjectionCharacterizationIsolatedPreparationDependencies` and `TestPortableProfilePublicGuardsRemainFirst` in the focused before/after gate, with their source unchanged. Public/default, executor-only, prepare-only bootstrap and isolated guards retain actual behavior. No additive fixture or helper is required.

Retain one compact actual receipt per command binding before/after consumed-source identities, candidate HEAD, tool and wrapper bytes, environment and effective Go settings, absolute CWD, argv array, numeric exit, wall elapsed time and immutable raw-log hashes. Bind the actual Perl executable, checker scripts and exact checker inputs before execution and after completion. Postcapture seals cannot retroactively provide missing execution-time checker binding. Shell entry must use `login:false`, `env -u BASH_ENV bash -c` and explicit `cd` into the assigned checkout. Reference existing evidence and wrapper conventions rather than copying bulk histories; rebind task-specific paths and inputs.

Retain formatting, complete selective-removal reconstruction, affected vet/errortype, configured unfiltered Runner lint and required repository `make lint-code-fast`, with pinned tools and fixes disabled. Generated validation, package architecture, private/public boundaries, both supported source-set static checks (`darwin/arm64` and `linux/amd64`) and all still-owned source gates require current evidence or exact reconciliation against valid retained evidence for every actual unaffected consumed input. A whole-fingerprint match is not a PASS extrapolation. Root serializes every shared Go/build/lint/vet/generator lane; this draft grants none.

## Root comparison and acceptance ownership

Root owns a future frozen ordinary comparison against the actual immediately preceding frozen baseline selected at execution time. It must bind the integrated candidate and enumerate actual terminal names from both runs. The combined68 packet above supplies retained historical RED and source research, not an invented immediate baseline or revised PRIMARY result. Permit outcome changes only in actual root-admitted fixture scopes; preserve every unrelated original and synthesize no parent/intermediate slash outcomes. Forecast no PASS total.

Compare all fifty complete current original-base lint blocks, requiring zero introduced or removed findings. Preserve actual aggregate nonzero exits and errortype reachability; focused tests, standalone analyzer success or diff-filtered fast lint cannot certify aggregate green. Fresh independent integrated source review and all remaining owned acceptance govern completion. Verified source progress may be checkpointed by root while RED50 and other source requirements remain open; this proposal supplies no formal Done, SHIP or aggregate acceptance.

## Exclusions

Exclude every other consumer, shared helper, import, comment, assertion, fault/result payload, target metadata, production/public/default path, real bootstrap decoder, real process transport/runtime forcing, replay/resume/minimize/guidance, simulation, adapter/installation identity, schema/generated input, runtime/toolchain and lint-policy change. Actual artifact publication/journal inspection and cancellation partial-state observations remain meaningful source controls; they establish neither target execution nor artifact replay/crash-resume qualification. The synthetic marker cannot satisfy the real bootstrap decoder, which remains unchanged.

Sibling tasks retain their own workers and evidence. Preserve dirty owner spec, `MILESTONES.md` including fn155 and unrelated user files. Native fn128/fn149 remain deferred and unverified. No native revival, full supported-native test-host result, runtime/replay/soak qualification, measured bound, CI, PR or push authority follows from this proposal.
