---
satisfies: [R4, R7]
---
# fn-195-modelsuite-a-models-test-file-holds-its.22 Native explicit conformance admission: exact binding and complete expectation/monitor validation

## Description
Add the bounded admission prerequisite for an authored conformance declaration with an explicitly supplied, fully constructed realization. This is partial R4/R7 coverage. It neither discovers a complete realization universe nor registers a conformance test, replays a path, grants producer eligibility or closes fn195.11.

Consume committed authored capture `.2`, native checked performed/waited coverage `.17` and the shared full-value expectation validator `.18`. The current partial S registration/execution source checkpoint `ff0dcaf55d460f887bf5fb56d7e31b4ecd92d7f1` is source context, not satisfaction of full `.8`. No dependency on full `.8`, `.9` or active `.21` is needed for this isolated admission boundary.

### Files and Touches

**Size:** M
**Touches:** [model/framework/dsl/NativeConformance.scala, model/framework/checker/ConformanceAdmission.scala, model/framework/checker/ConformanceAdmission.test.scala, model/framework/SEMANTICS.md]

Exactly four physical files:

- `model/framework/dsl/NativeConformance.scala` — new declaration-owned request and opaque result carriers.
- `model/framework/checker/ConformanceAdmission.scala` — new checker-private exact binding and expectation/monitor admission.
- `model/framework/checker/ConformanceAdmission.test.scala` — focused consumer controls, including inline source/library and declaration-only clients.
- `model/framework/SEMANTICS.md` — minimum accurate bounded admission contract.

### Implementation contract

`NativeConformanceRequest[M]` is private to the framework and DSL-owned. It retains the exact Model, supplied `ConstructedRealization`, full `RunExpectation`, immutable `RunBody[M]` containing its existing deferred targets/conditions, declaration identity and `RunPosition`. Request construction and declaration compilation/loading do not load checker implementation classes or evaluate deferred inputs, property predicates, effects or assertions. A realization is fully constructed at execution admission; passing `this` into an early superclass constructor does not establish that condition.

Construction completion is the framework-private caller precondition established by the downstream post-construction execution lifecycle, not a new runtime constructor detector. Preserve strict safe-init negative controls for constructor-this escape and a legal after-construction positive. Do not add JVM stack inspection, an author completion/registration protocol or a relocation restriction, and do not claim a general runtime completion proof here.

Checker-private `ConformanceAdmission.admit(request)` returns an opaque immutable checked binding snapshot retaining the checked coverage, canonical monitors and original request. The request may own a framework-private typed once-result cell, following existing owner-local acquisition patterns; immutable means retained metadata/binding values, not absence of a private cache. Admission computation stays checker-owned. A declaration-owned binding carrier is permitted only with framework-private, admission-controlled construction and no public copy/forge route or checker implementation type in a public DSL signature. It returns no eligibility. Check exact Model identity and consume `nativeCoverage(expectedModel)` with its native validation/acquisition ordering, immutable retained coverage and original failure/reentry semantics. Do not use global inventory membership, machine class, source name or candidate count as explicit ownership evidence. A second distinct supplied realization for the same exact Model remains admissible; a foreign same-class Model refuses. Later unrelated candidates do not revoke explicit binding.

Call the existing `validateConformanceExpectation` on the entire retained value, including raw apply/copy values. Preserve all nine independent fields, original monitor order and original monitor references. During actual execution admission, acquire the selected Model's monitor section once through `NativeCheck.section(exactModel, "monitors")`. Resolve raw names from actual returned member keys; references require exact membership in that selected section. Canonical identity is the exact monitor reference: aliases and name/reference combinations cannot evade duplicate detection. Unknown or ambiguous keys, foreign references and acquisition errors produce located refusals with retained original causes. Do not use `claimName`, declaration text, another Model or a global member scan as fallback ownership proof. Lookup does not execute predicate/next/violated/effect/assertion callbacks. Preserve existing selected-section initialization semantics and leave unrelated lazy realizations/helpers outside that section untouched. If required typed forms expose an acquisition safety gap, report that exact gap before widening scope or imposing author restrictions.

Keep retained binding data immutable, owner-local and collectible. New retained references may legitimately retain their selected owner while the request/result exists; release controls drop those results before requiring collection and preserve live State/Result neighbors. No global strong registry is introduced.

### Verification and handoff

Required investigation targets: `model/framework/realize/Constructed.scala`, `model/framework/realize/Expectations.scala`, `model/framework/checker/NativeCheck.scala`, `model/framework/dsl/NativeRuntime.scala`, `model/framework/checker/RuntimeLifetime.test.scala`, `model/framework/dsl/RunBody.scala`. Reuse their existing exact-owner, selected-section and deferred-carrier seams; do not introduce a second validator or registry.

At implementation admission, baseline the exact current checker/coverage/expectation controls reached, then run the new consumer controls and fresh source/separately packaged-library clients under unchanged strict flags, pinned tools and the frozen dependency. Capture terminal exits and nonzero executed counts. Prove checker-absent declaration compilation/loading, callback poison-zero, original failure retention, repetition/reentry and released-owner behavior. Do not rerun old scratch recipes as current proof or reuse historical pass counts.

SEMANTICS states only exact explicit admission and checked metadata; replay, eligibility and public registration remain downstream. Actual automatic selection, samples, generated capability paths, complete selector/Registry integration and full R4 stay in blocked `.11`. Historical `.10/.15/.16` reports and failed oracles remain unchanged. `.3` still waits for full `.11`; no old bridge retires here. Keep actual repository Make, Go, Temporal reads, export/lifting/IR/schema/backend/generated/regeneration, server/recorded-Run/Case operations and publishing held.

## Acceptance
- [ ] Exact explicit admission consumes `.17` checked performed/waited coverage for a fully constructed selected realization and preserves exact Model/action ownership, native validation ordering and immutable coverage/failure semantics. Controls admit another distinct realization for the same exact Model; refuse a same-primary-class foreign Model, malformed/unsupported coverage and changed owner association; keep a second lazy realization and unrelated lazy helper untouched. No inventory membership/cardinality or eligibility is inferred.
- [ ] The actual consumer calls `.18`'s single validator on complete retained full/raw/copied expectations and preserves every independent field. Selected-section monitor lookup runs once at execution admission, resolves actual member keys and exact selected references, rejects unknown/ambiguous/foreign/canonical duplicate monitors across aliases and name/reference forms, retains authored order/references and reports located sticky acquisition failures. Private-monitor/raw/copy bypass and valid-neighbor controls pass without executing predicate/next/violated/effect/assertion callbacks.
- [ ] The DSL-owned framework-private request retains exact Model/realization, immutable deferred RunBody, full expectation, identity and position; checker-private admission returns only an immutable checked binding snapshot with no eligibility. Repeated acquisition/reentry/partial failure retains deterministic original-cause behavior. Declaration construction/loading stays checker-independent and callback-free; released request/binding graphs collect after legitimate retained results are dropped, while live State/Result controls remain valid.
- [ ] Focused controls and fresh outside-package source/separate-library clients execute nonzero counts under unchanged strict flags and pinned tools, including checker-absent loading and unavailable-runtime neighbors. Receipts retain exact commands, terminal exits and immutable source/artifact bindings. SEMANTICS documents this bounded explicit admission only; full `.11` automatic/sample/capability/Registry integration, old dependency holds and `.3` retirement ordering remain open.


## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: