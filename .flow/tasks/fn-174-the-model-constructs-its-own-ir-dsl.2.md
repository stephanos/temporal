---
satisfies: [R3, R4, R5, R6]
---
# fn-174-the-model-constructs-its-own-ir-dsl.2 Realizations construct their messages; the driver prints them (size proof)

## Description
**Recut 2026-10-18 (one-language series):** fn-174 R5's oracle is now Scala checker answers, fn-203 Cases and fn-206/207 backends. 'The driver prints the IR file' becomes an in-process internal IR (fn-201): no checked-in IR, no `make umpire-check-backends`.

After fn-141.7 (realization factories build complete values). Put the IR jar on the DSL classpath. Each realization type in `model/framework/realize` and `model/temporal/realize` builds its `ir.Realization` (and nested messages) through a conversion beside the type; the descriptor-driven emitter in `export/Realizations.scala` is the model for the generic part. Write the driver in `model/framework/lift`: discover roots, `IrFile.construct`, collect messages, print with the existing printer; for now it takes realizations from the objects and everything else from the tree path, assembled into one `ir.Model` per file. Delete the lifter's realization reader. fn-169.2's seam loses realization lifting. Record the lines removed against the lines added (fn-141 R16's proof point, re-based).

**Names by reference (the resolved .2 conflict).** Today's reader names a command by following a helper parameter to the caller's reference (`Realizations.scala:2300`, `:2446`), so `scripted(original)` and `scripted(alias)` are `original` and `alias` although they construct one value (`.flow/tmp/fn1742-name-conflict.md`). Under the plugin host (spec amendment of 2026-10-14; `model/design/compiler-plugin-prototype.md` §2.2) the reify **names** phase rewrites a declaring-`val` reference passed where a nameable type is expected into `ref.named("<spelling>")`, and leaves parameters alone so helpers pass the caller's spelling through. This task gives the nameable DSL types the slot the phase targets, `named(spelling)` and `ref`, and makes the conversions read it: the owner chooses between a field on each nameable type (`Instruction`'s cases, `Command`'s existing id) and one wrapper type accepted by `everyCase`, `onPath`, `perform`, `aliasOf` and author helpers. Authors write nothing new; `.named(...)` never appears in a Model (R9). An argument of a nameable type that is neither a declaration reference nor a parameter is refused at compile time where today's exporter refuses it. The phase was prototyped in `.flow/tmp/fn174-plugin-prototype/plugin/LiftedPlugin.scala` (`LiftedNames`) against a stand-in `Instruction`; its rule must restrict to parameters whose declared type is not a top type, so `eq`, `==` and `println` leave references alone (the first version broke `original eq alias`). Cover default, named, vararg and constructor arguments (`Realization(scripts = Vector(scripted(alias)))`), which the prototype did not.

**Touches:** Makefile, model/framework/realize/**, model/framework/lift/**, model/framework/dsl/**, model/framework/reify/** (names phase and plugin packaging), model/framework/check/**, model/temporal/realize/**, model/temporal/check/{Coverage,Gate,MakeFeatures,ChangeCheck}*.scala, model/temporal/features/activity_mini/realization/**, model/README.md, .plans/UMPIRE_MODULES.md, model/temporal/migrate/README.md
**Current execution scope (owner, 2026-10-15):** complete the activity-mini/Temporal realization integration from the committed framework checkpoint. Authorize the Temporal realization conversion, mixed driver switch, complete realization-reader retirement, coverage registration, mini oracle, determinism sensitivity proof and the task's scoped Model gate. Also authorize the smallest production build seam that packages `model/framework/reify/plugin` before Model compilation and passes its jar to the Make, Scala gate, change-scoped Model-suite and framework-sample Model compile/package paths, with focused regression tests. The production compile's four exact missing family-flow trust cases are in scope: compiler-intrinsic context-function application, one artifact-identity anchor for the generated API, the `EvidenceRef` and `TypedEvidence` forwarding constructors, and only the canonical `framework.realize.Family` sentinel owner; opaque helpers, spoofed names and authored reserved literals remain refusals. A validated compiler-only contextual-family-package option may preserve the consuming realization package through shared `temporal.realize` helpers; every production Temporal Model path passes it, framework native/refusal tests remain plugin-free, and the frozen mini oracle proves the resulting evidence IDs and sources. This advances .1's explicitly deferred build-integration prerequisite because a test-only plugin-enabled mini jar would leave the standard pipeline broken; it does not claim .1's complete function-capture oracle. Do not edit real-feature sources, generated payloads, schemas, `tools/umpire/**`, Go, Cases or backend propagation, and do not regenerate checked-in IR. Full R5 consumer receipts remain deferred and uncredited. The skeleton prerequisite is the committed .1 framework plugin, not a claim that .1's full sample/mini capture oracle is complete. Keep the single-writer lane serialized, and leave MILESTONES updates to the conductor.
**Scope:** the constructed realization path for `activity_mini`, the shared framework/Temporal kit, the nameable slot and affected synthetic realization fixtures. The six samples remain regression inputs. No real-feature Model, generated payload, schema or Go source is edited or regenerated. R9 still forbids new Model spellings. The conductor verifies fn-141.7's committed local closure and fn-172's six committed task closures before dispatch. This task depends on complete realization values and on .1's plugin skeleton (the names phase); it does not depend on .1's body capture. The single-writer lane remains serialized. Finishing this kind does not satisfy .1 or retire the rest of the lifter.

### Source re-anchor and conversion boundary

- Re-read fn-141.7's final handover and its complete-value source, not just the historical task plan. Retained typed fields, selector paths, descriptor identities, scope contents, aliases and object relationships are the inputs to conversion. Its explicitly deferred declaration/member/fact/family identities belong to this task's construction context; do not invent names from object identity, function identity or initialization order; the carried spelling is the name.
- Convert beside each owning DSL/kit type and use the existing descriptor-driven realization lowering as the semantic reference. Do not add a second tree-side realization producer, span lookup, native evaluator, or hidden exporter fallback. After the switch, each realization is produced only from constructed values.
- Preserve callable values and authored typing. An opaque selector without a captured path is explicitly unsupported and must produce a located refusal; it must not become a guessed field path. Scoped reads and deadline expansion must either convert with all retained metadata or refuse explicitly rather than disappear.
- Preserve root/declaration context in construction failures and write no partial output. Traverse deterministically; names must match the old path, while declaration positions may be empty. Keep all non-realization kinds on their existing path until their own conversion tasks.
- Record removed and added production lines without importing the old line-count estimate as proof. The generic/Temporal ownership boundary stays enforced; fn-169.2's remaining plug-in work is separate.
- The three green pure-conversion snippets under `.flow/tmp/fn1742-pure-conversions/` are recoverable WIP (correlation, Name, typed Proto conversions and their tests); resume them explicitly incomplete.

### Verification cadence

- Register the new realization suites in the existing `model/temporal/check/Coverage.scala`; do not create a parallel registry.
- Start with focused, co-located construction/conversion tests and affected realization exporter fixtures. Register new suites in the existing coverage registry. Exercise every mini realization form, scoped and nested assignments/reads, typed operands, aliases, evidence identities and deadlines, including unsupported-path and construction-failure cases.
- Add the `original`/`alias` fixtures of `.flow/tmp/fn1742-name-probe/` as a names test: both compile, both export from constructed values, and their position-stripped IR differs only in the command id, `original` versus `alias`, as the old reader's does. Add a non-reference argument fixture that refuses at compile time, and a two-level helper fixture.
- Compare old and constructed-path realization messages on the mini and affected synthetic fixtures; review permitted position-only or other intentional IR differences explicitly. Test two separate JVM exports and demonstrate that the determinism assertion detects a seeded nondeterminism, restoring the task-owned mutation before final verification.
- Format affected Scala and run `make model-check-changed BASE=<committed predecessor>` once at task end against frozen source, with mini step-table equivalence. Reuse the preceding task's scoped record, not a redundant broad baseline. If the host still requires an explicit Go memory budget, record it; never present a constrained result as a default-memory pass.
- Broad/full Scala validation and independent review remain at the owner's source-batch boundary. R5's regenerated Cases and consumer/backend answers remain explicit propagation-phase obligations, not passes inferred from source fixtures or mini equivalence. Keep those obligations open in the spec and handover; no real-feature sweep or checked-in regeneration is selected here.
## Acceptance
- Every mini realization and affected synthetic realization fixture comes from the constructed value; `export/Realizations.scala` is deleted (R3). Real features remain held for the owner's sweep.
- Names equal today's; positions may be empty (R4). The `original`/`alias` fixtures give command ids `original` and `alias` from constructed values; a non-reference nameable argument is a compile error; a reference through two helper levels keeps the caller's spelling.
- The nameable slot is chosen and recorded (field per type or one wrapper); no Model writes `.named(...)` (R9).
- Source checkpoint: scoped gate and mini equivalence pass, with IR differences reviewed in the done summary. Full R5 remains: receipts, Cases and `make umpire-check-backends` unchanged after the authorized propagation/regeneration phase; deferred, not claimed by this source checkpoint.
- Two-run determinism test passes and fails on a seeded nondeterminism (R6).
- Line count before/after in the done summary.
## Done summary
Blocker resolved by the spec amendment of 2026-10-14: the compile-time host is a compiler plugin (`model/design/compiler-plugin-prototype.md`); the task is `todo` again with a rewritten description and acceptance. The handover below is kept for the record.

Blocked (2026-10-13, superseded):
# fn-174.2 incomplete: declaration-reference names are lost across helpers

`BLOCKED: DESIGN_CONFLICT`. The existing exporter accepts two fixtures with identical
constructed value graphs but emits command IDs `original` and `alias` respectively.
An ordinary helper erases which reference spelling the caller supplied; permitted
factory-side symbol metadata sees the same helper parameter in both. Full source,
native/IR outputs, literal metadata controls, commands and statuses are recorded
in `.flow/tmp/fn1742-name-conflict.md` and `.flow/tmp/fn1742-name-probe/`.

Task remains `in_progress` for the conductor to record its blocker. No source or
closure commit, Flow `done`, or review verdict was issued. HEAD remains
`97f0ff053fc833e0ccd20e15f641e17d2d39b938`.

Tier: session (jev-unavailable(no_key)); project implementer routing gpt-6.1-sol medium.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers independent review to source-batch boundary)

Baseline: green focused construction, 37 tests, exit 0,
`.flow/tmp/fn1742-baseline-construction.log`. The predecessor's broader bounded
record `.flow/tmp/checks/20261013-182854/record.md` was read as prior evidence, not
used as a blanket handoff for different commands.

### Recoverable incomplete conversion work

Three test-first conversion tests passed in
`.flow/tmp/fn1742-conversion-green.log` after the absent-API red observation in
`.flow/tmp/fn1742-conversion-red.log`. They exercise literal correlation identities/
bounds, typed protobuf fields without invoking their callable selectors, and an
opaque-selector refusal containing declaration context. This proves only those
initial conversion components, not a constructed realization or driver.

Exact task-owned WIP copies are preserved at:

- `.flow/tmp/fn1742-pure-conversions/Conversion.scala`: correlation, Name and typed
  Proto/ProtoValue conversions, originally `model/framework/realize/Conversion.scala`.
- `.flow/tmp/fn1742-pure-conversions/Conversion.test.scala`: three green tests,
  originally `model/framework/realize/Conversion.test.scala`.
- The sole other source hunk was `//> using jar ../../build/ir-scalapb.jar` immediately
  after the ScalaPB dependency in `model/framework/dsl/project.scala`.

The files/directive had already been restored before the conductor instructed
retention as WIP. The exact copies and evidence remain; no edits were replayed.
The WIP was not formatted, registered in Coverage.scala, or run through the full
scoped gate; it must remain explicitly incomplete if resumed. No production
source edits from this worker remain. All owner dirt, including README/Lift and
MILESTONES, was preserved and unstaged.

### Open acceptance and propagation obligations

The reader remains intact; no new driver exists. Complete generic/Temporal
conversion, declaration/source/family metadata, deadlines/status expansion,
construction failure atomicity, all affected fixtures, mini equivalence,
two-separate-JVM determinism and seeded sensitivity, net line-count proof,
formatting and the frozen-source `GOMEMLIMIT=6GiB make model-check-changed
BASE=97f0ff053f` gate were not reached or claimed. No final bounded/default-memory
gate pass is asserted. The small focused runs used no explicit Go memory budget.

Independent review and broad/full Scala validation remain owner-deferred at the
source-batch boundary. Full R5's regenerated receipts, Cases, consumer/backend
answers remain propagation-phase obligations, not inferred passes. No schema,
Go, real-feature Model, generated payload or checked-in generated IR was changed.
This blocker is independent of .1; it neither closes .1 nor authorizes its
excluded compiler-plugin host.

Suggested resolution: approve a concrete reference-metadata contract that survives
ordinary helper calls and bare val aliases, including the capture host or authoring
change required to obtain it. Existing names and accepted spellings cannot both
be preserved solely by converting the current constructed value graph.

Blocked:
The owner explicitly changed execution to framework-only and instructed ignoring `model/temporal` entirely, then requested disabling its Scala sources by extension rename. The remaining mini/Temporal integration and complete scoped gate for fn-174.2 are held. Three unfinished framework edits and all previous failed/interrupted verification evidence remain preserved. No Flow closure or complete gate pass is claimed. Re-scope framework work independently; do not resume Temporal verification until the owner restores that scope.
## Evidence
- Commits:
- Tests:
- PRs:
