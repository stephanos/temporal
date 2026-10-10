# Verified source progress — acceptance open

Task: fn-155-gomad-syscall-level-io-boundary-from.1
Base: 9f18f43e2127d3b448da3ca18f465e469eb1883c
Verified source commit: a28481d2eeffe7561b5a277770c88117d0ab24b4
Tier: session (jev-unavailable(no_key)); project implementer route: gpt-6.1-sol/high

Ported the standalone model through a bounded descriptor leaf, typed/generic syscall hooks, and direct runtime readiness. The internal test switch is off by default; no adapter was deleted, host fallback was not added, and no second network kernel was introduced. Root owns integration, independent acceptance review and shared Flow lifecycle; this handover does not complete the task.

## Source contracts

- Generic operation/FD-role decoding is nosplit/norace. Enabled virtual-range descriptor operations with no live token return EBADF before converting raw arguments. Host/non-FD calls retain upstream routing. Live ownership and pending readiness use internal/runtime/atomic, not race-instrumented sync/atomic.
- Pointer-bearing arguments cross together as tracked typed pointers before ordinary helpers. writev validates nested bases/lengths and gathers at most 64 KiB into owned storage; the backend retains no caller byte storage. Native moving-stack and pointer-map proof remains open.
- Darwin recognizes shared ARM64 branch targets, including an explicit libc_readv refusal trampoline. Linux message-batch/vector/socket escapes are explicit refusals; no unknown-number/unknown-function generic widening was made. Exact x/sys forwarding execution remains unverified.
- Existing connState/incoming/pending queues supply nonblocking try operations. Descriptor allocation is bounded, monotonic and disjoint from supplied reserved descriptors; binding and implicit port allocation share the existing model ledger.
- Poll registration lifetime is guarded by pollcache.lock through token/fdseq/atomic closing checks and lock-free netpollunblock extraction. No pd.lock nesting or lock-rank expansion was introduced. Extracted Gs are readied after unlocking, reader then writer. Virtual waiters omit both host increments and decrements.
- Atomic pending r/w bits publish before direct callbacks. Registration consumes them after registry publication and unlock, replaying through the same runtimeReady path. Portable fixtures cover pending connect completion and listener-close refusal before registration; callback tests restore the previous syscall-installed callback.

Grounded design refinement: the existing pollcache lock replaces the prepared registry→pd.lock proposal because the upstream lock order does not admit that nesting. A legacy blocking Write publishes committed-chunk peer readiness before its existing aggregate net.write record to prevent mixed-path capacity deadlock; task .3 owns descriptor transcript sequencing. Direct descriptor operations do not yet claim complete transcript coverage.

Task .2 must introduce early runtime-owned selected-mode state independently of backend readiness. Current SetEnabled requires a backend and leaves the testing switch disabled on missing-backend refusal; it is not the selected-before-init admission contract. Keep Token live-only. Task .8 owns compiler/closure admission and exact foreign forwarding coverage; .4 owns simulation/process adaptation, which currently refuses explicitly.

## Observed verification

All commands were serialized with pinned Go 1.27.1 linux/arm64 (SHA256 1675694ef690db0f18fbe7046a886170904bede1d9db6ec96ae27945c1705c64), test_dep and the dispatched cache environment. GOTOOLCHAIN=local, GOENV/GOWORK off, GOSUMDB off, file GOPROXY, empty GOFLAGS and exact worktree SANDBOX_START_DIR were used. Final source retry also set TMPDIR to the dispatched host-side GOTMPDIR.

| Observation | Result |
| --- | --- |
| Correct-worktree pre-edit Quick: make -C tools/gomad3 overlay-test test-toolchain test-simulation | exit 2: builder rejects linux/arm64; validators before it passed |
| Leaf activation regression before implementation | exit 1: register=Unsupported |
| Bind reservation regression before fix | exit 1: legacy listener claimed reserved port |
| Early-readiness mailbox before Notify fix | exit 1: readiness=0, want=233 |
| Implicit port reservation before shared allocator fix | exit 1: descriptor and legacy selected reserved ports |
| Final portable leaf, actual internal/gomadvfd | exit 0; 4 tests |
| Final portable existing-model backend | exit 0; 6 tests plus 2 allocator subcases; pending completion/refusal mailbox assertions included |
| Full syscall, runtime and internal/gomadio test-source compilation for linux/amd64 and darwin/arm64 | all 6 exit 0; latest backend test recompiled on both, exit 0 |
| make -C tools/gomad3 generate / validate | exit 0; final generated identities current |
| Focused host toolchain/descriptor/patch/version tests | final exit 0; archive-dependent inventory and pinned canonical regeneration cases SKIP because source archive is absent |
| TestPackageArchitecture | final exit 0; 1 collected test |
| make lint-code-fast with task base and pinned lint/errortype | final exit 0; 55 host packages, 0 new issues; 50 inherited findings filtered; overlays classified separately |
| git diff --check | exit 0 |

Portable backend harness uses the actual leaf/network/backend/test sources under a disposable stock GOROOT, with transcript/argument helpers stubbed and simulation/process paths inactive. It verifies model behavior, not seeded runtime, transcripts, zero sockets, virtual deadlines or native qualification. Harness hashes: leaf overlay 642e44985155bfc022b204be8846b690aacb8e2af1017da03c5c68afc5818e14; backend overlay 93a94e9f1bca0a72e8f3153436082ebe332cbfc164f254f8de5d80e42d408794; transcript shim f2fa435d9147459cd0ff9e900634f6bfc5c84049beb9121cc9812936367c4cba; inactive-sim shim 63d86b3fcde5a5459e65ac735c7a162702aac3a5d783347ad66a69f6b79c377b.

Final production freeze began 07:17:32 UTC. Patch SHA256: 2be96ca5c7d0b108e5ecc9278fa6ad681c83cc71265f20af8e5cf46dab6813dc. Descriptor SHA256: c0855d5f7d576276c3a7d1e9678f85827a65cb94effc89874af6b9d5bce9299a. Final test-only mailbox assertions SHA256: 8c2aa64e2fb559fdd5ba5c73fb65340b674806e77210407358b62e03619444a9. Six owned patch sections were rendered with canonical one-line context against the pinned installed Go source; other sections and original ordering were preserved. Archive-bound canonical equality is still unverified.

## Unavailable and inconclusive requirements

Native toolchain build, off-mode full Quick suites, native zero-socket TCP/accept/deadline and virtual-time progression, forced stack relocation of caller/nested pointer buffers, deadline/close/reuse races, native race/fork preservation and soak remain UNVERIFIED. Stock linux/arm64 or cross-compilation is not native evidence. These .1 requirements remain open; no native owner was revived and no qualification policy changed.

Initial BASH_ENV cwd-reset reads/baseline observed PRIMARY and were INCONCLUSIVE; correct-worktree anchoring and baseline were repeated before implementation, with no tracked primary edits. The overbroad go test ./toolchain/... invocation was INCONCLUSIVE because it included overlay sources outside their stdlib context. It additionally observed TestPatchCleanupRegenerate/work-after-publication failing its permission obstruction probe; that failure is UNRESOLVED/nonowned, not asserted inherited without same-base evidence.

A final focused retry initially failed cgo with ENOSPC under inherited /home/agent/.cache/codex-build/tmp. The overlay filesystem had exhausted inodes despite free bytes. No cache was deleted: setting TMPDIR to the dispatched host-side GOTMPDIR yielded exit 0. The temporary omitted-upstream-file Linux compile is historical nonacceptance evidence; the actual task-caused dot-import constant collision was fixed, and final complete upstream syscall tests compile without omissions.

Logs/harnesses remain under this worktree's .flow/tmp/fn155-* and .flow/tmp/*-overlay.json; compact observations here are committed. Two disjoint implementer children are reconciled, frozen and command-free. Fresh source-only audits belong to the root and are not formal implementation review or native acceptance.

stage: impl-review - skipped(policy: root owns integrated acceptance; native/unqualified Quick tree)
