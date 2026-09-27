---
satisfies: [R2]
---
# fn-93-simplify-the-lean-model.2 One axiom checker replaces every #print axioms pin (E2)

## Description
Lane E2. 81 `#print axioms` pins exist; 30 have no `#guard_msgs`, so they assert nothing. Add one checker command and convert every pin, including pins in modules lane B later deletes (a declined decision keeps them).

**Size:** M
**Files:** `model/Umpire/Shared/Test.lean` (or a new `model/Umpire/Shared/AxiomCheck.lean` it re-exports), `model/Umpire/Shared/Tests/AxiomCheck.lean` (new self-tests), the ~23 test files holding pins (list below), `model/UmpireTests.lean` (wire self-tests), `.plans/LEAN_GUIDELINES.md` trust-audit paragraph, `model/README.md` sentence on `#print axioms`
**Touches:** [model/Umpire/Shared/**, model/Umpire/**/Tests/**, model/Umpire/**/*Tests.lean, model/Temporal/**/Tests/**, model/Temporal/**/*Tests.lean, model/Temporal/Feature/Nexus/Success/Tests.lean, model/UmpireTests.lean, .plans/LEAN_GUIDELINES.md, model/README.md]
**Depends on other specs:** fn-88.5 and fn-92.3 add pins under `Search/Tests/*` and `Command/ComposeProofs`; convert what is there at start.

### Approach
- Command shape per spec §API Contracts (`assert_axioms [decls] allowing [axioms]`). Build on `Lean.collectAxioms` (see its use in the `machine` command's sorry guard at `model/Umpire/Command/Syntax.lean:2418,2462`); resolve names with `realizeGlobalConstNoOverloadWithInfo` so a missing name is a located error.
- Fail on `sorryAx`, on any axiom outside the allowlist, and on a missing declaration. Allowlist names are exact (`propext`, `Classical.choice`, `Quot.sound`, and `Lean.ofReduceBool`/`Lean.trustCompiler` only where `native_decide` is already used).
- Self-tests under `#guard_msgs` (use `drop warning` or pin the "declaration uses 'sorry'" line) for: seeded sorry, extra axiom, missing name.
- Convert each pin to one entry with the exact axiom set it prints today. The 8 duplicated targets (`nexusProduct`, `Execution.closed_property`, `Projection.Correlated.Monitor.admitMany_append`, `evaluateProperty`, `evaluateProperty_agrees`, `evaluatePropertyPredicate_agrees`, `Lowered.window_property`, `Lowered.evidence_validation`) keep one entry each; the per-machine "no sorryAx" guarantee stays one entry per machine.
- The checker module must stay inside `testSupportNamespaces` (`model/ModelLint/ImportGraph.lean:170`); importing `Lean.Elab` there is allowed for test support only.

### Investigation targets
**Required:**
- `model/Umpire/Shared/Test.lean` — current test-support home (23 lines)
- `model/Umpire/Command/Syntax.lean:2405-2470` — existing `collectAxioms` guard
- Pin sites with the most bare pins: `model/Umpire/Evidence/Tests/Compilation.lean:568-571`, `model/Umpire/Model/Tests/FiniteMachine.lean`, `model/Temporal/Feature/Nexus/Success/Tests.lean`, `model/Umpire/Property/Tests/Endpoints.lean`, `model/Umpire/Scenario/Tests/Authoring.lean`
**Optional:**
- `.plans/LEAN_GUIDELINES.md:164-173` — trust-audit prose to update

### Quick commands
```sh
grep -rn '#print axioms' model --include='*.lean' | grep -v .lake | wc -l   # 0 after, except comments
cd model && lake build UmpireTests TemporalModelTests
```

## Acceptance
- [ ] No `#print axioms` pin remains (comments aside); every former pin is one checker entry with its exact prior axiom set
- [ ] Self-tests pin the three failure modes; seeding `sorry` into a checked declaration makes `lake build` fail (receipt shows it)
- [ ] No declaration's axiom inventory widened; LEAN_GUIDELINES trust-audit text and README sentence updated
- [ ] `lake build` of all roots and `LEAN_NUM_THREADS=1 make lint-model` green


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
