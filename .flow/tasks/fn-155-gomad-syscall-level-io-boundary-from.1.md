---
satisfies: [R1, R4]
---
# fn-155-gomad-syscall-level-io-boundary-from.1 Port the virtual descriptor layer and syscall edge onto the Gomad toolchain

## Description
Carry the spike's syscall edge, runtime readiness hook and a virtual descriptor layer in the go1.27.1 Gomad toolchain, inert unless a run selects the boundary (selection itself lands in .2; this task uses an internal switch for tests). Back single-process descriptors with Gomad's existing standalone network model rather than the spike's separate kernel. Split from .2 so the toolchain and safety work can be reviewed before identity and CLI changes.

**Size:** M
**Files:** new overlay package for the descriptor layer (e.g. `toolchain/runtime/overlay/src/internal/gomadvfd/`), `toolchain/runtime/overlay/src/runtime/` readiness hook file, `toolchain/runtime/overlay/src/syscall/` edge files (unix, darwin, linux), patch edits to `runtime/netpoll.go`, `syscall/syscall_linux.go`, `syscall/syscall_darwin.go`, `syscall/zsyscall_darwin_arm64.go`, `syscall/zsyscall_linux_amd64.go`, `syscall/syscall_unix.go` (guard order) in `toolchain/runtime/go1.27.1.patch`; `toolchain/version/version.json` allowlists; `internal/gomadio/network.go` (readiness callback + non-blocking path); tests beside them.
**Touches:** [tools/gomad3/toolchain/runtime/**, tools/gomad3/toolchain/version/version.json, tools/gomad3/deterministicio/boundary/**, tools/gomad3/Makefile, tools/gomad3/choice/internal/wire/wire_generated.go, tools/gomad3/target/internal/livecap/protocol_generated.go]

### Approach
- Port from the spike worktree (`gomad-syscall-spike-20261009`, `tools/gomad3/spikes/syscallboundary/goroot/src/**` and `goroot.patch`), rebased from go1.27.0 to go1.27.1. New std files go in the overlay (patch may only edit existing files, `toolchain/patch.go:135-189`); edits to existing files go in the patch. Update `patch_allowlist`/`overlay_allowlist` (must equal the tree, `toolchain/version/descriptor.go:114-162`) and run `make -C tools/gomad3 generate`.
- The descriptor layer cannot import `internal/gomadio` from `syscall` (gomadio reaches `os`, which imports `syscall`). Put the descriptor table in a leaf package that `syscall` imports; let `internal/gomadio` register its connection/listener backend into it at init.
- Adapt the standalone model (`internal/gomadio/network.go`, `connState`, `waitForChange`/`deadlineTimer` at :498-540) to offer a non-blocking try and a readiness notification that calls the runtime hook. If that proves infeasible, port the spike kernel for single-process use and record why in the task summary (R8 cites it).
- The patched `syscall.Write` descriptor allowlist (fd 1/2/4) must admit virtual descriptors. Compile-time guard and closure-policy admission are .8's job, not this task's.
- Readiness: the goroutine that changes a descriptor's state unblocks its waiters directly on the seeded path (pollDesc unblock + ready), never through a `netpoll()` batch, which Gomad treats as host timing (`overlay/src/runtime/gomad.go:724-737, 789-792`). Waiters on virtual descriptors must not count in `netpollAnyWaiters()` for quiescence (`:1580`), or virtual time stalls.
- Typed hooks: darwin/arm64 and linux/amd64 `zsyscall` files. Generic hooks: linux dispatch by syscall number (arch-neutral `syscall_linux.go:51-100`); darwin trampoline decoding (arm64). Use a nosplit operation decoder to convert every pointer-bearing argument into tracked typed pointers before its first splittable call. Pass those pointers together to ordinary Go helpers, and never dereference the original uintptr addresses afterward. Audit nested iovec pointers and prohibit backend retention of caller buffers. Keep a bounded system-stack copy path as an operation-specific fallback only if typed handoff cannot satisfy the pointer-map and moving-stack checks.
- Choose the virtual range disjoint from `deterministicio/profile.go:208` reserved descriptors and inherited Gomad descriptors; make the generic check reject non-descriptor first arguments for calls that do not take a descriptor.
- Deterministic iteration everywhere (the spike's `findListener` map range must become ordered).

### Port admission (2026-10-10)
- The patch digest changes the generated choice wire identity. Admit only the corresponding host output at `tools/gomad3/choice/internal/wire/wire_generated.go`; its overlay counterpart is already under the runtime scope. Admit `tools/gomad3/target/internal/livecap/protocol_generated.go` only if editing `runtime/gomad.go` changes its declared input digest. Unrelated generated drift is not admitted.
- Extend `tools/gomad3/Makefile`'s explicit overlay-test package list for the new leaf descriptor package and for syscall tests if added. Descriptor allowlist edits alone do not require unrelated generated consumers.
- Use the selected pointer-handoff and direct-wakeup design in [.flow/artifacts/fn-155-gomad-syscall-level-io-boundary-from/task-1/safety-design.md](../artifacts/fn-155-gomad-syscall-level-io-boundary-from/task-1/safety-design.md). Virtual poll waits omit both the host waiter increment and its matching decrement; registry lifetime and generation checks protect pollDesc reuse.
- Gomadio backend registration is not guaranteed in raw-syscall-only programs. Test fixtures may explicitly link net; missing backend registration must refuse rather than open a host socket. Recorded selection and universal startup admission remain .2's scope.
- Native execution of the patched runtime is required for this task's acceptance on darwin/arm64 or linux/amd64. Portable stock-Go tests and cross-platform compilation are source evidence only. No existing native deferral automatically transfers this task's first-platform proof.

### Investigation targets
**Required:**
- `tools/gomad3/toolchain/patch.go:135-285` — patch path rules and prohibited runtime areas
- `tools/gomad3/toolchain/version/version.json:92-194` — allowlists
- `tools/gomad3/toolchain/runtime/go1.27.1.patch` — existing `syscall_unix.go` Write guard hunk
- `tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go` — standalone model to back descriptors
- spike `README.md` Limits section
**Optional:**
- `tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go:725-811` — recorded netpoll batches

### Key context
- x/sys/unix `SyscallNoError`/`RawSyscallNoError` on linux issue SYSCALL directly and escape the edge; they are not descriptor operations but must be listed as escapes.
- `prohibitedRuntimeArea` blocks `netpoll_*` files but not `netpoll.go`.
- go/build keeps host arch ToolTags when matching files for another platform (memory: gobuild-cross-platform-matchfile-keeps-2026-09-28); clear them in any allowlist or file-selection check you touch.
## Acceptance
- [ ] `make -C tools/gomad3 toolchain` builds; `validate-toolchain` passes with updated allowlists.
- [ ] With the boundary off: `make -C tools/gomad3 test-toolchain overlay-test test-simulation test-host test-runtime` pass unchanged on the platform in use; the toolchain also builds a linux/amd64 target binary (`GOOS=linux GOARCH=amd64`) from the platform in use.
- [ ] With the internal switch on, a single-process TCP fixture (listen/dial/read/write/close/deadline) runs over virtual descriptors with zero host sockets.
- [ ] Tests: generic data operation with a stack buffer across forced stack growth; non-descriptor first argument in the virtual range not routed; reserved descriptors never virtual; capability guard admits virtual descriptors; UDP under the boundary refused and reported.
- [ ] A server blocked in accept on a virtual descriptor does not block virtual-time advancement (deadline timers fire).
- [ ] Linux syscall-number dispatch tests compile on the platform in use; if that is darwin/arm64, their runtime coverage is recorded as deferred to fn-128.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
