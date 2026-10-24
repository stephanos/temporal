---
satisfies: [R1, R2]
---
# fn-237-activity-timeouts-heartbeat-and.2 Bounded nominal optional values through source binding, emission and decode

## Description
Moved from fn-227.10 on 2026-10-21.

Deliver the stored optional-value contract consumed by activity countdowns. Reuse fn-227.2's single authored range declaration and fn-227.7's typed binding. This is optional-value checkpoint O, allocated as fn-237-activity-timeouts-heartbeat-and.2 (formerly fn-227-parameterized-models-declared-worlds.10). Apply the approved fn-227 parent at `bbf0835fbf`; retained engineering contracts supply mechanisms, with obsolete pending-approval prose superseded. The native validity boundary below supersedes older engineering notes promising automatic generic checker validation.

**Size:** M, with a bounded adaptation risk. New nominal descriptors and their narrow operations extend the existing Type/Expr, SSA, CFG and decoder machinery. Roughly fifteen source consumers include mechanical exhaustive matches. Count every generated Type, traversal and decoder case; omit none to claim size. Arbitrary ADTs, Option matching/combinators or another evaluator would exceed this scope.
**Dependencies:** Consume native fn-227.2's range declaration/checked scalar API and fn-227.7's typed World binding (cross-spec prerequisites, satisfied through this spec's dependency on fn-227). If I (fn-227.8) supplies the shared source range-newtype descriptor/parser for action inputs, O consumes that completed I seam rather than duplicating it; the root must assign that single source owner in the canonical briefs. No F (fn-237.1) dependency unless the root explicitly assigns O a fixture consuming F; this draft uses a separate no-fact fixture. Keep F and O serial in scheduling where their files overlap.
**Files:** `model/umpire/lift/src/{lib,tree,shape,expr,emit,build,names}.rs`; `model/umpire/lift/src/body/{mod,typer,expr,effects,claim,progress,vector}.rs` for actual consumers; `model/umpire/backends/tlc/src/{module,decode}.rs`; `model/umpire/testing/src/backend.rs`; new `model/umpire/lift/tests/optional_ticks.rs` and owned source fixtures; existing decoder controls; new `model/umpire/tests/fixtures/optional_ticks/{model.rs,tests.rs,expected/**}` including an explicit test-local checked fixture factory; `model/README.md`. The shared bounded constructor belongs to native fn-227.2, not O.
**Touches:** [model/umpire/ir/**, model/umpire/front/**, model/umpire/author/dsl/**, model/umpire/backends/native/**, model/umpire/backends/tla/**, model/umpire/backends/tlc/**, model/umpire/backends/apalache/**, model/umpire/dev/testing/**, model/umpire/dev/fixtures/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

### Approach

1. Extend exact nominal metadata for fn-227.2's source-local bounded integer newtypes and Option of that newtype. Preserve backing width/name/range. Ticks admits 1..=3 and Remaining 0..=3. Consume fn-227.2's generated `try_new(raw: Backing) -> Option<Self>` and `contains(raw: Backing) -> bool`, both from its one authored field range, without changing Finite's enumeration signature. A test-local fixture factory calls try_new for configured values and names a rejected field; source binding independently validates native values and complete starts. Public/local tuple construction can bypass try_new. Do not claim World::new, Machine, Checker, Space or Run automatically rejects such bypasses.
2. Use the selected scalar representation. None is 0 and Some(n) is n+1; configured Ticks admits exactly {0,2,3,4}, Remaining admits {0,1,2,3,4}. Validate width, range and encoding overflow. Preserve typed metadata instead of erasing either descriptor to Int.
3. Admit canonical unshadowed prelude/core/std Option paths and typed None/Some, copy, equality, presence tests, eager unwrap_or and numeric payload projection. Admit only the canonical generated bounded-newtype contains(raw) associated call as a range predicate derived from that type's descriptor, allowing authored bounds invariants without a duplicated bound table. This does not admit arbitrary inherent methods. Native checked fixture/admission helpers may use try_new; this task does not need to interpret arbitrary helper construction inside source transitions. Preserve eager fallback errors on Some. Keep unsupported aliases, shadowing, inner types, generic arity, nested options, match/if-let/combinators, unwrap/expect and mutable access as located refusals. Preserve Vector get/default/presence behavior.
4. Extend all existing Type serialization, expression traversal/priming/check collection, source identities, full-state binding and main/safety/post/negative/progress CFG routes. World and before values stay unprimed as appropriate. Reuse conditional SSA and the current interpreter, never reached native graphs.
5. Decode exact code sets before state lookup, reconstruct authored wrapper Debug spelling, and replay under the same selected World. Exercise None/one/max selections of one source fixture and deliberately false claims with exact countdown lengths.

### Native validity boundary

Native World::new and suite constructors are infallible ownership/association operations. fn-227.2's checked scalar constructor is an opt-in building block, not a proof that every authored Rust value lies in its finite domain. O's native positive fixture Worlds/starts use its explicit factory; malformed direct native values are separately tested against that factory and the source-binding boundary. An authored bounds invariant on the fixture detects bad countdown starts or updates when explored; it reports Refuted, not a generic admission Refusal. Invalid source forms/selected bindings retain located refusals, and forged backend codes retain typed decoder refusals. These distinct boundaries must be reported honestly, without promising identical error variants for invalid/unsupported inputs. Agreement covers the admitted valid source/native scope. Do not add Machine validation hooks, reflection, fallible World::new or checker/suite admission APIs for this note.

### Investigation targets

Required:

- `.flow/artifacts/fn-227-parameterized-models-declared-worlds/optional-ticks-contract.md` for bounds, ordinal coding, path grammar and all consumer anchors.
- `model/umpire/lift/src/body/typer.rs` and `model/umpire/lift/src/body/expr.rs` for type admission and operations.
- `model/umpire/lift/src/body/vector.rs` for eager fallback and presence regression controls.
- `model/umpire/lift/src/emit.rs` and `model/umpire/lift/src/build.rs` for every CFG/generated type case.
- `model/umpire/backends/tlc/src/decode.rs` and `model/umpire/testing/src/backend.rs` for exact values and native identity.

### Verification

Quick from `model/`: `mise exec -- cargo test -p umpire-lift -p umpire-tlc -p umpire-testing -p umpire-tests`, with selected native/TLC controls repeated under `UMPIRE_BACKENDS=require`; scoped fmt and strict clippy. Baseline unsupported optional source first, then independent positive/refusal/mutated-dump results.

Full from root: `make model-check`, `UMPIRE_BACKENDS=require make model-check`, `make model-time`. Refresh changed linked guest bytes through `make umpire-guest` and run embedded slice/gaps. Compare the original artifact manifest, including metadata; only new fixture/source expectations are added. Retain counts, replay traces and actual timings.

### Paths after fn-244.1 (2026-10-24)

`model/umpire/dsl/macros` is `model/umpire/author/dsl/macros`; `model/umpire/lift` is `model/umpire/backends/tla` (crate `umpire-tla`); `model/umpire/tests/fixtures/<name>` is `model/umpire/dev/fixtures/fixtures/<name>` (crate `umpire-fixtures`, formerly `umpire-tests`); `model/umpire/testing` is `model/umpire/dev/testing`; `model/umpire/backends/tlc` is unchanged. Quick command crates: `-p umpire-dsl-macros -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures` as applicable.

### Actual shared owner after fn-238/fn-244/fn-237.1 (2026-10-24)

fn-227.8 used raw integer input fields and supplied no nominal source descriptor. Reuse the one native declaration parser `front/src/declare/fields.rs::{Finite::Newtype,finite,integer}` and generated `author/dsl/macros/src/finite.rs` checked constructors; this task owns the first IR/source nominal descriptor rather than duplicating bounds grammar. The read-only design prep is `.flow/tmp/fn2372-design-prep.md`; reanchor every proposed interface against current source before freezing it. F is complete; preserve its typed branch-fact and hidden decode-only bridge behavior and all new optional_facts artifacts.

Current exhaustive consumers additionally include author/dsl/src/value.rs, macros finite/lifted, native interpretation, front flow/SSA, main and safety TLA, names/traversal, typed shared schema in backends/tla/src/values.rs, dev/testing/bound/replay and mechanical Apalache compile-through cases. Keep the runtime scalar representation and exact nominal metadata, eager fallback checks and separate invalid-native versus source-binding/decoder boundaries. No general ADT evaluator, Machine validation hook, fallible World::new or Activity policy is added. After freezing a compiling shared descriptor/operation/export interface, use disjoint source lanes and serialized shared-cache builds; the task owner remains sole committer. The conductor owns MILESTONES.md.

Current Files ownership is the Touches list above: it supersedes stale pre-fn-238/pre-fn-244 lift paths. The task owner covers IR descriptor/expression/Print, front declaration/type/SSA/claims, native scalar export and interpreter bridge, both TLA/safety/name/schema consumers, TLC decode controls, mechanical Apalache exhaustive matches, bound/replay rendering and new fixtures. Source lanes use disjoint concrete files only after interface freeze.
## Acceptance
- [ ] fn-227.2's shared checked scalar constructor, O's explicit fixture factory and independent selected-binding/start validation distinguish absent, positive Ticks and exhausted Remaining zero. Forged raw values fail the factory/binding checks with the field named; authored native bounds-invariant controls refute invalid starts/updates. No generic native admission claim is made.
- [ ] The narrow typed source surface works through state/before/World/constants; unsupported forms refuse by location. Eager-default and existing Vector controls remain discriminating.
- [ ] Complete correlated/empty/deduplicated starts, every generated type case and all five CFG families carry identical selected bindings. Name-collision and priming controls distinguish before/state/World.
- [ ] Actual native/TLC evidence distinguishes None/one/max enabledness and lengths, with replayed false claims. Configured code 1, negative/out-of-range/wrong-kind values and malformed fields refuse; countdown code 1 reconstructs Some(Remaining(0)); a different selected World cannot certify agreement.
- [ ] Authored wrapper rendering and all pre-existing artifacts/metadata remain exact; required focused/full gates and timings are retained. No native exploration defines source behavior.


## Done summary
Stored bounded nominal values and their optional forms now preserve authored identity, backing and bounds through native export, source binding, IR/interpreter, TLA emission, exact TLC decoding and typed replay. The independent no-fact optional_ticks fixture covers None/one/max countdowns, malformed values, eager defaults, all five CFG families and selected-World replay; activity policy and World-dependent fairness authoring remain outside this task.

Tier: session jev-unavailable(no_key); explicit AGENTS implementer gpt-6.1-sol medium / reviewer gpt-6.1-sol high. Primary reviewer execution is evidenced by the explicit codex:gpt-6.1-sol:high command and three session receipts; review uses the same model family. No actual implementer model is inferred.

stage: impl-review - ran [2026-10-24T15:11:10.948654Z..2026-10-24T15:17:40.261785Z], SHIP. Correctness/contracts/integration each report 0 introduced and 0 pre_existing findings. Raw reports, raw output archives, metadata and final receipt are retained under .flow/artifacts/fn-237-activity-timeouts-heartbeat-and/optional-values-primary-*.

Baseline: green make model-check before source edits at e2f25dc292cf048d3b61e33b9b2b25e97fedaf97, actual kernel wait status/exit 0. The initial wrong-directory attempt is inconclusive and receives no credit. Early optional source refused before implementation. Compatibility controls for ordinary Some/None constants and fieldless Option enum failed 101 before the final no-bounded-descriptor guards, then all five source controls passed. Seven semantic mutants were detected by executed failing tests.

Final executable source: 6f0fb85f56b20dfa1e94fbeb286f17ce2ae61a41. Subsequent commits contain Flow/evidence only. Actual final gates:

- make model-check: exit 0, 498.206s, 679 passed test summaries.
- UMPIRE_BACKENDS=require make model-check: exit 0, 635.305s, 679 passed test summaries; real TLC agreement and replay required.
- make model-time: exit 0, wall 381.736s; gate 374.886s misses inherited 60s soft target, inner 6.660s meets 10s. No added wrapper kill timeouts.
- make umpire-check-model-slice: exit 0, 65.546s.
- make umpire-check-model-gaps: exit 0, 379.728s, both canonical mandatory parent assertions present. A stopped diagnostic reader may have consumed intermediate pipe stdout; diagnostic observation is inconclusive, with no verification credit or exact-output/count claim for that interval. The gate itself was unchanged and completed normally.
- Final workspace fmt check: exit 0. Final full gates include strict workspace Clippy, docs, tests and wasm/Go consumers; focused affected-crate tests/strict Clippy and required fixture controls also passed as retained in implementation evidence.

Final Verify classified FULL and the strict CLI honored all three final-source receipts:
GATE_SKIPPED:model-check:green-receipt 6f0fb85f
GATE_SKIPPED:model-check-required:green-receipt 6f0fb85f
GATE_SKIPPED:model-time:green-receipt 6f0fb85f

Native/interpreted None/one/max have states 2/3/5, edges 1/2/4, and exactly 0/1/3 Tick edges. Required TLC checks cover these selected Worlds and replay false claims. Native factories and source binding reject malformed selections; authored native invariants separately refute forged invalid starts/updates. Generic World::new/Machine/Checker remain infallible association/exploration boundaries. Apalache adaptation is compile-through, with no live optional-value claim.

Fresh parallel quality audits found correctness 1 and standards 4 issues initially; the main-CFG selected-binding anchor, continuation refusal and module/document extraction fixes resolved them. Final and compatibility follow-ups both report 0 remaining/0 new; reports are preserved verbatim.

All 827 pre-existing active artifacts and 15 original compatibility comparisons remain byte-exact, including metadata; only the new optional_ticks fixture has new expectations. Guest SHA-256 remains b4a4f57f1a64525ae568bd56a73eff85b0b253aeae44d9e4de3e2d88f546cf2a; no refresh or canary repin occurred. Original fn243 evidence and existing F fixture/bridge are preserved. The inherited broader legacy tools Go failure is not an all-repository green claim.

Continuation guard: parent fn237 and fn243/fn228 remain open. Root owns future fairness prerequisite, Model3, binding4, fn2435 and fn2375 sequencing; no cancellation/fairness policy or MILESTONES edits were implemented here. Full evidence is optional-values-implementation-evidence.json, protected authority is optional-values-preserved-manifest.json, and the reproducible guard is optional-values-preserved-audit.py in the same artifact directory.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 546dd249311d401762e64639869f1afce3f5b4a8, 5ae259444b4231d4a509ea4606ba4739122b0558, 6573d847d44afc37fda1e4b7364459d7deb13dc7, 8e089c6888f0097c30cd5bfa3274fcafe550e216, 04da78e42a2278de95f1dc08c9ec46bfff2c6981, 2daa834c671e4f64e8660d50b5b702bf0c11466e, 5fc7ee3413e350ff46b8d7586cc69e6fc12496e2, 968500b6de5d313549295ddd5d94cff5f7589f82, e58f21366e24b261f2dee8dcf6a52772430ad61f, 1fcf8dad86c9dca376533c4933085f83c83ebcff, 58027c27b8d5c33c9c24cc62544bd103a7a8a010, 47cf0beea6c2ddf4e727a5a1891c20d575e29011, 5c59b5edc92b4e7afcf02a82eceb18ca0a83d18d, 785651730bd37d1a73ce24884d4d5cd6b9609934, 6f0fb85f56b20dfa1e94fbeb286f17ce2ae61a41, cb8789a8e48d214841ec9f2f9cb540ddf419ce45, 2e50739c07dd0aef6aa87f2e9a8c7f4e61c5cfb0, 41561bce9100ade48132cc4eebaa2962db7520b0
- Tests: make model-check, UMPIRE_BACKENDS=require make model-check, make model-time, make umpire-check-model-slice, make umpire-check-model-gaps, cd model && mise exec -- cargo test -p umpire-front -p umpire-ir -p umpire-dsl-macros -p umpire-dsl -p umpire-checker -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures -p umpire-apalache, cd model && mise exec -- cargo clippy -p umpire-front -p umpire-ir -p umpire-dsl-macros -p umpire-dsl -p umpire-checker -p umpire-tla -p umpire-tlc -p umpire-testing -p umpire-fixtures -p umpire-apalache --all-targets --all-features -- -D warnings, cd model && mise exec -- cargo test -p umpire-tla --test optional_source, cd model && mise exec -- cargo fmt --all --check, GATE_SKIPPED:model-check:green-receipt 6f0fb85f, GATE_SKIPPED:model-check-required:green-receipt 6f0fb85f, GATE_SKIPPED:model-time:green-receipt 6f0fb85f
- PRs: