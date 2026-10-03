---
satisfies: [R8, R9, R10, R13]
---
# fn-120-adopt-what-quint-does-well-named.4 Add IR explorer and semantic-level refusals

Touches: [tools/umpire/model/**, tools/umpire/cmd/**, model/lifter/**, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md]

## Description
Implement Parts C and E after task 3's lint work settles the shared Go model/command surfaces, the choice schema and reader contract settle, and fn-112.10 closes its structural Case freeze. Fn-118's derived-wait work remains independently schedulable only where actual touched files are disjoint; shared lifter or generated-schema edits run serially.

**Size:** M
**Files:** Go model evaluator/command, lifter refusal fixtures, SEMANTICS.md.

### Approach
- The explorer is a command under `tools/umpire/cmd/` with reader-only dependencies. Amend the module map line that reserves `tools/umpire/explore` for it.
- Share one evaluator path between single commands and the interactive shell. Report bounded branch decisions with Scala positions, and say when the last decision was a wildcard arm.
- `state <key>` prints the per-state modality report (`.plans/MODALITIES.md` section 3): every class of the machine as MAY with its results and the Properties that pin them, MUST NOT with the guard at its line, or `?` for an H1/H2 hole, plus the transition Properties and progress claims that apply. `rules <class>` prints the per-operation table task 3's lint prints, grouped by the predicates the decision trace called and by the machine's capability parameters where fn-122 declares them, with gap, overlap and conflict lines for how the guards cover the state catalog. Both views are joins of the same table, trace and claim index; reuse task 3's view code, never a second computation.
- Name semantic levels and locate each reachable wrong-level expression; record type-impossible cases instead of inventing fixtures. Add the "Modalities" paragraph under Machines in `SEMANTICS.md` (R13): a row is permission with fixed results; a disabled pair is prohibition for a system action and silence for a party action; obligations are same-step Properties, progress claims and fairness; refinement narrows permission and does not by itself preserve obligation (the must half is #ZOOM's, Decision Context).
## Acceptance
- [ ] R8 lists starts, enabled classes and named outcomes; unknown names get useful refusals; `state <key>` prints the per-state modality report and `rules <class>` the per-operation table with gap/overlap/conflict lines, both from task 3's table, trace and claim index; a fixture Model with a wildcard arm shows `?`.
- [ ] R9 disabled reasons show bounded branch decisions at their Scala positions, including unbound classes and whether the last decision was a wildcard arm.
- [ ] R10 single-command and interactive modes agree on a fixture.
- [ ] R13 level contract, the Modalities paragraph and reachable refusal fixtures are documented and checked.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
