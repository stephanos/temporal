---
satisfies: [R3]
---
# fn-187-gomad-wasm-helper-fixture-tests-and.3 Delete inline tests the corpora already cover

## Description
Delete corpus-covered inline tests across network, volume, namespace, World,
clock, broker and the helper only after recorded coverage and complete assertion
equivalence. Add hand-named corpus cases for every genuine gap, regenerating
Go-generated fixtures on the Go side. Apply the same pass to duplicate Go WASI
namespace, volume, broker and network_v2 tests. Record before/after production
coverage per module and each retained invariant with the reason a corpus cannot
state it. Coverage loss on a model path blocks deletion. R3 remains in progress.

### Current continuation — 2026-10-19

R3 remains in `/home/agent/wt/wasm-fn187-functional`, HEAD
`95eed5c83a26eb1e93f2b1ba2f971e33bc1d51bc`. Its accepted R1/R2 index and
staged diff SHA256 `0a343ce3549c1b7ed42146caf23c7800f2cd48f9a60c78ebfd3954036833d299`
are unchanged. Primary `gomad` HEAD is `d200fb9be26b916fab433db1ebe882eb54e06e31`.
Do not integrate, rebase, stage or commit either checkout before conductor gates
and owner authority. No push, PR or CI action is authorized.

The current authoritative chronological checkpoint is
`.flow/artifacts/rust-hygiene-2026-10-18.md`; older source packets and exact
assertion dispositions remain in
`.flow/artifacts/fn187-lightweight-correctness-2026-10-18.md`. Retain failed
attempts and raw profiles, not only successful reports. Receipts are scoped to
their actual source/tool/object identities, never rebound to a changed tree.

Rust1.99.1/LLVM23.1.1 default full builds pass: original baseline352 passed with
two ignored manual benchmarks; candidate201 passed with two ignored. Candidate
corpus28 and invariant134 arms pass. Baseline Go full/corpus arms pass. Complete
candidate63-root Go preflight passes after Go-oracle regeneration of123 added
broker/network fixture provenance digests. Every old SHA256 reconstructs by
reversing only the implementation digest; repeated regeneration is byte-stable.
The separate183 original network cases remain byte-exact. No production source,
observation, generator or identity check was changed for that correction.

The helper-publication and broker-identity coverage phases retain failed Go
candidate attempts and are not final conservation certificates. All current
Rust/Go full, corpus, invariant and exact union/conservation comparisons remain
required, including fresh cursor proof. Use an actual complete original baseline,
not a functional subset, for each supported feature configuration. Default plus
diagnostic-profile, diagnostic-fuel-off, guest-profile and guest-profile-fuel-off
must each bind their exact compiled rosters and frozen objects. Conformance
helper execution is an arm, not a substitute sixth configuration.

A real Rust1.99.1 diagnostic helper-corpus preflight failed under terminal
session10159. It exposed missing explicit feature-profile expectations and a
cluster guard treating profile stderr as empty. A bounded test-side implementation
agent owns helper corpus/common helpers and additive feature expectations;
root owns compiler/runtime/LLVM execution. All originals stay intact pending
assertion and coverage proof. Public feature scenarios are not private waivers.
Retained diagnostic sample injection and genuine host profiling/filesystem/timing
invariants need precise per-root reasons and source bindings. New per-feature
tool-root copies must pin explicit selections and classifications without
altering existing frozen tools or weakening selection mismatch checks.

### Preservation and remaining acceptance

Keep all526 original JSON contracts,215 original helper-case bytes,407 current
helper inputs,183 original network inputs and CLOSED42. The two remaining public
originals stay whole until exact replacement/assertion and coverage proof permits
deletion; do not reclassify them to avoid missing proof. The existing142 retained
test/source bindings and reasons remain authoritative for their scoped roots;
any feature-specific admission requires its own explicit reason and source pin.
The exact-original read-sequence adapters are already implemented; their negative
controls, full assertion mapping and final feature conservation still apply.

Use pinned Rust1.99.1, stock Go1.27.1, exact matching LLVM tools, immutable
source/tool/object SHA/build-ID/feature inventories and separate raw pools.
Keep strict external diagnostic validation and fail on project/unexpected
diagnostics or skips. Never mix historical Rust1.94 profiles into current phases.
Sanitize inherited helper, generator/filter/seed and Rust/Go overrides. Do not run
the old hardcoded source_inventory.py or prepare_functional_selectors.py.
Never copy over a Cargo hardlink; ordinary helper publication/restoration uses
an independent inode and before/after verification.

The owner explicitly authorized WASM resource-sensitive checks, including the
original oversized reader, large context and RSS tests and unchanged capacity
budgets. Serialize all compiler/runtime/LLVM execution under root ownership.
Native execution remains deferred under fn128/fn149. Do not run fn185 profiling
or ignored benchmark roots. Memory admission and resource thresholds stay intact.

Final per-module before/after coverage, corpus/assertion mappings, retained-root
reasons, all-feature original-baseline conservation, lint, independent review and
exact primary integration are still required. R4 module ownership changes follow
completed R3; R5 retains original before/after counts, same-host test-wall-time
medians/ranges and documentation/handover scope. Neither spec is complete and
fn189's milestone remains in progress.

**Touches:** tools/gomad/internal/backends/wasm/wasmhost/src/**,
tools/gomad/internal/backends/wasm/wasmhost/tests/**,
tools/gomad/internal/backends/wasm/wasi/**,
tools/gomad/internal/model/network/**, tools/gomad/internal/model/volume/**,
task-owned coverage scripts/evidence and documentation
## Acceptance
- [ ] Coverage report before and after per module retained; no production line loses coverage
- [ ] Every deleted test names the corpus cases that replace it
- [ ] Remaining unit tests listed as invariants with reasons
- [ ] Go-side duplicate tests handled the same way

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
