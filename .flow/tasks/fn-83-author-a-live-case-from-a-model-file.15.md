---
satisfies: [R15]
---
# fn-83-author-a-live-case-from-a-model-file.15 Respell the command surface for minimal syntax

## Description
Respell the Model commands for minimal syntax, so a reader can tell framework words from author names without editor highlighting (R15). One rule governs the whole file:

- a **column-0 word** is a declaration kind (`enum`, `model`, `property`, `scenario`, `limits`, `query`, `case`) followed by the author's name;
- an **indented `word:`** is always a framework key, and what follows it is a value;
- **everything else** is an author name, a declared member, a number, or an operator (`+`, `→`, `←`, `[…]`).

Today the same `name:` shape marks author labels (`start: scheduled + awaitStart`, `require successState: …`, `[start: awaitStart, …]`) and bare framework words sit beside author words on one line (`find successfulResult`, `on lifecycle`). Both go.

### Normative sketch (the Nexus success Model after .12 and .13)

```lean
model lifecycle
  role: operation
  states: State
  actions: Action
  outcomes: Outcome
  starts: [scheduled]
  ends: [succeeded]
  steps:
    scheduled + awaitStart → started, outcome: acknowledged
    started + awaitSuccess → succeeded, outcome: completed

property successfulResult
  model: lifecycle
  when: awaitSuccess
  require:
    state: succeeded
    outcome: completed

scenario successfulCompletion
  model: lifecycle
  starts: scheduled
  actions: [awaitStart, awaitSuccess]

limits shortTrace
  steps: 2
  actions: 2
  search: 16

query completion
  find: successfulResult
  in: successfulCompletion
  limits: shortTrace

case asyncNexusSuccess
  fixture: "async-nexus"
  realizes: completion
  template: nexusOperation
  service: "umpire.case.service"
  operation: "complete"
  responds: async
  evidence:
    awaitStart ← nexusOperationStarted
    awaitSuccess ← nexusOperationCompleted
```

(.16 then removes `fixture:`, .17 removes `service:` / `operation:`.)

**Size:** M
**Files:** the Umpire command module from .10 and the Temporal `case` command, `model/Temporal/Feature/Nexus/Success/Model.lean`, `RaceSyntaxTests.lean`, `Tests.lean` (every `#guard_msgs` block and every command occurrence), the regenerated async-Nexus fixture, `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md`, `.6.md`, `.8.md` and the fn-83 spec's API Contracts grammar (respelled forms, including the `fault` line sketch)
**Touches:** [model/Umpire/**, model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/**, tests/testcore/testpilot/testdata/async-nexus-case.json, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.8.md, .flow/specs/fn-83-author-a-live-case-from-a-model-file.md]

### Approach
- **Keys.** Every value-introducing framework word takes `:` (`role:`, `states:`, `steps:`, `model:`, `when:`, `require:`, `starts:`, `actions:`, `find:` / `verify:`, `in:`, `limits:`, `realizes:`, `template:`, `evidence:`, inside a row `outcome:` and `facts:`), including the Known Gap lines .11 adds to `query`. Keys use column checks (`colGt` / `checkColGe`) so a missing or misplaced key is reported on its own line rather than as a parse error at the next declaration.
- **No author labels.** Derive every label the syntax asks for today:
  - step (relation) key from the row's source state and Action, which the `model` command already proves unique;
  - clause key from the requirement's kind and member (a duplicate requirement rejects in place);
  - occurrence key from the Action's position in `actions:`.
  Pick spellings that pass `FiniteCatalog.validKey` and Definition ID rules (for example `scheduled-awaitStart`, `state-succeeded`, `1`) and record them.
- **No redundancy.** `query` names no model: it resolves the model from its property and scenario, and a mismatch between them rejects in place. `property` and `scenario` name no role while a model declares exactly one (the model keeps `role:`). `scenario` `actions: [...]` is the exact sequence (the only form the command offers; a looser form later takes its own key). A step row is `before + action → after`, then `, outcome: x` and optional `, facts: [...]`; the `{ state := …, outcome := …, facts := [...] }` record form goes. `evidence` lines drop `history` (the only source kind; a second source kind later adds a prefix).
- **No retired arms for these forms.** Every call site is in-repo and migrates in the same commit; do not add located "retired spelling" arms for the pre-respell forms. Whether the fn-82 retired arms stay follows `make umpire-check-retired-vocabulary`; record the outcome.
- **Bytes.** Derived keys change relation, clause and occurrence Definition IDs. Clause IDs appear in the async-Nexus Contract, so the fixture changes; regenerate through `make umpire-gen-case-runtime-conformance`, list the diff in the receipt, and keep the live async-Nexus test green. Nothing else in the Case (Program, projection, bounds) may move.
- Update `.5` (fault line sketch in the respelled style, for example `fault: workerStop before start`), `.6`, `.8` and the fn-83 API Contracts grammar block to the respelled forms.

### Investigation targets
**Required:**
- the Umpire command module from .10 — the five command grammars, `successStep`, `successRequire`, `successOccurrence`, `resolveMember`
- the Temporal `case` command — `caseTemplate`, `caseEvidence`
- `model/Umpire/Model/Table.lean` — `FiniteCatalog.validKey`, row key uniqueness
- `model/Temporal/Feature/Nexus/Success/Tests.lean` — every command occurrence and `#guard_msgs` pin
- `.plans/LEAN_GUIDELINES.md` — syntax and whitespace-sensitivity conventions

### Key context
- Depends on .11 (Known Gap lines on `query`) and .13 (optional Facts) so the grammar is respelled once, after it stops growing. .16, .17 and .14 follow it.

## Acceptance
- [ ] Every indented framework word that introduces a value is spelled `word:`; no author-written label remains in `model`, `property` or `scenario`; `query` names no model; `property` and `scenario` name no role for a single-role model; step rows use `before + action → after, outcome: x[, facts: [...]]`; `evidence` lines have no `history`
- [ ] `Model.lean`, `RaceSyntaxTests.lean` and `Tests.lean` use only the respelled forms; derived relation, clause and occurrence keys follow a recorded rule and pass `FiniteCatalog.validKey`
- [ ] A missing key, a key on the wrong line, a query whose property and scenario name different models, and a duplicate derived clause each reject with a located message pinned by `#guard_msgs`
- [ ] The async-Nexus fixture diff is limited to derived Definition IDs and their fingerprints, listed in the receipt; the live async-Nexus test passes
- [ ] `.5`, `.6`, `.8` and the fn-83 API Contracts grammar use the respelled forms
- [ ] `cd model && lake build`, `make lint-model`, `make umpire-check-retired-vocabulary`, `make umpire-check-case-runtime-conformance`, `make umpire-check-live-tests` pass


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
