---
satisfies: [R1, R2, R3, R4, R10, R11, R12, R13, R16]
---
# fn-190-models-are-plain-scala-state-as-vars.9 Phase 2: lift vars, assignment effects, transitions and check blocks; State record, schema and init from declarations

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Recut 2026-10-18 (one-language series):** Keep the lifting: fn-206/207's TLA+ emitter and fn-203's Case producer need vars, effects and transitions. Re-anchor to `framework/lift` (fn-201) and fn-198 case folders, not the export suites; no checked-in IR is written.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Starts after phase 1 (fn-190.7: the six samples compile in the plain form and their native suites pass).

The plugin and lifter half of fn-190.1, .2, .4 and .6, moved here so the IR and plugin change once, against the final DSL:
- From .2: the plugin collects a Machine's vars in declaration order, synthesises the IR record `<machine>.State`, derives its schema from the declared types (bounded aliases from the type tree, enums, records, Option, Set, Map, List, Inbox) and lifts `init` from the initialisers; refuse a Machine without vars and a var whose type has no schema with the spec's texts. Re-cuts fn-189 R4 (fn-189.6 superseded).
- From .4: lifter rules and refusals — an effect body lifts to one `copy` per path with sequential substitution (`+=`, `if`/`match` arms, `choose` arms, `reject`, `records`, `crash`); var reads lift to `field(s, name)` in guards, claims, monitors, refinement and evidence; `before(x)` reads the transition's first parameter; `invariant:`/`transition:`/`after(action):` check blocks reuse today's `check`/`oneOf`/`branch`/`records` lowering (reify/body `requirements`); transitions' `if`/`match` chains lift as nested `if`; progress claims, `only`/`fair` scenarios, the start list from `oneOf(domain)` initialisers, and the spec's refusal texts.
- From .6: remove `Phased`, the projection/writer, `$facet-context-v1` placeholders, substitution, FacetOutcome and the three facet payloads from the plugin; the Facets fixture lifts to the same bindings except positions and phase access shape.
- The prototype's stand-in lifting from .1 becomes the production plugin path here.
- Preserve fn-189 R2/R7's deferred full-mode proofs: Machine's Lifted marker covers authored nested containers without special section spellings, and assignment effects must lift through the single carrier. Fn-189.7 runs the full Model-with-effects/warm-helper-edit and samples/Facets integration proofs after this lowering lands; the earlier native-source checkpoints did not close them.
- Re-anchor the framework export/lint suites to the final DSL in this batch (the parked fn-164.3 bridges work, branch `work/fn1643-bridges`, is input).
- Native `Hole.reached[A]` now carries the expected result type so Scala does not erase later independent declarations after a naked hole. The hole lifter must recognize `TypeApply(Select(hole, "reached"), ...)` as well as the former bare selection; the current `Expressions` matcher still requires that update. Predicate and effect fixtures must retain the same Hole identity and unknown result. Native execution evidence does not establish this lifting behavior.

**Size:** L

### Verification
- A fixture Machine with every supported var type lifts to the expected State record and schema; refusal fixtures report the exact texts.
- Focused export/reify suites pass; the full framework gate runs once at the end of phase 2.
## Acceptance
- [ ] Supported state declarations produce the ordered State record, schema and starts; missing vars and unsupported types yield the specified located refusals.
- [ ] Sequential assignments, branches, reads/before, claims, outcomes/facts, scenarios and exact input domains retain the specified semantics, covered by positive/negative fixtures.
- [ ] Typed Hole.reached forms preserve Hole identity and unknown results; facet removal leaves no stale carrier/context representation.
- [ ] Focused source/lift fixtures pass and full-mode/warm-helper obligations remain owned by fn-189.7; native-only results cannot close them.
- [ ] On explicit resumption, use the recorded lift/case-folder successors, without restoring checked-in IR or running archived paths as current gates.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: