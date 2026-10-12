---
satisfies: [R1, R2, R4, R5]
---
# fn-169-a-temporal-free-framework-the-lifter.1 Mechanical move: generic lift/check/migrate into model/framework, Temporal parts into model/temporal; Temporal-free build boundary; docs

## Description
Runs after fn-166.13 and before fn-141 (owner, 2026-10-12). Read the spec's survey and target layout first.

1. **Move.** Generic code from `model/irgen`, `model/check` and the `model/migrate` harness moves into `model/framework/{lift,check,migrate}` with `framework.*` packages. Temporal-specific parts go to `model/temporal/{lift,check,migrate}`: the realization-kit lifting, layout conventions, Temporal API use, the Gate, ProtoLiterals, kit syntax rules, and every task rule with its proof. Where a file mixes both, split it only as far as a mechanical move allows; the full plug-in seam is fn-169.2, after fn-141.8. Record the residual coupling it leaves.
2. **Build boundary.** `model/framework` compiles with no Temporal API jar and no `temporal.*` source on its classpath. A gate/source check fails on a violation.
3. **Migration harness (owner, 2026-10-12):**
   a. Parameterize it by feature. `prove-mini.sh` becomes `prove.sh`, selecting the feature with `FEATURES` (default `activity_mini`) and deriving the copied source set from fn-166.13's scope logic instead of today's hard-coded list (`framework`, `temporal/foundations`, `actors`, `capabilities`, `realize`, `Bounds.scala`, `Activity.scala`, `features/activity_mini`). A rule's rewrite target is the runner's argument, not a hard-coded `activity_mini/protocol`.
   b. Reuse `framework/check` for the before/after lift comparison, including the declared-delta case (moved source lines, recorded identity maps) that proofs now handle with their own scripts (`RewriteGapsMini compare`, `check-*-delta.mjs`). Retire the plain `cmp` comparison and the per-proof comparison scripts that this replaces.
   c. Split by home: the generic harness goes in `framework/migrate`; the per-rule proof runners stay beside their rules in `temporal/migrate`. Move `GuideSourcesTest` (a docs-excerpt check) out of `migrate/test` into `check`.
   Every existing rule proof still passes through the new harness.
4. **Follow the move.** Update Make targets, scala-cli project files, fn-166.13's coverage registry, the worker docs and every doc naming the old paths: `model/README.md`, `model/migrate/README.md`, `AGENTS.md`, `.plans/UMPIRE_MODULES.md` (record fn-114.12/fn-122.8 as superseded).
5. **Proof.** The mini lift is byte-identical except for recorded source-path changes; mini step-table equivalence holds; the moved suites pass at their new homes.
## Acceptance
- `model/framework/{lift,check,migrate}` exist with `framework.*` packages; Temporal parts are under `model/temporal/{lift,check,migrate}`; old `model/irgen`, `model/check` and `model/migrate` directories are gone or hold only what the receipt names.
- The framework builds with no Temporal API jar or `temporal.*` source; a deliberate violation fails the boundary check.
- `prove.sh` proves every existing rule on mini through `framework/check`'s comparison, with feature selection by `FEATURES`; the per-proof comparison scripts it replaces are removed; `GuideSourcesTest` runs from `check`.
- Mini lift byte-identical apart from recorded source paths; mini equivalence passes; docs updated.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
