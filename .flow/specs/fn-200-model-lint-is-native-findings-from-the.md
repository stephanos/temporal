# Model lint is native: findings from the graph, acceptances in source

## Conversation Evidence

> user (2026-10-18): "what functionality do we have in temporal/tools/umpire that should actually be in temporal/model/framework?"
> owner (2026-10-18): approved filing the one-language series; lint moves second because the DSL already declares every acceptance subject and nothing writes it.

## Goal & Context

The Go lint reads the IR and the Go interpreter to report what a Model declares that nothing reaches, takes, asks, evidences or realizes, with acceptances kept in a `.lint.json` beside each IR file. None of its kinds exists in Scala, while the Scala side already declares a `@gap` subject for every kind except H5 and resolves it to the same `(kind, owner, subject)` key, then writes nothing. The native checker's exploration already holds the evidence the graph kinds need (reached states, edges, admitted classes, holes, completeness) and already reports "never fires" vacuity. This spec computes every surviving kind from DSL declarations or from the native exploration, makes `@gap` the only acceptance, and deletes the IR lint, its command and its acceptance files. The modality tables and hole kinds H1, H2, H3 and H5 need a decision trace that native closures do not produce; their disposition is the owner's and is recorded here before any implementation.

## Architecture & Data Models

- A lint run takes a checked instance and its `Search` result. Static kinds walk the instance's declarations: unasked Property, unperformed action, unread refinement and observation, explicit wait, witness-only Query, unevidenced fact. Graph kinds read the search: unreachable value per state field, never-enabled per class (action plus input values, not per action), stuck state (no successor, not an end, not reaching a hole), unproduced outcome and fact, unreachable binding and uncovered class, unfired verify (a verify claim whose trigger never fired), untaken choice (needs the choice token carried on each native `Step`).
- The API-value kind walks poll and Run Event guard operands against ScalaPB descriptors; the DSL's `unmapped(GeneratedEnum | GeneratedOneof)` is its acceptance subject.
- Acceptance is `@gap` on the owning declaration, resolved to `(kind, owner, subject)`. A finding with no matching gap fails; a gap that matches no finding is stale and fails. Capability waivers remain a gap kind written by the lifter and judged nowhere.
- Coverage is population minus findings per kind, printed per Model; it never fails a run.
- The modality tables and H1/H2/H3/H5 are a separate module gated by the owner's decision: keep over the IR until the rename, re-implement by instrumenting native step functions, or drop. Until decided, the native lint reports those kinds as "not computed" and the Go command stays available for them.

## API Contracts

- `Lint.of(instance, search)` returns findings, each `(kind, owner, subject, position)`, plus the per-kind coverage tally. Kinds are a closed enum mirroring the Go kind names so existing acceptance reasons carry over verbatim.
- `Lint.judge(findings, gaps)` returns unaccepted findings and stale gaps. Both nonempty sets fail the gate with the kind, owner and subject of each item.
- The framework check runs lint for every sample after its native checks; a cut exploration lints only static kinds and says so.
- `Step` carries the chosen choice token, or none for an unnamed branch.

## Edge Cases & Constraints

- A budget-cut exploration cannot say "never" about anything; graph kinds are suppressed with one warning naming the cut.
- Never-enabled is per class; an action that fires for some inputs and never for others reports the unfired classes.
- A composition lints its own classes and syncs; member findings belong to the member.
- Modality tables stay out until decided; no partial implementation ships.
- No `.lint.json` is read or written once this closes; the lifter's waiver file becomes a gap input only.

## Acceptance Criteria

- **R1:** Static kinds (unasked, unperformed, unread refinement, unread observation, explicit wait, witness-only, unevidenced) are computed from DSL declarations with the Go kind names and subjects. Errors: a Model with no Queries lints without findings of the Query kinds; no error surface beyond that.
- **R2:** Graph kinds (unreachable value, never-enabled per class, stuck state, unproduced outcome and fact, unreachable binding, uncovered class, unfired verify, untaken choice) are computed from the native `Search`; a cut exploration suppresses them with one warning. Errors: a hole-reaching state is never reported stuck.
- **R3:** `@gap` is the only acceptance; `judge` fails on an unaccepted finding or a stale gap and names each; capability waivers are accepted and never judged. Errors: a gap naming an unknown kind or an owner outside the Model refuses at compile time.
- **R4:** The API-value kind runs over ScalaPB descriptors with `unmapped` as its acceptance. Errors: a guard operand naming a field absent from the descriptor is itself a finding, not an exception.
- **R5:** Coverage prints population minus findings per kind per Model and never fails a run (no error surface).
- **R6:** The framework check runs lint over the six samples; every sample passes with its gaps in source, and the Go `umpire-lint` command, `umpire-check-lint` target and every `.lint.json` are deleted together with the IR lint package's static and graph kinds. Errors: none beyond R3.
- **R7:** The owner's decision on modality tables and H1/H2/H3/H5 is recorded in Decision Context before any of those kinds is implemented or deleted; until then the native lint reports them as not computed. No error surface.

## Boundaries

- Modality tables and hole kinds are decided, not built, here.
- No change to what lifting writes beyond the waiver file remaining a gap input.
- No Temporal Model lint runs; Temporal sources stay disabled.
- Go lint tests for deleted kinds are deleted, not ported.

## Decision Context

2026-10-18, phase renumbering: the owner inserted Phase 3 "Temporal Models native" (re-enable and check `model/temporal` against the final DSL) between DSL completion and the IR batch; the IR batch is Phase 4 and the one-language series (fn-199 through fn-209) is Phase 5. This spec runs in Phase 5 unless its Decision Context says otherwise.

Porting the Go lint over the IR into Scala would keep the IR as a lint input and the `.lint.json` files as a second source of truth beside `@gap`. Computing from the native exploration reuses a search the gate already runs, makes acceptance a compile-checked annotation, and removes a file format, a strict decoder and a stale-acceptance judge. The one kind family that cannot follow is the modality tables, because H1/H2/H3/H5 depend on `interp.Why` and `interp.Reads`, the interpreter's trace of which arm disabled a pair and what a claim reads; native step functions are closures. The options are kept open for the owner: keep that slice over the IR until the rename, instrument native bodies (the claim-section bytecode rewriter already exists), or drop the tables, which are the most-accepted findings today. Depends on fn-199 for the verdict kinds and fn-191 for sample lint placement. Filed 2026-10-18 as part of the one-language series (fn-199 through fn-208).

2026-10-18, one-language series: absorbs by reference fn-167 R2-R4 and R6/R7 (gap lowering, inertness, waivers, matching, expiry, all native in .3), fn-173.3's overlap, coverage and growth lint (.1/.2), and fn-123.4's acceptances (gaps in source). The 691 `.lint.json` findings and the H2 review listed under MILESTONES "Open for the owner" are inputs to .5's modality-table decision.

2026-10-18, owner decision (R7): the modality tables and H1, H3 and H5 are dropped; the graph kinds cover the unreachable cases. H2 silent-rejection is decided after the review of the 691 accepted reasons: kept and instrumented alone (the firing rule or its absence per pair, no arm trace) only if accepted H2 findings demonstrably led to Model fixes, otherwise dropped too. A new lint kind, unbounded arithmetic (an arithmetic effect over a var with no bound), is added under R2 per the fn-201 numeric decision.

