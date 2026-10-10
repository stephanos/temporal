# fn-155.8 contained socket admission preparation

Proceed after .2 with entry-specific compiler routing, the .1 edge's argument checks, and an explicit source-admission receipt. The present closure evidence cannot prove a reachable socket-only call set. Removing forbidden imports, adding unconditional socket guard exemptions, or making the existing runtime guard return whenever the profile is selected would admit more than R11 permits.

This is a source recommendation, with no implementation, execution, qualification or acceptance claim. Task .8 remains TODO behind .2. Requested research model was gpt-6-astra/high; dispatch reported session fallback `jev-unavailable(no_key)` and exposed no actual-model telemetry. Prose follows `/home/agent/.codex/docs/flow-next/prose.md`.

## Current enforcement and evidence

| Surface | Observed contract and consequence |
| --- | --- |
| [Compiler guard][guard], `Apply`, lines 21-42 | With `-gomadguard`, guards exported, non-init Go bodies in forbidden packages, except generated exemptions or modeled boundaries. Prepends a **zero-argument** `runtime.gomadCapabilityGuard()` and marks the definition non-inline. There is no entry identity or argument in that call. Bodyless declarations and assembly are not instrumented. |
| [Runtime guard][runtime], lines 217-221 | When Gomad is enabled, runs exit hooks and throws `GOMAD_CAPABILITY_DENIED`. Preserve this denial function for every non-admitted entry. |
| [Protocol source][protocol], lines 507-590, 632-700 | Reads `target/internal/livecap/livecap.json` and `livecap.go.tmpl`, strictly validates their exemptions/import lists, and generates both host and compiler protocol files. The task's description of `protocol.go` as the source is incomplete without these inputs. Implementation identity covers guard, linker, encoder, validator and projection source. |
| [Patch][patch], lines 12-13 and 656-669 | Interception runs before guard insertion. A later prepended guard therefore executes before an intercepted body. `syscall.Write` separately guards every fd except 1, 2 and 4 before its existing modeled write hook. .1 must supply virtual routing before that check; .8 must preserve other descriptor denials. |
| [Closure policy][policy], lines 75-133 | Reviews every nonstandard package's forbidden imports, foreign source files and linkname directives. It has no function references, call graph, trap constants or argument evidence. [Collection][collection], lines 150-180, records source digests and textual linknames only. |
| [Preparation][target], lines 703-728, 781-796 | Closure rejects findings before build; it requests neither compiler guards nor linker capability metadata. `finishGoTarget` extracts metadata only for linked/guarded modes. [Review][capability], lines 289-290, similarly returns closure findings before compilation. |
| [Linker][linker], lines 34-85; [encoder][encoder], lines 29-61 | Scans relocations of reachable symbols, records the referenced symbol, and recognizes guards by any relocation to the guard symbol. This is evidence of a retained reference, not proof of invocation arguments or unconditional protection. Conditional guard references must not be mistaken for the old unconditional denial guarantee. |
| [Projection][projection], lines 54-115 | Collapses executable references into package/import capability keys. A modeled socket and `Kill` in one package need separate admission decisions before import findings can be removed. Foreign/linkname findings remain active independently. |

Existing [native fixtures][fixtures] cover direct calls, initialization, function values, interfaces, reflection and inlining for import reachability. They establish no static socket-argument proof. Their platform skips also mean they cannot supply both-platform admission evidence without extension.

## Recommended bounded implementation

Define one generated admission table keyed by platform, package, exact entry/signature, edge operation and argument roles. Bind its canonical digest and implementation digest to .2's selected profile and preparation carrier. The table should describe a verified wrapper-to-edge route, including its argument validation/refusal contract. Start from .1's actual implemented operations, not the task's aspirational name list.

Keep the existing no-argument denial guard. For reviewed typed socket wrappers, the compiler can emit a separate runtime guard taking a generated entry ID. It may return under the selected profile only after the table's wrapper-to-edge contract has been validated. Preserve declaration/return types, no-inline protection and the existing denial when the profile is off. This requires no x/sys source adapter. The edge remains the argument-specific enforcement owner. Merely allowing a function name is insufficient when its body has a host fallback, so validate wrapper signatures and reviewed body fingerprints or equivalent compiler structure. `Socket`, `Accept`/`Accept4`, option helpers, `CloseOnExec` and `SetNonblock` are distinct wrappers even when several terminate in one lower operation. Unsupported family/type/protocol, fcntl command, option/value/length and non-owned descriptors must reach the existing denial or the edge's explicit recorded refusal, never a host fallback. If .1 cannot establish that route for an entry, do not emit its conditional admission.

Generic entry admission belongs at .1's operation decoder. Linux dispatch first identifies the syscall number; Darwin dispatch first identifies a reviewed libc target. Only then may it interpret descriptor positions. `socket` has a domain in argument one; an unrelated trap with argument one inside the virtual interval must not become socket I/O. Descriptor operations need live ownership, reserved/inherited-descriptor exclusion and stale-descriptor handling, not an integer-range test. Preserve existing authorized output/bootstrap paths explicitly.

The compiler must not prepend a splittable returning guard ahead of the generic decoder's typed-pointer handoff. Pinned Go's `Syscall*` and `RawSyscall*` contracts use `nosplit` because `uintptrkeepalive` does not relocate scalar addresses. A new early policy helper must be a checked nosplit scalar-only leaf, or run after all pointer arguments become tracked. The [task .1 safety design][safety] owns that proof; .8 must consume it rather than introduce a second raw-pointer seam.

### Closure proof choice

Recommend a bounded, conservative typed source-use analysis for ordinary callers, with exact reviewed summaries for the syscall/x/sys boundary implementation. Count every executable reference in the compiled source set, including dead functions and values assigned to variables. This deliberately over-approximates reachability and preserves closure's conservative treatment. Types, constants and verified pure error helpers may be classified separately. A source package containing both socket use and `Kill` remains rejected, even when the `Kill` branch is dead.

Resolve selectors through Go type/object information, including import aliases and dot imports. Propagate only finite known function-value sets; join branches by union. A set containing `Kill`, an unresolved indirect target, unsupported linkname, reflection-created executable target, unsafe function construction, or unproved forwarding stays denied. Generic calls require an exact modeled operation proof or a reviewed typed-wrapper summary. Do not infer their trap from the name `Syscall6`. Dynamic fd and socket-option values still require runtime checking; a static receipt claims containment, never that every argument succeeds.

Scanning every function inside x/sys itself would find its complete syscall API and reject the useful socket wrapper too. Treat the approved wrapper/bridge summaries as a narrow platform-specific boundary substrate, pinned to actual compiled sources, module version and sum. Validate initialization as well as wrapper bodies. This retains per-version bridge review, but need not introduce per-application adapters or rewritten imports. Unknown x/sys versions or mutated bridge bytes fail preparation. Existing foreign-source/linkname findings cannot simply disappear with the import finding.

An alternative is profile-selected two-phase preparation with compiler-emitted operation summaries and a conservative linker projection. It can exclude dead helpers, but existing relocation facts alone are insufficient. It also changes closure's build/review behavior and needs explicit evidence and record changes. [Record validation][validation], lines 417-420, currently forbids a linked manifest in closure mode. Root must choose the conservative source route or admit that larger compiler/linker contract before implementation. Neither route can promise the full excluded-adapter gRPC closure passes until its selected source set, initialization and nonnetwork references have been evaluated.

For the source route, retain normalized admission receipts in the capability closure and revalidate them in `reviewRecordedClosure`, provenance, preparation and cache restoration. Receipt fields should bind platform/profile, policy implementation, owner/source-set digest, referenced entry/summary, argument-role constraints and refused unknowns. Existing [cache identity][cache], lines 44-63 and 117-129, already hashes the closure; add the selected policy carrier explicitly so a boundary-off binary cannot be restored for a boundary-on review. The profile digest alone does not replace per-target proof. Coordinate these fields with .2; its preparation/inspection carrier is a dependency.

## ABI and escape inventory

The actual primary module and gRPC v1.83.2 require **x/sys v0.47.0**, per [root go.mod][gomod] and gRPC's module file. The earlier spike survey inspected v0.48.0. This report checked v0.47.0 directly.

| Route | Required treatment |
| --- | --- |
| gRPC `NetDialerWithTCPKeepalive` | Its `RawConn.Control` callback calls `unix.SetsockoptInt(fd, SOL_SOCKET, SO_KEEPALIVE, 1)`. Prove the typed wrapper route and let the edge check virtual ownership. Keep the negative `KeepAlive` value semantics. |
| Darwin x/sys `syscall_syscall6` | Bodyless declaration linknames `syscall.syscall6`; option wrappers pass their own `libc_*_trampoline_addr`. Those addresses differ from stdlib trampoline addresses. Compare validated resolved libc targets/approved trampoline forms, not raw function-PC equality. Review the generated assembly/data and cgo-import-dynamic declarations as evidence even with cgo disabled. |
| Darwin other aliases | `syscall`, `syscall6`, `syscall6X`, `rawSyscall`, `rawSyscall6` need individually proved routes. `syscall9`, `rawSyscall9`, `syscallPtr` and exported assembly-backed `Syscall*` are not admitted merely because another variant works. Bodyless linknames evade Go-body prologues. |
| Linux x/sys assembly | `Syscall`, `Syscall6`, `RawSyscall`, `RawSyscall6` jump to stdlib counterparts through ABI wrappers. `SyscallNoError` and `RawSyscallNoError` execute `SYSCALL` directly. Deny those escapes and unreviewed assembly; whole-file foreign approval cannot stand for socket-only reachability. |
| Mixed or changed bridge | A wrapper linked to `Kill`, a different libc trampoline, a new alias, raw trap, unsupported ABI or unknown source digest remains denied. Existing compatibility-pack approval must not be mistaken for new socket admission. |

Primary ABI sources are the cached [x/sys Darwin declarations][xsysdarwin], [Darwin wrappers][xsyswrappers], [Darwin assembly][xsysasm], [Linux assembly][xsyslinux], [Go Linux entries][golinux] and [Go Darwin entries][godarwin]. The [spike decoder][spike] demonstrates only two option targets and does not establish the complete production classifier.

## Focused fixtures and scope corrections

1. Matrix boundary off/on against closure/guarded and unseeded/seeded. Preserve exact boundary-off denials and findings. Verify typed TCP plus the real gRPC keepalive callback succeeds only on the admitted profile, and `Kill` retains its denial.
2. One package references socket and `Kill` through direct calls, aliases, function-value joins, an interface implementation, init and dead code. Source closure must retain the nonmodeled finding; guarded execution must stop before the nonmodeled call reaches the host.
3. Model a constant generic socket/data operation; refuse an unknown/dynamic trap under the static proof policy, wrong descriptor role, a high first argument to an unrelated trap, stale/reserved fd, unsupported fcntl/option/length, and UDP creation. Verify error/transcript behavior rather than absence of a guard string alone.
4. Exercise actual pinned Darwin linkname/trampoline and Linux ABI forwarding. Negative fixtures cover no-error direct assembly, `syscall9`/pointer aliases, modified source/shape and function values of generic entries. Reuse .1's moving-stack fixtures with guards enabled so the prologue cannot invalidate them.
5. Mutate/delete the admission receipt, policy hash, profile selection, source digest and prepared-cache evidence. Preparation, provenance and replay must reject each mismatch. A socket finding and a nonmodeled finding sharing the same import must never collapse to an admitted import.

The existing `.8` Touches already cover `target/**`, compiler/linker overlay code, protocol generation and `profile.go`, including the livecap JSON/template and both livecap generated outputs. Correct the description to name those inputs. Changes to `runtime/gomad.go` also regenerate **all three choice implementation outputs**. Add `choice/internal/wire/wire_generated.go` and `toolchain/runtime/overlay/src/{internal/gomadchoicewire/wire_generated.go,runtime/gomad_choicewire_generated.go}` to scope. The latter two are outside the current `overlay/src/cmd/**` and single runtime-file allowance. Confirm generator output from its actual input list before allowing further drift.

Depending on root's selected mechanism, add exact syscall/leaf admission-hook paths, patch and version allowlist inputs; record schema/types/validation and their publication consumers if a new external receipt is required. A linked-closure design definitely requires the record validation change above. New guard/edge source files must enter implementation hashing; the current livecap input list hashes `runtime/gomad.go`, not an arbitrary new guard file. Extend table/universe hashing for the admission policy rather than silently reusing the old exemption identity. These are scope decisions for root, not changes made here.

Task-prescribed generated validation, toolchain/interception/host gates, project lint and both source-set checks remain necessary after implementation. No Go command, test, generator, lint or native run was performed during this preparation. Current-candidate native proof remains unavailable here; no acceptance or native ownership was transferred.

## Retained source identity

Dispatched base was `9f18f43e2127d3b448da3ca18f465e469eb1883c`; observed primary HEAD during inspection was `eaa5de72ffd53ab7f5e1b4acabfca6fd40d873f8`. A read-only `git diff --stat` from the base over `tools/gomad3`, the .8 task files and fn-155 spec returned empty. This report therefore describes stable pre-port sources, not .1's candidate. Hashes below bind the principal consumed contracts and implementation inputs. Paths under `G/` mean `tools/gomad3/`; `X/` means `/home/agent/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/`; `T/` means `/home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/`.

| Input | SHA-256 |
| --- | --- |
| fn-155 .8 task markdown | `3600dfeb951a7a88f9ebaaff5647baf467897b39caa99f098003e6ccb09d2540` |
| fn-155 .8 task JSON | `1ebffba5e6e93b4cdcf67a26fc7db2be3e027c720aed65d09921cc07a0009198` |
| fn-155 spec markdown | `a3120b8f052395a370cc69d12983f3bc5c8dcc00091e6d09d395d5756f17d5c1` |
| G/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go | `3fbea44376cf2c223017381b3aa885ed271cbab7aadcfef8e707b1ba0817aa48` |
| G/toolchain/runtime/overlay/src/runtime/gomad.go | `90c7831f647bc3fd1019329b0b1c9c1904c9e0de7f98240b8d3d39dc25c73836` |
| G/internal/gomadtool/generation/protocol/protocol.go | `3d179a9736d39b929bcee66321aacb3e5124c836a8f8612f356fa5a1aa7f021b` |
| G/target/internal/capabilitypolicy/policy.go | `943b5bce7a64e76fa7bd40573d1946f0ea4323f6cea7e7d1b4a70dadb05f00e6` |
| G/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go | `923326a3c3126f783e7cb43e3e85c1a8eb810d198114782f88fb02ba4cd7437f` |
| G/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go | `f0d2115f7b5321c7bef9c41c93d2d2bb48e40d540187e3cd3f394bd2bd95ee79` |
| G/target/internal/livecap/project.go | `914ee78048512c602f18a74bc0943481c8e4200d0979594e0dfbf3345f7d9745` |
| G/target/internal/livecap/livecap.json | `11a0cb4bbb49e43e790bc1e454e75da388f0da65f697d035bbd03f759c5ee42a` |
| G/target/internal/livecap/livecap.go.tmpl | `d6490e804e65ae06f578705cfa238317a4943045a12b4976772ba2e676616adc` |
| G/target/target.go | `bf5a1c8e193650913220fd3a1de6ae2aa0dbf264f1a77a1c77bb52bb9a848bf7` |
| G/target/capability_collection.go | `8770de8f3465c6dec35918cdf7d8a1ba31c30f3de42e6fffa7cb5ebcb9de99ce` |
| G/target/capability.go | `e443c4072e9797e3971a64978ba6a02e8aedebb38796b563335d35b14c5b7a2e` |
| G/target/prepared_cache.go | `13d96303a2eafe579d87f82f38c6c4c8f78416647dc42ea38de21bbc44d02c01` |
| G/record/validation.go | `bacfb8beede09172831ad9b8bfc8360c5d999b1bfcfef11470ee0c9001da519b` |
| G/deterministicio/profile.go | `d068900a3b76bc91e5b94f0d67e00e67e225c50c0f3d23cbf92ce992ef0c5f21` |
| G/toolchain/runtime/go1.27.1.patch | `4b13066eeacc5e9d33a5ada7d3924ff346786ccaf578b9856e1be5f41a94c6f9` |
| X/asm_linux_amd64.s | `14c826e5d2db337e49c32e0b5a66317b58da198874a0eb950c33aac571e9573c` |
| X/syscall_darwin_libSystem.go | `0c327ad9b9845e19b1e097dfb7b569bc9793b670874e24be4a87ac9bd4647557` |
| X/zsyscall_darwin_arm64.go | `630f26d7c5679ac8ac11dfe0e7e1861aec0c801e1ef7ca503030f8e8736469a3` |
| X/zsyscall_darwin_arm64.s | `5daa70eefd10942e6ba8da79d69152b330da1981874d6726d1d09cf8d8a0d30e` |
| X/syscall_unix.go | `d851dcf05549674486f35d58b0357bb5c1ea9378bd234b1646534a35dc4a6da5` |
| T/syscall/syscall_linux.go | `e5fa95a3d18df64cf660e47d832c0fcf32fcc4681606aebcad6269ce522ef632` |
| T/syscall/syscall_darwin.go | `07eea080aae7b97b37943bb0964594a198c73b8ec35697b2b8912c0323cd129a` |
| cached grpc@v1.83.2/internal/tcp_keepalive_unix.go | `e8bfe03234b391d24006a3a274590111f0f8705fc5b25d9a78391bfdde3df32c` |
| task-1/safety-design.md | `9fa0ae2462ca2edda9a6f8c378720f0f20c5a1a32140a7a2a55c80e18fb3bbcd` |
| task-1/root-admission.md | `664827d8be45857166a709d2df011612936f1fbb992b423821766175692bc3f5` |
| docs/research/gomad/2026-10-10-fn155-syscall-port-readiness.md | `53e32e3608e08724891be90a479ecc6c11b8663e353ee9d960ff739f0982b6db` |

[guard]: ../../../../tools/gomad3/toolchain/runtime/overlay/src/cmd/compile/internal/gomadguard/guard.go
[runtime]: ../../../../tools/gomad3/toolchain/runtime/overlay/src/runtime/gomad.go
[protocol]: ../../../../tools/gomad3/internal/gomadtool/generation/protocol/protocol.go
[patch]: ../../../../tools/gomad3/toolchain/runtime/go1.27.1.patch
[policy]: ../../../../tools/gomad3/target/internal/capabilitypolicy/policy.go
[collection]: ../../../../tools/gomad3/target/capability_collection.go
[target]: ../../../../tools/gomad3/target/target.go
[capability]: ../../../../tools/gomad3/target/capability.go
[linker]: ../../../../tools/gomad3/toolchain/runtime/overlay/src/cmd/link/internal/ld/gomadcap.go
[encoder]: ../../../../tools/gomad3/toolchain/runtime/overlay/src/cmd/internal/gomadcap/encode.go
[projection]: ../../../../tools/gomad3/target/internal/livecap/project.go
[fixtures]: ../../../../tools/gomad3/target/internal/livecap/toolchain_test.go
[validation]: ../../../../tools/gomad3/record/validation.go
[cache]: ../../../../tools/gomad3/target/prepared_cache.go
[gomod]: ../../../../go.mod
[safety]: ../task-1/safety-design.md
[xsysdarwin]: /home/agent/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/syscall_darwin_libSystem.go
[xsyswrappers]: /home/agent/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/zsyscall_darwin_arm64.go
[xsysasm]: /home/agent/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/zsyscall_darwin_arm64.s
[xsyslinux]: /home/agent/go/pkg/mod/golang.org/x/sys@v0.47.0/unix/asm_linux_amd64.s
[golinux]: /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/syscall/syscall_linux.go
[godarwin]: /home/agent/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.1.linux-arm64/src/syscall/syscall_darwin.go
[spike]: /Users/stephan/Workspace/skunkworks/gomad-syscall-spike.wt/tools/gomad3/spikes/syscallboundary/goroot/src/syscall/vkernel_darwin.go
