---
satisfies: [R1, R2, R3, R4]
---
# fn-236-host-independent-guest-compiler-inputs.1 Derive independent guest identities and verify native cross-OS builds

## Description
Implement the parent spec's self-contained target-identity wrapper and generated-path remapping. Reuse the existing guest builder, preserve native host compilation and Cargo cache filenames, and fail explicitly on absent/corrupt dependency identity evidence. This is the separately scoped implementation prerequisite for fn-232's renewed certification.

**Touches:** model/scripts/guest.sh, the focused rustc-wrapper helper and script tests, model/temporal/hosts/go/umpire_temporal_guest.wasm, model/GUEST_BUILDS.md, model/DESIGN.md and task-owned proof artifacts. Preserve concurrent source/lifter changes.

### Verification

Run focused script/wrapper tests covering cross-host/path equivalence, variant separation, dependency propagation, malformed identity and actual compiler failure; then the full model gate and actual stock/control and Temporal host suites. Record four independent frozen source snapshots/targets, matching pins, manifests, hashes and actual comparisons. Do not create a commit: final fn-232 committed-checkout certification waits for the user-created source commit, as the parent requires.
## Acceptance
- [x] R1: Derived portable target identities preserve package, feature, profile and target dependency distinctions; missing/corrupt identities fail explicitly.
- [x] R2: OUT_DIR remapping removes host directory hashes; the existing target, unwind, ABI/protocol, dependency versions and exit/check contracts are preserved.
- [x] R3: Sixteen focused tests, both native host/control suites and the full required-backend macOS model gate passed; the refreshed asset matches gate and proof outputs.
- [x] R4: Four frozen source snapshots with initially absent independent targets passed all twelve within-OS/cross-OS comparisons with manifests and toolchains retained. fn-232 subsequently passed its separate fresh clean-checkout certification at committed source 3c314fb5b85427471a1525d6f190fe5d1fb9f3b2.
## Done summary
The guest builder now derives portable target crate identities from normalized compiler, package, configuration and dependency inputs, and remaps generated paths independently of Cargo host hashes. Cargo retains its filenames and fingerprints; native macro compilation, unwind flags, ABI, protocol and exit/check contracts remain intact. The launcher preserves Cargo jobserver descriptors and the macOS loader environment.

All twelve actual byte comparisons passed for both guests across four independent frozen Linux/macOS source snapshots. Sixteen focused script tests, both native stock/control and Temporal host suites, and the full macOS UMPIRE_BACKENDS=require make model-check gate passed. Gate outputs match the four-path proof and refreshed embedded Temporal asset. Independent review reported no remaining concrete findings; reviewer and writer share the GPT family.

Evidence is retained at .flow/artifacts/fn-236-host-independent-guest-compiler-inputs/proof.md. This session created no commit. A concurrent commit supplied source pin 3c314fb5b85427471a1525d6f190fe5d1fb9f3b2, and fn-232 subsequently passed its separate fresh four-clean-checkout certification and native host suites at that pin.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3c314fb5b85427471a1525d6f190fe5d1fb9f3b2
- Tests: python3 -B -m unittest discover -s model/scripts/tests -p 'test_*.py' (16 passed), run-snapshots.sh native Linux and macOS (both exit 0; actual stock/control and Temporal hosts passed), compare.py (12 actual cmp calls passed), UMPIRE_BACKENDS=require CARGO_BUILD_JOBS=2 CARGO_TARGET_DIR=task-owned mise exec -- make model-check in fn236-candidate (exit 0), gate-built stock/Temporal cmp against snapshot artifacts and refreshed embedded asset (passed), bash -n guest.sh/run-snapshots.sh and scoped git diff --check (passed), Separate fn-232 certification at committed source 3c314fb5b8: four clean clones, 12 actual comparisons, both native host/control suites (passed)
- PRs: