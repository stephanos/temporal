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
- [ ] R1: independently derived target identities preserve package/feature/profile/dependency distinctions; no platform reference map or binary rewriting; missing identities fail.
- [ ] R2: generated paths are host-independent while target, unwind/ABI/protocol/dependency versions, failure codes and check-mode immutability stay intact.
- [ ] R3: focused negative/positive regression tests, full model gate and actual guest-host tests pass for the refreshed artifact with recorded commands/results.
- [ ] R4: four clean exports and fresh targets establish actual within-OS and cross-OS byte identity with manifests/toolchains/hashes retained; this does not claim fn-232's later committed-checkout certification or authorize a commit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
