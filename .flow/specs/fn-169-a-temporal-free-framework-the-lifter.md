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

**Successor amendment (owner, fn-174, 2026-10-13):** fn-141.8 is superseded by fn-174.2.
The remaining .2 seam follows that task's committed local closure and covers layout configuration
and registered API descriptors. Realization conversion belongs beside each framework/kit type;
the seam no longer supplies a tree-side realization-kit lifter. Historical sequencing and kit
lifting references below are read under this amendment, not restored as duplicate producers.

Target layout [user-approved proposal]:

```
model/
  framework/                 package framework.*   (no Temporal jar on its classpath)
    dsl/                     package framework.dsl: the authoring DSL
    realize/                 generic realization core (as today)
    export/                    today's irgen minus the Temporal parts, plus a plug-in seam:
                               layout conventions as configuration,
                               realization-kit lifting as a registered kit,
                               target API descriptors injected, not imported
    check/                   gate driver, Tools, ModelSources, metrics
      lint/                  generic source rules (+ fn-141.1's order/structure/marker lints)
      equivalence/           step-table equivalence, rename maps, ledgers (used by gate and migrate)
    migrate/                 the rule-proof harness (prove-mini runner, patch application)
    samples/                 one-file Temporal-free sample Models: Kettle, Bank, TwoPhaseCommit (P/Quint counterparts)
  temporal/                  package temporal.*    (everything Temporal)
    Bounds.scala, actors/, capabilities/, foundations/, features/   (the Models)
    realize/                 the Temporal kit (as today)
    export/                    the Temporal plug-in: kit lifting, layout config, API descriptors
    check/                   Temporal gate configuration (protos, Go checks)
      lint/                  kit syntax rules, Temporal API proto-name rule
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
- fn-141 rewrites the lifter. The mechanical move runs before it, so fn-141 lands in the final homes; most kit coupling disappears in fn-141.8 (realizations exported from constructed values), and the residue moves to `temporal/export/` after it.
- Retention (owner, 2026-10-12): the rule-proof harness is permanent and lives in `framework/migrate`. `temporal/migrate` holds only rules the sweep has not yet applied; once the sweep applies a rule to the real features, the rule, its proof and its README section are deleted (git keeps them). The guide's instruction to ship a rule with each DSL change stays.
- Make targets, scala-cli project files, fn-166.13's coverage registry and every doc that names `model/irgen`/`model/check`/`model/migrate` paths follow the move.

## Acceptance Criteria

- **R1:** The generic lifter, checks and migration harness live under `model/framework/{export,check,migrate}` in `framework.*` packages; Temporal-specific parts live under `model/temporal/{export,check,migrate}`.
- **R2:** `model/framework` (DSL, realize core, lift, check, migrate harness) compiles with no Temporal API jar and no `temporal.*` source on its classpath, enforced by a build/gate check that fails on a violation.
- **R3:** Temporal layout conventions and target API descriptors reach the generic export module only through a plug-in seam supplied by `temporal/export`. Realization conversion belongs beside the framework/kit types under fn-174, not a registered tree-side lifter. Protobuf support is generic framework: a system registers its API proto roots and descriptor sets, which the proto lint, the gate's proto stage and constructed realization conversion read; the generic proto authoring sugar lives in `framework/realize`.
- **R4:** The mini lift is byte-identical except for recorded source-path changes; mini step-table equivalence holds; the targeted suites of the moved code pass at their new homes.
- **R5:** `MILESTONES.md` Direction, `.plans/UMPIRE_MODULES.md`, `model/README.md` and `AGENTS.md` state the new layout and that the framework is Temporal-free; the fn-114.12/fn-122.8 position is recorded as superseded.
- **R8:** `framework/samples/` holds one-file, Temporal-free sample Models of rising complexity (Kettle, Bank, TwoPhaseCommit) with P/Quint counterparts named, checked by the gate.
- **R6 (held, sweep):** The Umpire IR is Temporal-free, with no extension schema either: an Umpire-owned namespace and a generic realization layer, where Temporal's vocabulary is data (message type names, methods, field paths, kind names) validated against the system's descriptor set and each Driver's declared capabilities.
- **R7 (held, sweep):** Go consumers follow the `framework.dsl` IR type names (`tools/umpire/interp/eval.go` `StepType`/`deliveryType`, test strings); the `tools/umpire` adapter separation is fn-170.2's, and Testpilot's `internal/execution` API use moves behind `testpilot/temporal/`, with an import rule.

## Boundaries

Not in scope: changing Model meaning; the real features' sources; publishing the framework as a separate artifact.

## Decision Context

- 2026-10-13, owner: the Umpire IR must have no Temporal specifics, not even as an extension schema (R6, .3). The Temporal vocabulary is data validated against descriptors and Driver capabilities, produced by the Temporal kit and interpreted by the Temporal Driver.
- 2026-10-12, owner: the Quint-specific parts of Go `tools/umpire/export` move into `export/quint`; with the rest of the `tools/umpire` layout this moved to fn-170.2.
- 2026-10-12, owner: port the Python table comparator (`model/testdata/table-pins`) to Scala in `framework/check/equivalence` (.6); no Python under `model/`.
- 2026-10-12, owner: tests are colocated with what they test (no `test/` folders); `Fixtures.test.scala` sits beside the lifter's entry point until fn-141 splits it per construct.
- 2026-10-12, owner: add `framework/samples/` with one-file sample Models over a range of complexity; conductor picked Kettle (Quint), Bank (P ClientServer, Quint README) and TwoPhaseCommit (P and Quint) for comparison.
- 2026-10-12, owner: protobuf support is framework, not Temporal; a system registers its API protos (.2), and the generic proto sugar moves to `framework/realize` (.1).
- 2026-10-12, owner: `check/` is the umbrella for verification, with `check/lint/` and `check/equivalence/` beneath it and the gate driver at its root.
- 2026-10-12, owner: the Model-to-IR module is named `export` (`framework/export`, `temporal/export`), not `lift`; the lifter's vocabulary in docs and diagnostics may stay. Go's `tools/umpire/export` (the Quint export) keeps its name.
- 2026-10-12, owner: the DSL moves into `framework.dsl` in .1; IR type names may change with it (`framework.dsl.Step`, ...), and the Go interpreter's hard-coded names follow in .4.
- 2026-10-12, owner: migration rules are temporary, the harness permanent (see Edge Cases).
- 2026-10-12, owner: framework must be self-contained and Temporal-free; approved the layout above and the sequencing: mechanical move after fn-166.13 and before fn-141; plug-in seam completion after fn-141.8; Go and IR namespace at the sweep.

2026-10-18, deferred (owner): all specs other than fn-210 (the Rust prototype of the framework in `temporal/model2`) are deferred until fn-210's write-up recommends whether to extend the Rust prototype or stay on Scala. Flow has no deferred status; this line is the record. MILESTONES.md was reset to the prototype alone and no longer lists this spec. Nothing in this spec changes; resume or close it after that decision.

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
| R8 | .5 |

## Redesign impact (fn-173, 2026-10-13)

Owner-approved redesign `fn-173` (reachable exploration; `model/design/reachable-exploration.md`) drops full catalog tables, Scala step tables and equivalence sealing, and Query totals; the Behavior Fingerprint becomes an IR structural hash. For this spec:

- **.3:** the namespace move and its step-table proof should share one schema change and regeneration with fn-173 Phases 1 and 3.
- **Unaffected:** .2/.4.
