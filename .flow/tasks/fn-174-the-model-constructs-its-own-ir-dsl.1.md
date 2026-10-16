---
satisfies: [R1, R2, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.1 The reify module: body lifter moved into a capture macro, `@lifted` at the definition site

## Description
Create `model/framework/reify`, a Scala compiler plugin compiled against the pinned `scala3-compiler` before the DSL and packaged as a jar like the lifter. Move `Expressions`, `Types`, `Expansion`, `Roles`, `Equality`, `Constants` and the name/position parts of `Context` from `model/framework/export` into it, still written against `Quotes` (obtained in a phase from `QuotesImpl`, as the spike's plugin host did), with two seams: definition of a symbol (the current run's typed tree, or the `@liftedIr` carrier read from TASTy) and refuse (`report.error` at the refused tree). Register the standard phases after `posttyper` and before `pickler`: **store** (a def with the plain `@lifted` marker is lifted at its definition and the `ir.Function` is added to its symbol as `@liftedIr`), **capture** (each capture-marked DSL parameter's argument is lifted and the call rewritten to supply the `ir.Function` beside the callable; `choose` applies `namedCopy` and step cardinality to the stored IR of the defs its alternatives call, refusing at the alternative), and the **names** phase's hook (its DSL slot is .2's; the phase itself is written here and exercised by .2). Rule: a capture lifts its own compilation unit; a reached def in another file must be `@lifted`; `@sugar` defs expand from jar TASTy; anything else is refused naming the def and the fix. Add `-Xplugin:<reify jar>` to the Model builds (`model/project.scala`, `dsl/project.scala`, samples, fixtures); do not add `-experimental` or `-Yretain-trees`. Decide the nested `records` case with a fixture (phases run before inlining, so it is expected to be seen intact). Store and construct the `ir.Function` within the 64 KB constant limit (chunked constants or bytes in the carrier; generated code at the capture); measure on the largest function. Decide refusal testing with the first refusal fixture: `typeCheckErrors` if it runs plugin phases, else a compile of the fixture. The TASTy lifter keeps working through the moved module until .8 (its `Index.defs`/`LiftError` seams).

The prototype to start from is `.flow/tmp/fn174-plugin-prototype/` (verdict: `model/design/compiler-plugin-prototype.md`): `plugin/LiftedPlugin.scala` has the three phases with a step-count stand-in for the lifter, `lib/` the marker and carrier annotations, `helper/`, `caller-*/` and `warm/incr/` the boundary fixtures over the real framework DSL. Its logs show the stored carrier read across a jar by a plugin phase and by a macro, the refusal at `Refused.scala:14:34` with today's text, and the warm helper-only edit recompiling the caller. Replace the stand-in with the moved lifter; keep the fixtures as this task's boundary proof.

**Touches:** model/framework/reify/**, model/framework/export/**, model/framework/inspect/**, model/framework/dsl/**, model/framework/check/**, model/framework/samples/**, model/framework/lint/Conventions.scala, model/framework/lint/Conventions.test.scala, model/README.md

**Current execution scope (owner, 2026-10-14):** focus on framework Model code. Do not edit or run `model/temporal/**`, `tools/umpire/**`, or their integration gates for now. Implement and verify the real plugin foundation against framework fixtures and the six samples; use framework-only compiler/JVM checks, not the Go sample checker or the mini gate. Keep the original mini function oracle, Temporal build integration, module-map update and `make model-check-changed` obligation below open for the later integration boundary. A framework checkpoint does not complete this task or satisfy those deferred acceptance criteria. The conductor owns tracking and MILESTONES updates; no generated schema, checked-in IR or real-feature changes are selected.
**Scope:** plugin foundation and its old-host coexistence, framework/shared code, six samples and `activity_mini`. Kind conversion/capture integration stays with .2–.7. Existing Model spellings stay; only R9-authorized cross-file `@lifted` markers and the fixture-proved nested `records` decision may alter Model source. The named shared TaskQueue helper is read by the mini's cross-file function chain; no other real-feature Model edit or sweep is selected. Generated payloads, IR schema and Go are unchanged. Any source-change boundary not covered by R9 is reported to the conductor before editing, not waived by a local passing fixture.

**Execution prerequisites:** the conductor verifies `done` and committed local closures for fn-172.1–.6 and fn-141.7 before claiming this task. Flow's task dependencies cannot cross specs, so these checks are explicit execution preconditions, not fabricated task edges. Fn-172's whole-spec review/closure remains open for the owner's source-batch boundary; it is not required before the plugin source lane starts.

### Source re-anchor (2026-10-14)

Read `.flow/tmp/fn174-body-port-research.md`, `.flow/tmp/fn174-capture-api-research.md` and `.flow/tmp/fn174-boundary-conflict.md` while they exist, then verify their cited source against the task's baseline. These are primary-source surveys and a recorded conflict, not implementation receipts; the conflict is resolved by the plugin host, as `model/design/compiler-plugin-prototype.md` measures.

- The nominated traits are not a closed module: their `self: Lifting` closure reaches declaration-owned channel/finite metadata, phase projections, composed fact keys, block syntax and declaration callback services. Separate shared body semantics from temporary old-host adapters; do not import a second declaration evaluator into the plugin or use post-compile/span lookup as a fallback.
- A single `ir.Function` contains no dependency catalogs. Define the collection/linkage seam for transitive functions and types without changing the IR schema or losing names. Stored cross-file function IR must support `choose`'s existing helper-body rewrite and `$choice` copies without reading the external body; the prototype's `viaSingle` shows a stored def reading another stored def's carrier.
- Preserve nested sugar diagnostic enrichment before the plugin reports: replacing every inner `LiftError` with an immediate error would bypass the existing enrichment wrapper. Positions are exact everywhere; keep the refused tree as the anchor.
- Most author APIs and generic claim/capability helpers currently accept plain callable functions. Preserve native callable values and public typing beside captured IR; the capture rewrite supplies the IR without changing the author-visible signature. Abstract `end`/`toProduct` overrides and structural evidence methods need concrete definition-site association.
- Start with the prototype's two-file boundary proof (an annotated helper and a `choose` using it), now with the real lifter: complete function closure, warm-server helper-only edit, missing-marker refusal and refusal locations. This supplements, never replaces, the all-samples/mini oracle and the nested `records` fixture.
- The archived spike proves nine Kettle functions in a plugin phase and the staleness defect; the prototype proves the carrier, the cross-boundary refusal and zinc invalidation with a stand-in lifter. Neither proves typed message/chunk emission or full sample/mini coverage. Do not inherit those claims.

### Verification

- Start test-first with affected body-engine and boundary fixtures; register new suites in the existing coverage registry. Preserve the old host's fixture IR and refusal contracts while it uses the shared implementation.
- Exercise refusals at the refused tree inside captures and inside `@lifted` defs, marked and unmarked cross-file calls, transitive closure/choice transformation, generic argument/evaluation semantics, nested `records`, type/channel/phase metadata, public typing and native callable retention. The stand-in does not satisfy the complete function oracle.
- Run the warm-server helper-only edit regression through the real compile path, not a clean rebuild or a hash guard: the dependent unit must recompile in zinc's second cycle and its IR or refusal must change. Record the observed cycles as the prototype's `logs/warm-2.log` does.
- At task end run `make model-check-changed BASE=<committed predecessor>` once with source frozen and cite the record and mini equivalence. Use the preceding source task's scoped evidence as baseline; no redundant broad baseline, bespoke proof scripts, immutable manifests or checked-in regeneration. Format changed Scala and check the task diff.
- Independent review and broad/full Scala validation remain at the complete source-batch boundary. Real-feature migration and full consumer propagation remain held; no credit for their answers follows from the mini oracle.
## Acceptance
- The plugin's capture on every function of the six samples and `activity_mini` gives a message equal, positions stripped, to the TASTy lifter's (R1).
- Refusals keep today's text and point at the refused tree, inside a capture argument and inside a `@lifted` def (R1).
- A reached def without `@lifted` in another file is a compile error naming the def and the fix (R2).
- A `choose` alternative calling a `@lifted` helper that gives two steps is a compile error at the alternative with today's text, when the helper is in another file of the same run and when it is in a jar; the helper bound directly as a rule stays accepted (R2; the prototype's `caller-refused` and `caller-allowed`).
- Staleness regression test: edit a `@lifted` helper's body in another file under a warm Bloop server; the dependent unit recompiles and its IR or refusal changes (R2).
- Models compile with `-Xplugin` and without `-experimental` or `-Yretain-trees`; the compiler pin and the internal API surface the plugin uses are recorded beside it.
- The TASTy lifter still passes its fixtures through the moved module.
- The nested `records` decision, the 64 KB carrier measurement and the refusal-testing decision (`typeCheckErrors` or compile) are recorded with their fixtures (R9, R8).
## Done summary
Blocker resolved by the spec amendment of 2026-10-14: the compile-time host is a compiler plugin (`model/design/compiler-plugin-prototype.md`); the task is `todo` again with a rewritten description and acceptance. The handover below is kept for the record.

Blocked (2026-10-13, superseded):
# fn-174.1 boundary handover

BLOCKED: DESIGN_CONFLICT
Task: fn-174-the-model-constructs-its-own-ir-dsl.1
Summary: Definition-site generated storage is not available through the supported Quotes member lookup for the existing caller-specific compile-time `choose` refusal.
Impact: The full .1 implementation and downstream function-capture integration remain open.
Suggested resolution: Read `.flow/tmp/fn174-boundary-conflict.md` and choose an explicit metadata/staging design or authorize the narrow stored-helper validation timing change before implementation.

Task status remains `in_progress`. No production or tracked edit, staging, commit, review or `flowctl done` occurred. Owner dirt is preserved; HEAD/base remains `97f0ff053fc833e0ccd20e15f641e17d2d39b938`. All probe commands exited; the private probe Bloop server stopped successfully.

Evidence and exact limits: `.flow/tmp/fn174-boundary-conflict.md`, with the frozen native/old-host contradictory case, compiler phase logs, helper-first and binary visibility controls, supported-inline-generation refusal, narrow warm stored-string result, and alternatives. These probes do not satisfy task acceptance or the complete sample/mini function oracle. All implementation obligations remain open. The task-end `GOMEMLIMIT=6GiB make model-check-changed BASE=97f0ff053f` was not reached.

Additional conductor-requested control: an ordinary generated accessor with an encoded literal-singleton return type compiles but remains invisible to a compiled consumer's Quotes. A helper-only jar edit does re-expand the consumer under warm Bloop (actual macro invocation trace, unchanged consumer SHA256) and changes the runtime stored value. This proves classpath invalidation for the staged scratch case, not a compiler-visible IR seam. The report quantifies its two disjoint compile invocations and distinguishes them from a forbidden duplicate compilation of the same Model input set. Both private probe servers are stopped.

baseline: prior scoped evidence `.flow/tmp/checks/20261013-182854/record.md` read as dispatched; no blanket BASELINE_HANDOFF inferred and no redundant broad baseline run.

stage: impl-review - skipped(config: REVIEW_MODE=none; owner defers independent review to the source-batch boundary)

Tier: session (jev-unavailable(no_key)); project implementer routing gpt-6.1-sol medium.

Blocked:
Owner override. Native Umpire framework DSL authoring, typing, runtime, caller/API retirement and native guidance come first. This original mixed/integration task remains open and held while native fn164.2/.10, fn165.7-.9, fn173.10/.11, fn123.9/.10 and fn153.7 complete. Consume native results later without repeating or claiming their deferred IR/Go/backend proofs. Retain all original exact IR, metadata, schema, compatibility, consumer and integration requirements. Full spec closure is not a native prerequisite. Ignore model/temporal entirely; no task resumes Temporal work without new owner authority. After native DSL completion, re-anchor this task to the final API before release. This is a deliberate priority/scope hold, not evidence of completion or a technical failure.
## Evidence
- Commits:
- Tests:
- PRs:
