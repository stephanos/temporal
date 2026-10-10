# fn-155.1 syscall source audit

This audit found three Important source concerns in the initial frozen candidate. Root and the source owner received each concern before this report. All three are resolved in the subsequent frozen source assessed below. No additional Critical, Important or Minor finding was identified within this bounded syscall pointer/dispatch axis.

This is a source-only correctness audit of syscall pointer and dispatch containment. It supplies neither a formal implementation-review verdict nor task acceptance, native qualification, merge authority, or a determinism claim. Root retains lifecycle and acceptance ownership. The prose contract is `/home/agent/.codex/docs/flow-next/prose.md`.

## Findings in the initial source window

Critical findings: none identified.

1. **Important. Stale and never-allocated virtual-range descriptors fall through to the host.** `tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_vfd_unix.go:386` requires `gomadvfd.IsFD(a1)` before handling a recognized descriptor operation. `internal/gomadvfd/descriptor.go:125` returns zero for a stale token, and `:334` clears the token on close. After successful virtual close, a second generic close, read, write, or fcntl therefore returns `handled=false` and reaches the host syscall. Never-allocated values in the reserved virtual range do the same. Generated typed wrappers use the same live-only predicate, including materialized `zsyscall_linux_amd64.go:334` and `zsyscall_darwin_arm64.go:580`. The failure is the host boundary crossing, even if the host currently returns EBADF. Route enabled reserved-range values only after identifying the operation's FD role, return handled EBADF for absent live tokens before converting caller addresses, and retain live Token/IsFD for poll registration and notifications. Unknown non-descriptor operations must preserve their scalar first argument.

2. **Important. Explicit unmodeled socket operations classify as unknown and reach the host.** `syscall/gomad_vfd_linux_amd64.go:61` refuses several vector/message calls but omits RECVMMSG and SENDMMSG. These operations on a live virtual descriptor become unknown at `:66`, then take the host path through `gomad_vfd_unix.go:383`. Darwin's table at `gomad_vfd_darwin_arm64.go:31` similarly omits readv. The exact root dependency `golang.org/x/sys@v0.47.0/unix/zsyscall_darwin_arm64.go:2522` forwards readv through `syscall.syscall`; its assembly at `zsyscall_darwin_arm64.s:741` supplies a valid NOSPLIT JMP libc_readv trampoline. The decoder returns unknown and calls libc. Add explicit refused-operation entries for these known FD-taking operations, including a Darwin readv target with the pinned trampoline shape. Preserve unknown non-FD calls and leave broader foreign-call admission with .8.

3. **Important. Disabled raw syscalls gain race-runtime calls on the post-fork path.** `internal/gomadvfd/descriptor.go:118` implements Enabled with `sync/atomic.LoadUint32`. A disabled modeled raw close/read/write/ioctl takes RawSyscall6, the nosplit decoder, and Enabled. In a race build, the pinned `runtime/race_amd64.s:211` implementation forwards this load through LoadInt32 and racecallatomic into TSAN. The upstream post-fork path expressly forbids locks and runtime instrumentation in `syscall/exec_linux.go:127` and uses these raw operations after fork. The outer norace annotation does not suppress the separately instrumented sync/atomic implementation. This adds race-runtime synchronization and callbacks to a path that must remain safe with inherited locks and the fork stack guard. Use runtime-safe internal atomic reads for the nosplit selection/token primitives and suppress instrumentation in their wrappers; ordinary setup stores may keep their synchronized implementation. This concern is established by the static call chain, without a race or fork execution claim.

Minor findings: none identified.

## Pointer and dispatch strengths

The generic Linux and Darwin operation classifiers, generic dispatcher and pointer handoff are nosplit. Virtual handling precedes entersyscall, so allocating and locking helpers retain the P. The Linux host path calls the internal syscall implementation directly after entersyscall, avoiding reentry into an allocating virtual handler without a P.

Each modeled buffer crosses the first splittable seam as a pointer-typed argument. Sockaddr output and its length pointer cross together in accept/name calls; getsockopt passes both value and length pointers together. Helpers do not dereference the original uintptr values after that seam. The pinned Darwin ARM64 and Linux AMD64 Iovec layouts both declare Base as *byte. Writev retains the typed array/slice, reads nested Bases from it, validates vector count, nil bases and aggregate integer overflow, and gathers at most 64 KiB into owned storage before calling the backend. Partial successful writes return the accepted count.

The concrete leaf/backend chain does not retain caller buffers. Descriptor entries store handles, generation and flags; readiness notices store descriptor identity and mode. Backend read copies synchronously into the supplied destination. Backend write calls tryWriteLocked, which copies accepted bytes into an owned network chunk. Neither destination nor source enters descriptor state, a deferred request, or a future notification.

Address output copies at most the supplied length and the finite encoded IPv4 sockaddr size, then reports the full length. Getsockopt writes a finite four-byte value and rejects short buffers. Socket creation refuses unsupported families, types and protocols, and socketpair refuses while enabled. Fcntl and socket-option helpers enumerate supported scalar commands/options and reject unsupported ones. The virtual Write branch bypasses the existing fd 1/2/4 guard and uses the typed write route; boundary-disabled paths preserve that guard and upstream host forwarding.

## Inspected sources and identity

Primary checkout is `/Users/stephan/Workspace/skunkworks/gomad/temporal`. Candidate checkout is its `.worktrees/fn-155-gomad-syscall-level-io-boundary-from` worktree, based on `9f18f43e2127d3b448da3ca18f465e469eb1883c`. Candidate source is uncommitted. This audit read AGENTS.md, Gomad README, MILESTONES.md, task .1, safety-design.md and root-admission.md. It read every new syscall production/test file, the leaf descriptor API, concrete backend and caller-copy transitions, every changed syscall patch hunk, and each changed generated wrapper's complete materialized function. Complete generic syscall contexts came from the candidate's `.flow/tmp/patched-goroot/src/syscall/` and pinned stock Go source.

The initial production window began at root's 07:04 UTC freeze on 2026-10-10. Initial hashes matched the supplied patch digest. The source owner ended that window after receiving the findings, and root instructed this auditor to stop reading moving production source. The audit resumed only when a subsequent freeze was announced. Generated choice-wire refreshes are outside the syscall pointer/dispatch axis.

| Initial production input, relative to candidate | SHA-256 |
| --- | --- |
| tools/gomad3/toolchain/runtime/go1.27.1.patch | a8c227ad80f81fefa22acc350f51c5f001dea923ad7d370d43a99d7626ae8873 |
| tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_vfd_unix.go | ad0af6e909b8dee13d402370a4b710977bc5f01083fe5d5d704051fa8dd2c28e |
| tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_vfd_linux_amd64.go | 3612af6677419d207a7895ee2f9b76f785c5462b1ec74d51dbc79c4ea8a43641 |
| tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_vfd_darwin_arm64.go | fc8567fc360c740ab66b403f772347befee93e8f69a6728244a65d8ea5d77eb6 |
| tools/gomad3/toolchain/runtime/overlay/src/syscall/gomad_vfd_unsupported_unix.go | e64b3234f341a654be338e8fe79b345aa187cba08b38bdaf0516b705c332aaf5 |
| tools/gomad3/toolchain/runtime/overlay/src/internal/gomadvfd/descriptor.go | d2f93e013712ce965887727c490423a331fc0cced3bbd6bb936fea5a8ecea378 |
| tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/descriptor_backend.go | 50bdcf1328f1c9b3a9ce95b1821a8218a6d78dc53eaf2a120bbdc585ba8752f7 |
| tools/gomad3/toolchain/runtime/overlay/src/internal/gomadio/network.go | 96e69a0030428de79eda655782c55fec976bd9dba20cc0ffcbfa1fae4791286a |
| tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad_vfd.go | 76de9e659f46021f7db840867e9b8c839910d2dc8878335c70dc1e66521086fb |

The exact x/sys forwarding source hashes are `0c327ad9b9845e19b1e097dfb7b569bc9793b670874e24be4a87ac9bd4647557` for syscall_darwin_libSystem.go, `630f26d7c5679ac8ac11dfe0e7e1861aec0c801e1ef7ca503030f8e8736469a3` for zsyscall_darwin_arm64.go, and `5daa70eefd10942e6ba8da79d69152b330da1981874d6726d1d09cf8d8a0d30e` for its assembly. Pinned stock syscall_linux.go hashes to `e5fa95a3d18df64cf660e47d832c0fcf32fcc4681606aebcad6269ce522ef632`; syscall_darwin.go hashes to `07eea080aae7b97b37943bb0964594a198c73b8ec35697b2b8912c0323cd129a`.

Requested reviewer route is gpt-6.1-sol at high. The accepted dispatch tier is session fallback, `jev-unavailable(no_key)`; actual execution-model telemetry is unavailable. Reviewer and writer belong to the GPT family. No bridge, subagent, test, Go command, compiler, lint, generator or native command was run by this auditor. The sole write is this primary artifact.

## Open proof and owner gates

The linux/arm64 host cannot qualify the patched native runtime. Portable backend tests and cross compilation support source progress only. Forced stack relocation, zero host sockets, logical virtual deadlines, disabled-runtime preservation and the first-platform acceptance remain unverified here.

The current read/write fixture checks a moved caller-buffer address and adjacent sentinels. The writev fixture checks bounds, owned-copy limits and partial bytes, but its backend stack growth occurs after vectors have already been copied. It does not witness relocation of both the typed iovec array and nested caller buffers at the handoff/copy seam. Native paired sockaddr/length and getsockopt value/length relocation, X/syscall6X variants, and exact x/sys forwarding execution also remain open. Compiler escape and pointer-map evidence must bind those fixtures to the final source candidate.

Task .1 uses a default-off internal scalar switch and requires a registered backend to enable it. Task .2 must supply recorded selection independently of backend readiness, refuse missing-backend selection, and reject unsupported platforms before target initialization. This audit does not classify absent .2 activation as an introduced .1 defect. Task .8 owns capability admission and must keep generic guard prefixes nosplit before typed pointer handoff. Darwin raw trap assembly, Linux no-error syscall assembly, unsupported foreign trampolines, cgo and foreign threads remain outside the inspected modeled route and need explicit escape/admission treatment. Runtime readiness acceptance and later transcript, simulation and determinism tasks retain their own gates.

## Subsequent frozen source assessment

Root and the owner announced the final production freeze at 07:17:32 UTC on 2026-10-10. The reassessment consumed that freeze; before/after hashes remained identical through 07:19:26 UTC. Patch and version.json matched the announced identities. Root additionally supplied the sorted overlay aggregate `84a1ec45be6c838e9e0f75d25537dba4b38eca78c0d6a86d61cea048b5658471`; this auditor verified individual consumed files, not that aggregate recipe. The initial findings remain retained against their original hashes.

1. **Resolved in source. Reserved-range routing and stale rejection.** Final `syscall/gomad_vfd_unix.go:27` uses Enabled and InRange for typed FD operations. The generic dispatcher at `:386` applies InRange only after operation classification, then returns handled EBADF at `:390` when Token is zero, before any caller-address conversion at `:400`. Unknown operations return unhandled at `:383`, preserving non-FD arguments. Token and IsFD remain live identity checks. This meets the requested syscall ownership distinction without adding a historical allocation bit or changing .2 selection ownership.

2. **Resolved in source. Known unsupported message/vector operations.** Final `syscall/gomad_vfd_linux_amd64.go:62` explicitly refuses RECVMMSG and SENDMMSG; SENDMMSG's declared number 307 matches the pinned Linux AMD64 source. Darwin's `gomad_vfd_darwin_arm64.go:38` classifies the dedicated readv target as refused. Its declaration at `:58`, dynamic import at `:60`, and new `gomad_vfd_darwin_arm64.s:7` NOSPLIT JMP target match the pinned trampoline shape. The decoder recognizes the target through the ARM64 branch instruction rather than guessing arbitrary foreign function semantics. The final known-shape fixture checks its own trampoline and branch target. Exact external x/sys forwarding execution remains a native proof gate.

3. **Resolved in source. Runtime-safe disabled raw prefix.** Final `internal/gomadvfd/descriptor.go:119` implements nosplit/norace Enabled using internal/runtime/atomic.Load; Token at `:127` uses Load64, and TakeReady at `:152` uses Xchg only after validating live generation. Production no longer imports sync/atomic. The pinned AMD64 primitives are direct nosplit loads, ARM64 primitives use NOSPLIT LDAR loads, and the compiler's runtime-package exclusions cover internal/runtime/atomic. The disabled generic prefix short-circuits before token indexing, pointer conversion, pending-bit access, locks or callbacks. Token bounds are guarded by InRange, and TakeReady checks identity before indexing. No reachable bounds panic or instrumented callback remains in that inspected disabled scalar prefix. This removes the identified static race-runtime call chain; native race/post-fork execution remains unverified.

The reassessment reread the final syscall production files and revised fixtures, leaf atomic/readiness callbacks, runtime installation signature, concrete buffer-copy backend chain, changed patch hunks, and complete materialized generic syscall and Write contexts. Existing generated wrapper host bodies remain preserved after their early virtual branch. Unsupported-platform source still declines these routes; selected unsupported-platform rejection belongs to .2 and .8, not an invented host route.

| Final consumed production input, relative to tools/gomad3/toolchain/runtime | SHA-256, identical before/after |
| --- | --- |
| go1.27.1.patch | 2be96ca5c7d0b108e5ecc9278fa6ad681c83cc71265f20af8e5cf46dab6813dc |
| ../version/version.json | c0855d5f7d576276c3a7d1e9678f85827a65cb94effc89874af6b9d5bce9299a |
| overlay/src/syscall/gomad_vfd_unix.go | c18bba832662583aeb2e2d5c0dd2c53e86f0a2e744b20633064d2c0525398d1b |
| overlay/src/syscall/gomad_vfd_linux_amd64.go | 00cc8bb6acc1b8674439b8618437a33b4bba8d0d79c564397bcf69024ab2187b |
| overlay/src/syscall/gomad_vfd_darwin_arm64.go | 1dd87ba159ff7e5c53ed326271a0e857086c1582987f39a935e1f327fc3180ac |
| overlay/src/syscall/gomad_vfd_darwin_arm64.s | 15b68af6cd689298edd03d16e756d51515d6c270156be138f94bf91e4d5b423c |
| overlay/src/syscall/gomad_vfd_unsupported_unix.go | e64b3234f341a654be338e8fe79b345aa187cba08b38bdaf0516b705c332aaf5 |
| overlay/src/internal/gomadvfd/descriptor.go | 3c1610d306e7f03859fe544539429bffc3a857a39c612243f60401f35680005c |
| overlay/src/internal/gomadio/descriptor_backend.go | 30d5b525a6df65b94e21c7495346d2efe68d618f8527a1505dcaf4bb171c3dac |
| overlay/src/internal/gomadio/network.go | 7d7fa5d652bf7a08ce5b087ca2f33fb0fa7df2cbf728d5614d3ddf11fde422ee |
| overlay/src/runtime/gomad_vfd.go | b44968c6602f60c8459e9900bd6994f0c466ee87b473bfa51769d32ce517a719 |

Final syscall fixture hashes are `8f5c36e8d2d5f93028b1108109d59972a79dd8f1cea3c2f7bfc9705530c6130c` for gomad_vfd_darwin_arm64_test.go, `72971f4fe5e1c6e90090761d074abe06a023833be3c23d56f337357ab14066da` for gomad_vfd_darwin_export_test.go, `409fbea5823f0e501a36ee7bccda994915de9405060814750c9e9388e7e5b842` for gomad_vfd_export_test.go, `1ea1188e83e4185a2fe1ad291aa6b10768ffc488641320567db5a74102e4d16e` for gomad_vfd_linux_amd64_test.go, `b6f9bd43414394bb9c0506cc368846631420595c4b94a8fa2f86df940f760934` for gomad_vfd_stack_test.go, and `40da99de71b3197444785923a6390838c4a3f70f6b9f6f952c9a2fd64415914d` for gomad_vfd_unix_test.go. These were also stable across the reassessment.

The source assessment closes the three listed source concerns and identifies no further finding within the bounded axis. It does not close any native proof or owner gate above and is not an acceptance or lifecycle verdict.
