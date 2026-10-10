# Ordinary Runner source frontier after task75

The retained task75 ordinary run has 537 PASS, 130 FAIL and 12 SKIP terminals, with 87 distinct failing top-level tests. Source inspection identifies 17 self-contained tests with 20 fresh scripted call sites eligible for a proposed existing-helper correction. No new correction owner, implementation admission, test result or review verdict follows. Fn155 remains first and owns the shared Go/build/lint/generator lane.

This current classification supersedes the residual-count assumptions in the historical [source frontier](source-frontier-20261010.md) and older 123-test inventory. Their evidence remains unchanged. The research route remains requested `gpt-6-astra/high`, with the previously supplied judge state `session`, unavailable `no_key`; actual model telemetry is unverified. No rerouting or judge retry occurred. Drafting follows `/home/agent/.codex/docs/flow-next/prose.md`.

## Evidence binding

PRIMARY HEAD was `a226b92b48f5a1580860f3f851afd7284d1851c9` throughout this follow-up. Worker W is PRIMARY's `.worktrees/fn-109-75-six-attachments-candidate`. Its BASE is `cc0c1c90fa5d2ac3f386c7a13fb51637d144c857`, source A is `4bb2f25bd7c694aeea5a201def5ffa795e10cc08`, and packet B is `71b6e95cf2dae0ca3fe856aa08cb256a03d64acd`. PRIMARY integrated source A as `46699234193d33e385297905a7d2492246065ea1` and packet B as `87ab8ebdfb8d8545e72d1bd2fc07abdafd2cce18`.

Raw evidence remains under W's `.flow/tmp/fn10975-evidence/`. Read-only rehashing and JSON classification reproduced the following identities and counts.

| Retained input | SHA256 |
| --- | --- |
| `after-ordinary.log` | `f24a23b7dfa70f51dcd1122e354b7456f3dfafc5970fdf6fd7e1ff4cc3b3a7d4` |
| `after-ordinary.json` | `ff0cd62ebac95cfbbad92f0832da81299b910b817d401c041a78ae6787920c91` |
| `after-ordinary-binding.json` | `b6c31eb4116fa4f6db17232d15aed57c3846adba1690179e545c82f786458756` |

The retained command is `go test -tags test_dep -count=1 -json ./runner`, cwd W's `tools/gomad3`, exit 1, elapsed 84.476 seconds. It ran from 2026-10-10T11:46:09.416412Z to 11:47:33.892461Z with stable captured inputs and terminal owned processes. This report executed no Go command.

The committed Runner source comparison from A to current HEAD is empty. The prior [integrated source assessment](../../fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-retention-diagnostics-inspection-20261010/integrated-source-assessment-20261010.md) retains the broader 5,169-product-entry equality and execution binding. Its [reconciliation](../../fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-retention-diagnostics-inspection-20261010/post-task75-reconciliation-20261010.md) and [complete verifier result](../../fn-109-gomad-deepen-modules-and-tool-interfaces/scripted-retention-diagnostics-inspection-20261010/root-source-ordinary-complete-verification-20261010.json) retain 14 prior FAIL-to-PASS changes, six newly reachable original bounds leaves, no unselected normalized outcome/diagnostic changes and `product_acceptance_pass=false`. The six task75 attachments are repaired boundaries and are excluded from new proposals.

## Current boundary map

Counts below are disjoint failing top-level tests. Call-site counts are lexical fresh `exploreWith` sites, including loop-local invocations and the crash test's child helper.

| Group | Tests | Fresh sites | Scope and next boundary |
| --- | ---: | ---: | --- |
| Self-contained scripted campaigns | 17 | 20 | Existing helper can supply explicit preparation/bootstrap while retaining actual orchestration and semantic decoders. Eleven ordinary tests and six guidance tests already supplying scripted replayers. |
| Fresh setup followed by resume | 15 | 19 | Fresh executors are scripted. Successful continuation reaches unconditional retained adapter validation and bootstrap; new resume scope is required. |
| Mixed fresh campaign and plan/shard | 4 | 4 | Diagnostic profile, frozen guided selection, empty guided shards and regression-mode plan identity. Fresh attachment alone cannot repair plan/shard setup. |
| Default real guidance replay | 1 | 1 | `TestRunGuidanceWithoutReplayCapabilityFailsClosed` selects the default artifact replayer and enters real replay preflight. |
| Plan/shard without fresh Explore | 8 | 0 | Seven current `portable_plan_test.go` failures and the shard injection-characterization test. Direct plan preparation and internal shard preparer need their own scope. |
| Scripted retained replay/minimize | 27 | 0 | Shared real-executable fixture publisher, installed identity, retained profile/adapter validation and direct bootstrap encoding. Existing campaign helper is insufficient. |
| Real process/toolchain/profile | 15 | 0 | Actual compiler, adapter, coordinator, bootstrap, World child transport, I/O, mount or watchdog paths. Preserve their real boundaries. |
| Total | 87 | 44 | Proposals only; no broad migration or passing-count forecast. |

The smallest cohesive self-contained groups are failure policies (2 tests/2 sites), seed evidence/path/mount forwarding (3/3), choice evidence/inspection/watchdog classification (3/3), simulation exploration success/deduplication (2/2), physical target sharing (1/1), guided corpus admission ordering (1/1), guided identity/coverage (3/4), and guided selection/accounting (2/4).

The six guidance candidates already use `matchingReplayer` or `replayRecorder`. Those callbacks open actual artifacts and inspect corpus state, without calling `replayWith`. Preserve real corpus publication, index ordering, rejection cleanup, selection identities and exact replay callback counts. They establish scripted orchestration coverage only. Existing-helper eligibility never permits synthetic bootstrap bytes to enter a real bootstrap decoder or target process; existing World, choice, transcript, simulation, artifact and journal validation stays real.

## Smallest failure-policy proposal

The next proposed two-assignment group is in [runner_test.go](../../../../tools/gomad3/runner/runner_test.go). Both current diagnostics are the unsupported deterministic-I/O profile refusal, before either executor reaches its original behavioral assertions.

| Existing test | Proposed insertion before current call | Required original assertions |
| --- | --- | --- |
| `TestRunFirstFailureCancelsActiveTargetsWithoutPublishingThem` | Line 548, after final `testConfig` | Attempts 3, failures 1, cancelled 2, distinct failures 1, first-failure stop, one artifact and two cancelled partials; preserve three-executor rendezvous and cancellation. |
| `TestRunBudgetCountsDistinctSignatures` | Line 578, after `FailureBudget = 2` | Attempts/failures 4, distinct failures 2 and budget stop; preserve repeated signature before seed 4's distinct result. |

Each proposed statement is exactly `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)`. Use the final preparer and outer executor. The current errors are at lines 550 and 580. `testConfig:2504` supplies executor-only dependencies; [preparation_dependencies.go](../../../../tools/gomad3/runner/preparation_dependencies.go) therefore selects real preparation, whose validation reaches the supported-host guard before scripted execution. `firstFailureExecutor.Run:2484` and `fakeExecutor.Run:2332` ignore bootstrap bytes.

Preserve the complete original file after removing only the two proposed assignments, including earlier attachments, helpers, comments, assertions, payloads and budgets. No helper, production, public/default guard, timeout, cache or error-wording change is proposed. Root would need a new bounded owner/admission and serial lane grant after fn155; existing fn109.65-.75 admissions enumerate already integrated sites. Acceptance consumers remain fn109.63/.21 and fn112.10.

## Boundaries requiring another owner

[resume.go](../../../../tools/gomad3/runner/resume.go) reconstructs and verifies the retained target, then unconditionally calls `Default().VerifyAdapters`, even for empty adapters. That operation invokes the platform guard. An injected executor skips the later installed-toolchain check but not adapter admission. Resume skips fresh preparation and uses bootstrap for continued executions. Preserve original plan/target checks, journal recovery, novelty, committed rounds, frozen guidance and dependency forwarding. In `TestRunResumeRejectsChangedEvidence`, the Runner identity leaf can reject earlier; the retained-success corruption leaf reaches later validation. A partial leaf improvement supplies no complete resume proof.

[portable_plan.go](../../../../tools/gomad3/runner/portable_plan.go) calls `preparation.Prepare` directly and does not consume the private injected preparation operation. [campaign_shard_execution.go](../../../../tools/gomad3/runner/campaign_shard_execution.go) builds its own `campaignPlanPreparer`; binding the helper to an outer fake preparer would not preserve that contract. Keep portable target/mount identity, ordinal partition, completeness, deduplication and public process-executor guards.

The 27 scripted retained tests comprise eleven replay-operation tests excluding the two World process transports, all nine current minimization-operation failures, diagnostic replay, minimization inspection, minimizer target sharing, three replay/minimize injection-characterization tests, and unsupported watchdog-choice rejection. The existing [retained-fixture research](replay-fixture-boundary-20261010.md) identifies the missing private retained-execution contract. The publisher reads actual executable bytes/build information and installed identity; replay preflight verifies profile, identity, adapters, choice/World/mount evidence and build information. Replay/minimize encode bootstrap directly. Preserve independent expected identity and actual negative validation; neither copying expected identity from the artifact nor substituting nil-returning validators is a correction.

The 15 real-boundary tests are the three coordinator transport failures, two environment integration failures, native guided-corpus reproduction, covered-binary rejection, three replay-I/O integration failures, two World child-transport replays, captured-input watchdog replay, pinned exploration benchmark and real Sprig adapter preparation. The Sprig test has a fake executor but deliberately no fake preparer. None admits a synthetic target/bootstrap conversion. No CLI-package failure belongs to this 87-parent Runner domain; CLI acceptance needs separate evidence.

## Remaining evidence limits

The 130 terminal failures group by observed diagnostics into 64 explicit platform refusals, 42 missing-toolchain errors, 13 preparation-validation errors, two non-resumable lifecycle errors, one unexplained crash-helper exit and eight parents without their own diagnostic. The 42 missing-toolchain terminals include scripted retained fixtures and cannot all be transferred to native owners.

Task74's earlier `TestRetentionKeepsTheSameRunsAndArtifactsForEveryStrategy/failures/choice-exploration` failed with `artifact_publication`, `sync artifact store: context deadline exceeded` and one committed observation against three expected. [Diagnosis](../../fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/retention-deadline-diagnosis-20261010.md) locates the error after rename but cannot attribute the consumed ten-second budget. The [disposition](../../fn-109-gomad-deepen-modules-and-tool-interfaces/task-74/deadline-failure-disposition-20261010.md) preserves that unresolved observation. Task75's policy-table pass supplies no retrospective cause. The divergence crash helper exited before readiness with empty captured stderr; that cause is also unproved. Preserve its actual SIGKILL/reap protocol rather than labeling the historical exit as demonstrated platform refusal.

Ordinary RED130, collateral RED22, Runner RED6, aggregate RED50 and unreached integrated errortype remain source acceptance gaps. No retry or count subtraction closes them. Native full-host/soak qualification remains deferred and unverified under fn128/fn149; those deferrals neither excuse portable failures nor block separately admitted source corrections. Fn155 retains its own first-platform proof and current priority. No native revival, CI, push, PR, SHIP or Done follows.

Only this new evidence report was written. Requested raw seals/counts matched with no discrepancy. The historical source-frontier report was preserved unchanged at SHA256 `5f5f1cc0a8db695f1962b7332b13f0739f06e7433d5af69391e0788aeba08c47`. No product/index/Flow lifecycle mutation, Go command, test, lint, compiler, generator, delegate or review verdict occurred.
