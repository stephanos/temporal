# fn-155.1 descriptor and readiness source audit

SOURCE-ONLY assessment of the frozen uncommitted candidate based on `9f18f43e2127d3b448da3ca18f465e469eb1883c`. The original Important R1 and Minor R2 findings below are historical findings addressed by the final frozen production source on reread. No remaining Critical or Important source bug was established on this axis. Exact completion/refusal portable fixture source was reread after the approved test-only window; native evidence remains limited as described below. This assessment supplies no formal implementation-review, SHIP, Done, merge, native qualification, or determinism verdict.

Primary checkout is `/Users/stephan/Workspace/skunkworks/gomad/temporal`. Candidate paths below are relative to its `.worktrees/fn-155-gomad-syscall-level-io-boundary-from` checkout. Inspection began at 2026-10-10 07:04:43 UTC. The closing source-hash check for this report ran at 07:10:51 UTC. The four production files on this audit axis and their four test files had equal opening and closing hashes. The patch and version descriptor still matched the dispatched hashes at that check.

Root authorized a syscall-containment-only thaw during inspection. Its proposed changes concern descriptor-role classification and refusal of stale descriptors before pointer conversion. No changed syscall source was present at this report's closing hash check. Later containment edits or other production fixes need their own candidate binding. Generated choice-wire refresh is outside this audit axis.

Requested reviewer was `gpt-6.1-sol/high`. Dispatch returned Tier `session` with `jev-unavailable(no_key)`; actual execution-model telemetry was unavailable. Reviewer and writer belong to the GPT family. This audit used a fresh context and the prose contract at `/home/agent/.codex/docs/flow-next/prose.md`.

## Original-window findings

### Important R1. Connect completion before poll registration loses its wake

Locations are `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_vfd.go:73`, `internal/gomadio/descriptor_backend.go:247`, and `internal/gomadio/descriptor_backend.go:289`, with the latter two paths relative to the same overlay `src/` directory. Pinned upstream context is candidate `.flow/tmp/patched-goroot/src/net/fd_unix.go:48`, `:73`, `:118`, and `:126`; `internal/poll/fd_unix.go:689` forwards WaitWrite straight to the poll waiter.

Fill the 64-connection listener backlog and start one additional connect. The backend puts that socket in its waiting queue and returns WouldBlock, translated to EINPROGRESS. Upstream `netFD.connect` calls connect before initializing its poll descriptor. If another goroutine accepts a connection in that interval, backend Accept or resumeWaiting establishes the queued connection, sets its completion status, and publishes the write notice. `gomadVirtualReady` discards that notice because the registration has no pollDesc yet. Poll open then initializes the write semaphore to pdNil. Upstream waits for write readiness before examining SO_ERROR or Getpeername, so the completed connection can wait forever or incorrectly expire at its deadline. A listener close in the same interval loses the pending connect's refusal notice in the same way.

The read/write try-before-wait argument in the safety design does not cover this upstream connect sequence. The existing registry test registers before every notification; the backend completion test observes a callback without exercising poll registration. Neither covers the interleaving above. This is a source-grounded reachable interleaving, not a natively reproduced hang.

Preserve a token-bound completion/readiness hint across registration, or otherwise make virtual poll admission provide the retry needed by upstream connect before its first WaitWrite. A bounded initial write-ready retry hint is another mechanism to evaluate against the upstream SO_ERROR retry loop. Any admission mechanism must avoid acquiring model or Go table locks under runtime locks and must retain stale-token rejection. Add regressions that complete and refuse a backlog-waiting connect before its poll registration, plus the corresponding after-registration case. Root and the source owner received this finding immediately.

### Minor R2. Implicit connect ignores explicitly reserved local ports

Location is `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go:185`. Bind reserves a local port in `networkState.boundPorts` at lines 112-117; ListenTCP also consults that reservation. Connect assigns `networkState.nextClientPort` directly without checking that ledger or existing listeners.

From the initial client-port state, bind one virtual socket to loopback port 40000 and keep it open. Connecting a different unbound socket to a listener also assigns local port 40000. The new binding reservation therefore does not prevent implicit client allocation from claiming its endpoint. The resulting connections still use distinct model objects, so this finding concerns socket/address ownership rather than cross-connection byte delivery.

Use a bounded deterministic search for an available implicit local port, and retain the chosen reservation for the connection lifetime if the model's ownership contract requires it. Test an explicit bind in the client-allocation range before an implicit connect. Root and the source owner received this finding separately.

No Critical finding was established on this axis.

## Source strengths and assessed invariants

- The descriptor table allocates monotonically within `[1048576, 1114112)` and stops after 65536 successful allocations. Generations are nonzero and unique within that allocation sequence. Token loads and ownership checks use atomics in the nosplit prefix; they acquire no Go mutex. Table lookup/closing state and references use the existing Go mutex outside that prefix. Backend calls occur after releasing the table lock. Close prevents new acquisitions and keeps an operation reference until backend closure finishes; in-flight entries survive until their final release.
- Socket/listener/connection handles point into the existing standalone network model. The new bound-port ledger and pending-connect queue contain lifecycle metadata, rather than another byte-delivery kernel. Process and simulation backends explicitly refuse the port in this task. Delayed-delivery support remains task .4 scope.
- Read copies synchronously into the supplied destination. Write copies accepted bytes into an owned existing-model chunk with a 64 KiB limit and a 64-chunk receiver queue. The descriptor table and readiness notices retain no caller buffer, slice, pointer, or buffer-capturing callback. Pending chunks and accepted connections clear consumed entries. The inherited one-current-read-chunk state remains bounded in addition to the receiver queue.
- Listener enqueue publishes read readiness. Queue dequeue publishes peer write readiness. Pending-connect acceptance and refusal publish write readiness. Shutdown and close publish both modes for affected descriptor identities; listener close also resets unaccepted descriptor-backed peers. Notices sort by descriptor and mode and validate the current ownership token. The model releases its locks before calling the runtime hook. Both legacy and descriptor operations share the extracted try transitions; the new backend never invokes the blocking legacy Read, Write, Accept, or Dial loop.
- Closed, refused, address-in-use, capacity, not-connected, EOF, broken-pipe, reset, invalid, and unsupported outcomes retain distinct leaf statuses and syscall translations. Descriptor-table exhaustion calls the existing refusal recorder. Successful descriptor-operation transcript coverage is task .3 scope. Legacy blocking Write still records its aggregate operation after completing or failing its loop; its per-chunk direct notices occur earlier, so task .3 must account for that existing aggregate-record shape when specifying cross-path transcript ordering.
- The runtime registry uses the already-initialized `pollcache.lock`. No generated lockrank edit is needed. Registration records the fd-indexed slot, actual leaf ownership token, pollDesc pointer, and atomic fdseq. The open path finishes pollDesc initialization before registry publication. Unregistration clears the slot under the same lock before pollcache.free changes fdseq or exposes the descriptor for cache reuse. The normal internal/poll operation references still prevent wrapper Close from freeing a descriptor used by a blocked operation.
- Ready holds `pollcache.lock` while validating slot token, atomic fdseq, and atomic closing state and while extracting semaphore waiters. `netpollunblock` performs atomic semaphore operations and acquires no pd.lock. This refinement avoids the proposed registry-to-pd lock nesting. After unlocking, Ready dereferences only the extracted Gs and calls netpollgoready in read/write order. It never retains an unlocked pollDesc for a later notification and never calls netpollready, injectglist, a host-poll batch, or a channel-wait helper goroutine.
- The virtual park callback retains the upstream pdWait-to-G CAS and omits the host waiter increment. The netpollunblock exemption omits the matching decrement for every extracted virtual G, including readiness, deadline, close, and cancellation. Genuine host poll descriptors retain both existing sides of the accounting. Upstream error rechecking, deadline sequence validation, and pdNil/pdWait/pdReady protocol remain in place.
- The reserved names in deterministicio/profile.go are backed by concrete launcher bindings at 0-14, with bounded optional simulation, choice, replay, and diagnostic additions ending below 32 in launch_plan_unix.go. These actual Runner-owned values are disjoint from the virtual range. SetEnabled rejects any supplied reservation within the full virtual range. Early runtime control descriptors can also be supplied numerically, so universal enumeration/admission belongs to the recorded-selection handoff in .2. The internal switch requires backend registration and stays off by default. This report makes no universal startup-admission claim.

## Evidence limits

The reviewer ran read-only file discovery, source reads, diffs, and SHA-256 checks. No Go process, compiler, test, formatter, generator, build, lint, worktree operation, bridge, or lifecycle mutation ran. The sole write is this report in the primary checkout.

The available host is unsupported Linux ARM64 for patched-runtime execution. Portable leaf/backend results and cross-platform source compilation supplied by the worker are outside this audit's execution evidence. The runtime tests inspected here cover synthetic semaphore states, host/virtual accounting arithmetic, duplicate notices, token mismatch, closing state, and notification after unregistration. They do not establish real park/commit races, concurrent deadline/close races, physical pollDesc reuse, moving-stack safety, or logical-time advancement while accept/read/write are blocked.

Native stack growth and pointer maps, first-platform zero-host-socket execution, virtual deadlines/quiescence, disabled upstream preservation, native races, and the task's build/lint/generated-validation/static gates remain open unless root binds independent evidence to the final candidate. Task .1's first-platform proof has no automatic transfer to fn-128 or fn-149. Independent early selection is a .2 handoff prerequisite; capability admission is .8 scope. Root owns all completion and scope decisions.

## Consumed source identity

All candidate-axis paths below start with `tools/gomad3/toolchain/runtime/overlay/src/` unless another prefix is shown. Opening and closing hashes are equal for these eight files.

| Source | SHA-256 |
| --- | --- |
| internal/gomadvfd/descriptor.go | `d2f93e013712ce965887727c490423a331fc0cced3bbd6bb936fea5a8ecea378` |
| internal/gomadvfd/descriptor_test.go | `0c17f3c70f2ce95b740ad2e36233b0f854d98a4d8cdaeb2eaa09d2b91f827916` |
| internal/gomadio/descriptor_backend.go | `50bdcf1328f1c9b3a9ce95b1821a8218a6d78dc53eaf2a120bbdc585ba8752f7` |
| internal/gomadio/descriptor_backend_test.go | `d883bbb841d31a7953a28419f48537f4e42e474bfc71ff2bf52ed9c46db93b6d` |
| internal/gomadio/network.go | `96e69a0030428de79eda655782c55fec976bd9dba20cc0ffcbfa1fae4791286a` |
| runtime/gomad_vfd.go | `76de9e659f46021f7db840867e9b8c839910d2dc8878335c70dc1e66521086fb` |
| runtime/gomad_vfd_export_test.go | `38f55c1e084bf39e015e1b07bd665c66253dceec9c8662f3e1f8bc676422278a` |
| runtime/gomad_vfd_test.go | `a0edefc9d4c73c83a7e0a77b0cb83bea54b2a7f7683f93c8995e7dd6840704c0` |

Supporting production inputs were inspected in the same window. The complete patch remained byte-identical; its netpoll section hash includes the opening `diff --git` line through the next `os_linux.go` section's opening `diff --git` line.

| Supporting source | SHA-256 |
| --- | --- |
| tools/gomad3/toolchain/runtime/go1.27.1.patch | `a8c227ad80f81fefa22acc350f51c5f001dea923ad7d370d43a99d7626ae8873` |
| netpoll patch section | `cc17614c57aaa75c45761f95386b1acaa8d44e12007d7c439cd477efe4efc711` |
| tools/gomad3/toolchain/version/version.json | `c94b51756bbbea8d94b3180f10a572764f495eebfe5a43823a0dbafae46974cb` |
| overlay src/syscall/gomad_vfd_unix.go | `3c40961e4bb186a4995dedef85aab2db64212ffe0d60fbbb11827bc579b93d8d` |
| overlay src/runtime/gomad.go | `90c7831f647bc3fd1019329b0b1c9c1904c9e0de7f98240b8d3d39dc25c73836` |
| overlay src/internal/gomadio/libc.go | `e56fe97787d4b1b6224d97a2d9867826e67743b7d859ba332d7d6d0d97874ef4` |
| tools/gomad3/deterministicio/profile.go | `d068900a3b76bc91e5b94f0d67e00e67e225c50c0f3d23cbf92ce992ef0c5f21` |
| tools/gomad3/runner/internal/execution/launch_plan_unix.go | `52e6c04e83e59be86ca580aa700130108280d65fdcb33da8f47e5082e69bf964` |
| .flow/tmp/patched-goroot/src/runtime/netpoll.go | `4107f96dbca01b536616558a50faa252c701ed1abf9af7966a0512e180b5a246` |
| .flow/tmp/patched-goroot/src/internal/poll/fd_poll_runtime.go | `88fda43019c5dcf42545effdebd1747aaea377560dd041a35df8499538ee07c5` |
| .flow/tmp/patched-goroot/src/internal/poll/fd_unix.go | `675f74e8cbfc73170e08b43947c131706cd59e498d722555fe3b793618f5f162` |
| .flow/tmp/patched-goroot/src/net/fd_unix.go | `c9bf26c2e83fb733f5eaeefe39f7dabdc315c5019873e1270e3b1c0fe12a2126` |
| stock go1.27.1 linux-arm64 src/runtime/netpoll.go | `f7c606d3fa0a3b9499fd5f42b81918a98f0b99a6029fca2010cabc710fc9ea65` |

Primary instructions were read before the source assessment.

| Primary input | SHA-256 |
| --- | --- |
| AGENTS.md | `8d634df5cbbbffd7dbada06e32b4d20d879707273f8be07bdf253b387211e6f3` |
| tools/gomad3/README.md | `fb85ed4952fb925ca31768b516fa01285d73fa2738551d9781cd6264cda0f610` |
| MILESTONES.md | `327a5eb97ac78b14e8fa31fe076fc1689bf653183fe8ae6f672728b5be925a51` |
| .flow/tasks/fn-155-gomad-syscall-level-io-boundary-from.1.md | `de37596cb110ec1491fec0b8a25b0821ba658790be78537b9a5068bc5fa73447` |
| task-1/safety-design.md | `9fa0ae2462ca2edda9a6f8c378720f0f20c5a1a32140a7a2a55c80e18fb3bbcd` |
| task-1/root-admission.md | `664827d8be45857166a709d2df011612936f1fbb992b423821766175692bc3f5` |

## Final frozen source reassessment

Root authorized a narrow mailbox/replay, shared implicit-port allocator, and callback-restoration fix after the original audit. The owner froze the final production source at 2026-10-10 07:17:32 UTC. Final-axis reads and hash checks ran through 07:20:42 UTC; the consumed production hashes below matched the post-freeze opening hashes. The original-window findings and hashes above remain historical evidence, not the identity of the final candidate.

R1 is addressed in source by `internal/gomadvfd/descriptor.go:155` and `:190`, together with `runtime/gomad_vfd.go:45`. Notify records one atomic bit per readiness mode before invoking the direct callback. A notification before poll registration therefore survives even if the immediate runtime callback finds no registered pollDesc. Poll open publishes the registration under pollcache.lock, unlocks, then consumes the token-validated scalar mailbox through TakeReady and replays through the existing lifetime-safe Ready path. There is no Go mutex or model callback in the nosplit mailbox prefix and no leaf/model lock acquisition under pollcache.lock.

The mailbox is a fixed 65536-entry array. The descriptor allocator never reuses an fd in the process, so pending bits cannot migrate into a new descriptor generation. A stale token cannot consume a live descriptor's bits. If close intervenes after TakeReady validates the token, Ready still rejects a closing, unregistered, or mismatched registry slot. A notification racing publication and mailbox exchange either contributes to the exchange or invokes Ready against the published registry. Duplicate hints are safe under the unchanged pdReady/pdWait semaphore protocol. The registration caller still owns the initialized pollDesc during this synchronous replay; no unlocked pointer is retained for future notifications.

The same backend paths continue to publish write readiness for both backlog completion and listener-close refusal. Thus the mailbox repair covers the two source interleavings behind R1, while preserving direct seeded callbacks without a host polling batch or helper goroutine. This is source reasoning, not native race reproduction.

R2 is addressed by `internal/gomadio/network.go:489` and the descriptor Connect call to that shared allocator. The bounded monotonic search skips both listeners and boundPorts and reports exhaustion. Legacy DialTCP uses the same helper. The new portable backend regression tests both descriptor and legacy allocation with a listener at the next port and a bound socket at the following port, expecting the next unreserved port.

RegisterReady now returns the previous callback at `internal/gomadvfd/descriptor.go:144`; backend fixtures restore that exact callback instead of installing nil at teardown. This preserves the syscall-installed runtime callback when the tests run inside the patched runtime. No parallel callback-swap fixture was observed.

Final netpoll patch serialization changed during canonicalization. Comparison of the entire applied netpoll.go against pinned upstream still shows the same seven semantic adaptations: pollDesc ownership field, ownership resolution before cache allocation, initialized ownership assignment, virtual open, unregister-before-cache-free, virtual park commit, and virtual waiter-decrement exemption. The cache lock, atomic fdseq/closing checks, lock-free netpollunblock, deferred goready, and matching omission of both host waiter accounting sides retain the original assessment above. No generated lockrank changes were introduced.

### Portable fixture reread and native limits

The new leaf fixture retains r/w notices before registration and rejects stale tokens. The runtime fixture replays a synthetic early write bit after publication. The first final-freeze backend inventory still lacked the exact two connect regressions. Root and the source owner were notified and root authorized a test-only window, without changing production source. The resulting backlog fixture was reread at 07:21:46 UTC and its final hash was checked at 07:22:05 UTC. It asserts no pending readiness before completion, completes a backlog-waiting connect through Accept, verifies Remote succeeds and TakeReady returns write readiness before any poll registration, then queues a separate waiting connect behind the refilled backlog, closes the listener, verifies Remote reports Refused and TakeReady returns both modes. Listener cleanup now tests live ownership before closing, avoiding a second close after the explicit refusal setup. These assertions exercise the actual completion/refusal backend transitions and mailbox in portable source, not a real native parked poll waiter.

The owner reported leaf 4/backend 6 portable tests green and two meaningful regressions red before the fixes and green afterward. This reviewer did not execute or independently reproduce those results. Real parked-G wakeups, deadlines/quiescence, concurrent close and physical pollDesc reuse, moving-stack safety, first-platform execution, disabled-preservation gates, and root's exclusive build/lint/generated/static gates remain outside this source-only evidence. The syscall containment changes were consumed as supporting identity, not separately certified on this axis.

### Final consumed identity

All overlay paths below start with `tools/gomad3/toolchain/runtime/overlay/src/`. Opening and closing final-window production hashes match. The backend test changed only in the separately approved test-only window: its first final-freeze hash was `5ff4974090ac6a47bda098c919b3aa90b9ee02615f085bc4fff65037478b569b`, and its final consumed hash is listed below. Production hashes were checked again unchanged at 07:22:05 UTC.

| Final source | SHA-256 |
| --- | --- |
| internal/gomadvfd/descriptor.go | `3c1610d306e7f03859fe544539429bffc3a857a39c612243f60401f35680005c` |
| internal/gomadvfd/descriptor_test.go | `a4463728c02bad81a1449610255b6ce9eff4d8c51382bc8222ecd0497b4415fa` |
| internal/gomadio/descriptor_backend.go | `30d5b525a6df65b94e21c7495346d2efe68d618f8527a1505dcaf4bb171c3dac` |
| internal/gomadio/descriptor_backend_test.go | `8c2aa64e2fb559fdd5ba5c73fb65340b674806e77210407358b62e03619444a9` |
| internal/gomadio/network.go | `7d7fa5d652bf7a08ce5b087ca2f33fb0fa7df2cbf728d5614d3ddf11fde422ee` |
| runtime/gomad_vfd.go | `b44968c6602f60c8459e9900bd6994f0c466ee87b473bfa51769d32ce517a719` |
| runtime/gomad_vfd_export_test.go | `1978d6ffc9f796d045e4b257ac98a960773c6cff0578b59ed694c16fdb1a5346` |
| runtime/gomad_vfd_test.go | `a0edefc9d4c73c83a7e0a77b0cb83bea54b2a7f7683f93c8995e7dd6840704c0` |
| syscall/gomad_vfd_unix.go | `c18bba832662583aeb2e2d5c0dd2c53e86f0a2e744b20633064d2c0525398d1b` |
| tools/gomad3/toolchain/runtime/go1.27.1.patch | `2be96ca5c7d0b108e5ecc9278fa6ad681c83cc71265f20af8e5cf46dab6813dc` |
| tools/gomad3/toolchain/version/version.json | `c0855d5f7d576276c3a7d1e9678f85827a65cb94effc89874af6b9d5bce9299a` |
| final netpoll patch section | `df028c17015f14099858c8503632cbeea54dc435bc7443946df8594565cd7282` |
| final .flow/tmp/patched-goroot/src/runtime/netpoll.go | `0433d0024eca34b7cb0387fc194b83fab8d763b6bb3bbd013992c7cda9f655f5` |

Root supplied final overlay-inventory aggregate `84a1ec45be6c838e9e0f75d25537dba4b38eca78c0d6a86d61cea048b5658471`; this reviewer bound the consumed individual files above rather than independently deriving that aggregate.
