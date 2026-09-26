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
- [x] `lake exe umpire-lint` passes on the tree; a planted direct import from a `Temporal.Feature` production module to `Umpire.Model` exits 1 with the exact diagnostic, asserted by the Makefile; `Temporal.Case` realizations and the Implementation Link do not trigger it
- [x] `make lint-model` green (LEAN_NUM_THREADS=1)


## Done summary

Done 2026-09-20; self-review. Commit e4c3db4.

### The rule

`ModelLint/ImportGraph.lean`: `Rule.authoringPathIsolation`, label `authoring-path-isolation`, a
direct-import rule. `Policy` gains `authoringPathRoots` (`Temporal.Feature`, `Umpire.Examples`)
and `authoringPathExceptions` (`Temporal.Case`, `Temporal.System.Nexus.ImplementationLink`; neither
is under a root today, and naming them keeps a move under a root from silently putting one inside
the rule). `Policy.isAuthoringPathModule` is production (`isProductionModule`, so the test-consumer
and test-support predicates exclude what they already exclude), under a root and not an exception;
`checkAuthoringPath` reports every direct import of an `authoringOwners` module (`Umpire.Model`,
`Umpire.Property`, `Umpire.Scenario`, `Umpire.Query`, `Umpire.Operation`, `Umpire.Case`) by such a
module, sorted by module then import, and `check` returns those before the reachability results, so
the lint executable, the synthetic suite and the controlled-violation mode all see the rule through
the one entry point. `Violation.render` gives the rule its own form,
`[model-import-graph/authoring-path-isolation] forbidden direct import: <module> -> <owner>`, and
keeps the path form for every reachability rule. The hand-written inventory reconciliation stays as
it was (its roots also cover `Temporal.Testpilot`, whose `CaseSupport` and `Conformance` remain
ledger-listed); under the two authoring-path roots the rule makes a direct owner import a violation
whether or not the ledger lists it.

### The proof it fails closed

`ModelLint/ImportGraphTests.lean`: `testAuthoringPathIsolation` plants a `Temporal.Feature`
module importing `Umpire.Model.Table` and an `Umpire.Examples` module importing `Umpire.Query`
beside `Umpire.Command` (each one violation with its path), checks that a `Tests` module, a
`Temporal.Case` realization importing `Umpire.Case.Producer`, the Implementation Link importing
`Umpire.Property.Elab` and a module importing only `Umpire.Command` are outside the rule, pins the
order of two owner imports from one module and the rendered diagnostics.
`umpire-lint-tests --controlled-authoring-violation` prints the planted
`Temporal.Feature.Planted -> Umpire.Model` diagnostic and exits 1; `Makefile` `lint-model` asserts
that stderr byte for byte beside the existing `shared-independence` assertion. The tree is clean
under the rule: after .7 no production module under either root imports an owner directly
(`Temporal/Feature/Workflow/Start/Tests.lean` imports `Umpire.Operation.Parameterized` and is a
test module).

### Gates

`lake build umpire-lint-tests umpire-lint`; `lake exe umpire-lint-tests` passes; both controlled
violations exit 1 with their exact diagnostics; `LEAN_NUM_THREADS=1 make lint-model` at the .1
baseline (the import graph passes; the declaration linters' 40 warnings and the two generated
`Temporal/API/Proto.lean` findings are the baseline).

## Evidence
- Commits: e4c3db4
- Tests: `cd model && lake build umpire-lint-tests umpire-lint && lake exe umpire-lint-tests && lake exe umpire-lint-tests --controlled-authoring-violation`; `LEAN_NUM_THREADS=1 make lint-model`
- PRs:
