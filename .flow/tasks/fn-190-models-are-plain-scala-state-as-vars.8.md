---
satisfies: [R10]
---
# fn-190-models-are-plain-scala-state-as-vars.8 Reconcile fn-189 and re-record the plugin counts

Touches: .flow/specs/fn-189-the-reify-plugin-keeps-only-the-body.md, MILESTONES.md, model/README.md

## Description
> Deferred: the parent spec's 2026-10-21 execution hold applies. These criteria preserve the filed scope; they do not authorize work on the frozen Scala tree. Revalidate against current Rust/successor work before resuming.

**Phase 2 (IR and plugin), owner phasing 2026-10-17.** Runs after phase 1 (fn-190.7) against the final DSL, batched with fn-190.9/.10; one full framework gate at the end. Reconcile fn-189 after fn-190.9 lands the plugin changes.

Mark fn-189.6 superseded by fn-190.2, close fn-189's facet boundary by reference to fn-190.6, and re-record the fn-189 R9 counts (phases, modes, annotations, non-lifter lines) on the final source. Update MILESTONES for both specs.

**Size:** S

### Verification
- Counts reproducible from the recorded command.
## Acceptance
- [ ] Reconcile fn-189.6 and facet obligations by exact requirement-to-checkpoint references after their dependencies; no unimplemented obligation is marked complete by association.
- [ ] Record reproducible final counts for phases, modes, annotations and non-lifter lines with command and source pin.
- [ ] Update both specs' scheduling/documentation consistently, preserving the owner hold until explicitly lifted.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
