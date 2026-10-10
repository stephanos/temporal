# fn-109.73 bounded source assessment

The frozen Unix mode fixture preserves its seven real filesystem permission checks and adds exactly the admitted private preparation assignment. This source assessment identifies no introduced code defect. Execution evidence, integrated preservation and source acceptance remain conductor-owned and open.

## Scope and identities

Assessment date is 2026-10-10. Requested reviewer route is `gpt-6.1-sol/high`, in the same GPT family as the writer. Tier is session with `jev-unavailable(no_key)`; actual host model telemetry is unproved. This fresh-context assessment is code-only and supplies no formal implementation-review, SHIP, Done or readiness verdict.

Candidate C is `/Users/stephan/Workspace/skunkworks/gomad/temporal/.worktrees/fn-109-73-unix-mode-candidate`. Its observed HEAD and comparison BASE are `ca4d8b88cf95da0b0a911efdc3a18f89bcbb68ae`. The working file, rather than committed HEAD, contains the edit. Its SHA-256 is `dd24ed23e85d82a61ec4526a8cdc3908d6ea1195df9caee32ab5450ca4f862f6`, with 40 lines. BASE blob `d2d6f8cbcca71f4727b4d1da4078c8245a98f4b0` has whole-file SHA-256 `ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168`, with 39 lines. Candidate product status and the BASE diff identify only `tools/gomad3/runner/runner_mode_unix_test.go` as changed under `tools/gomad3`.

Authoritative inputs came from PRIMARY P, `/Users/stephan/Workspace/skunkworks/gomad/temporal`. The dirty owner spec was read from P, including R5/R18/R19; the isolated historical copy was not substituted.

| PRIMARY input | SHA-256 |
| --- | --- |
| `AGENTS.md` | `8d634df5cbbbffd7dbada06e32b4d20d879707273f8be07bdf253b387211e6f3` |
| `tools/gomad3/README.md` | `fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610` |
| `MILESTONES.md` | `a16476b5a8bc7a56d7d2d4086cafca59a9667959ea0687b9053d0dfe64345991` |
| `.flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md` | `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c` |
| `.flow/tasks/fn-109-gomad-deepen-modules-and-tool-interfaces.73.md` | `1ccb6e2dfcd264c66af4c53826b2cf2bcd4ba825f65223ea82199f90195de092` |
| `task-73/admission.md` | `9fb8b3ad698b5d355e45229b48c96865d21ee13de1966bd7bd0bb45a464ee44a` |
| `task-73/worker-admission-20261010.md` | `f029f4773c87b0d9756406d9beb403e207ceb6a850eebfe913e2930c181d6c64` |

The two `task-73/` inputs above resolve under `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/` in P and were read completely. The project guide, Gomad README and milestone delivery requirements were read before assessment.

Initial commands supplied C through the tool's `workdir` argument without an explicit inner `cd`; their observed `pwd` was P and they read the original 39-line PRIMARY fixture. Those reads and the initial apparent candidate mismatch are excluded from candidate evidence. Subsequent candidate reads used `login:false`, `env -u BASH_ENV bash -c` and explicit `cd C`; observed `pwd`, HEAD and edited hash confirmed C. The correction was sent to the conductor before conclusions.

## Strengths

- `tools/gomad3/runner/runner_mode_unix_test.go:18` inserts exactly `configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)`, immediately before `exploreWith`. A read-only checker required exactly one occurrence, removed only that full assignment line and compared the resulting whole file with the actual BASE blob using `cmp`. The command exited 0 and the reconstructed hash is `ebdc20609fd89c246bf345e0df40f3c126b09765d898770dabe4389518228168`. The checker used `/usr/bin/perl`, SHA-256 `0953404d494ccb2618aaf418313376fc217a243ec21574c7f2a0dfa005e0acc3`, bound immediately before its invocation. This mechanical check supports exact source preservation only.
- `runner_mode_unix_test.go:14` still creates `newFakePreparer`, `fakeExecutor` and `testConfig` before `syscall.Umask(0o777)` at line 15. The saved umask and deferred restoration at line 16 remain byte-identical. The assignment, `exploreWith` and all assertions remain inside that umask interval. The fixture contains no `t.Parallel` or new parallel execution; its original campaign parallelism remains 1. A `t.Fatal` still runs the installed defer.
- `preparation_fixture_test.go:17` retains the supplied executor directly in the returned private dependencies. Its preparation closure validates the forwarded preparer and nonempty preparation root, calls that preparer's `Prepare`, validates the resulting target metadata and calls real `prepared.Verify`. `campaign_options.go:211` carries those dependencies into the request, and line 263 carries `config.Preparer`. `runner_local.go:250` passes the final target, environment and preparer into the operation; lines 84-86 select the injected outer executor. `runner.go:712` invokes that executor with the prepared target and bootstrap bytes.
- `runner_test.go:1981` creates the original target before the restrictive umask. `runner_test.go:2012` reads those target bytes and writes a real copy into the journal-owned preparation directory, explicitly applying mode 0500. `target/target.go:305` verifies the regular executable's actual hash and size through `hashRegularFile` at line 889. The helper and local completion/finalization retain those integrity checks at `preparation_fixture_test.go:35` and `runner_local.go:527`/`:857`.
- The permission observations still cover actual campaign publication. `runner_local.go:197` creates `CampaignJournal`; `campaign_journal.go:190`/`:200` create and chmod the campaign root, and line 203 creates `failures` and `.partial` through `makePrivateDirectoriesContext`. That function explicitly chmods created directories at line 642 and the final directory at line 646. `segmented_journal.go:170` creates `executions` through the same owner. `runner_local.go:716` appends the successful execution. The segment opens through real `os.OpenFile` and applies 0600 at `segmented_journal.go:257`/`:261`, then seals through a real rename at line 292. Its index uses `atomicWriteContext` at line 207. `runner_local.go:872` calls campaign publication, which writes `campaign.json` through that same atomic writer at `campaign_journal.go:387`. The writer applies 0600 to its real temporary file at line 574 before rename. The mutation observer in `campaign/filesystem.go:31` only checks a context hook; the fixture uses `context.Background` and no filesystem substitution.
- `runner_mode_unix_test.go:23` preserves the complete table and `:32` preserves actual `os.Stat(filepath.Join(summary.CampaignPath, path))`, fatal handling of stat errors and the permission comparison. All seven original observations remain.

| Observed path | Required permissions | Source row |
| --- | --- | --- |
| `.` | `0700` | `runner_mode_unix_test.go:24` |
| `failures` | `0700` | `runner_mode_unix_test.go:25` |
| `.partial` | `0700` | `runner_mode_unix_test.go:26` |
| `executions` | `0700` | `runner_mode_unix_test.go:27` |
| `campaign.json` | `0600` | `runner_mode_unix_test.go:28` |
| `executions/index.json` | `0600` | `runner_mode_unix_test.go:29` |
| `executions/00000000000000000000.jsonl` | `0600` | `runner_mode_unix_test.go:30` |

The helper's bootstrap value remains the explicitly synthetic marker at `preparation_fixture_test.go:15`/`:41`. The unchanged `fakeExecutor.Run` at `runner_test.go:2332` records its request and returns a scripted result. This fixture establishes no real bootstrap decoding or process-launch coverage. Public `Explore` still constructs empty private dependencies at `runner.go:373`, default preparation and bootstrap still reach their real owners at `preparation_dependencies.go:17`/`:24`, and isolated execution still rejects substitutions at `runner.go:382`. The unchanged default/bootstrap controls in `preparation_dependencies_test.go:205` and isolated controls in `executor_injection_characterization_test.go:151`/`:168` retain those refusals. The sole test-local edit widens no public/default admission.

## Critical findings

None identified in the admitted source change.

## Important findings

None identified in the admitted source change.

## Minor findings

None identified in the admitted source change.

## Evidence limits and code-only conclusion

This assessor ran no Go, build, lint, vet, generator, bridge or native command, and reviewed no pending worker gate as completed evidence. The exclusive execution lane remains with the worker. Missing current execution receipts are acceptance requirements for the conductor to reconcile, rather than introduced code defects. In particular, source reasoning cannot establish that every unchanged assertion executed successfully.

The supplied retained aggregate lint RED50 remains acceptance-open. The task still requires candidate-bound RED, controls, execution, applicable source checks, complete original-base lint-block preservation and the conductor's actual-predecessor integrated comparison and fresh independent integrated source/evidence review. This source-only report cannot close those requirements. Native fn-128/fn-149 remains deferred and unverified; no supported native full-host pass, replay claim or soak bound follows from this assessment.

The inspected working file matches the exact admitted one-assignment scope, preserves all other BASE bytes and still routes the seven permission assertions through real filesystem publication. No corrective product edit is indicated by this bounded source assessment.

## Consumed candidate source identities

Paths below resolve under `C/tools/gomad3/`. The edited-file hash was reconfirmed after the source inspection.

| Source input | SHA-256 |
| --- | --- |
| `runner/runner_mode_unix_test.go` | `dd24ed23e85d82a61ec4526a8cdc3908d6ea1195df9caee32ab5450ca4f862f6` |
| `runner/preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `runner/preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `runner/preparation_dependencies_test.go` | `c74d91de827cb171fb9565f697c6254f02b2091c69e51cbf916f76ce0921ecb9` |
| `runner/runner_test.go` | `d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96` |
| `runner/campaign_options.go` | `f6081a272e1b4657a8ce18effaaf81123016b0cb7badc922efcbf9c2b9eec1fa` |
| `runner/runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner/runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| `internal/preparation/preparation.go` | `52f02f1dc0d1f90910eb9c7e409093e1731e01c91467084c040f4369417247f6` |
| `runner/internal/campaign/campaign_journal.go` | `acd68ff0131f4d2d1f6050a0cb6e53e8a4ebc8e6a58cd5299a2cbaa7cc8b2f25` |
| `runner/internal/campaign/segmented_journal.go` | `b724320128fa5d111ce42329226b6408cf4b794eda3a78b5dd23ac59f0d7b38f` |
| `runner/internal/campaign/filesystem.go` | `b29fc8ddd25839cc38616437954bc94d7335498dbef095385f519fa10ba5b573` |
| `target/target.go` | `bf5a1c8e193650913220fd3a1de6ae2aa0dbf264f1a77a1c77bb52bb9a848bf7` |
| `runner/executor_injection_characterization_test.go` | `b225b832599a605854e297b662c71d4c4e3684a903b40ccf6805c9ff7ef7ef0d` |
