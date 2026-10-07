---
satisfies: [R6, R7]
---
# fn-137-capabilities-read-phase-roles.5 Default end for Phased objects (needs fn-136's Closed role)

## Description
A `Phased` machine or composition gets `end` = "the phase is `Closed`" by default. The lifter synthesizes it, and the objects whose hand-written `end` says only that drop it. Starts only once fn-136 has landed the role traits and role-test lowering. Behaviour pin: regeneration is byte-identical.

**Size:** M
**Files:** model/umpire/Syntax.scala, model/umpire/Machine.scala, model/umpire/Compose.scala, model/irgen/Declarations.scala, model/irgen/Compositions.scala, model/irgen/testdata/** (default-end fixtures), model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/nexus/workflow/system/{System,TrustingCaller}.scala
**Touches:** [model/umpire/**, model/irgen/**, model/temporal/features/activity/standalone/system/System.scala, model/temporal/features/nexus/workflow/system/System.scala, model/temporal/features/nexus/workflow/system/TrustingCaller.scala]
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.

### Approach
- `Machine.end` is abstract (Machine.scala:280), and `Composition`'s `end` is too. `Phased` supplies a concrete `end` reading the `Closed` role through a type witness for `Closed` over `P`, resolved at the `Phased[...]` parent (the same witness mechanism as spec R3).
- A trait can't see an override without reflection, so the framework refusal fires on the default `end`'s first evaluation: no case of the `Finite` phase type is `Closed`, so it fails naming the object and the phase type. The lifter refuses statically.
- Build the "cases of P with role R, refusing when empty" check once, in the lifter and in the framework, so tasks 6 and 7 reuse it (plan-review maintainability note).
- Lifter: a machine's `end` is read as a DefDef member (Declarations.scala:213-218, 257-259) and a composition's at Compositions.scala:199 and :238. When there is none and the object is `Phased`, synthesize `end` as the `Closed` case-set membership of the projection, using fn-136's role lowering. Refuse when there is no `Closed` case. Derived and derived compositions keep forwarding their source's `end`.
- Drop `end` where it is exactly the terminal-phase test: activity System.scala:62 and the composition at :412; nexus workflow System.scala:81 and the composition at :431; TrustingCaller.scala:28. First confirm that fn-136 made each `terminal`/`terminalPhase` set equal to the `Closed` cases. If one differs, stop and report rather than absorb the difference.
- ActivityRecord (Record.scala:98) keeps its override.

### Acceptance
- [ ] Five hand-written `end`s removed, and `make umpire-gen-model` leaves model/ir and model/cases byte-identical.
- [ ] Framework test: the default `end` holds exactly in the `Closed` phases, and an override wins over it.
- [ ] Relying on the default with no `Closed` case is refused on first evaluation (framework test) and by the lifter (refusal fixture), each naming the object and the phase type.
- [ ] The role-case check is one shared helper per layer.
- [ ] `make umpire-check-model` and `make lint-model` pass.

## Acceptance
- [ ] TBD

## Done summary
Added the Closed-role default end to Phased machines and compositions, backed by one witnessed finite role-case helper in the framework and the existing shared role-case check in the lifter. The lifter synthesizes the same end for omitted definitions and refuses a phase with no Closed case by object and type. Removed exactly the five equivalent Activity/Nexus workflow/System/composition ends; all other explicit Phased ends gained only the required override modifier and retained their bodies. Explicit no-Closed overrides and derived source ends remain authoritative. The illegal derived-machine rephasing case is now rejected earlier by Scala because Derived.end stays final; the derived-composition initialization guard remains. No generated IR/Cases/goldens were changed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f85bfe0528
- Tests: scala-cli test model/project.scala model/umpire model/temporal (53 passed), scala-cli test model/irgen --test-only umpire.irgen.DefaultEndsSuite (3 passed), make lint-model-models lint-model-irgen lint-model-syntax (exit 0), scala-cli fmt --check all 285 Model Scala files (exit 0), exact audit: five ends removed; remaining explicit end bodies modifier-only, batch deferral: authoritative IR/Case byte comparison at single regeneration against 96de1fd92d
- PRs: