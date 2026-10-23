I’ll read the spec and task requirements, then trace them through the diff and focused verification evidence.
The spec’s recorded decisions move timeouts and cooperative cancellation to fn-237. I’m checking the remaining requirements: declared worlds and inputs, source/TLC agreement, activity retries and refinement, live request admission, and documentation.
The activity implementation includes the retained graph comparison, stale-attempt rejection, retry-cap controls, and transactional Start admission. The broader diff also contains recorded fn-233 envelope migration and fn-238 front-end work; I’ll account for those decisions when checking compatibility. Cargo tests are blocked by the read-only build-cache lock, so I’m checking source and retained test evidence.
I found coverage for every active R-ID. The focused cached test binaries passed 58 tests, and all 15 retained activity artifacts passed hash verification; these are not fresh Cargo builds. The retained final gate records required TLC execution. Its timing report openly records the gate-target miss, while the inner loop met 10 seconds.
## Requirements Extracted

1. Declare one typed World with derived value traits and a field-order constructor; preserve unit-World behavior.
2. Pass optional trailing `&World` through handlers, invariants, postconditions, assumptions, scopes and progress, classified by type before positional arity.
3. Support method-level computed starts, preserving correlation, deduplication and empty starts; refuse conflicting declarations.
4. Retain one explicitly selected World through suite operations and retained Runs.
5. Supply `Finite` for bool, fieldless enums and bounded integer newtypes, including checked scalar construction.
6. Enumerate explicit, Finite and field-level World-dependent input products in declaration order; reject invalid declarations.
7. Preserve static Actor and path APIs while providing explicit World-bound domains and refusing unbound wildcards.
8. Admit catalogs under a configurable default limit of 4096, with exact count/overflow refusals before product materialization or stepping.
9. Propagate catalog admission through checker, suite, backend and Wasm consumers; report empty variants.
10. Preserve exact concrete payload identity in action restrictions, fairness, decoding and replay.
11. Export selected World, start and action values without introducing a Model dependency on the lifter.
12. Emit World CONSTANTs/configuration values, correlated Init states and existential finite input relations.
13. Admit the activity’s Phased, rejection and closure-law semantics through source emission.
14. Provide two-World shared fixtures, all configuration families, negative controls and actual TLC agreement.
15. Keep admitted IR and embedded/emitted representations linear across conditional returns and repeated updates.
16. Extend the existing activity with bounded retries, attempt-aware Poll and retryable failures.
17. Reject stale Polls and preserve terminal laws across phases and attempts.
18. Preserve original activity evidence and prove exact projected lifecycle refinement, including mutation controls.
19. Update three-attempt activity goldens and state the cover test’s `Cases(n)`.
20. Transport optional actual request evidence at Observation tag 4 while preserving established payload fields and encodings.
21. Capture owned, after-fill request/raw-result pairs from the same invocation, including concurrent calls and retries.
22. Admit live Worlds transactionally from valid Start evidence; enforce identity, replay, freeze, Reset and publication rollback contracts.
23. Prove actual SDK retryable failure followed by completion through real Describe attempts; preserve existing live controls.
24. Document delivered APIs, activity state, native/TLC counts and timing evidence; compile the README example and preserve declared boundaries.

R5’s timeout, heartbeat and cooperative-cancellation requirements are explicitly deferred to fn-237. Recorded decisions also supersede the earlier separate source-admission architecture, envelope identities and 30-second soft gate target.

## Coverage Verification

1. **COVERED** — World derivation and constructor: `model/umpire/dsl/macros/src/state.rs:14`, `:135`; unit/declared association: `model/umpire/dsl/macros/src/machine.rs:585`.
2. **COVERED** — Typed parameter admission: `model/umpire/front/src/declare/mod.rs:231`; adapters: `model/umpire/dsl/macros/src/machine.rs:188`, `:221`, `:403`, `:521`, `:539`. Positive controls include `model/umpire/dsl/tests/world_dispatch.rs:98`.
3. **COVERED** — Conflict and method admission: `model/umpire/front/src/declare/mod.rs:427`, `:698`; invocation: `model/umpire/dsl/macros/src/machine.rs:552`; correlated/duplicate/empty controls: `model/umpire/lift/tests/world.rs:188`. New trybuild cases cover duplicate World, mixed starts and misplaced/mutable/duplicate parameters.
4. **COVERED** — Owned suite binding: `model/umpire/suite/src/space.rs:59`, `:134`; retained Runs: `model/umpire/suite/src/run.rs:65`; explicit-World suite controls: `model/umpire/suite/tests/explicit_world.rs:1`.
5. **COVERED** — Trait/bool: `model/umpire/dsl/src/finite.rs:12`; enum/newtype derivation and checked constructors: `model/umpire/dsl/macros/src/finite.rs:5`; domain controls: `model/umpire/dsl/tests/finite_domains.rs:1`.
6. **COVERED** — Input derivation and ordered products: `model/umpire/dsl/macros/src/action.rs:23`, `:441`; misuse diagnostics are pinned by the new `action_domain_*` and `action_finite_missing` UI cases.
7. **COVERED** — Value capability: `model/umpire/dsl/src/action.rs:18`; explicit domains: `model/umpire/dsl/macros/src/action.rs:350`; unbound refusal: `model/umpire/dsl/src/path_build.rs:167`.
8. **COVERED** — Default limit and checked catalog: `model/umpire/dsl/src/catalog.rs:6`, `:42`; generated admission: `model/umpire/dsl/macros/src/machine.rs:309`. Boundary/product/overflow/zero-factor controls: `model/umpire/dsl/tests/finite_domains.rs:227`.
9. **COVERED** — Checker admission and empty-variant warnings: `model/umpire/checker/src/explore.rs:124`, `:233`; progress revalidation: `model/umpire/checker/src/progress.rs:75`; suite forwarding: `model/umpire/suite/src/search.rs:41`; Wasm admission: `model/umpire/wasm/src/runtime.rs:351`.
10. **COVERED** — Exact scoped/fair relations: `model/umpire/lift/src/emit.rs:535`, `:560`; typed replay: `model/umpire/testing/src/backend.rs:174`; discriminating fairness mutations: `model/umpire/tests/fixtures/action_inputs/tests.rs:174`.
11. **COVERED** — Derived exports: `model/umpire/dsl/macros/src/state.rs:114`, `model/umpire/dsl/macros/src/action.rs:124`; selected binding: `model/umpire/testing/src/bound.rs:26`. The fn-238 decision explicitly authorizes the embedded shared-tree route.
12. **COVERED** — CONSTANT/configuration emission: `model/umpire/lift/src/emit.rs:43`; whole-state Init: `:81`; existential input relation: `:487`; bound value/type refusals: `model/umpire/lift/src/bind.rs:20`.
13. **COVERED** — Shared phase/rejection/law admission and refusal controls: `model/umpire/lift/tests/phased.rs:17`, `:78`; activity law mutants: `model/temporal/feature/activity/tests.rs:220`.
14. **COVERED** — Parameterized fixture and selected-World replay: `model/umpire/tests/fixtures/parameterized_world/tests.rs:15`, `:52`; input fixture: `model/umpire/tests/fixtures/action_inputs/tests.rs:18`, `:127`; five-family execution: `model/umpire/lift/tests/world.rs:370`.
15. **COVERED** — Linear-growth, collision and compilation controls: `model/umpire/lift/tests/phased.rs:122`; compiled large-handler fixture: `model/umpire/lift/tests/fixtures/linear_returns.rs:1`. Task16 artifacts retain measurements and justified TLA+ changes.
16. **COVERED** — World, actions and retry transition: `model/temporal/feature/activity/model.rs:14`, `:53`, `:110`; cap/nonretryable controls: `model/temporal/feature/activity/tests.rs:271`.
17. **COVERED** — Stale-attempt rejection: `model/temporal/feature/activity/model.rs:112`; phase/attempt law matrix and mutation controls: `model/temporal/feature/activity/tests.rs:151`, `:180`, `:200`.
18. **COVERED** — Stated projection: `model/temporal/feature/activity/compatibility/README.md:10`; exact retained graph comparison and changed-edge negative: `model/temporal/feature/activity/compatibility/checkpoint.rs:81`, `:93`.
19. **COVERED** — Explicit `Cases(3 * 7)`: `model/temporal/feature/activity/tests.rs:27`; updated report/rows/states/coverage and new activity TLC artifacts are present under `expected/`.
20. **COVERED** — Request tag 4 and unchanged remaining field numbers: `model/umpire/proto/umpire/realization/v1/realization.proto:9`; absent-request wire pins: `model/temporal/hosts/go/parity_test.go:109`. The generic package migration is explicitly settled in fn-233’s envelope contract.
21. **COVERED** — Owned lexical capture: `tests/testpilot_model_annotations_test.go:40`, `:93`; controls cover after-fill equality, caller mutation, raw/local error separation, reversed concurrent responses and retry request IDs.
22. **COVERED** — Admission/receipt/freeze: `model/temporal/realization/src/live.rs:171`; identity validation: `:239`, `:314`; commit-only effects: `:254`; rollback matrix: `model/temporal/realization/tests/admission/publication.rs:28`; actual Wasm allocation controls: `model/temporal/hosts/go/allocation_test.go:185`.
23. **COVERED** — Attempt-aware realization: `model/temporal/feature/activity/temporal.rs:103`, `:192`; actual retry/Describe sequencing: `tests/testpilot_model_slice_test.go:392`. Task5 retains successful seven-scenario slice and gap/privacy execution.
24. **COVERED** — Delivered readiness/API table: `model/DESIGN.md:554`; selected-World counts: `:593`; activity/live evidence: `:612`, `:632`; compiling synchronized README example: `model/umpire/dsl/tests/readme.rs:1`. DESIGN has 969 lines. Final timing evidence records the soft-target miss explicitly.

Verification in this session: 58 focused tests passed using existing cached binaries; all 15 original activity artifacts passed hash verification; the README example matches its compiled fixture. Fresh Cargo execution was blocked by the read-only build-cache lock. The retained final required-backend gate passed, its 13 archived logs verified against recorded hashes, and no non-Flow files changed between that verified source checkpoint and the reviewed head.

## Reverse Coverage (untraced changes)

The range contains collateral work from several separately specified efforts. The following classifications apply to every matching changed file in the supplied scope; patterns group files sharing the same disposition.

| Changed files | Classification | Reason |
| --- | --- | --- |
| `.flow/specs/fn-{119,123,125,128,129,138,140,141,144,146,147,148,149,150,151,152,153,155,156,157,158,159,160,161,162,163,164,165,166,167,168,169,170,171,172,173,174,189,190,191,192,193,194,195,196,197,198,199,200,201,202,203,204,205,206,207,208,209}-*` | UNRELATED_CHANGE | Historical Scala backlog audit/deferment updates do not implement fn-227. |
| `.flow/tasks/fn-{137,156,171,173,190,191}-*` | UNRELATED_CHANGE | Historical task annotations accompanying that backlog audit. |
| `.flow/specs/fn-211-*`, `.flow/tasks/fn-211-*` | UNRELATED_CHANGE | Apalache planning remains outside this spec’s implementation scope. |
| `.flow/specs/fn-228-*`, `.flow/tasks/fn-228-*` | UNRELATED_CHANGE | Runner/Run planning belongs to the separate successor spec. |
| `.flow/specs/fn-{219,220,222}-*`, `.flow/tasks/fn-{219,220,222}-*` | UNRELATED_CHANGE | Other backlog/lifecycle documentation updates. |
| `model/umpire/reference/kettle/**` | UNDOCUMENTED_ADDITION | The service, guest, catalog, realization, probes and evidence implement fn-233 rather than a fn-227 requirement. |
| `model/umpire/hosts/go/reference/kettle/**` | UNDOCUMENTED_ADDITION | Independent Kettle server, validators, mutants and generated transport belong to fn-233. |
| `model/umpire/hosts/go/kettle_annotations_test.go` | UNDOCUMENTED_ADDITION | Kettle annotation integration proof. |
| `model/umpire/hosts/go/kettle_mutants_test.go` | UNDOCUMENTED_ADDITION | Kettle differential mutant proof. |
| `model/umpire/hosts/go/kettle_test.go` | UNDOCUMENTED_ADDITION | Kettle live service proof. |
| `model/scripts/proto-kettle-go.sh` | LEGITIMATE_SUPPORT | Generation/freshness infrastructure for the separately specified Kettle addition. |
| `.flow/specs/fn-233-*`, `.flow/tasks/fn-233-*`, `.flow/artifacts/fn-233-*/*` | LEGITIMATE_SUPPORT | Explicit specification and evidence for that collateral addition; it is documented elsewhere. |
| `.flow/specs/fn-{232,236,239}-*`, `.flow/tasks/fn-{232,236,239}-*`, `.flow/artifacts/fn-{232,236,239}-*/**` | LEGITIMATE_SUPPORT | Separately owned guest reproducibility and certification evidence supporting refreshed guest verification. |
| `model/scripts/guest-lock.py`, `model/scripts/guest-rustc.py`, related guest scripts/tests and `model/GUEST_BUILDS.md` | LEGITIMATE_SUPPORT | Host-independent guest construction/freshness infrastructure. |
| `.flow/specs/fn-238-*`, `.flow/tasks/fn-238-*`, `.flow/artifacts/fn-238-*/**`, `model/design/shared-front-end.md` | LEGITIMATE_SUPPORT | Recorded architecture amendment consumed by fn-227’s later source/IR tasks. |
| `model/design/rust-verdict-measurements.md` | LEGITIMATE_SUPPORT | Historical measurement material retained while consolidating DESIGN documentation. |

The remaining implementation, fixtures, generated mirrors, manifests, documentation and fn-227 artifacts trace to requirements 1–24 above. The collateral additions and unrelated planning changes do not create a completion gap.

## Gaps Found

None under the current spec and its recorded amendments.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | met | World/adapters/starts/suite coverage in verification items 1–4 and 14. |
| R2 | met | Finite/domains/catalog/fairness coverage in items 5–10 and 14. |
| R3 | met | Binding, emission, phase/rejection, replay and linear-IR coverage in items 11–15. |
| R4 | met | Retry semantics, terminal laws, retained refinement and activity goldens in items 16–19. |
| R5 | deferred | Explicitly retired here and transferred to fn-237 by owner decision. |
| R6 | met | Request transport/capture, transactional admission and live SDK proof in items 20–23. |
| R7 | met | Delivered documentation, compiling example, counts and timing evidence in item 24. |

Unaddressed R-IDs: []

Classification counts: 0 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":0,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict> - All requirements implemented (R-IDs all met or deferred)
