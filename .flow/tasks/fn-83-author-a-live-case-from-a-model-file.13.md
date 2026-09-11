---
satisfies: [R13]
---
# fn-83-author-a-live-case-from-a-model-file.13 Optional Facts: stop authoring Facts that restate a state

## Description
Make Facts optional in the `model` command and stop authoring Facts that only restate a state (R13). In the Nexus success Model every step records one Fact named after the state it reaches (`facts := [started]` on the row into `started`, `facts := [succeeded]` into `succeeded`), so `require successFact: fact succeeded` restates `require successState: state succeeded`. It adds no runtime check either: `Umpire.Case.Producer` confirms a whole Step (state, outcome, facts) from the one history event mapped to its Action, so both clauses are answered by the same `nexusOperationCompleted` event.

A Fact carries information a state cannot when it distinguishes two paths into the same state, marks something that happened without a state change, or records several claims in one step. The command keeps Facts for those Models and stops requiring them from Models that have none.

**Size:** S
**Files:** the Umpire command module from .10 (the `model` grammar: `facts <Type>` optional, per-row `facts := […]` optional; the `property` command's `fact` clause), `model/Temporal/Feature/Nexus/Success/Model.lean` (no Fact domain, no `successFact` clause, rows without `facts`), `RaceSyntaxTests.lean` stays with Facts (its `lose` row records none, and its Facts are not state mirrors), the command tests, the regenerated async-Nexus fixture, `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md` and `.6.md` (their Models drop a Fact that only mirrors a state), the R3 parenthetical in the fn-83 spec if .5's Model changes
**Touches:** [model/Umpire/**, model/Temporal/Feature/Nexus/Success/**, tests/testcore/testpilot/testdata/async-nexus-case.json, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md, .flow/specs/fn-83-author-a-live-case-from-a-model-file.md]

### Approach
- Grammar: `facts <Type>` may be omitted from the `model` header; a row may omit `, facts := […]`, meaning `[]`. A row that names a Fact in a Model with no Fact domain rejects in place, listing that the Model declares no Facts. A `require …: fact …` clause on such a Model rejects the same way at the clause.
- An omitted domain elaborates to an empty Fact catalog. Table validation already admits an empty `facts` catalog with rows that record none (`Umpire/Model/Tests/FiniteTable.lean:106`). How the empty type is supplied (a generated model-scoped empty inductive, or a shared empty type with the required `BEq`, `DecidableEq`, `Repr` instances) is a task decision; it must not be a type the author declares.
- Check the empty domain end to end, not only at the table: vocabulary construction, Property and Scenario admission, Search, the Producer's projection (`stepOf` with `facts := []`), coverage, and Case compilation. Any stage that assumes a nonempty Fact list is a finding to fix or record.
- Nexus success Model: remove `enum Fact`, the `facts` header line, both row `facts` fields, and `require successFact`. The async-Nexus fixture changes: the Contract carries two correlated clauses instead of three, and the model's canonical description and fingerprint move because its Fact domain is empty. Regenerate through `make umpire-gen-case-runtime-conformance` and list the diff in the receipt; the live async-Nexus test must still pass.
- `.5` (worker outage: one Fact `completed` beside state `completed`) and `.6` (sync Nexus: Fact `resultReturned` beside state `succeeded`): update their task wording to omit a Fact that only mirrors a state, and adjust the fn-83 R3 parenthetical to match. Keep a Fact only where the task can name what it claims beyond the state.
- `RaceSyntaxTests.lean` keeps its Facts so the declared-Fact path, a row with `facts := []`, and a `fact` clause stay pinned.

### Investigation targets
**Required:**
- the Umpire command module from .10 — the `model` grammar (`successStep`), `domainConstructors` for the fact domain, the `property` command's `fact` arm
- `model/Umpire/Model/Table.lean` — `validate`, `validateCatalog`, `modelValue` for facts
- `model/Umpire/Case/Producer.lean` — `patternHolds` (`.observation`), `stepOf`, the projection declaration
- `model/Umpire/Model/Canonical.lean` — how the Fact domain enters the canonical description
- `model/Temporal/Feature/Nexus/Success/Tests.lean` — guards that count three Property clauses (`checked.property.clauses.length == 3`) and read `factAt`

### Key context
- Depends on .12 (same grammar, same Model file). .5, .6 and .8 depend on this task so their Models and the tutorial author Facts only where they carry a claim.

## Acceptance
- [ ] A `model` block may omit `facts <Type>` and a step row may omit `facts := […]`; the author declares no Fact type for a Model without Facts
- [ ] A Fact named in a row, or a `require …: fact …` clause, on a Model without a Fact domain rejects with a located message pinned by `#guard_msgs`
- [ ] The Nexus success Model declares no Fact domain and no `successFact` clause; the query still finds its witness; the regenerated async-Nexus fixture diff (two correlated clauses, moved canonical description) is listed in the receipt; the live async-Nexus test passes
- [ ] `RaceSyntaxTests.lean` still elaborates with declared Facts, including its `facts := []` row and its `fact` clause
- [ ] The empty Fact domain passes vocabulary, admission, Search, projection, coverage and Case compilation, or each stage that cannot is recorded
- [ ] `.5` and `.6` task wording and the fn-83 R3 parenthetical no longer author a Fact that only mirrors a state
- [ ] `cd model && lake build`, `make lint-model`, `make umpire-check-goldens`, `make umpire-check-case-runtime-conformance`, `make umpire-check-live-tests` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
