# Retained-success preparation fixture slice

Recommend exactly two existing call-site attachments in `tools/gomad3/runner/retention_test.go`. Both fixtures already meet `scriptedPreparationDependencies`' target and argument checks. The helper can supply their private preparation and synthetic bootstrap while leaving real target copying, target sharing, full-byte capacity, artifact publication, campaign journal validation and disk assertions in place. No metadata repair, shared-helper change, production change, new seam or assertion change is needed by the inspected paths. This is a feasibility finding, not an observed passing result.

This report is read-only source research except for this one authored Markdown file. No Go, test, build, lint, vet, generator, Flow operation, Git mutation, native operation, CI, PR or push ran. Root retains admission and verification ownership. The task68 worker retains its separate exclusive Go lane and `runner_test.go` implementation scope.

## Bound source and retained RED

The inspected primary checkout is `/Users/stephan/Workspace/skunkworks/gomad/temporal`, HEAD `7727b062b0c263046f0409e8f9d6cf5e58e7c0ef`. The owner-amended fn-109 spec has SHA-256 `851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c`. Its opening amendment removes byte/format compatibility requirements while retaining capabilities, classifications, transaction guarantees and resource lifetimes. The proposed two assignments change no recorded contract. `MILESTONES.md:42` prioritizes fn-112.10's retained source acceptance and preserves separate native ownership.

The retained ordinary capture is `.flow/artifacts/fn-109-gomad-deepen-modules-and-tool-interfaces/combined-66-67/ordinary-runner.log`. Its companion `ordinary-runner.json:2` records `go -C tools/gomad3 test -tags test_dep -count=1 -json ./runner`; line 15 records exit 1 and line 21 records unchanged capture inputs. `run-binding.json:3` binds the isolated capture directory, line 7 binds raw frozen HEAD `f699252450b8e67f1edb50ed8e4cff4cb6e644c0`, and lines 8-10 identify Linux/aarch64. That raw source identity is separate from the primary research HEAD.

`outcome-comparison.json` records 673 named outcomes across all levels, with 389 PASS, 272 FAIL and 12 SKIP. Both selected top-level names are failed outcomes. Raw lines 2281-2284 record the shared-target test failing its first measured run at `retention_test.go:111`, with Attempted 0 and the unsupported linux/arm64 preparation error. Raw lines 2287-2289 record the same preparation error at line 140 in the same-output test. These failures precede the selected behavioral checks. They establish preparation RED on the bound capture, not a retention defect or a predicted GREEN after attachment.

The raw capture's `source-before.json:789`, `:801` and `:806` contain the same preparation-helper, retention-test and runner-test hashes listed below. Independent `git show 41727a2ce2ac6a59562db7f62613c06d7f30502b:<path> | sha256sum` checks matched the current retention and preparation-helper bytes. `git diff --exit-code 41727a2ce2ac6a59562db7f62613c06d7f30502b HEAD --` for those files plus `runner_test.go` returned 0. The corresponding diff from raw frozen `f699252450b8e67f1edb50ed8e4cff4cb6e644c0` to primary HEAD for `runner/`, `artifact/` and `target/target.go` also returned 0. This reconciles the inspected paths; it does not assert that every repository input or execution environment is identical.

| Evidence under combined-66-67 | SHA-256 |
| --- | --- |
| `ordinary-runner.log` | `f97441bc0b8f4ebc4da6673b0e3b097a9420fe6e5bd1c1871de83c6f333eb032` |
| `ordinary-runner.json` | `7b746348fa9739d1e7e926d8958d70e68bfc0dec52a83bd037438ebd38b40c55` |
| `outcome-comparison.json` | `5e4c48dd7caef408a4f4c5dcffc6be045219e7ffafc6b0533a424c2fb7129a7b` |
| `run-binding.json` | `45f31ddfd586858fda9d0b9afb76e421a0f989c659957ba0727fe23f55073bb5` |
| `source-before.json` | `d83f75ce0a33e327c057b83b7d45b17f34b608102a38f54782849cf733043598` |

The neighboring convention report is `post-task66-next-slice.md`, SHA-256 `b2e54c6b48f714e53e83e4f0f129c025346c6db9d06da7899e56c576e0934033`. Its earlier combined64/65 counts remain historical. This report uses the later combined66/67 capture.

## Exact proposed Touches

The complete proposed source Touches set is `tools/gomad3/runner/retention_test.go`. Insert this existing assignment at each site below.

```go
configDependencies = scriptedPreparationDependencies(t, config.Preparer, configDependencies.executor)
```

| Existing function | Exact insertion point in the inspected source | Existing behavior retained |
| --- | --- | --- |
| `TestRunCountsASharedTargetInFullAgainstTheSuccessByteLimit` at line 86 | Inside its local `run` closure, after line 106 sets `SuccessBytesLimit` and immediately before line 107's `return exploreWith(...)` | One syntactic attachment applies to both original `run` invocations. The generous run measures two distinct artifacts and requires their target files to be the same inode. The reduced-limit run must return `success_retention_capacity` with one retained success. |
| `TestRunRetainsSameOutputSuccessesWithMatchingDiskAndJournalCounts` at line 129 | After line 137 sets `SuccessBytesLimit` and immediately before line 138's `exploreWith(...)` | Two identical-output successes remain two artifacts with distinct seeds and paths, matching summary, on-disk directory and journal counts and byte totals. |

Deleting exactly those two added lines must recover the original file. Keep all imports, comments, fixtures, policies, seeds, limits, invocations and assertions unchanged. In particular, keep the assignment inside the first test's `run` closure, so each newly constructed preparer and executor gets its own dependencies. `testConfig`, `newFakePreparer`, `fakePreparer.Prepare`, `fakeExecutor.Run`, `processResult`, `completeEmptyTranscript` and `scriptedPreparationDependencies` remain unchanged because they also serve excluded consumers.

## Metadata and executor requirements

`runner_test.go:2501` constructs `KindGoRun`, source `.`, no target arguments, no adapter replacements and the supplied preparer. It returns only `executionDependencies{executor: executor}` at line 2508. Both selected tests preserve that target and preparer. `newFakePreparer` at line 1978 supplies matching kind/source, argv `[gomad3-target]`, empty tags/compatibility, default-profile Go/platform fields, a fixed build key, and an actual file with its hash and size. Neither selected path needs adapter selection, installation identity or replay preflight.

The shared-target test writes `bytes.Repeat([]byte("target bytes "), 8<<10)`, which is 106,496 bytes, after making the source fixture writable. Line 98 updates both SHA-256 and Size before preparation. `fakePreparer.Prepare` at `runner_test.go:2009` reads that real source, writes the campaign's `.prepared/target`, applies mode `0500`, copies all metadata and changes only Path. The helper at `preparation_fixture_test.go:24` invokes this actual preparer. Its line 32 checks kind, source, argv0, argument equality and absence of adapters; line 35 calls `Prepared.Verify`; line 38 supplies the explicit empty adapter slice. `target/target.go:305` checks compatibility and actual hash/size. Its `hashRegularFile` at line 889 rejects symlinks and non-regular/non-executable files. Exact `0500` is imposed by the preparer and checked by the existing forwarding control, rather than being an exact-mode requirement of `Prepared.Verify` itself.

The original default route reaches `preparation.Prepare` through `preparation_dependencies.go:17`. `internal/preparation/preparation.go:54` selects real profile validation, and lines 97-103 call the custom preparer before the validation-stage failure. `runner_local.go:243` owns this fresh preparation before scheduling. The proposed helper replaces just that private operation and `bootstrapFrame`, whose default remains at `preparation_dependencies.go:24`.

`runner.go:681` obtains the bootstrap bytes and lines 692-712 forward them into `execution.Spec.IO.Config`. `fakeExecutor.Run` at `runner_test.go:2329` records the request, derives the seed from Env, and invokes the existing result callback. It neither executes target bytes nor decodes `IO.Config`, reads its transcript backing, or writes the partial stdout/stderr handles. The callback already supplies captured output with real hashes through `processResult`/`output` at lines 2525-2535 and a complete empty transcript through line 2552. The helper's explicitly synthetic bootstrap marker therefore satisfies this scripted consumer. It establishes no real bootstrap-frame or runtime execution proof. Real partial output files still open and close through `runner.go:664` and `:730`.

## Real retention and disk boundaries

`runner_local.go:527` verifies the prepared target again before assessing completion. `recordSuccessfulExecution` at line 651 calls the unchanged retention decision, composes an actual execution manifest and publishes through `artifact.PublishArtifact` at line 676. That call retains `StoreKeyExecution`, the campaign's real successes directory, remaining success-byte budget and `artifact.TargetPool(config.Artifacts)`. The dependency attachment supplies none of those outputs.

`artifact/publication.go:52` publishes the target from its real prepared path with mode `0700`, plus actual output payloads. Lines 78-81 add the complete empty transcript payload, and lines 130-143 add World payloads. `artifact/store.go:123` calls `placeSharedPayload` for the target. `artifact/target_pool.go:42` uses `os.Link`, and lines 59-78 create/verify the pool link or take the existing private-copy fallback. The first test's `os.Lstat` and `os.SameFile` checks at `retention_test.go:113` and `:119` still require real sharing. A filesystem that takes the fallback cannot satisfy that assertion; the attachment provides no excuse to skip or weaken it.

Each artifact's stored-byte count includes every file and its manifest at `artifact/store.go:278`, independent of inode sharing. `runner/retention.go:43` supplies the remaining budget, and `artifact/store.go:149` compares full stored bytes with that budget before final publication. `successPublicationFailure` at `retention.go:59` maps `artifact.CapacityError` to `success_retention_capacity`. The first test reduces the generous run's measured total by half the target size, 53,248 bytes, at `retention_test.go:122`; line 124 requires the second run to retain exactly one success before the capacity error. Keep this relative measurement and negative assertion. No new byte estimate or expected value is needed.

The identical-output test exercises the real execution-identity collision branch at `artifact/store.go:192`. The store distinguishes campaign/ordinal/seed at line 292 and derives an execution path at line 299 when another execution already occupies the same outcome-signature path. `runner_local.go:682` annotates the actual published path and StoredBytes, lines 683-685 advance summary retention, line 716 appends the journal record, and line 872 publishes the campaign summary. Their ordering remains unchanged.

`retention_test.go:142` reopens the real campaign, and line 146 independently lists its successes directory. Line 150 requires two summary successes, two summary paths, two disk entries, two journal executions and two published campaign successes. Lines 156-175 verify each journal path against its returned path, reopen each artifact, preserve Close error checking, require seed 1/2 and matching journal seed, check the full stdout hash for `same output`, and require equal outcome signatures with distinct paths. Line 177 compares summed reopened StoredBytes with both summary and published-campaign totals. These are logical full-file byte counts; they do not measure allocated filesystem blocks or total pool disk usage.

The reopen operation itself stays substantive. `campaign/open_campaign.go:67` reads the published execution journal, line 71 validates summary counts, and line 75 validates retained artifact capacity. Lines 228-242 sum valid per-execution success references and bytes; line 282 compares those with the campaign record. Line 308 calls `ResolveRetainedEvidence`; `retained_evidence.go:24` opens the actual artifact and line 40 checks ordinal, seed, success classification and stored-byte identity. `artifact/open.go:33` pins the real directory, checks its mode and manifest, validates the listed payload tree at line 68, and computes full stored bytes at line 77. No journal, artifact or disk-return substitution is proposed.

## Controls and failure boundary

The first selected test already supplies the decisive byte-capacity negative control after a successful sharing measurement. The second supplies the same-output collision regression and independent disk/journal/summary agreement. An admitted implementation needs no new helper or test to preserve these original checks. Its unchanged-source RED must be retained at the actual admitted candidate before making the two insertions, and its selected result must be measured afterward. Task68's different choice-exploration slice supplies no result for these names.

Keep these existing controls in the focused verification selection, with their source unchanged.

- `TestRunFailsClosedWhenSuccessRetentionCountIsExhausted` and `TestRunRejectsSuccessfulRetentionWithoutReplayTranscript` at `runner_test.go:412` and `:429` distinguish count exhaustion from missing replay evidence. Task66 already attached their own call-local dependencies; neither moves into a shared helper.
- `TestPreparationDependenciesForwardRealFixtureInputs` at `preparation_dependencies_test.go:31` checks real copied bytes, mode/size, input forwarding, synthetic marker delivery and closure of real output files. `TestPreparationDependenciesOperationErrorsRemainUnchanged` at line 135 preserves exact return/error forwarding.
- `TestPreparationDependenciesFailuresStopAtOriginalStages` at line 165 requires zero executor calls after prepare/bootstrap failures and preserves their distinct attempted counts and error classification. `TestPreparationDependenciesKeepRealDefaultsAndBootstrapGuard` at line 205 retains real public, executor-only and prepare-only refusal on an unsupported host. A prepare-only attachment must still fail at the real bootstrap guard.
- `TestInjectionCharacterizationIsolatedPreparationDependencies` at `executor_injection_characterization_test.go:168` rejects prepare/bootstrap injection before the coordinator starts, while preserving resume preflight precedence. `TestPortableProfilePublicGuardsRemainFirst` at `deterministicio/profile_portable_test.go:162` retains unsupported-host ordering for public profile operations.

An attachment can expose an additional failure at preparation metadata verification, campaign planning, artifact publication or journal validation. Retain that diagnostic before proposing any scope change. The first test's earliest behavioral checks remain line 110's two distinct artifacts, line 119's shared inode and line 124's one-success capacity result. The second test's downstream boundary begins with real campaign opening at line 142 and continues through counts, seed/output identity and full-byte totals. Failure at any of these boundaries provides new evidence; it does not authorize changing an assertion, fabricating a transcript, forcing target sharing, disabling validation or repairing production inside this two-assignment scope.

## Source hashes and exclusions

All paths in this table are relative to `tools/gomad3/` and bind the cited primary source.

| Source | SHA-256 |
| --- | --- |
| `runner/retention_test.go` | `35a5599194d809a364751743318dc3c9e7304810bbeb8fd819e69dc8d3f14194` |
| `runner/preparation_fixture_test.go` | `c43c4fb18ad07b9b9bbb6efba9dc5a6194a99d86ae2405cd0dc4b6088943402e` |
| `runner/runner_test.go` | `7045165b88318f57fb147882b051cb7bfd2aac8c3182039a0cca0fce6f5af8e0` |
| `runner/preparation_dependencies.go` | `4f9e93b79fc75e984a34e6fa7591bf4ff077db5dd1e96330697483b9af434e56` |
| `runner/preparation_dependencies_test.go` | `c74d91de827cb171fb9565f697c6254f02b2091c69e51cbf916f76ce0921ecb9` |
| `runner/executor_injection_characterization_test.go` | `b225b832599a605854e297b662c71d4c4e3684a903b40ccf6805c9ff7ef7ef0d` |
| `runner/runner.go` | `dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e` |
| `runner/runner_local.go` | `162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692` |
| `runner/retention.go` | `3ed085af90aa61a0360c0c8147708a81b95a4e62a059f3102e0e547fdbbb1f6d` |
| `target/target.go` | `bf5a1c8e193650913220fd3a1de6ae2aa0dbf264f1a77a1c77bb52bb9a848bf7` |
| `internal/preparation/preparation.go` | `52f02f1dc0d1f90910eb9c7e409093e1731e01c91467084c040f4369417247f6` |
| `artifact/store.go` | `f93601eccc225ab158f3c6f922b95e85deb4358de34cd7ff889b7d5533acddab` |
| `artifact/target_pool.go` | `2f16b5c0ec47226bd80b63d309f37e28ccfe55c24a6e2b54e2118a7fd0b12b8e` |
| `artifact/open.go` | `69cd4af287008c5985050b6d9a7a03f3b5c96ad695bc1b005eb9c3fe2431b0de` |
| `artifact/publication.go` | `44d9d007b0b09397654e62e24596f8321e14366a6797a6f31fc63ec1fb827671` |
| `runner/internal/campaign/open_campaign.go` | `1003d279b9b66cc34a51254c3205db602ec69b98af3cc61ecf71c816ebe8fc0f` |
| `runner/internal/campaign/retained_evidence.go` | `fcadd5148a33de7fc1593c3f6c2f471c190323b161df9302e42bc46c6a81a337` |
| `deterministicio/profile_portable_test.go` | `6dfd6c349c6c6a05995d5f9343541e0529ab7eb2e453607de3026c02dd4932c4` |

Exclude every other call site, shared helper, production owner, real compiler/bootstrap test, replay/minimize/resume path, guidance/corpus operation, adapter/installation identity hook, schema, generated input and native toolchain. Existing artifact and campaign storage stay unchanged, including the code assigned for later replacement by fn-152/fn-153. This recommendation changes fixture reachability only and grants no additional implementation authority.

Ordinary source coverage, lint, both-source-set checks, generated validation, preservation obligations still applicable under the current owner amendment, and integrated source review remain with their existing owners. There is no new full-suite PASS, measured outcome reduction, D10 completion, supported-native replay proof or determinism bound. Deferred fn-128/fn-149 retain native qualification; this report revives neither. User dirty files and all cited Go sources remain untouched.
