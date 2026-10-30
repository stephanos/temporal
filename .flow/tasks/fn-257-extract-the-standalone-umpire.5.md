---
satisfies: [R3, R4, R5]
---
# fn-257-extract-the-standalone-umpire.5 Prove the exact standalone and paired local development loop

## Description
Execute the migration proof after the owner validation hold ends; bind every result to the actual final pair.

**Size:** M
**Files:** this spec's source/fixture/guest provenance, isolated and paired validation logs, local-change proof records.
**Touches:** [.flow/artifacts/fn-257-extract-the-standalone-umpire/**]

### Approach
- Reconcile current input hashes before reusing historical evidence. Run the standalone native gate and prescribed release tests from a fresh clone/target with no Temporal sibling, then the actual paired consumers from fresh revision-pinned siblings.
- Regenerate protocol outputs and require tracked cleanliness. Compare all fixture metadata/payloads and protocol/ABI identities; freshly reproduce the committed Temporal guest.
- In disposable copies, apply recognizable uncommitted Rust and Go edits, observe consumer failures, restore exact bytes and observe green, with no commit/push/registry operation between runs.
- Reuse historical baselines; measure final timings without relabelling existing fn-254 misses or unrelated product failures.

### Investigation targets
**Required:** MIGRATE.md verification evidence; task 1 inventory; tasks 3/4 interface map; model/GUEST_BUILDS.md; model/design/gate-speed.md.

### Quick commands
Standalone make check/time, mise exec -- cargo test --workspace --release --locked, and the exact paired forwarding/Go gates listed by the current migrated docs. Never execute this task under the validation hold.

## Acceptance
- [ ] Fresh standalone and paired proofs cover the actual current source, all migrated checks and required negative controls.
- [ ] Both uncommitted-language red/green proofs and generated/fixture/guest identity checks pass with retained full output.
- [ ] Final timings and any pre-existing failures/platform limitations are explicit; no historical or stale-artifact pass is substituted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
