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
**Touches:** [model/umpire/lift/**, model/umpire/backends/tlc/**, model/umpire/testing/src/backend.rs, model/umpire/tests/fixtures/optional_ticks/**, model/README.md, model/temporal/hosts/go/umpire_temporal_guest.wasm]

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
## Acceptance
- [ ] fn-227.2's shared checked scalar constructor, O's explicit fixture factory and independent selected-binding/start validation distinguish absent, positive Ticks and exhausted Remaining zero. Forged raw values fail the factory/binding checks with the field named; authored native bounds-invariant controls refute invalid starts/updates. No generic native admission claim is made.
- [ ] The narrow typed source surface works through state/before/World/constants; unsupported forms refuse by location. Eager-default and existing Vector controls remain discriminating.
- [ ] Complete correlated/empty/deduplicated starts, every generated type case and all five CFG families carry identical selected bindings. Name-collision and priming controls distinguish before/state/World.
- [ ] Actual native/TLC evidence distinguishes None/one/max enabledness and lengths, with replayed false claims. Configured code 1, negative/out-of-range/wrong-kind values and malformed fields refuse; countdown code 1 reconstructs Some(Remaining(0)); a different selected World cannot certify agreement.
- [ ] Authored wrapper rendering and all pre-existing artifacts/metadata remain exact; required focused/full gates and timings are retained. No native exploration defines source behavior.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
