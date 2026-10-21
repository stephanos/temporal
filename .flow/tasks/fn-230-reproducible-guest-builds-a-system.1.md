---
satisfies: [R1]
---
# fn-230-reproducible-guest-builds-a-system.1 Guest profile and guest.sh in Umpire; reproducibility proven on two clean checkouts

## Description
Add profile.guest inheriting release with debug=false, strip="symbols", lto="fat",
codegen-units=1, overflow-checks=true, panic="unwind". Add the system-agnostic
model/scripts/guest.sh <crate> <output> [--check] using wasm32-wasip1, --locked,
the pinned mise toolchain, and remapped workspace/registry/sysroot paths. The script
must not encode system crate names or paths. Exit 0 for successful build/freshness,
1 for build failure, 2 for stale bytes with sizes and first differing offset.
Keep checker release profiling unchanged and preserve caught-panic judgments.

Prove both umpire-temporal-guest and umpire-stock-guest byte-identical in two clean
source checkouts at different absolute paths with independent target directories.
Record exact commands, hashes, sizes, platform and toolchain in DESIGN.md. Verify
the stock guest's actual Go-host reports still match the native goldens under this
new profile, including panic controls. Do not substitute a release-profile parity
run as evidence about profile.guest.

This task proves Linux/aarch64 path independence. The owner moved macOS and
cross-OS proof to fn-232-linux-macos-guest-byte-identity; it does not gate this
task or task .2. Never claim Linux directories prove cross-OS identity. The
caught-panic requirement remains unchanged. Task .2 owns embedding and freshness wiring.

**Touches:** `model/Cargo.toml`, `model/Cargo.lock`, `model/scripts/guest.sh`, `model/scripts/tests/**`, `model/umpire/hosts/go/**`, `model/umpire/wasm/**`, `model/temporal/hosts/go/allocation_test.go`, `Makefile`, `model/DESIGN.md`, `model/GUEST_BUILDS.md`
**Quick:** bash syntax and script controls for fresh/stale/build-failure exit codes; actual caught-panic/overflow reports, reset and unexpected-trap controls through the production Go host; restricted import denial/signature controls; the two-clean-checkout proof for both guests.
**Full gates:** `make model-check`; `make model-time`; stock/control Go parity against artifacts built with profile.guest.
Use a lane-specific Cargo target and isolated proof targets. Proof source copies
are temporary artifacts, not retained worktrees. No new dependencies or Model,
ABI, protocol, export or Realization changes.
Keep DESIGN.md below the documented 1,000-line threshold: retain the actual proof
summary, hashes/sizes, source/platform/toolchain and unresolved requirements there,
with exact detailed reproduction/diagnostic commands in a linked focused
model/GUEST_BUILDS.md. This documentation layout does not waive caught-panic
obligations or the successor's cross-OS proof, and must not move unrelated older design sections.

### Proof-driven unwind implementation (2026-10-21)

The conductor's isolated proof retained the pinned Rust 1.99.1 compiler and
`wasm32-wasip1`, rebuilt `std` and `panic_unwind` with final WebAssembly exception
handling, and ran the actual existing effect-panic fixture through the existing ABI.
The full native report matched on three calls with successful Reset. The same bytes
failed under wazero 1.12's ARM64 compiler but passed under its existing interpreter.
Implement this route rather than weakening the caught-panic requirement:

- Keep the exact compiler pin and existing dependency versions. Rebuild the standard
  library with command-scoped `RUSTC_BOOTSTRAP=1`, `-Zbuild-std=std,panic_unwind`,
  explicit unwind and modern EH code generation. Preserve path remapping and locked
  builds. Document the unstable build-policy tradeoff; do not set bootstrap globally,
  switch to nightly, patch the standard library, or add an alternative WASI SDK.
- Use the existing wazero interpreter and EH feature in the transport-only Go host.
  Preserve bounds, cancellation, serialization, exact import/export admission and
  invalidation after uncaught traps. This changes no Go/SDK panic handling.
- Rebuilt std additionally imports `fd_close` and `fd_seek`. Admit only their exact
  signatures and supply unconditional WASI BADF denial stubs, with no filesystem,
  descriptor lifecycle, seek or output-memory effect. Link no general WASI module.
  Negative tests must call the registered imports with arbitrary descriptors and
  offsets and prove rejection and unchanged output memory.
- Retain permanent guest-profile caught-panic and overflow controls using existing
  checker fixtures/semantics, including a subsequent call and Reset. Keep uncaught
  trap/invalidation controls separate. No Model, ABI or protocol semantics change.
- Wire the relevant guest build/host checks so the actual tested artifacts use the
  guest profile. Do not use a release-profile green check as unwind evidence.
  Task .2 still owns committing/embedding Temporal's artifact and freshness wiring.
- Repeat the stock and Temporal two-clean-checkout proof for this code generation;
  old aborting-artifact hashes do not certify it. Record source/toolchain/platform,
  independent targets, commands, hashes, sizes and actual timing.

## Acceptance
- [ ] profile.guest and guest.sh present with the specified exit codes
- [ ] Two clean checkouts produce identical bytes for both guests; sizes in DESIGN.md
- [ ] Stock guest report byte-identical to goldens; gate passes

## Done summary
Model authors can build generic guest-profile artifacts that preserve native caught-panic and overflow judgments. Two clean Linux/aarch64 clones of integrated source ab9c8edb38318d497e4ce06cd6548fd96b8772c5 independently produced identical stock and Temporal bytes.

Task: fn-230-reproducible-guest-builds-a-system.1
Tier: session (jev-unavailable(no_key)), explicit project routing pin.
stage: impl-review - ran [2026-10-21T06:59:05Z..2026-10-21T07:03:19Z] (model: gpt-6.1-sol at high)

The formal Codex three-lens review covered b27d38ecd059f1b160e6a0285c7d0deed66935b5..3ba18ed020422e423897c972620560bb86afe2f4, before the first builder implementation through the integrated proof. All three draws returned SHIP with zero findings; finalization and dispatch exited 0. Reviewer and writer are the same family. No fix loop or memory capture was required. The receipt is /tmp/impl-review-receipt-8f37faba39e2-fn-230-reproducible-guest-builds-a-system.1.json; the committed implementation-review artifact preserves its scope, sessions and limits.

Acceptance evidence:
- Generic profile/builder retains pinned Rust 1.99.1, locked compilation, path remapping, overflow checks and unwind; script controls cover build/fresh exit 0, failure 1 and stale 2 with sizes/offset. Checker release profiling is unchanged.
- Scoped bootstrap rebuilds std/panic_unwind with final EH. Existing wazero interpreter/EH retains memory/cancellation/serialization/admission and trap invalidation. Exact fd_close/fd_seek imports always deny with BADF and no memory effect.
- Actual Tiny EffectPanic/Overflow native full responses and existing report goldens match guest-profile calls, including three calls and Reset. TestGuestProfileUncaughtPanicsTrap separately verifies trap invalidation; descriptor controls verify arbitrary arguments, unchanged full memory and wrong contracts. TestKettleParity preserves stock native goldens.
- Clean source clones and initially absent independent targets are under .flow/tmp/fn2301-integrated-proof.IJl2tf/. Both were clean before/after; target identities are 58:453312922 and 58:453329518. Both stock files are 192255 bytes, SHA-256 5ec57e2e5371723adf318665c1048b5a4021e9f4203ed56884c5f849e7a2dfc1. Both Temporal files are 3330375 bytes, SHA-256 0d4e1b112097ba62f62f53c6a95ecc7776bfbf907df8594592586d3fe767ece0. Final gate artifacts compare equal to the proof.
- DESIGN is 999 lines. GUEST_BUILDS contains exact reproduction, toolchain/platform and timings. macOS/cross-OS verification remains open in fn-232; task .2 owns artifact embedding and freshness wiring.

Baseline and verification:
- The inherited focused script baseline was green. This late-phase continuation freshly ran integrated default and required-backend make model-check, proof and actual profile parity rather than reusing old lane results.
- Default and UMPIRE_BACKENDS=require full gates exited 0. Both Go modules passed vet; syntax/Python exit-contract controls and diff checks passed.
- Initial model-time exited 0 with full/inner 163.764/3.295 s. Mandatory post-review Verify classified FULL; final model-check and model-time both exited 0. Final full/inner were 159.170/1.694 s. The full gate misses its 30 s soft target; the inner loop meets its 10 s target. This is not a full-gate performance pass.
- Gate receipts refused unrelated dirty MILESTONES.md. No reusable receipt or GATE_SKIPPED line is claimed; all required commands actually ran. Source/native goldens and gates were not weakened.

Durable evidence is .flow/artifacts/fn-230-reproducible-guest-builds-a-system/{task-1-integrated-verification,task-1-implementation-review,integrated-quality-audits,unwind-lane-retirement}.md. Raw runner, proof manifest, review JSON and gate/control logs are in .flow/tmp/fn2301-integrated-artifacts/. The retired implementation lane's ignored recovery archive is named in unwind-lane-retirement.md. No command remains running.

This continuation changed proof documentation and evidence only. No new dependency/version or Model, ABI, protocol, Realization or original Go/SDK panic behavior changed. The new entry route is model/scripts/guest.sh; make model-check now verifies actual guest-profile controls. Plan-sync is disabled, tracker inactive and QA off. Unrelated MILESTONES and fn-233 plans were preserved and excluded from task commits.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d3a845f413167bba4742339eeadcd6577a2a9437, 0da21c3a639887dc86da3e80b82dc438bef66e36, bdcc3f8ca19a666b6673f914b3372811f615846f, 103209f5528a74444bf57090d0ae387669e5f40f, df58e04a076f9d44392a0151e812ad3ccf0824fc, 8509564cfba84f394f630fc610b28ad382ad2b40, 4b04d0a3a7f0f877526be013d9e8b2311f3b2861, edb007881f15870ca3673942abc8c08601a3c39d, aeec53b3f771c93f2b67b9d630556741f2bec8cb, f6ba7ea43e28e1194d8e0b9bbadad545166e36eb, 351191e5d811b469bfe4c56243aed69d663396b2, 0a30d59b1df98dafda200153520c98e5cd22fce8, 665e8d533483d34289b45a535e643e4c04e567ea, 7dce30c8dd13238f4b0ccf4a52279f29277abcfb, dc071f05c1b216119274a96ad5937d479e56dc6d, f98fe8271786d8d896f6169b047f84ad90ee4e8a, ac51ec6198266705260867ea1134d9607b6dba1a, 5d8e8050c328aec6fc4504f8e0ede61191dea7a1, 879c70ca2c0626395d235337629244880a9a9fc8, 8afe887b9f05d3288e71cef98448678c0f5903b0, 02bfd95d48f83ac14ad2a870a6a62ddb28960a4e, 40240a5c554f7f6ca739984cdf25b17824cd92fe, a21bcd022da1b602344890fea8e22fc2dad7e906, 00364f8e0f3fbd42ead1fd600989b655bdb005f0, cdf7f3c9e6766aadeb7aab6cadb89b15b40616c6, e921f3a52686d993d3fd790b35d4c7461a38444a, ab252ce645ae70fe0766af22f5954f4424c4884a, c4310fd81dcaa7a2cc894d32d3c2016211bc22bf, 9d0db31fa5080af7548c8ff7ea0811bde8960400, 16f113317b10899fd47a54b55e746772c301e830, 6a35d57bf9c786cff666a4172a4ce27228ad4252, 73328af3aa26f9c0e9c194754e1a6ce4e97ee598, 5d5637a2ab76adc4b06253ab0688bef84c7bf3bf, 7791ce6b1d4818e5629dbe6cbdb916e961f66258, de413dfa3398da2d1d63b993eca16ae87a89bc3f, 183e22fccce1f1badde0cc8ab2eb3ad78355f03e, 54503f3799615a041a7190036fe5e400f06fbef0, 557925363eeff0d92ba4e91bf6990f63c4bf6856, 9438ccbf6dfb48b52d85688dc406ddc47a65fce6, 1b06b2ff66a5926f731e29fe85661e04caed4a52, 7e5eb61c036f8299a1c7b4d16424a7f054104ddd, 3d6d5b4a7d717213b782839d363fb2d135222bb1, d1fc4437645f6c0cd0dcb7ad8a8d4a7b9f0dd077, 56aa9bd2252e0ae62a535e8606400d49e73a6f43, 6d6aaba056e5decef94cec4a84fbea9738ca35a7, 9682dd5da60223827e007936544e0615c41af550, 55a27ddd024a20ea94c40f0209eeb7d6408794a4, 53d51ac71cdd52365a86e4e029ee741a47b8ad07, bf74e3f15279b3c3312fd451f8ccdc1bed409f33, e32de4627db91a1c8ffa3050ea00fe8cf27f0d6d, b49d6879056f23ccf78c2e284d499ebffbf02b3c, 5c9d19cca65f6920ceb8fd988d107f4eb866618f, 99527a1f3b4e3a4bd27b2412b045019f8a7a0e99, f231ef28337d5c1be0b865483fa782f582c75c1d, 39c8034c8146552e195ef34a2eaf1ae0b9beb7c3, 1691aa14ce849d628a752bafcec34212248ef0c3, 8af0871a1f7a55b5a1f0c16c6a33b04e5f34d977, 9340b4f188e03fcf2f70db2afed84ef4284a9e63, ad42587f3e743d04d285fe6a82bc9f37a81ba6c8, e9d2e554c31e923359ee17ba363001c28d452096, a23540b5446e834df6fc1e478aae093d991fb38f, 7f5479c34690ebd2d06da34f41035134906a7684, 66c37fdeeda85d4e4ab6c0f61600719f3d83a8ae, 4a09f982588177f4425958f3abbcac132f52fd67, 9baf1f772e79f4332fb20906fc5b7af0432224f4, 944d9bd336ecea3ad7c65f9f31c6b45fcd6aace1, f65a1f356851fc3885be946f6c136e31b64c9ceb, 7073840836e87e49ebe29c89e7d2c22bf58a0dd7, 0322c83c21cf495e1236d6d7da83fdf1189930e5, 1a92390e19d8a3a82b1c4ebb0bb16943d991cd6b, 3dc12ffc6e84892f9fbb8f1cc548ea2e3dc9d0cf, 0821646dd2a24f145c406ad75b88dd93de4aeee0, daa8228314732975f2afbfcff7308a4f8d3be6d0, feb4621de85e1889f4dd071ef3875b97a2ad8d22, 50463fdea741c0553da1fd41288d9654d1530bf8, 405889a77ac5364b53fa71acabc516778b531a80, 7fb747199b1786914da0ada312885238696576dc, 5bf8bc014cfe8345936f4fdba57b2c15ce78f2a3, cb739765e88dd5cbd754bf8b46b3bb7653446d88, 284c28d410a64d1f910b5035d233fef52af4b5ad, 91c5dfd56f3dff0fea45c3d778a930888f425713, c723626828c891b69d35235d0195866aba25b160, 851c0e92a92637c5a2fef0fc073d83a206092670, 3ff6aaf25d11f45e67f5dcd40adf1b1dcca4f679, 1c4770b9260eae3cddbdbfd96d1a0b2a4a907a5e, e101cda4322a20bc086069ee19e39fb9cd49fc19, 44c419f10988116059c886994f792f74100fd3b1, 19f71945cba152f566ef532cee0c25c56ccbbfde, 799b0d79eaf494f668f2a94427612bf30f5d3773, f55725be9b1ea31908870e265ed3a2ae42c89209, edd4829162610c9d840dd50e4cf5ef56b746208c, 43cf9d0e50705493be83ed91d9a1ad0c481a850a, a4ac04c739469f7de89a5ff3d671ead01174012f, 424f6eb7a7b8c160b7270c8ce21b639aa3aedaea, 616d378223cd29b38af4771f11fa126542377406, 8f59fcf4e85cb7fc7cf6f9f519338f5bee4c1885, 870ee1de9ac33b9e87428cac03c8a3f2e7b243b5, 06caf99bcc0614463c4316db7414eb0f184ae71d, 01c90fbce385e3fa9c4eeec734bedb8fdde2569e, 2fc8a58d6b4f86fa2e27da19cc9bba41b0a6d4b6, 6e4f1b4b3200e6772ed96b612dbb93a07de5d80d, 703eb87c653e3ea6f8c164d083eadb7ce50aaa7d, 77cf858054fa8072d90f6b2ae106f228ef9a76c5, 0b9b0ae78ef3aea6f622b20c192727926f2a1be8, ab9c8edb38318d497e4ce06cd6548fd96b8772c5, 3ba18ed020422e423897c972620560bb86afe2f4
- Tests: bash -n model/scripts/guest.sh model/scripts/tests/build-guests.sh model/scripts/tests/guest-parity.sh, python3 -m unittest discover -s model/scripts/tests -p 'test_*.py', timeout 600s env CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go make model-check, timeout 600s env UMPIRE_BACKENDS=require CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go make model-check, timeout 600s env CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go make model-time, timeout 600s env -u RUSTFLAGS -u CARGO_ENCODED_RUSTFLAGS bash /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn2301-integrated-artifacts/run-proof.sh, timeout 600s env CARGO_BUILD_JOBS=4 CARGO_TARGET_DIR=/home/agent/.cache/codex-build/fn2301-unwind/cargo GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go UMPIRE_WASM=/home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/umpire_stock_guest.wasm UMPIRE_STATEFUL_WASM=/home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/stateful_control.wasm UMPIRE_CAUGHT_PANIC_WASM=/home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/caught_panic_control.wasm UMPIRE_OVERFLOW_WASM=/home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/overflow_control.wasm UMPIRE_PANIC_NATIVE=/home/agent/.cache/codex-build/fn2301-unwind/cargo/panic-native bash /Users/stephan/Workspace/skunkworks/umpire/temporal/model/scripts/tests/guest-parity.sh, timeout 600s env GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go CGO_ENABLED=0 mise exec -- go -C /Users/stephan/Workspace/skunkworks/umpire/temporal/model/umpire/hosts/go vet -tags test_dep ./..., timeout 600s env GOCACHE=/home/agent/.cache/codex-build/fn2301-unwind/go CGO_ENABLED=0 mise exec -- go -C /Users/stephan/Workspace/skunkworks/umpire/temporal/model/temporal/hosts/go vet -tags test_dep ./..., cmp .flow/tmp/fn2301-integrated-proof.IJl2tf/stock-a.wasm /home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/umpire_stock_guest.wasm, cmp .flow/tmp/fn2301-integrated-proof.IJl2tf/temporal-a.wasm /home/agent/.cache/codex-build/fn2301-unwind/cargo/wasm32-wasip1/guest/umpire_temporal_guest.wasm, git diff --check
- PRs: