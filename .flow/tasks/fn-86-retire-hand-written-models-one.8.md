---
satisfies: [R7]
---
# fn-86-retire-hand-written-models-one.8 The authoring-path-isolation lint rule

## Description
Add the `lint-model` rule (R7): a production module under `Temporal.Feature` or `Umpire.Examples` may not import `Umpire.Model`, `Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, `Umpire.Operation` or `Umpire.Case` directly; only `Umpire.Command` may. It is a direct-import rule (every module reaches those owners transitively through `Umpire.Command`), reported in the checker's one-line form, with realizations in `Temporal.Case` and `Temporal.System.Nexus` outside it; a planted violation proves it fails closed.

**Size:** S
**Files:** `model/ModelLint/ImportGraph.lean` (`Rule.authoringPathIsolation`, its label, the direct-import check beside `forbiddenRule?`, the scope predicate reusing `isProductionModule`), `model/ModelLint/ImportGraphTests.lean` (planted violation and the two carve-outs), `Makefile:870-878` (a second controlled-violation assertion with the exact stderr), `.plans/UMPIRE4_SPEC.md` (task .9 drafts the rule text; this task only names the label)
**Touches:** [model/ModelLint/**, Makefile]

### Approach
- Diagnostic: `[model-import-graph/authoring-path-isolation] forbidden direct import: <module> -> <Umpire owner>`; the Makefile asserts it for the planted case the way it asserts `shared-independence`.
- Scope: `Policy.isProductionModule` and the three roots; `testConsumerModules` (which lists `Temporal.Tool.Goldens`) and test-support namespaces stay excluded; `Temporal.Case.*` and `Temporal.System.Nexus.ImplementationLink` are named carve-outs in the policy.
- Land after tasks .6 and .7 so the tree is clean under the rule the day it is added (the hand-written inventory rows are all migrated or deleted by then).
- Adjusted 2026-09-19 after fn-85 .7 landed: the carve-out for realizations is the whole
  `Temporal.Case.*` namespace as written, which today holds `Temporal.Case.Realization.Nexus`
  (imports `Umpire.Case.Producer`), `Temporal.Case.Syntax` (the set-realizing `case` block, imports
  the realization and `Umpire.Command`), `Temporal.Case.Registry`, `Schema`, `EventKind` and
  `Conventions`; the `Temporal.Feature.Nexus.Tests.*` and `Success.Tests.*` modules fall under the
  existing test-consumer predicate (`nameHasComponent name "Tests"`, `ImportGraph.lean:209`) and
  need no carve-out. A `Temporal.Feature` production module imports `Umpire.Command` for every
  command including `set`, and reads `Umpire.Case.Producer` types only through the Temporal
  `case` block, so the rule holds on the tree fn-85 leaves without exceptions beyond the two named.

### Investigation targets
**Required:**
- `model/ModelLint/ImportGraph.lean:43-65,87-166,204-215,232-293` — rules, policy (`implementationLinkConsumers` at `:120`, `testConsumerModules` at `:126`), labels (`:152-166`), predicates, `forbiddenRule?` (`:232`), `check` (`:281`)
- `model/ModelLint/ImportGraphTests.lean:556-627` — `controlledViolations`, `runControlledViolation`, `main`
- `Makefile:875-885` — the lint-model target and its controlled-violation assertion

**Optional:**
- `.plans/UMPIRE4_SPEC.md:197-198` MOD-11

### Key context
- The rule is enforced under MOD-11; task .9 adds it to MOD-11's list.

## Acceptance
- [ ] `lake exe umpire-lint` passes on the tree; a planted direct import from a `Temporal.Feature` production module to `Umpire.Model` exits 1 with the exact diagnostic, asserted by the Makefile; `Temporal.Case` realizations and the Implementation Link do not trigger it
- [ ] `make lint-model` green (LEAN_NUM_THREADS=1)


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
