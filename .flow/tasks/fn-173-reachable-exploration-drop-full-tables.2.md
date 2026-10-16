---
satisfies: [R2]
---
# fn-173-reachable-exploration-drop-full-tables.2 Phase 1: the reachable graph is the only table (roots, reach bound, open-graph lint, receipt fields, IR structural fingerprint, parity test, one regeneration, Voting drops inv)

## Description
Implement R2 after the macro/source-construction migration in milestone order: the reachable graph
is the only production table, with on-demand rows, separately attributed root regions, a per-machine
reach bound/default, honest open/closed receipts and per-Query `.open(because = ...)` acceptance.
Replace graph-dependent behavior identity with the approved IR structural dependency hash. Preserve
existing answers and witnesses, and complete Case bytes apart from fingerprint changes. Remove
Voting's invariant-as-guard and prove its expectations through reached behavior/refinement.

**Scope:** the owner-approved Phase 1 Go exception inside the Scala batch. Framework/shared code,
`activity_mini`, the six framework samples and synthetic fixtures are selected. Real-feature source,
IR, Cases and legacy payloads remain held for the sweep. One scoped regeneration is permitted here;
it is not permission to regenerate every feature or publish functional/canary pins for unselected
features. Native collections and Scala catalog/totals retirement belong to later phases.

### Re-anchor before implementation

Read `.flow/tmp/fn173-reachability-preflight.md` while it exists and verify its cited owners against
the actual committed predecessor. It is a read-only survey, not passing evidence. Re-read the
approved GOV-02 rules, full spec/design and module import restrictions. The historical design's
ordinary double replay interpretation is stale: fn-172.6 already shares the same-Model binding.
Distinct-Model witness replay must remain independent without retaining an eager fallback.

- Replace eager state production and admission, not just the main Query caller. Follow binding,
  refinement, composition, progress, lint, lowering, conformance, export and Exploration/replay
  consumers. Keep row/typed-transition association and bounded work semantics consistent.
- Native starts, detached Scenario starts and Exploration's currently inherited Scenario start
  remain distinct. Do not append extra roots to native `Starts`, change initial refinement obligations,
  or let a closed region borrow completeness from an unrelated region. Refined images outside a
  product's native region still require valid reached carriers.
- An unexpanded frontier is not a disabled transition. Preserve complete result lists, hole/limit
  precedence, fair-cycle/deadlock semantics and independent work limits. An accepted open graph is
  not a waiver for an actually incomplete Query search or unsupported evaluation.
- Replacement type admission must preserve channel capacity and unordered canonical-order checks;
  current `Conforms` alone is weaker than catalog membership. Do not defer finite-channel parity
  to the native-collection phase.
- Structural identity follows semantic dependency closure and own starts, not explored rows or a
  whole-Model serialization. Preserve meaningful names/order while erasing incidental bindings.
  Respect module layering and prove stale-version rejection through the actual reader path, not
  merely a changed hash salt. Resolve the cited `Step.because` compatibility tension explicitly
  before implementing an erasure that could hide behavior changes.
- Lowering's `fixedRequirements` currently probes all table states. Case parity must compare actual
  Contract clauses and identifiers, not conceal drift behind a fingerprint-stripping projection.
  Keep the old eager relation only as a test oracle, never as a production fallback.

### Verification scope

Capture a fresh, actual mini/sample/fixture baseline at the committed predecessor. The retained
16-table mini source baseline contains no generated Case/live proof; do not infer either from it.
Compare old eager and new reached answers, witnesses, refinement/progress semantics and complete
Cases across the selected scope. Explain changed representation/work counters rather than silently
ignoring differences in outcomes or coverage. Exercise independent/detached roots, exact-bound
closure, partial frontiers/result lists, mapped product roots, typed key collisions, finite channels,
changed-Model replay and the fingerprint mutation/compatibility matrix. Run focused changed-package
tests with `-tags test_dep`, formatting/lint, then the scoped integrated gate and one authorized
regeneration. Review every selected generated diff; preserve unselected artifacts and pins.

Broad/full gates and independent review follow the owner's current batch cadence. Held real-feature
parity and consumer/live obligations remain explicit; no mini-only receipt proves them complete.

## Acceptance
- [ ] R2: every live producer/consumer uses the reached graph only; no eager state-table fallback remains outside tests.
- [ ] R2: root-region reach, bounds, open/closed receipts and per-Query open acceptance are implemented without false completeness or altered hole/search-limit semantics.
- [ ] R2: structural fingerprint dependency/canonicalization and reader-version compatibility are proven; unresolved identity tensions are settled explicitly.
- [ ] R2: selected old-vs-new answers and witnesses match, and complete Cases differ only in permitted fingerprints; representation/work counter differences are exposed and justified.
- [ ] R2: Voting drops `inv` with unchanged expectations; one scoped regeneration is reviewed and leaves real-feature payloads/unselected pins unchanged.

## Done summary
Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
