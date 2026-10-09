# Ordinary Runner fixture boundary research

An explicit private operation boundary could let genuine Runner orchestration units run on the ordinary host while keeping public preparation and runtime validation intact. This is a read-only research recommendation, not an admitted implementation or a passing coverage claim. Root requested gpt-6-astra/high; actual execution telemetry is unobserved. The researcher executed no gates or writes and inspected primary source at ce4b95ea06.

## Failure inventory

The retained combined57-59/full-ordinary-runner.stdout contains77 distinct failed top-level tests before the diagnosed startup hang. This is a historical partial observation, not the current candidate's failure count.

| Family | Historical failed top-level tests | Boundary |
| --- | ---: | --- |
| Choice divergence, including crash/resume | 5 | Fake-executor orchestration with real journal/crash assertions |
| Completion characterization | 3 | Orchestration and error precedence |
| Diagnostics | 4 | Explore/plan/shard/resume/replay orchestration |
| Guidance identity/selection | 8 | Fake execution with real corpus/selection behavior |
| Retention characterization/retention/modes | 8 | Orchestration with real filesystem assertions |
| Portable planning | 7 | Fake preparation with real plan/mount/shard assertions |
| Inspection | 3 | Projection with orchestration-built fixtures |
| Minimization | 9 | Fake execution with real identity prerequisites |
| Executor-injection characterization | 5 | Mixed private-unit and public/default-path coverage |
| Replay operation | 13 | Mostly fake replay, including two real transport cases |
| Runner core before the hang | 2 | Fake preparation/execution orchestration |
| Coordinator/environment/coverage/preparation/replay-I/O integration | 10 | Real boundaries excluded from blanket migration |

TestReplayRejectsFirstDivergentWorldTransitionBeforeTargetMutation and TestReplayExecutesMatchingWorldPlanThroughChildTransport use recordReplayIOTranscript and real process transport in [replay_operation_test.go](../../../../tools/gomad3/runner/replay_operation_test.go). Sharing a file with fake replay tests does not make these unit fixtures.

## Existing owners and propagation

Public Preparer changes target preparation only. [The preparation owner](../../../../tools/gomad3/internal/preparation/preparation.go) still attaches adapters and validates. Its private services are unavailable to Runner. [executionDependencies](../../../../tools/gomad3/runner/runner.go) currently contains only executor. No existing lawful switch covers the following operations.

- Complete preparation in runner.go and [portable_plan.go](../../../../tools/gomad3/runner/portable_plan.go). Any private substitute must replace the operation as a unit. The default retains preparation.Prepare, adapter attachment, validation order and stage errors.
- BootstrapFrame in runner.go, [replay_operation.go](../../../../tools/gomad3/runner/replay_operation.go) and [minimize_operation.go](../../../../tools/gomad3/runner/minimize_operation.go). Explicit unit substitutes must assert target, Runner identity and seed arguments and cannot claim validated runtime input.
- Recorded-adapter verification in [resume.go](../../../../tools/gomad3/runner/resume.go) and replay, including verify-only replay.
- Toolchain identity reads in replay preflight and existing process-executor shard/resume branches. Replacing whole preflight would remove real artifact, World, build-info and capability coverage and is excluded.

[campaignRequestFromSpecWith](../../../../tools/gomad3/runner/campaign_options.go) copies only executor. Resume reconstruction and [default guidance replay](../../../../tools/gomad3/runner/guidance.go) also rebuild from only executor. Shards, resume, minimization sessions and default replayers need an explicit propagation inventory. Isolated Explore must reject explicit substitutes before launch; defaults must remain lawful and functions must never enter coordinator wire data. The shared replay-artifact fixture separately reads the real .toolchain and serves mixed callers. A blanket constructor replacement would silently change coverage.

## Options and recommendation

1. A campaign-only pilot supplies preparation, bootstrap and adapter verification to an explicit list of Explore/plan/shard/resume units. It leaves replay, minimization and default guidance replay incomplete.
2. A four-operation private dependency value, introduced through the campaign pilot and a separately admitted identity-dependent slice, retains the original entry paths and real journal/artifact/preflight/retention/comparison logic. Research recommends this option. Dropped dependencies during reconstruction and accidental use by real-boundary fixtures are its principal risks.
3. New post-preparation/post-preflight entry points avoid some dependency fields but change wider control flow and can stop tests exercising their original paths. This has the larger preservation cost.

Likely full product scope is runner.go, campaign_options.go, portable_plan.go, resume.go, campaign_shard_execution.go, replay_operation.go, minimize_operation.go and guidance.go within tools/gomad3/runner. Same-package fixtures require an explicit per-test inventory and constructor. No deterministicio or preparation API expansion is recommended. Do not globally replace testConfig, the shared replay-artifact constructor, mixed public/private calls or real transport fixtures.

## Required proof before acceptance

Zero dependencies must retain exact real operation calls, arguments, ordering, concrete error types, wrapping and precedence, including unsupported-host and missing-installation rejection. Spy controls must prove preparation before bootstrap before execution, no downstream work after failure, one preparation per campaign and complete resume/shard/guidance/minimization propagation. Every migrated unit retains its original assertions and real filesystem/artifact validation; wrong identity/argument and injected-error controls must fail causally.

Real preparation-stage failure/cancellation, public profile guards, adapter/identity checks and artifact/build-info mutation tests remain independent real-boundary coverage. A public call replaced by a private call changes coverage and needs explicit admission. No global hook, public API field, automatic fake-executor bypass, profile mutation, skip, fake-host substitution or native qualification follows. A new correction owner must establish this scope beyond tasks6/7 before implementation. Root will first finish the current wave and measure the complete ordinary Runner candidate; the historical partial inventory alone cannot close its acceptance.
