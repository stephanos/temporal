---
satisfies: [R2, R11]
---
# fn-107-scala-umpire-prototype-for-standalone.16 Make Scala v2 independent and restore the legacy Scala baseline

## Description
**Touches:** [model/scalav2/**, Makefile, model/scala/umpire/Action.scala, model/scala/umpire/Compose.scala, model/scala/umpire/Domain.scala, model/scala/umpire/Machine.scala, model/scala/umpire/Refine.scala, model/scala/umpire/Search.scala, model/scala/umpire/Sets.scala, model/scala/umpire/Assume.scala, model/scala/umpire/Channel.scala, model/scala/umpire/Monitor.scala, model/scala/umpire/test/Declarations.test.scala]

Make the prototype's Scala sources and tooling independent of model/scala. This is a bounded adoption of the reviewed framework and selected activity/Nexus models into model/scalav2/scala, followed by guarded removal of task 2's legacy delta. Preserve package names and semantics. No feature policy, Go API or IR schema change is required.

**Size:** L
**Files:** v2 authoring project/framework/models, compiler wrapper and proto-jar/gate wiring, provenance fixtures, README, Scala Makefile prerequisites, exact eleven legacy paths above.

### Approach
- Copy the necessary framework, case producer and selected native models/tests into model/scalav2/scala. Keep comments and user baseline bytes. Own compiler helper, toolchain directives and both generated Java jars inside v2; reuse shared protobuf inputs and generic Go models.
- Rewire v2 generation, fixture compilation/lifting, native tests and lint. Native declarations compile independently. The Makefile discovers the new scala/project.scala. No legacy source/helper/jar may be a build or runtime input.
- Actual copied-source lint probe found 51 DisableSyntax findings (19 casts,5 nulls,18 vars,6 loops,2 runtime type checks,1 return): .flow/tmp/fn-107/isolation-lint-probe/lint.log, rc96. Resolve lint with behavior-preserving changes or justified existing line-scoped conventions; preserve comments and lint coverage/config. Record normalization; do not weaken semantic or identity pins.
- Regenerate IR from source. Compare strictly after accounting only for source-location/provenance relocation and recorded formatting line moves. Tables, IDs, refinement, evidence, stuck states and behavior fingerprints stay equal.
- Before restoring each legacy path, compare its current hash to task2-review-r2/manifest.json. For seven existing paths restore exact task2-baseline/bytes; remove four new task-owned files only after adoption is verified. Preserve any later user drift and reconcile it rather than overwrite it. No Git reset/revert/checkout/stash.
- Run meaningful positive and negative isolation controls in a temporary snapshot lacking model/scala and v2 generated jars. Regenerate, compile, native-test, lift, check, lint and compare deterministic artifacts there. The saved pre-isolation gate must fail in the same snapshot. Never move/delete the user's live legacy directory.

### Investigation targets
**Required:** model/scalav2/{run.sh,gen.sh,lifter/Lift.scala,README.md}; model/scala/{project.scala,scala.sh,gen-proto.sh}; selected umpire/temporal sources; Makefile Scala section; task2 baseline bytes/review manifest; current task15 admission/interpretation API; isolation-spec drafts and lint probe in .flow/tmp/fn-107.
**Optional:** legacy proof/view code for dependency inventory only; existing shared Go/Lean/Testpilot fixture inputs.

### Quick commands
GOFLAGS=-tags=test_dep make umpire-check-scala; make lint-scala; mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/...; scoped Go IR lint if its diagnostic paths change. Final isolation snapshot runs the same gates and deterministic regeneration. Record inherited global lint failures separately.

## Acceptance
- [ ] V2 owns its authoring framework, selected models, compiler helper and generated jars; build/lift/test/runtime inputs do not read model/scala. Existing package names and source-linked IDs are preserved.
- [ ] In an isolated snapshot without the legacy Scala tree or prebuilt v2 jars, generation, native compile/tests, IR checks, focused Go tests and configured Scala lint pass. The pre-move gate fails in that same snapshot, and repeat generation produces byte-identical artifacts.
- [ ] Strict pre/post semantic comparison preserves tables, Definition IDs, refinement rows, evidence, stuck states and fingerprints. Only documented source provenance/positions and normalization line moves differ; no pin is weakened and feature IR is not handwritten.
- [ ] Existing comments and preexisting user edits are retained. Only the eleven guarded task-2 legacy paths are restored/removed; all other legacy source bytes remain unchanged, and the restored legacy native baseline still passes in a separate snapshot.
- [ ] Legacy-dependency checks inspect matched content and IR source-location fields accurately, with an explicit negative control; they cannot discard every hit by filtering the containing v2 filename. Every v2 IR source position resolves within v2.
- [ ] Configured lint covers the adopted Scala project without relaxing rules or dropping source coverage. Exact normalization and exception reasons are recorded; all required live/snapshot gates and command closures have actual exit-code receipts. No staging, commits, pushes or worktrees.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
