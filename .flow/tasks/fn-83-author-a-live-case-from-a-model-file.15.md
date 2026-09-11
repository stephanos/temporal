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
- 2026-09-10: the `case` abstraction is under redesign (query sets per purpose, one Case per Query, a separate Temporal binding; see the block reason on .16 and .17). Respell only `enum`, `model`, `property`, `scenario`, `limits` and `query`; leave the `case` command and its fixture unchanged, and drop the `case` block from the acceptance evidence.

## Acceptance
- [ ] Every indented framework word that introduces a value is spelled `word:`; no author-written label remains in `model`, `property` or `scenario`; `query` names no model; `property` and `scenario` name no role for a single-role model; step rows use `before + action → after, outcome: x[, facts: [...]]`; `evidence` lines have no `history`
- [ ] `Model.lean`, `RaceSyntaxTests.lean` and `Tests.lean` use only the respelled forms; derived relation, clause and occurrence keys follow a recorded rule and pass `FiniteCatalog.validKey`
- [ ] A missing key, a key on the wrong line, a query whose property and scenario name different models, and a duplicate derived clause each reject with a located message pinned by `#guard_msgs`
- [ ] The async-Nexus fixture diff is limited to derived Definition IDs and their fingerprints, listed in the receipt; the live async-Nexus test passes
- [ ] `.5`, `.6`, `.8` and the fn-83 API Contracts grammar use the respelled forms
- [ ] `cd model && lake build`, `make lint-model`, `make umpire-check-retired-vocabulary`, `make umpire-check-case-runtime-conformance`, `make umpire-check-live-tests` pass


## Done summary
One rule now governs a Model file: a column-0 word is a declaration kind followed by the author's
name, an indented `word:` is a framework key introducing a value, and everything else is an author
name, a declared member, a number, or an operator.

Derived keys, so nothing an author writes is a label the framework only hands back:
- **relation** `<source>-<action>` (`scheduled-awaitStart`) -- the row's own coordinates, which the
  command already proves unique by rejecting a second row leaving the same state on the same Action;
- **clause** `<kind>-<member>` (`state-succeeded`) -- so a repeated requirement is a repeated key and
  rejects on the line that repeats it;
- **occurrence** the 1-based position in `actions:` -- what distinguishes two occurrences of the
  same Action.
All three pass `FiniteCatalog.validKey` (alphanumerics, `-`, `_`).

Redundancy removed: a `query` names no Model (its Property and its Scenario each name one, and a
disagreement rejects on the Scenario); `property` and `scenario` name no role while a Model declares
exactly one; `actions:` is the exact sequence; a Step row is
`before + action → after, outcome: x[, facts: [...]]` and the record form is gone. The Known Gap
lines .11 added are respelled to keys (`gap:` / `code:` / `subject:` / `detail:`).

`Umpire.Command.Registry` gained a `PropertyEntry`, a Model's `role`, and the Model each Scenario
runs on, which is what lets `query` resolve the Model rather than be told it.

Retirement: no retired arms were added for the pre-respell forms, and fn-82's retired arms
(`initial`, `terminal`, `transitions`, `selected_actions`, `candidate_evaluations`, `behavior`,
`witness`, `all`, `when action`, `resultingState`) went with the shapes that carried them -- every
call site is in this repository and migrated in the same commit.
`make umpire-check-retired-vocabulary` then needed no exemption at all, so both
`model/Umpire/Command/Syntax.lean` and `model/Temporal/Feature/Nexus/Success/Tests.lean` were
removed from `allowedNegativeFixture`, and the gate is green.

Fixture diff, exactly the derived identities:
  contract.correlated.clauses[0].clauseId   ...successfulResult.successOutcome -> ...outcome-completed
  contract.correlated.clauses[1].clauseId   ...successfulResult.successState   -> ...state-succeeded
  contract.correlated.projectionFingerprint moved, because the clause ids did
  provenance.producerData                   moved, for the same reason
Nothing else moved -- not the Program, not the projection rules, not the bounds.

Scope note, per the 2026-09-10 line in the task: the `case` command is deliberately untouched, since
the abstraction it belongs to is under redesign. Its `evidence ... ← history <kind>` lines therefore
still carry `history`, and the acceptance line about them is not claimed.

Deviation on pinning: a missing key and a key on the wrong line are **parse** errors, and
`#guard_msgs` cannot capture those -- the command it wraps never parses, so the `#guard_msgs` block
fails to parse with it. Both messages are located on the offending token
(`unexpected identifier; expected 'states:'`, `unexpected token 'outcomes:'; expected 'actions:'`);
they are recorded verbatim in a comment beside the pins that `#guard_msgs` can hold (the duplicate
requirement and the Model mismatch, both pinned).

`make umpire-check-regression` is exit 0 end to end (571 Lean jobs, 9 passing live identities);
`make lint-model` reports 0 findings outside generated `Temporal/API/Proto.lean`;
`make umpire-check-retired-vocabulary` passes.

Review: SHIP, no introduced findings; the one observation (`let _ := prior`) was taken and the
binding removed.
Pinned reviewer `claude:claude-fable-5-1:high` is account-limited for this session, so the review
ran on `claude:claude-sonnet-4-5:high` -- a same-family fallback, not an equivalent cross-family
review.

stage: impl-review - ran (model: claude-sonnet-4-5, high; fable pinned but account-limited)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fd8420b270, bbb4b1a9e4
- Tests: cd model && mise exec -- lake build, make umpire-gen-case-runtime-conformance (fixture diff listed in the summary), make umpire-check-case-runtime-conformance, make lint-model (0 findings outside generated Temporal/API/Proto.lean), make umpire-check-retired-vocabulary, make umpire-check-live-tests (9 passing identities), make umpire-check-regression (exit 0)
- PRs: