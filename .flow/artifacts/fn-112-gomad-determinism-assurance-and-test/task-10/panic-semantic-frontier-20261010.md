# Eight remaining panic sites, 2026-10-10

The smallest proposed correction is the single choice-inspection guard in `runner/inspect.go:647`. Its callers already return errors and own artifact cleanup. Replacing its panic with an error still changes internal-failure semantics and needs a separate dated admission. None of the eight sites has a demonstrated public-input path to its explicit panic in the inspected source. This is a bounded source finding, not proof against future defects, unsafe mutation or races.

This report extends [lint-policy-frontier-20261010.md](lint-policy-frontier-20261010.md). The current primary fn-109 owner and the October 7 decision remain authoritative. The latter admits only two different controller panics and expressly excludes others (`.flow/artifacts/source-unblocking-20261007/owner-decisions.md:11`). No authority, task status or source changed here. No Go, build, lint, generator, test or native command ran.

## Exact retained findings

Paths below are relative to `tools/gomad3/`. The retained task 73 log names eight statements in six files, rather than eight files. `runner.go`, `process_unix.go` and `world/replay.go` supply caller/state context only.

| Site | Exact panic operand |
| --- | --- |
| artifact/manifest_copy.go:59 | `fmt.Sprintf("artifact manifest cannot copy a %s field", source.Kind())` |
| deterministicio/profile.go:211 | `fmt.Errorf("encode deterministic I/O inventory: %w", err)` |
| runner/campaign.go:11 | `"gomad3: duplicate execution completion"` |
| runner/campaign.go:25 | `"gomad3: execution completion order has an unresolved gap"` |
| runner/inspect.go:647 | `fmt.Sprintf("unknown validated choice kind %d", kind)` |
| runner/internal/execution/launch_plan_unix.go:321 | `fmt.Sprintf("unknown descriptor resource %q", resource)` |
| world/world.go:202 | `"invalid request state"` |
| world/world.go:216 | `"queued request has no queued event"` |

The log SHA-256 is `430ff67ecce2756ba007e5270488ddfa1a64dea2bf53698b1d76d3e9d52a07d3`; diagnostic locations are log lines 38, 47, 59, 62, 68, 71, 182 and 185. The complete fifty-block SHA-256 `034b5959d8f6689fefbd9215234326d66b3809e204cf628c62b244e256398eea` is inherited from the prior independently checked frontier report, not recomputed here.

## Caller, state and test findings

### Artifact copy

`Opened.Manifest` and `Opened.Snapshot` call `cloneManifest`, which recursively calls `deepCopy` (`artifact/open.go:100`, `artifact/manifest_copy.go:15`). Only the fixed `record.ExecutionRecord` enters from production. Decoding a user artifact cannot inject a new Go field kind. The guard rejects interface, function, channel and unsafe-pointer fields, including nil values. Existing full-record population tests exercise the current record shape; the guard would become production-reachable after an incompatible type change.

The destination can already contain copied fields, allocated pointers, maps and slices when recursion rejects a later field. That destination is a local clone; the original manifest stays unchanged. The helper opens no resource. Its caller may hold an `os.Root`, and metadata remains accessible after Close (`artifact/open.go:44`, `:84`, `:100`). Changing signatures must preserve those handle-lifetime guarantees.

Existing tests are `TestManifestCopiesCannotChangeOpenedHandle`, `TestCloneManifestSharesNoMemory`, array/reference isolation, nil/empty and scalar preservation, and eight explicit panic assertions for unsupported nil/nonnil kinds (`artifact/opened_test.go:210`, `:244`, `:257`, `:284`, `:327`). These last assertions explicitly preserve the current failure mechanism; fn-109's current owner records that preservation at line 677.

A credible redesign makes a checked generic copier return an error and validates the record shape at an error-returning construction boundary, or uses a typed/generated clone with complete field-coverage checks. Neither a partial clone nor silently shared unsupported values is acceptable. Preserving the current public no-error accessors while retaining generic future-shape rejection needs an explicit construction/clone contract. Admission must specify panic-to-error behavior, supported helper shapes, nil/empty preservation and test migration. Moving the panic into a helper or deleting the rejection tests is not a correction.

### Deterministic profile initialization

Package initialization calls `mustSpec` once for `deterministicProfile`; same-package platform-golden tests also call it (`deterministicio/profile.go:185`, `deterministicio/profile_test.go:39`, `profile_portable_test.go:144`). Its inventory contains strings, string-backed digests, slices and structs (`profile.go:47`). There is no public profile-definition constructor. The current private encoder first marshals, then encodes into a bytes buffer with HTML escaping disabled (`deterministicio/domain.go:40`). Current inventory values contain no functions, channels, cycles, floats or custom marshalers that provide an ordinary encoding-error route.

Before panic, only local inventory assembly and encoding occurred. The inventory and implementation digests are assigned after the guard (`profile.go:213`). Initialization opens no resource or transaction; an actual failure stops package initialization before callers can handle it. Existing tests pin both platforms' inventory bytes, digests and bootstrap frames, and immutable Default behavior (`profile_test.go:39`, `:79`, `:89`). There is no direct encoding-failure characterization for this fixed type.

The credible choices are a generated, validated inventory representation that removes fallible runtime initialization, or an error-returning constructor/resolver with failure propagated to every profile consumer. The latter changes `Default`/initialization semantics and has a wider caller surface. Ignoring the encoder error or returning an empty profile is not credible. Coordinate this site with fn-153's encoding work; its R2 requires standard encoding and pin regeneration, but replacing this private helper with `json.Marshal` alone leaves a fallible API and does not discharge the panic (`fn-153...md:37`, `:50`, `:60`).

### Campaign completion ordering, two sites

`localCampaign.runSeeds` starts the orderer in a goroutine, fed by `launchSeed → runSeed`; the scheduler consumes ordered completions (`runner/runner_local.go:318`, `:432`, `:448`; `runner/runner.go:647`). Every current `runSeed` return path sends exactly one completion bearing the internally assigned job. Private executor injection can change its result/error, not directly forge another completion send or job ordinal.

The duplicate guard checks only entries still in the pending map. Example sequence 1,1 while ordinal 0 is outstanding reaches it. The second send is rejected before overwrite, but earlier arrivals and even earlier emitted results may already exist. A duplicate of an already-emitted ordinal instead remains pending and can reach the gap guard. The gap guard fires after input closes with nonempty pending, for example input 1 without expected 0. It does not diagnose a completely missing trailing completion when pending is empty. Normal first-failure/budget cancellation can leave unissued selection ordinals and must stay valid (`runner/campaign.go:5`).

Workers have begun partial-execution journals and may have executed children. Normal completion closes output handles before sending; earlier ordered completions can already have published artifacts or journal entries (`runner/runner.go:650`, `:727`; `runner_local.go:498`, `:595`). A panic in this goroutine cannot be recovered by the caller goroutine's defer. Simply returning an error from the orderer can instead strand workers or the scheduler. Input closure currently occurs only in `runSeeds`' deferred cleanup, after scheduling and `finishSeedCampaign` (`runner_local.go:352`, `:377`), so a late ordering failure must be observed before publication in a redesigned protocol.

Existing controls cover out-of-order completion and novelty retention, first-failure cancellation, budgets, shard/resume ordinal filtering and controller statistics (`runner/runner_test.go:289`, `:544`, `:567`; `campaign_shard_test.go:50`; `campaign_test.go:9`). No direct duplicate/gap panic test was found in the scoped test search.

A bounded owner must add a terminal ordering-error channel, stop admission, cancel and join workers, drain sends without inventing successful completions, retain already committed evidence and incomplete partials, then return a distinct HostError before final publication. Tests must include pending and already-emitted duplicates, a gap after an emitted prefix, early stop with an unissued suffix, cancellation and blocked-sender cleanup. This is a process-fatal-to-recoverable-failure migration, not a local replacement. Coordinate transaction changes with fn-152, whose R2-R5 replace persistence; that spec does not explicitly delete this in-memory orderer (`fn-152...md:56`, `:106`).

### Choice inspection

`Inspect → projectChoices → choiceKind` maps projected sites; `projectChoices → projectReplayDecisions → choiceKind` maps tape decisions (`runner/inspect.go:396`, `:557`, `:613`). Stored traces are decoded and projected before either call. Wire validation rejects unknown kind bytes, and `ProjectTrace` revalidates the stored bytes (`choice/internal/wire/wire_generated.go:192`, `:232`; `choice/trace.go:182`). Therefore malformed artifact bytes receive existing decode errors before the panic. The guard represents an internal projection/enum disagreement.

Only local projection slices have changed. Inspect holds a validated artifact handle and defers Close, joining close errors after a projection error (`runner/inspect.go:401`). It performs no publication. Existing tests cover validated runnable sites, all three trace kinds and tape decision ordinals, malformed trace evidence, and inspection error/cleanup behavior (`runner/inspect_test.go:92`; `runner_test.go:1066`; `choice/trace_test.go:136`; `inspect_cleanup_test.go:16`, `:74`). No direct invalid-kind mapping test was found.

Proposed next unit changes `choiceKind` to `(string, error)`, keeping all three known strings and returning the existing diagnostic text for an unknown kind. Both projection loops must propagate failure and return no partial report. Add known/unknown-kind table tests and a malformed stored-kind case with a valid recomputed trace digest so it reaches kind validation; retain existing cleanup and replay-ordinal tests. A small checked projection helper can expose invalid internal site/decision values to tests without bypassing public validation or adding a production injection seam.

The dated admission must explicitly replace this one panic with an inspection/projection error, retain malformed-input precedence and zero-report cleanup, and choose its CLI classification under the existing inspection error route. No fallback string such as `unknown` is acceptable. Neither fn-152's campaign inspection rewrite nor fn-153's encoders expressly replace this artifact choice-kind mapper.

### Descriptor resource lookup

`Run` and the supervisor call `resources.closeInherited → closeInheritedStage → descriptorSpecFor` after successfully starting their child (`runner/internal/execution/process_unix.go:301`; `supervisor_unix.go:189`, `:201`; `launch_plan_unix.go:303`). The lookup key comes from `descriptorLayout`, built from private specs or fixed choice/simulation/diagnostic resource constants. All currently emitted constants have a spec (`launch_plan_unix.go:118`, `:150`). A missing public resource file already fails in `filesForStage` before Start (`:287`); the panic requires internal layout/spec drift.

Earlier inherited ends can already be closed when lookup fails. The child is live, and deferred resource cleanup alone does not execute the explicit early-supervisor or kill/reap error branch. Existing error paths at `process_unix.go:304` and `supervisor_unix.go:201` already supply that cleanup. Tests verify missing files, retained versus closed ends, pipe ownership, stage layouts and descriptor-number stability (`process_unix_test.go:18`, `:61`, `:86`, `:154`).

The smallest error-based design returns a checked lookup result and propagates it through closeInheritedStage, preserving prior accumulated close errors and using the existing child cleanup branch. An alternative stores close policy directly in each validated binding and eliminates the second lookup. Either needs coverage across every capability combination and fault-injected unknown bindings after earlier closes, including child reaping and primary-error precedence. Admission must define internal descriptor-plan failure as a returned infrastructure error instead of panic. fn-152/fn-153 do not own this launch plan.

### World cancellation, two sites

Public `Model.Cancel`, mailbox cancellation and snapshot-transition reconstruction reach both guards (`world/world.go:183`; `world/mailbox/mailbox.go:95`; `world/snapshot.go:164`). Unknown request IDs return an existing classified error. Private state is created Pending, changed to Queued by Ready, and changed to Delivered by Quiesce under the same mutex (`world.go:120`, `:171`, `:273`). Restore reconstructs transitions into a new model and compares state rather than installing arbitrary supplied request states (`snapshot.go:69`). Thus neither an arbitrary public RequestState string nor a corrupt snapshot directly installs the guarded internal state.

At line 202, an existing request with a state outside the four constants panics after transition-capacity checking but before transition construction or mutation. At line 216, cancellation has passed recording/replay checks and set `request.state = RequestCanceled`; a nonzero event ID then resolves to an event whose state is not Queued or whose heap index is negative. A missing map entry would cause a separate nil dereference before this explicit guard. The current check also does not protect an out-of-range positive heap index or a mismatched queue entry.

The mutex unlocks on panic. At the second guard, the event and heap are unchanged, history/digest/cursor have not committed, but the request is already canceled and recording.pending may have been set (`world/recording.go:184`). `checkReplay` is read-only; replay payload/cursor and recording used bytes change only in `commitTransition` (`world/replay.go:260`, `:356`). There are no file handles or process resources inside Model.

Existing controls cover pending/repeated/post-delivery cancellation, queued cancellation across snapshot restore, corrupted-snapshot refusal, replay no-mutation and recording capacity no-mutation (`world_test.go:98`; `snapshot_test.go:9`, `:61`; `replay_test.go:54`; `recording_test.go:8`). No direct internal-state panic assertions were found.

A credible correction validates state and the event/map/heap relationship before recording reservation or request mutation, then returns a distinct model-invariant failure. Admission must specify that failure's identity, process-terminal classification and whether the damaged model can be used again; relabeling corruption as ordinary invalid user input is unjustified. It also must explicitly admit removing the partial mutation at line 212 on this failure. Add same-package corruption tests for invalid state, missing event, wrong state, negative/out-of-range index and wrong queue entry; assert unchanged request/event/heap, history, digest, recording pending/used and replay cursor/payloads. Preserve normal cancellation and existing public error precedence. Neither future storage/encoding owner replaces these state transitions.

## Source binding and limits

HEAD before and after source inspection was `c2b92ec179115228bc47ad439d23bd3c63bff908`. During report authoring, HEAD advanced to `f71bc9133bbda2b73d670a44f7b9c8674bc28087`. The final binding check at that HEAD found zero changes in the forty consumed source, test, owner and log files below. The six panic files and authoritative owner/log identities were bound at initial inspection. Caller-name discovery and the first read of the future-owner specs preceded those supporting files' first hash readings; their detailed follow-up inspection and final hashes stayed within the binding window. These hashes bind this report, not any current-candidate test pass. Research/prose skill instructions governed source attribution and wording only.

All eight direct guards currently have observable panic semantics for their internal invalid inputs. No proposed error-based removal preserves those semantics exactly. Structural removal of an impossible state can preserve valid executions, but still needs explicit characterization and the appropriate owner admission. The next reviewable corrective unit is the choice-inspection migration above; the broader eight-site migration is not admitted by this report.

```text
4bf26d4bcf510564b6bb0745b689667f2e00c3590e9bf78e9a8891a83f678928  .flow/artifacts/source-unblocking-20261007/owner-decisions.md
851151bc3b5ea0ac9bfda873f108a593653a9becbb66323d241244955274fd2c  .flow/specs/fn-109-gomad-deepen-modules-and-tool-interfaces.md
c36ff356093f9b04bfef0e1b5dde6b2291358494bf80ad208b5428405450c7fe  .flow/specs/fn-152-gomad-runner-storage-on-one-append-only.md
00a21e051a9b325981d51f386f9e06941f7332136bd4126bb2922a9b530b1e10  .flow/specs/fn-153-gomad-retire-canonical-json-and-private.md
430ff67ecce2756ba007e5270488ddfa1a64dea2bf53698b1d76d3e9d52a07d3  .worktrees/fn-109-73-unix-mode-candidate/.flow/tmp/fn10973-evidence/after-aggregate-lint.log
f1dad686d9f2fc59ebde1f0f484bafa6dcfad7fb1f84d1d3fe7878eeb44d7dae  tools/gomad3/artifact/manifest_copy.go
69cd4af287008c5985050b6d9a7a03f3b5c96ad695bc1b005eb9c3fe2431b0de  tools/gomad3/artifact/open.go
016bb31d600047178bc355784dd2635ef44dbbb6d2b351a8687913a8623b76b4  tools/gomad3/artifact/opened_test.go
b8b1c24b4c9257759e3cf152259d3f02a1a13bf2df3fef8bf7b5e42c39f607f5  tools/gomad3/choice/internal/wire/wire_generated.go
e7c8c7774e25d83a0007ba3c234d7004d9ec49affa062c5b448d993299e0559e  tools/gomad3/choice/tape.go
be764b5f667258179864c81bad5e0bd547bed5822a59da0356494fe3fda51dba  tools/gomad3/choice/trace.go
f163d39e14483acb07296a21e28220c7a2d1e12c97580c682eb31bb7783a0b87  tools/gomad3/choice/trace_test.go
7c028b4d403b6fac1df5469a82575905c5be8f4ab3e5c9f71c4d796a0fc589fb  tools/gomad3/choice/wire.go
18bd47fd50a389e55e02270a083c7f83c6611403b9d0a358ef650b90a701b066  tools/gomad3/deterministicio/domain.go
d068900a3b76bc91e5b94f0d67e00e67e225c50c0f3d23cbf92ce992ef0c5f21  tools/gomad3/deterministicio/profile.go
6dfd6c349c6c6a05995d5f9343541e0529ab7eb2e453607de3026c02dd4932c4  tools/gomad3/deterministicio/profile_portable_test.go
48b6dd1b8d6b363d3ef4ed3e2b73754a708a2a46683bc429ef927de047fc24d8  tools/gomad3/deterministicio/profile_test.go
738375708f711bb44094f57a346ec9131159f5ef61da402b3a62ec82b15f9a41  tools/gomad3/internal/canonicaljson/canonical.go
f76d2c59da4fc899587c5ef4f048cecdde608589cd332dd85d0ee149a93d83e9  tools/gomad3/runner/campaign.go
9a1f90864f5bc0b80f15b18cf7518c9c1979b8ae4bc863883940a2f358e4e393  tools/gomad3/runner/campaign_shard_test.go
f34a5e236c0f880e4fb9e4e2339f985c5c98f054d523531a1ce136bc1743f885  tools/gomad3/runner/campaign_test.go
94d69de99dbb7a30c9a307dfd4a2e51b35f24b49aaf0ae0d1da5f77766af7359  tools/gomad3/runner/inspect.go
965db096bedfb28bdacc24a96c01dfa1f1ea461dd066d8e31e3b8a8dfc354fa5  tools/gomad3/runner/inspect_cleanup_test.go
140e6688d3e1dddb8dbc1e85ae53de0de15fd676f655227efae6fa09282ca8df  tools/gomad3/runner/inspect_test.go
52e6c04e83e59be86ca580aa700130108280d65fdcb33da8f47e5082e69bf964  tools/gomad3/runner/internal/execution/launch_plan_unix.go
388f3756e72b11c5e52bc7c71911f04816af2a308d9b83966ce1c72138674df6  tools/gomad3/runner/internal/execution/process_unix.go
9f9f6158d8a75c58ac6956dff4b4f825752e7322bb3d387590ddff95db415701  tools/gomad3/runner/internal/execution/process_unix_test.go
7f7155151a4fedb732891bb458f15bab9bd27b5857d5ad0f0aafd03825ebac70  tools/gomad3/runner/internal/execution/supervisor_unix.go
dcfe7f2d14c4bbddba89bf536a010eddd2b690e6b47f0aecc5bc2a800664160e  tools/gomad3/runner/runner.go
162dbed7bf6f82da56b356ebb5c4ec17c402434ca86507a2f6b90311dd7cc692  tools/gomad3/runner/runner_local.go
d251cc9c5f32b95821fcede257df76ad2fe147c4e915128812a2e1ef275a5e96  tools/gomad3/runner/runner_test.go
3bc2d360dab408f4486a5df47b6618057f7aee7b0b3071e3a112cfdc9d896818  tools/gomad3/world/mailbox/mailbox.go
fc00c9af42ebb2cb74fa593b9c026205a2df19bf4634e99a532f697c7fb4fc27  tools/gomad3/world/recording.go
20848d717654c0df7e1ad3da0f7db9e4b20cdf8fae66520dd79a6fcd25b37365  tools/gomad3/world/recording_test.go
3e32d751adb715f3ae1c3d0e66fdd40f05946ca77d251fb0c7ade417cde302ee  tools/gomad3/world/replay.go
52cd78fb24c9fa62a6a785614f8f59cda01dbf12d8bf54f8cb62f3672aa152fe  tools/gomad3/world/replay_test.go
065d173b05de29f83babc444818f2fd9491df90fc9d5d073ce495daf5ed78219  tools/gomad3/world/snapshot.go
303aeb7a66d393f4800f7c04801e55846833030d5af3bd2062d6dc4e904934e7  tools/gomad3/world/snapshot_test.go
5de21e439b3f04aa8bb9dc44b6ce21c95b556952fb4b70d98f403dbb2d4b68c6  tools/gomad3/world/world.go
36f0c4b4186009888a723b4d90b8c82b25cc56086e8733bd8fe8e8c267092ede  tools/gomad3/world/world_test.go
```

