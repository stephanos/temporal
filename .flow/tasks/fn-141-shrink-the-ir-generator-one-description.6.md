---
satisfies: [R12]
---
# fn-141-shrink-the-ir-generator-one-description.6 Refusal ledger: one recorded outcome per refusal kind

## Description
About 450 refusal sites decide most of what Part B may delete. Classify each kind once, before the migration, so no semantic rule is lost when its tree matcher goes.

**Size:** M
**Files:** a ledger under `.plans/`, read from `model/irgen/**` and `model/irgen/testdata/lifts/expected/rejects.txt`
**Touches:** [.plans/**]
**Deferred:** see MILESTONES.md, Deferred, fn-141. Planned 2026-10-06 against the tree before the DSL batch. The files and names below are that tree's and carry no line numbers on purpose: when the spec is revived, re-read them and recount before starting, since the DSL batch and fn-140 rewrite them.

### Approach
- Group the sites by kind. For each kind record its reject fixture and one outcome. Deleted: the spelling it refused will export, and the fixture becomes a lift fixture. Kept: where it will be enforced, by the compiler, by the framework at construction or by the exporter. Left to Go: the Go reader already repeats it, with the file that does.
- A kept refusal names the author's file and line. Note for each whether a capture point supplies it.
- Carry over every validation of a path that is folded away; memory `consolidated-extractor-dropped-a-2026-09-27` records a regression from skipping this.
- Name the Part B task that handles each kind (8 to 13), so each of them has its checklist.

### Investigation targets
**Required:**
- `model/irgen/*.scala`: every `fail(` and `LiftError(`
- `model/irgen/testdata/lifts/{Rejects,MarkerRejects,ScriptRejects,CapabilityRejects}.scala` and `expected/rejects.txt`
- `model/umpire/*.scala`: the `require` and `throw` sites that already enforce a rule at run time
- `tools/umpire/ir` and `tools/umpire/interp`: what the Go reader validates again

## Acceptance
- [ ] The ledger covers every refusal kind in the lifter at the task's start, with fixture, outcome and task.
- [ ] No kind is marked deleted unless the declaration it refused is legal Scala that will export.
- [ ] The owner has seen the count per outcome.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
