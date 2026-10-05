---
satisfies: [R5, R6, R14]
---
# fn-126-read-each-feature-top-to-bottom-one.3 Group actions by actor in transparent section objects

## Description
Group every feature's actions by actor (R14) and keep every Definition ID through transparent section objects (R6). This is the first lifter change of the spec.

**Cross-spec entry gate:**
- Task 2 is done.
- Never alongside fn-124.8.
- Before fn-124.7.

**Size:** M
**Files:**
- `model/umpire` (the markers `Section` and `Actor`; `Party` opened so an object can be one);
- `model/irgen/Context.scala` (`definitionId`, `pinOf`) and the party naming, with fixtures;
- every feature file's signature, its Scenarios and realizations (call sites);
- `tools/umpire/internal/golden/config.json`.

**Touches:** [model/umpire/**, model/irgen/**, model/temporal/**, tools/umpire/internal/golden/**, model/ir/**, model/cases/**]

### Approach
- `Section` is a marker trait. `Actor` is a `Party` and a `Section`, named by its object's name. Today's `val caller = Party()` becomes `object caller extends Actor`, and its members are the actions it takes. The party name is unchanged.
- Lifter rule: when computing a Definition ID, a section object is skipped. A member takes the ID it would take as a direct member of the section's enclosing owner, or of the file's package object at the top level, so the file's pin applies. Refuse, at their line:
  - a section inside a section;
  - a section anywhere other than the top level of a Model file or directly in a machine, composition or derived-machine object (task 4 adds the last three);
  - two members that would share an ID (`idTakenBy`, as today).
- Groups:
  - parties `caller`, `handler`, `network` and the shared `worker`;
  - sections `timers`, `deadline`, `history` (internal steps), `queue`, `faults`;
  - the standalone activity's `object worker extends Section`, whose actions are taken by the shared party, imported as `process` (R14).

  Actions keep their val names (`worker.attemptStart`), so every ID stays. Task 6 renames them.
- Move the deadline inputs out of `object Inputs`. Keep only inputs named like an action of their actor object.
- Rewrite call sites in Scenarios, Properties, capabilities, compositions and realizations. Record function-symbol and position deltas.

### Investigation targets
**Required:**
- `model/irgen/Context.scala:225-335`
- `model/umpire/Action.scala` (Party, action naming), `model/irgen/Declarations.scala:20-40` (action lifting)
- `.plans/DSL_SIMPLIFICATION.md` section 4b (the per-feature table)
**Optional:**
- `model/temporal/shared/worker/Model.scala:44-60` (why the prefix stayed)

### Quick commands
```bash
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks && make lint-model
go test -count=1 -tags test_dep -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower
```

### Execution constraints
- Zero Definition-ID, party-name, action-class or Case-content change beyond positions and function symbols.

## Acceptance
- [ ] Every action of every feature is declared in an actor or section object, and every call site shows its actor (`caller.start()`, `worker.attemptStart`).
- [ ] `Section` and `Actor` exist in `model/umpire`. Section objects are transparent to Definition IDs, with a lifting fixture and refusal fixtures for a nested section, a misplaced section and a duplicate ID.
- [ ] Definition IDs, party names, action classes, tables, answers and Case bytes are unchanged apart from recorded positions and function symbols.
- [ ] `object Inputs` keeps only action-named inputs.
- [ ] All gates of the spec's Verification pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
