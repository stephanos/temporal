---
satisfies: [R1, R2, R9]
---
# fn-174-the-model-constructs-its-own-ir-dsl.1 The reify module: body lifter moved into a capture macro, `@lifted` at the definition site

## Description
Create `model/framework/reify`, compiled before the DSL and packaged as a jar like the lifter. Move `Expressions`, `Types`, `Expansion`, `Roles`, `Equality`, `Constants` and the name/position parts of `Context` from `model/framework/export` into it, generic over `Quotes`, with two seams: definition of a symbol (`Symbol.tree`) and refuse (`report.errorAndAbort`). Expose `Reify.function[F](inline f: F): ir.Function` and the `@lifted` macro annotation (SIP-63, `-experimental`), which lifts a def's body at its definition and stores the `ir.Function` beside it. Rule: a capture lifts its own compilation unit; a reached def in another file must be `@lifted`; `@sugar` defs expand from jar TASTy; anything else is refused naming the def and the fix. Add `-Yretain-trees -experimental` to the Model builds (`model/project.scala`, `dsl/project.scala`, samples, fixtures). Decide the nested-inline case (`records` inside a Property) with a fixture. Construct the `ir.Function` with generated code or chunked constants (64 KB limit). Unit-test refusals in-process through `typeCheckErrors`. The TASTy lifter keeps working through the moved module until .8 (its `Index.defs`/`LiftError` seams).

**Touches:** model/framework/reify/**, model/framework/export/**, model/framework/inspect/**, model/framework/dsl/**, model/framework/check/**, model/framework/samples/**, model/project.scala, model/temporal/check/**, model/temporal/capabilities/**, model/temporal/features/activity_mini/**, model/temporal/foundations/taskqueue/product/Queueing.scala, model/temporal/migrate/README.md, Makefile, model/README.md, .plans/UMPIRE_MODULES.md
**Scope:** macro foundation and its old-host coexistence, framework/shared code, six samples and `activity_mini`. Kind conversion/capture integration stays with .2–.7. Existing Model spellings stay; only R9-authorized cross-file helper annotations and the fixture-proved nested `records` decision may alter Model source. The named shared TaskQueue helper is read by the mini's cross-file function chain; no other real-feature Model edit or sweep is selected. Generated payloads, IR schema and Go are unchanged. Any source-change boundary not covered by R9 is reported to the conductor before editing, not waived by a local passing fixture.

**Execution prerequisites:** the conductor verifies `done` and committed local closures for fn-172.1–.6 and fn-141.7 before claiming this task. Flow's task dependencies cannot cross specs, so these checks are explicit execution preconditions, not fabricated task edges. Fn-172's whole-spec review/closure remains open for the owner's source-batch boundary; it is not required before the macro source lane starts. No default-memory fit follows from the toolchain closures.

### Source re-anchor (2026-10-14)

Read `.flow/tmp/fn174-body-port-research.md` and `.flow/tmp/fn174-capture-api-research.md` while they exist, then verify their cited source against the task's baseline. These are primary-source surveys, not passing tests or implementation receipts.

- The nominated traits are not a closed module: their `self: Lifting` closure reaches declaration-owned channel/finite metadata, phase projections, composed fact keys, block syntax and declaration callback services. Separate shared body semantics from temporary old-host adapters; do not import a second declaration evaluator into the macro host or use post-compile/span lookup as a fallback.
- A single `ir.Function` contains no dependency catalogs. Define the collection/linkage seam for transitive functions and types without changing the IR schema or losing names. Stored cross-file function IR must support `choose`'s existing helper-body rewrite and `$choice` copies without reading the external body.
- Preserve nested sugar diagnostic enrichment before the macro host aborts: replacing every inner `LiftError` with an immediate compiler abort would bypass the existing enrichment wrapper. Keep argument-tree and definition-site refusal anchors distinct.
- Most author APIs and generic claim/capability helpers currently accept plain callable functions. Preserve native callable values and public typing beside captured IR; prove capture before ordinary wrappers erase trees. Later kind tasks integrate those APIs. Abstract `end`/`toProduct` overrides and structural evidence methods need concrete definition-site association, not an assumption that a factory sees a later virtual override. Retain R9's source-change boundary.
- Start the high-risk boundary proof with two authored files: an annotated single-step helper using jar sugar, and a capture using it inside named `choose` alternatives. Check its complete function closure, warm-server helper-only edit, missing-annotation refusal and direct/definition refusal locations. This supplements, never replaces, the all-samples/mini oracle and nested-inline `records` fixture.
- The archived spike proves nine Kettle functions and demonstrates the staleness defect. It does not prove `@lifted` storage, staleness repair, typed message/chunk emission or full sample/mini coverage. Do not inherit those claims from the design projection.

### Verification

- Start test-first with affected body-engine and macro-boundary fixtures; register new suites in the existing coverage registry. Preserve the old host's fixture IR and refusal contracts while it uses the shared implementation.
- Exercise both direct and definition-site refusals, annotated and unannotated cross-file calls, transitive closure/choice transformation, generic argument/evaluation semantics, nested inline `records`, type/channel/phase metadata, public typing and native callable retention. A toy macro or untyped spike does not satisfy the complete function oracle.
- Run the warm-server helper-only edit regression through the real compile path, not a clean rebuild or a hash guard. Record the actual build dependency and annotation behavior.
- At task end run `make model-check-changed BASE=<committed predecessor>` once with source frozen and cite the record and mini equivalence. Use the preceding source task's scoped evidence as baseline; no redundant broad baseline, bespoke capture/proof scripts, immutable manifests or checked-in regeneration. Format changed Scala and check the task diff.
- Independent review and broad/full Scala validation remain at the complete source-batch boundary. Real-feature migration and full consumer propagation remain held; no credit for their answers follows from the mini oracle.

## Acceptance
- `Reify.function` on every function of the six samples and `activity_mini` gives a message equal, positions stripped, to the TASTy lifter's (R1).
- Refusals keep today's text; inside a capture argument they point at the refused tree; inside a `@lifted` def at the def (R1).
- A reached def without `@lifted` in another file is a compile error naming the def and the fix (R2).
- Staleness regression test: edit a `@lifted` helper's body in another file under a warm Bloop server; the stored IR changes (R2).
- The TASTy lifter still passes its fixtures through the moved module.
- The nested-inline decision is recorded with its fixture (R9).

## Done summary
Blocked:
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
## Evidence
- Commits:
- Tests:
- PRs:
