# fn-155.1 generic buffers and virtual readiness design

Recommendation for the approved runtime port. This report proposes source interfaces; it records no implementation, test pass, native qualification, or determinism bound. The prose contract is `/home/agent/.codex/docs/flow-next/prose.md`.

## Decision

Use a small `nosplit` syscall decoder that converts each operation's pointer arguments into GC-tracked typed pointers before calling ordinary Go helpers. Keep the existing standalone network model behind nonblocking descriptor operations. Publish readiness directly from the goroutine that commits a model change through `netpollunblock` and `netpollgoready`. Exclude virtual waits from both increments and decrements of the host poller waiter count.

This selects the typed-pointer alternative invited by the design assignment. It satisfies R4's moving-stack requirement if its pointer-map and no-retention invariants pass the executable checks below. Task .1 currently names system-stack or fixed-buffer handling in its approach text; its implementation handover must record this selected mechanism explicitly. No second network kernel is justified by the current model's mutexes, allocations, or channel notifications.

## Source observations

- `internal/gomadio/network.go` bounds listener backlog and pending chunks at 64, and each chunk at 64 KiB. `standaloneConn.Read`, `Write`, `standaloneListener.Accept`, and `DialTCP` wait on channels after releasing model locks. Those operations require extracted nonblocking attempts, rather than direct invocation by the syscall edge.
- `connShared.signal` and `standaloneListener.signal` close and recreate channels while holding their Go mutexes. `networkArguments` allocates, and `record` enters `gomadtrace.Record`. These functions belong on an ordinary goroutine with a P. They cannot be called on g0.
- `gomadInjectHostList` marks host netpoll batches as host-timed. `gomadChoiceShuffleSeeded` changes its draw source during those batches. `gomadCheckDeadTime` refuses standalone clock advancement when `netpollAnyWaiters()` is true.
- The spike's `vkGenericSlow` converts incoming uintptr addresses after entering a splittable function. The spike's `vkernelReady` releases its registry lock before using a `pollDesc`, then calls `netpollready` and `injectglist`. Porting these functions unchanged would preserve the two defects under examination.
- Pinned Go 1.27.1 `syscall_linux.go` and `syscall_darwin.go` expressly say `uintptrkeepalive` does not relocate uintptr arguments on stack copying. Compiler `escape/call.go:273-341` creates a typed keepalive temporary only for direct unsafe-pointer-to-uintptr argument conversions. `noder/lex.go:69-75` makes `uintptrescapes` imply keepalive and heap escape.
- Pinned `internal/poll/fd_writev_unix.go:14-31` uses generic `syscall.Syscall(SYS_WRITEV, ...)`. Linux writev is necessary for normal standard-library networking and cannot be deferred as an unused spike operation.

## Alternatives

| Approach | Specific mechanism | Cost and decision |
| --- | --- | --- |
| Typed-pointer handoff | Convert uintptr to `unsafe.Pointer`, `*Iovec`, `*RawSockaddrAny`, or `*_Socklen` in the nosplit prefix; pass only pointer-typed data to splittable helpers. | Recommended. Preserves the existing allocating backend and avoids a new buffer pool. Requires pointer-map, nested-pointer, and no-retention checks. |
| Fixed scratch with bounded system-stack copying | Copy bytes on g0 into preallocated scratch, run the backend on the user G, then copy results back through a tracked pointer or a runtime-managed stack-relative address. | Feasible fallback for one data operation. Read copyback still needs address relocation, and allocation/acquisition cannot occur while an untracked address may move. Pool bounds, contention, writev metadata, and partial progress add interfaces and failure modes. g0 must perform only bounded pointer access/copy, never backend mutexes, allocation, transcript recording, or channel operations. |
| Heap escape at generic entry | Replace or add `//go:uintptrescapes` on each visible forwarding declaration so direct conversions escape before the call. | Insufficient as the general solution. It covers only syntactic conversion arguments, does not propagate through scalar forwarding or assembly automatically, and escaping the outer iovec array does not independently prove every nested buffer escaped. It also changes disabled-path allocation behavior. Keepalive alone supplies liveness only. |

## Generic data interface

Keep exported `syscall` and x/sys linkname signatures unchanged. Add a nosplit operation classifier, proposed as `gomadGeneric`, and operation-specific ordinary helpers such as `gomadGenericRead(fd int, p unsafe.Pointer, count uintptr)`, `gomadGenericWrite`, and `gomadGenericWritev(fd int, vectors *Iovec, count uintptr)`. The prefix passes typed pointers directly into these helpers. A helper's stack-growth prologue can then relocate its incoming pointers before it constructs slices or calls the backend.

The prefix first identifies a modeled descriptor-taking operation. It then checks boundary activation and descriptor ownership. A trap whose first argument is a size, address, or unrelated integer must never be routed merely because that integer lies in the virtual range. Creation operations such as socket require their own operation classification and refusal path. Virtual data operations enter before `entersyscall`; they retain the P and may use ordinary Go synchronization. Host operations keep their upstream paths and guards.

All pointer-bearing arguments must cross the typed seam together, including sockaddr output plus length pointer, getsockopt value plus length pointer, and iovec array plus its pointer-bearing elements. After the first splittable call, neither the generic entry nor any helper may dereference the original uintptr addresses. Scalar counts remain scalar. Validate nil-plus-positive-length, representability in `int`, vector count, and arithmetic overflow before constructing a slice. Define a finite vector bound with a refusal test; 1024 is a proposed port bound and needs the platform contract check before implementation.

`Iovec.Base` must have the pinned platform's GC-pointer shape. Helpers retain the outer typed vector pointer/slice across calls and obtain each `Base` from that tracked representation. Do not flatten vectors into a heap array containing pointers into the caller's stack. For writev, gather only accepted bytes into an owned chunk of at most 64 KiB, stop at backpressure, and return the accepted byte count across vector boundaries. A partial successful transfer returns its count without pretending the whole request completed.

The leaf descriptor module must not retain a supplied buffer, slice, pointer, or closure that captures one. `TryRead` copies into the caller's tracked destination synchronously; `TryWrite` copies accepted bytes into the existing network chunk before returning. Transcript code may hash/copy bytes synchronously but must not retain a caller alias. No caller stack pointer may enter the descriptor table, a network chunk, a heap request, or a future notification. This is a source invariant to audit through the concrete backend, rather than a property guaranteed by a callback's Go type.

Preserve existing `uintptrkeepalive` declarations for valid direct-conversion callers. An independently saved uintptr later passed as a pointer is outside Go's supported unsafe conversion contract; adding a later `runtime.KeepAlive` cannot repair it. A passing heap-buffer test or an escape-only test does not satisfy the requested stack-buffer regression.

## Descriptor and model interface

`syscall` imports a leaf `internal/gomadvfd` module. It cannot import gomadio, which imports os and therefore syscall. Gomadio registers the sole backend into the leaf module at init. Proposed backend operations are `TryAccept`, `TryConnect`, `TryRead`, `TryWrite`, `Close`, `Shutdown`, and address/option queries. They return a finite status such as success, would-block, closed, refused, or capacity; the syscall adapter owns errno translation. Would-block returns `EAGAIN` with zero progress. Runtime `internal/poll` supplies blocking and deadline behavior.

Extract the shared model state transition from current blocking methods. `TryRead` considers `pairedConn.pending`, `connState.incoming`, EOF, reset, and close under the existing shared connection lock. `TryWrite` admits only available queue space and owns the accepted copy. `TryAccept` pops one pending connection or returns would-block. Connect attempts must respect the 64-entry backlog and avoid the current waiting `DialTCP` loop. Existing standard-library-level methods continue to wrap the same transitions with their existing wait/deadline behavior when the syscall switch is off.

Model changes capture bounded notices identifying descriptor, incarnation/generation, and read/write readiness. Publish them synchronously on the committing goroutine after releasing model locks and finishing the corresponding operation's transcript handling. Keep stable descriptor order when one change wakes several endpoints. A stale ready hint is harmless if the subsequent try returns EAGAIN; it must never authorize access to another descriptor incarnation.

Read readiness covers listener enqueue, available incoming/pending bytes, EOF, reset, and read-side closure. Write readiness covers connection completion, receiver queue capacity regained by dequeuing a chunk, peer read closure, reset, and local write closure. Close/revocation wakes both modes so the retry observes its existing error. A future delayed delivery cannot announce read-ready early; its existing logical-time delivery event must publish the notice when bytes become readable. Task .4 owns simulation and process-backend adapters, using this interface and their existing incarnation checks.

Use bounded monotonic descriptor allocation for task .1 unless reuse is required by a concrete workload. Allocation must validate disjointness from every inherited/reserved Gomad descriptor; a proposed numeric range alone is insufficient. Exhaustion returns a recorded deterministic capacity outcome. Slot reuse, if introduced, requires a generation carried in every notification and registration. Serialize table lookup/closing transitions, retain an operation reference while calling the backend, and release the table lock before backend entry. Close marks an entry closing and disallows new operations, then closes the model outside the table lock. Existing in-flight operations observe model closure safely.

## Runtime readiness and lifetime

Add an immutable `gomadVirtual` classification to `pollDesc` for its open lifetime, set by `poll_runtime_pollOpen` before publication. Virtual open/close register and unregister the descriptor in a bounded runtime-owned registry and skip `netpollopen`/`netpollclose`. Reinitialize the classification on poll-cache reuse. Check actual virtual ownership during registration; do not classify an inherited host descriptor solely by numeric range.

Proposed runtime hook `gomadVirtualReady(fd, generation, mode)` executes on the seeded committing G, or the existing logical timer delivery path. It takes the runtime registry lock, finds the matching live registration, then takes `pd.lock` and validates the descriptor token, `fdseq`, and closing state. It calls `netpollunblock(pd, 'r'/'w', true, &delta)` while the registration remains protected. It releases both locks and calls `netpollgoready` for the extracted Gs in fixed mode order. It uses neither host polling, `netpollready` batching, `injectglist`, nor a helper goroutine that waits on `changed`.

Registry unregistration uses the same protection before `pollcache.free`. Notifications must not save an unlocked `pollDesc` pointer for later use. Once a waiter G has been extracted, waking it uses that G rather than dereferencing the descriptor again. Keep the upstream internal/poll operation references that prevent the close path from freeing a descriptor still used by an I/O operation.

The runtime lock order is registry lock then `pd.lock`; no path may acquire the registry while holding `pd.lock`. Existing deadline/unblock paths use only `pd.lock`. Backend model locks never nest with these runtime locks. Descriptor-table locks never nest with model locks. The existing standalone `networkState` then listener lock order stays intact. Runtime registration must not call back into gomadio under a runtime lock.

In `netpollblock`, select `gomadVirtualPollBlockCommit` for a virtual pollDesc. That callback performs the same `pdWait` to G CAS as `netpollblockcommit`, but omits `netpollAdjustWaiters(1)`. In `netpollunblock`, omit the matching `delta -= 1` for a virtual pollDesc whenever an actual G is extracted. This single decrement exemption covers readiness, deadline, close/unblock, and cancellation paths. Host increments/decrements retain their existing behavior. Do not filter virtual waits only at `gomadCheckDeadTime`; that would leave scheduler host-poll decisions and close/deadline deltas inconsistent.

Keep the upstream `pdNil`, `pdWait`, `pdReady`, and G-pointer protocol. An event before park sets pdReady, an event racing the commit cancels or extracts the waiter once, and an event after commit readies the parked G once. Preserve error rechecking after installing pdWait. The ordinary internal/poll sequence attempts the syscall before waiting, so readiness before initial registration is observed by that try. Direct consumers that bypass this try-before-wait sequence would require an explicit readiness snapshot contract and are not admitted implicitly.

Excluding virtual waiters allows `gomadCheckDeadTime` to advance to the next logical deadline while a server waits in accept. Virtual waits with no runnable work or future logical event should still report deadlock. Genuine host poll waiters continue to prevent that standalone advancement. Process transport quiescence retains its separate external-request and arrival accounting.

## Required executable checks

1. A generic read and write fixture keeps a fixed array on the caller G's stack. A private test backend seam invokes bounded noinline recursive stack growth after typed handoff and before touching bytes. Compare numeric addresses only as witnesses and require the stack-buffer address to change while all transferred bytes and adjacent sentinels remain correct. A moved stack with a heap buffer is not sufficient evidence.
2. Repeat for generic writev with a stack-resident typed iovec array and nested stack buffers, including partial progress across vectors. Witness relocation of the array and buffers. Include sockaddr/length and getsockopt/length output pairs because those raw addresses can have the same hazard.
3. Inspect compiler escape output/pointer maps for the fixture and the helper seam, and audit backend stores. Where possible run forced shrink/GC while the helper is parked on ordinary Go synchronization, in addition to growth. Test instrumentation belongs in test overlays/fixtures, without introducing a production runtime growth hook.
4. Exercise all generic variants actually modeled, including Syscall, Syscall6, RawSyscall, RawSyscall6 and Darwin libc forwarding. Test the ABI/linkname forwarding shape used by the pinned x/sys package. Linux dispatch runtime evidence remains with fn-128 when execution is on Darwin; portable compilation alone proves no stack movement on Linux.
5. Drive readiness before park, at the pdWait/commit race, after commit, twice, and concurrently with deadline and close. Assert one wake, no lost event, balanced host waiter count, and refusal of stale generation/registration notifications after descriptor and pollDesc reuse.
6. Block accept, read, and backpressured write separately with a logical deadline as the only next event. Each deadline must fire without host socket activity or wall-time polling. Include simultaneous read and write waiters.
7. Fill the listener backlog and receiver chunk queue to their bounds, then accept/dequeue to verify write-side readiness and FIFO delivery. Include EOF, half-close, reset, raw descriptor close, and refused connect.
8. Show a genuine host poll waiter still follows the existing host count path. Use ordinary seeded/conformance instrumentation; this design adds no native clock-audit authority.
9. Feed a non-descriptor operation a first argument in the candidate virtual range; test every reserved/inherited descriptor collision. Under the internal boundary switch, UDP/nonstream/unmodeled socket operations must be recorded and refused, never delegated to a host socket. The virtual Write route must precede the existing fd 1/2/4 guard. Compile-time and closure-profile admission belong to .8.
10. Keep boundary-disabled behavior and test tiers unchanged. Check package architecture, exact overlay/patch allowlists, generated validation, both source sets and the task's prescribed build/lint/test gates only when the root admits the worker. No gate was run by this design task.

## Open risks and failure handling

The typed-pointer mechanism is a recommendation pending actual pointer-map and moving-stack evidence. If an admitted forwarding path moves the stack before typed conversion, or backend code must retain caller pointers, stop that operation's admission and select a bounded-copy design for it. Invalid memory supplied through unsupported unsafe pointer patterns cannot be made safe by dispatch; valid direct-conversion callers and the declared vector ABI are the supported contract.

Backend registration has a linkage risk for raw-syscall-only programs that never link gomadio. Before startup admission can claim support, establish a real existing init anchor or a checked link contract that guarantees the registered backend. Missing registration must produce a clear unsupported outcome; it cannot fall back to a separate kernel or host network. Boundary selection on unsupported generic dispatch platforms must fail before user initialization. Task .1's switch stays internal and off by default; task .2 owns recorded selection/profile identity and startup admission.

At ten times the current queue demand, try operations return backpressure or capacity without enlarging the model's bounds. A missed capacity notice can stall write indefinitely, so queue-bound tests are required. Reordered or duplicate notices can cause an extra retry, but the upstream semaphore and generation checks must prevent lost wakes, double readiness, count underflow, and cross-incarnation access. A crash/stop later handled by .4 must revoke descriptors through the existing network incarnation outcome.

Darwin decoder expansion must cover each newly modeled libc target explicitly and validate the arm64 trampoline shape. Unsupported shapes and unmodeled socket operations fail closed. The current spike's blanket socket-option acceptance and unbounded zeroing are not a production option contract. Keep exact modeled option values, finite output sizes, transcript coverage, and refusal tests. Assembly raw syscalls, Linux no-error x/sys variants, cgo, and foreign threads remain declared escapes for R8's admission cost assessment.

## Retained input identity

The primary inputs are the approved fn-155 spec/task .1, AGENTS.md, MILESTONES.md, Gomad README, current runtime overlay/patch, `network.go`, `transcript.go`, patch policy, and version descriptor. Root owns their final integrated candidate identity. No Git or Flow state was mutated by this task.

The supplied spike checkout HEAD is `6723dfd8a293f60aae22085cf19788f31ab9443c`. Its consumed spike files are untracked inputs and are bound by SHA-256 below, relative to `tools/gomad3/spikes/syscallboundary/` in `/Users/stephan/Workspace/skunkworks/gomad-syscall-spike.wt`.

| File | SHA-256 |
| --- | --- |
| README.md | `9c675e2cd5fc30af456f328e09416ebd69ea2e6a8e5427d22a66a031c8b6bd6c` |
| goroot.patch | `fd0e117eb020e07cecd0108a8196b482f6dd0c7564c522b47f6b66d09a920cb2` |
| goroot/src/syscall/vkernel_linux.go | `77c9f02e167cd4af4bec13ba7759d277362ae50de393b03aec6007d498b02985` |
| goroot/src/syscall/vkernel_darwin.go | `201f9b233a385656833202666297b83ee910d7484889dc815cbbb818d3916721` |
| goroot/src/syscall/vkernel_unix.go | `edfe1e20fb6a9af0215e1c69daa79cd4f682562f79cbdd037d02dc9abb6139f7` |
| goroot/src/runtime/vkernel.go | `5ef0023ca4e9bb2884230b51bad776288b99e67b798eb96e785fc28f0372a808` |

Pinned compiler/poll semantics were checked against the existing source directory `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src`. Reading its source is not a linux/arm64 patched-runtime build or qualification.

| Source | SHA-256 |
| --- | --- |
| cmd/compile/internal/escape/call.go | `be49c90365fb9d53b4cc7951a6f5244255d9c91e9fae0e048d5bf10005fb7e4f` |
| cmd/compile/internal/noder/lex.go | `6c7ddb7db0c27b7cc4851ed80092fcd5d557305dfa7aaec5ad01c8af83836077` |
| syscall/syscall_linux.go | `e5fa95a3d18df64cf660e47d832c0fcf32fcc4681606aebcad6269ce522ef632` |
| runtime/netpoll.go | `f7c606d3fa0a3b9499fd5f42b81918a98f0b99a6029fca2010cabc710fc9ea65` |
| internal/poll/fd_writev_unix.go | `9e3040f0f7bdf1737bc49687a6231ec4f0b2ae20df86f37dd09c426a8a8d0ee4` |
