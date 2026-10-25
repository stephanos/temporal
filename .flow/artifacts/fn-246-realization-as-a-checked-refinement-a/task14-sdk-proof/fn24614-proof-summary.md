SDK v1.48.0 delivers the tested same-task heartbeat histories when native heartbeat uses an activity-value-preserving context bound to the Run from its first batching window. The test-only checkpoint is committed; fn-246.14 remains in_progress for production continuation, schema/admission controls and conductor review.

Tier: explicit implementer gpt-6.1-sol at medium; judge no_key fallback already documented by conductor. Executed model metadata was not exposed to this worker.
stage: impl-review - skipped(policy: conductor owns the integrated review)

Workspace: /Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/fn24614-sdk-proof
Base: 0e88397afc3a4c8466643f058db863db0bb45932
Commits: c5b77bf2d3dcea8a310ac576ac275bc0324253a4, f2b05ef81888cff6d7bdc1ae9a96dbea80781b38
Preparation SHA-256: e0762c6acb7e5b399d0cd8d3f23c15f3c0d1b0b8fdf02daa3eeb19fa85bed52b

Similar code search found sdk/namespace_test.go. The fixture follows its actual lazy-client/loopback WorkflowService/worker pattern. Production sdk.go, program.go, raw.go, schemas, Rust, Testpilot and Flow lifecycle records remain unchanged.

The native activity calls activity.RecordHeartbeat. The client dial interceptor sits inside the SDK retry interceptor, clones the request before actual invocation and clones each actual response/error afterward. Invocation numbers distinguish retries and out-of-order transport completions. One assigned task carries attempt 7 and its opaque token throughout. Background empty polls supply no continuation. The Run owns the detached context and deadline, its AfterFunc callbacks and delayed completion publishers. Cleanup joins those components, the handler, public worker Stop/client Close and server Serve.

| Control | Actual result | Test anchor in sdk/heartbeat_test.go |
| --- | --- | --- |
| Repeated false, false-to-true, after-true | Actual false,false,true,true completions on the same assigned token, independently ending in Completed, Failed and Canceled RPCs | TestNativeHeartbeatDetachedFromFirstWindow, line 375 |
| SDK values across first-window detachment | Custom converter metadata, propagated scenario, activity attempt and identical activity logger survive; original deadline is retained | heartbeatConverter, line 38; newHeartbeatWorker, line 148 |
| Original server cancellation | Actual true response causes temporal CanceledError on the original SDK handler context | Same positive control and ordering controls |
| Public Stop flush isolates opening context | With the Run alive and an hour throttle, the queued heartbeat fails Canceled/no server call for an original opening context and completes actual true for a detached-from-first opening context; both then send Completed and join Stop | TestNativeHeartbeatStopFlushUsesOpeningContext, line 401 |
| Throttle/coalescing and failure flush | Calls for details 1,2,3 in the existing hour window produce one actual tail heartbeat carrying 3 before actual Failed | TestNativeHeartbeatCoalescesAndFlushesLatestBeforeFailure, line 455 |
| Original unknown/false/true terminal subset | Unknown and false each select actual Completed/Failed; true independently selects Completed/Failed/Canceled, with one terminal request | TestNativeHeartbeatTerminalSelection, line 479 |
| Public client alternative | A true Client.RecordActivityHeartbeat completion returns CanceledError but leaves the native handler uncanceled | TestClientHeartbeatDoesNotCancelNativeHandler, line 506 |
| Completion before SDK delivery | A real true transport completion is held before returning to SDK; no native cancellation appears until release | TestNativeHeartbeatCompletionPrecedesSDKCancellationDelivery, line 527 |
| SDK cancellation before observer publication | Original SDK cancellation arrives while the immutable actual completion waits in a Run-owned publisher; release publishes that same invocation | TestNativeHeartbeatSDKCancellationPrecedesCompletionPublication, line 691 |
| Run cancellation and Close | Poll, handler, heartbeat RPC, batch, delivery hold and terminal hold all shut down and join; unpublished actual completion is retained separately and publisher joins | TestNativeHeartbeatRunCancellationJoinsBlockedWork, line 549; TestNativeHeartbeatRunCancellationJoinsUnpublishedCompletion, line 715 |
| SDK retry | Actual Unavailable attempt 1 and same-request successful true attempt 2 have distinct invocation identities; neither API return nor failed attempt supplies a bit | TestNativeHeartbeatRetryCapturesEveryTransportAttempt, line 606 |
| Other cancellation causes | NotFound, PermissionDenied, pause and reset preserve distinct actual causes and never qualify as temporal CanceledError | TestNativeHeartbeatOtherCancellationCausesAreNotServerCancel, line 635 |
| Malformed wire response | Invalid protobuf response produces actual Internal/unmarshal failure, followed by bounded Run shutdown; no successful completion is admitted by the test | TestNativeHeartbeatMalformedResponseIsNotFalse, line 735 |
| Crossed token/namespace and reordered completions | A public-client true completion for another token/namespace finishes invocation 2 before the outstanding native false invocation 1; the original handler stays uncanceled | TestNativeHeartbeatCrossedClientCompletionDoesNotCancelHandler, line 758 |

| SDK path | Representability established by this phase | Limit |
| --- | --- | --- |
| Native heartbeat with Run-bound detachment from first call | Tested repeated-false, false-to-true, after-true and all three required terminal choices | Tested finite histories through depth four; production row matching and completeness are open |
| Native heartbeat using original activity context, then switching after true | Public Stop flush fails the pending original-context window while the paired detached-from-first window completes actual true | Switching only after true cannot replace the original window context |
| Public Client.RecordActivityHeartbeat alone | Returns the actual server cancellation error | Does not deliver cancellation to the native handler; cannot substitute for native proof |
| Current production raw and SDK followers | Existing terminal-only suites pass | Neither production follower gained continuation in this phase |

Baseline was green for the focused SDK package before edits. Verification passed `go test -tags test_dep ./...` in the Temporal host module and `go test -race -tags test_dep ./reference/worker/sdk -count=10` with the assigned cache, GOMAXPROCS=2 and GOFLAGS=-p=2. The initial race run reported 20.242 seconds; the correction race run reported 19.443 seconds. Changed-file golangci lint reported zero issues, formatter diff was empty and errortype vet passed. Root `make lint-code-fast` selected no packages because it excludes nested modules; the explicit module lint supplied the actual changed-Go check. Initial unfiltered lint also exposed the inherited sdk.go:83 ignored assertion result, which remains unchanged.

The first error-cause characterization failed because PermissionDenied canceled the handler. Source explains the distinction between retry's raw gRPC code and its later converted serviceerror, whose absent GRPCStatus is seen as Unknown by native retry.IsRetryable. The final test asserts the concrete cause. A later provisional API-return assertion failed because native batching can return before the true response; the final fixture keeps API return, actual completion and cancellation separate. Exact failed logs and final green logs are retained in fn24614-proof-logs with hashes in the evidence JSON.

Pinned source anchors are internal/internal_activity.go:397-414, internal/internal_task_handlers.go:2182-2330,2388-2505,2586-2643, internal/internal_task_pollers.go:1515-1545,1584-1655 and internal/grpc_dialer.go:101-132,147-182. internal/common/retry/interceptor.go owns the PermissionDenied conversion distinction. `go list -m go.temporal.io/sdk` returned v1.48.0. The evidence records exact source hashes.

Remaining integration requirements are explicit. Production exact-history keys, schemas, codecs, startup admission and both follower continuations are still open. Production notifier rejection of wrong/crossed attempt/token/namespace, duplicates, reordered/ambiguous completions, missing continuations, exhausted bounds and unsupported fields is unproved. The crossed-completion control captures actual identity/order and native cancellation isolation; it does not implement those admission checks. Terminal validation-failure/wrong-kind mutants, production multi-task reconciliation and activity-deadline expiry remain open. SDK private goroutine internals were not introspected; the proof uses public Stop/Close and joins every fixture-owned callback/publisher/handler. Broader converter/interceptor combinations remain unproved.

No actual Temporal service ran. fn-237.4 retains every original cooperative live witness and its evidence journal/wrapper/barriers/replay/gates. The independent protocol Model retry-contract conflict remains unresolved by this SDK work. Parent protobuf/Rust/guest/live gates and the task's done/review lifecycle remain conductor-owned.

All owned commands are terminal. The checkout is clean and retained at this planned phase boundary. Before/after disk checks reported 132 GiB free; the assigned Go cache is retained for continuation.


Proof review correction

The conductor accepted the P2 timing finding in fn24614-sdk-proof-review.md. The original negative control assumed that its 100 ms batch window remained open until the detached second call. Scheduler delay could instead let the empty window expire and make the second call start a valid fresh RPC. The earlier green runs remain historical evidence and did not exclude that race.

Correction commit f2b05ef81888cff6d7bdc1ae9a96dbea80781b38 replaces that timing dependency with paired `TestNativeHeartbeatStopFlushUsesOpeningContext` cases. An hour throttle exceeds the 10 second Run bound. Both cases finish an actual true first heartbeat, receive the original SDK CanceledError, and acknowledge the queued detached second API call before starting public worker Stop with the Run alive. SDK activityWorker.Stop closes stopC before waiting, and the native batching goroutine selects that same channel. The original-context case captures an actual canceled invocation and no mock server heartbeat. The detached-from-first case captures an actual successful true server response. Both keep the handler commandable until that transport result, then select actual Completed and join the once-owned Stop call. Fixture cleanup reuses the same waitable Stop owner. This tests the native timer/workerStopChannel path rather than the separate invoker.Close failure-flush path.

Pinned anchors are SDK internal/internal_worker.go:672-675 and internal/internal_task_handlers.go:2186-2189,2201-2228. No private SDK APIs, synthetic server cancellation, raw surrogate or production admission bridge was added. The correction changes only heartbeat_test.go. Its current SHA-256 is 77a494e981ab8f65a738049e8d1972e9c28a51f14ff072b042ce14ca9c88969d.

The pre-edit focused SDK baseline was green through its cached unchanged package result. The paired focused test passed in 0.056 seconds. The full host module passed, with unchanged packages cached and the SDK package executed in 1.665 seconds. Ten SDK race runs passed in 19.443 seconds. Changed-file golangci lint reported zero issues, formatter diff was empty and errortype vet passed. Every launched command exited, every owned Stop goroutine joined, and the checkout is clean. Exact commands, commit identities and new fn24614-proofreviewfix-*.log hashes are in the evidence JSON. Earlier failed logs and pre-correction handovers are retained in fn24614-proof-logs. The complete fn-246.14 integration and live obligations above remain open.
