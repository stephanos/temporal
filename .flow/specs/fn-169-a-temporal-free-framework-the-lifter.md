# A Temporal-free framework: the lifter and checks join `model/framework`

## Conversation Evidence

> user (2026-10-12): "why are temporal/model/irgen and temporal/model/check not in temporal/model/framework? aren't they framework, non-temporal-specific things? we want to have framework/ be sth we could re-use for other things if we wanted to later. it needs to be self-contained and temporal-free. propose a package layout"
> user (2026-10-12): "and what about temporal/tools/umpire; are temporal and non-temporal concerns clearly separated?"
> user (2026-10-12), on the conductor's proposed layout and sequencing: "yes, approved"

## Goal & Context

`model/framework` is meant to be reusable for systems other than Temporal: self-contained and Temporal-free [user]. Today only the DSL lives there. The lifter (`model/irgen`, 15.4k lines), the checks (`model/check`, 3.8k) and the migration harness (`model/migrate`) sit beside it, although most of their code is generic [inferred]. This reverses the earlier recorded position that the lifter and the Testpilot IR are Temporal's driver tooling by design (fn-114.12, fn-122.8; `MILESTONES.md` Direction, `.plans/UMPIRE_MODULES.md`) [user, paraphrase].

Measured coupling (conductor survey, 2026-10-12) [inferred]:
- `framework/`: one Temporal reference, a comment in `StepTables.scala`.
- `irgen/`: the Temporal realization kit (`Realizations.scala`, 2.7k lines, and parts of `Syntax.scala` lift `temporal.realize` constructs such as `Realizes`); layout conventions (`Structure`, `Lifting`, `Context`, `Admission` hard-code `temporal/features`, `foundations`, `capabilities`, `actors`); Temporal API protos (`Conventions.scala`, and the build's `api-scalapb.jar`); the IR schema namespace `io.temporal.server.api.umpire.v1`.
- `check/`: `Equivalence`, `StepLedger`, `Tools`, comment and block-form rules are generic; `Gate`, `ProtoLiterals` and the kit part of `SyntaxRule` are Temporal's.
- `migrate/`: a generic rule-proof harness; every rule and proof is Temporal/mini.
- Go `tools/umpire`: core packages (`ir`, `interp`, `internal/engine`, `check`, `explore`, `export`) import no Temporal API, but activity/nexus specifics live inside generic packages (`lower/activity_reset.go`, `lower/activity_external.go`, `lower/realization.go`, the matching `realization/` files, Temporal binding kinds in `lint/`, evidence/guard logic in `conformance/`). Testpilot has a generic core and a `testpilot/temporal/` adapter; `internal/execution` imports the Temporal API in six files. `tools/canary` is Temporal deployment tooling by nature.

## Architecture & Data Models

Target layout [user-approved proposal]:

```
model/
  framework/                 package framework.*   (no Temporal jar on its classpath)
    dsl/                     package framework.dsl: the authoring DSL (IR type names kept stable)
    realize/                 generic realization core (as today)
    lift/                    today's irgen minus the Temporal parts, plus a plug-in seam:
                               layout conventions as configuration,
                               realization-kit lifting as a registered kit,
                               target API descriptors injected, not imported
    check/                   Equivalence, StepLedger, step tables, generic source rules, Tools
    migrate/                 the rule-proof harness (prove-mini runner, patch application)
  temporal/                  package temporal.*    (everything Temporal)
    Bounds.scala, actors/, capabilities/, foundations/, features/   (the Models)
    realize/                 the Temporal kit (as today)
    lift/                    the Temporal plug-in: kit lifting, layout config, API descriptors
    check/                   the Gate, ProtoLiterals, kit syntax rules
    migrate/                 task rules and their mini proofs

proto/ (or model/framework/ir/)  umpire.ir.v1   the IR schema, out of Temporal's namespace

tools/umpire/                generic Go consumers
tools/umpire/temporal/       Temporal adapter (API builders, activity/nexus lowering and realization specifics,
                             Temporal lint binding kinds, descriptor registration)
common/testing/testpilot/    core; internal/execution's API use moves behind testpilot/temporal/
```

Enforcement: `framework/` compiles without the Temporal API jar or `temporal.*` sources on its classpath; a source rule refuses `temporal.*` packages/paths under `framework/`; generic Go packages may not import `go.temporal.io/api` (Go part held).

## Edge Cases & Constraints

- Mini-only scope (MILESTONES): only `framework/`, the lifter/check/migrate code, shared code and `features/activity_mini` change; real features stay behind. Moves must keep the mini lift byte-identical apart from recorded source paths, and mini step-table equivalence must hold.
- The IR schema rename and the Go reorganisation touch proto, Go and regeneration: held for the owner's sweep.
- fn-141 rewrites the lifter. The mechanical move runs before it, so fn-141 lands in the final homes; most kit coupling disappears in fn-141.8 (realizations exported from constructed values), and the residue moves to `temporal/lift/` after it.
- Retention (owner, 2026-10-12): the rule-proof harness is permanent and lives in `framework/migrate`. `temporal/migrate` holds only rules the sweep has not yet applied; once the sweep applies a rule to the real features, the rule, its proof and its README section are deleted (git keeps them). The guide's instruction to ship a rule with each DSL change stays.
- Make targets, scala-cli project files, fn-166.13's coverage registry and every doc that names `model/irgen`/`model/check`/`model/migrate` paths follow the move.

## Acceptance Criteria

- **R1:** The generic lifter, checks and migration harness live under `model/framework/{lift,check,migrate}` in `framework.*` packages; Temporal-specific parts live under `model/temporal/{lift,check,migrate}`.
- **R2:** `model/framework` (DSL, realize core, lift, check, migrate harness) compiles with no Temporal API jar and no `temporal.*` source on its classpath, enforced by a build/gate check that fails on a violation.
- **R3:** Temporal specifics reach the generic lifter only through a plug-in seam: layout conventions, realization-kit lifting and target API descriptors are supplied by `temporal/lift`.
- **R4:** The mini lift is byte-identical except for recorded source-path changes; mini step-table equivalence holds; the targeted suites of the moved code pass at their new homes.
- **R5:** `MILESTONES.md` Direction, `.plans/UMPIRE_MODULES.md`, `model/README.md` and `AGENTS.md` state the new layout and that the framework is Temporal-free; the fn-114.12/fn-122.8 position is recorded as superseded.
- **R6 (held, sweep):** The IR schema moves to an Umpire-owned namespace (`umpire.ir.v1`).
- **R7 (held, sweep):** Go `tools/umpire` separates a `temporal/` adapter from generic packages, and Testpilot's `internal/execution` API use moves behind `testpilot/temporal/`, with an import rule.

## Boundaries

Not in scope: changing Model meaning; the real features' sources; publishing the framework as a separate artifact.

## Decision Context

- 2026-10-12, owner: the DSL moves into `framework.dsl` in .1, with IR type names kept stable by a fixed name table.
- 2026-10-12, owner: migration rules are temporary, the harness permanent (see Edge Cases).
- 2026-10-12, owner: framework must be self-contained and Temporal-free; approved the layout above and the sequencing: mechanical move after fn-166.13 and before fn-141; plug-in seam completion after fn-141.8; Go and IR namespace at the sweep.

## Quick commands

```sh
FEATURES=activity_mini make model-mini   # once fn-166.12 lands
```

## Requirement coverage

| Req | Task |
| --- | --- |
| R1, R2, R4, R5 | .1 |
| R3 | .2 |
| R6 | .3 |
| R7 | .4 |
