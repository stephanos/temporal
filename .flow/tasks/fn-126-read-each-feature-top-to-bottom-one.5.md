---
satisfies: [R5, R9, R13, R15, R16, R17]
---
# fn-126-read-each-feature-top-to-bottom-one.5 Write the remaining Models as machine objects and retire the builder forms

## Description
Convert every remaining Model to the object forms of task 4 (R15-R17), inline its single-use Scenarios (R13), retire the builder forms, and bring the docs to the final declaration shape (R9).

**Owner decisions 11-17 (spec, "Later owner decisions").** Convert the remaining Models with `init`, `states`, `adopts`, `refinement` and `IrFiles` as task 4 built them, and rename the Nexus caller's `object Control` to `ForgedCaller` (pin unchanged).

**Cross-spec entry gate:**
- Task 4 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** L
**Files:**
- `model/temporal/features/{nexuscaller,nexuscaller/closepolicy,nexusoperation}/**`, `model/temporal/shared/{taskqueue,worker}/**`;
- `model/umpire` and `model/irgen` (the builder forms and their reading removed) and every lifter fixture restated in the object forms;
- `tools/umpire/internal/golden/**`;
- `model/README.md`, `model/SEMANTICS.md`, `.plans/UMPIRE_MODULES.md`, `.plans/UMPIRE4_VISION.md`, `AGENTS.md`, `.plans/DSL_OPERATORS.md`, `.plans/QUINT_MODULE_LAYOUT.md`.

**Touches:** [model/temporal/**, model/umpire/**, model/irgen/**, model/check/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**, tests/testcore/testpilot/testdata/generated/**, model/README.md, model/SEMANTICS.md, .plans/**, AGENTS.md]

### Approach
- **Nexus caller and Nexus operation:** as the activity.
- **Close policy:** `RejectAfterClose` holds the family's effects (`deliver(policy, redelivery, s, r)`, `reset`, …). Each of the nine designs becomes a `Derived` object whose derivation names its policy, reset and channel, with its own `queries`, so `designQueries` splits across the objects. Per-class rules where they read better (`handler.complete(…)`).
- **Task queue and worker:** machine objects (`DispatchQueue`, `MatchingQueue`, …, `Polling`), with `worker.workerStop`/`workerResume`/`serve` bound by rules.
- **Retire the builder forms.** Remove `machine[S, O, F] { … }`, `steps`, `starts`/`ends` and the `compose(…)` value form from `model/umpire` and their reading from the lifter. Restate every passing lifter fixture in the object forms, and confirm by its expected IR that nothing moved apart from recorded deltas.
- **Docs (R9):**
  - the README's "Writing a Model" shows the object forms, rules and effects, the sections and R2's order;
  - SEMANTICS states the rule lowering and disjointness;
  - `.plans/DSL_OPERATORS.md` records the owner's reversal of its rejected guard helper: `when` and `in` are rule headings, never guards inside a step;
  - QUINT_MODULE_LAYOUT and DSL_SIMPLIFICATION mark what landed.

### Investigation targets
**Required:**
- `model/temporal/features/nexuscaller/closepolicy/Model.scala:515-580` (derivations)
- task 4's done summary (the wiring and the recorded deltas)
- `model/irgen/testdata` (fixtures to restate)
**Optional:**
- `.plans/UMPIRE_MODULES.md:30,321`

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 ./tools/umpire/...
grep -rn "machine\[\|steps(\|starts(\|ends(" model/temporal model/irgen/testdata --include=*.scala
```

### Execution constraints
- R5 holds. A table, answer or Contract change stops the task.

## Acceptance
- [ ] Every Model is written in the object forms with rules, effects and sections. The close policy's designs are `Derived` objects with their own Queries.
- [ ] The builder forms are gone from `model/umpire` and the lifter. Every lifter fixture uses the object forms, and the grep in Quick commands finds none.
- [ ] Tables, IDs, names, answers, lint findings and Contracts are unchanged apart from recorded R5 deltas.
- [ ] The R9 docs describe the final declaration shape. `.plans/DSL_OPERATORS.md` records the guard-helper reversal.
- [ ] All gates of the spec's Verification pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
