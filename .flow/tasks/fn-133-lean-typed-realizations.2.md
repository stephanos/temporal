---
satisfies: [R1, R2, R12]
---
# fn-133-lean-typed-realizations.2 Kit evidence modules: described status, history evidence, request base

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part A, R1, R2, R12.

- **`describedStatus`.** One declaration of a describe method, its info field, its operation key and a fact→status table. It yields the evidence of each fact and an `await(fact)`. The activity and standalone Nexus realizations use it, and their `status`/`awaitStatus` helpers and per-fact await vals go.
- **History evidence.** One declaration of a workflow's history-event kinds: fact → attributes field, keyed by the field naming the operation (e.g. `scheduledEventId`). It yields the exhaustive kinds and the set the closing read `closes`. The Nexus caller's five `historyKind` calls go.
- **A request base.** A realization declares the fields every call on a role carries (namespace, operation id from `run`) once. `rpc` and `await` apply them, and an explicit assignment overrides them.

Refusal fixtures: a fact listed twice in a table; an await of an unlisted fact; a history entry with no operation key. Lint: an override equal to the base.

## Acceptance
- [ ] No feature file defines `status`/`awaitStatus`/`historyKind`, or assigns `namespace` or the operation id without overriding the base.
- [ ] Each refusal fixture is refused at its line; the lint reports a redundant override.
- [ ] A before/after projection is identical.
- [ ] The spec's Verification gates pass.

## Done summary
Added the kit's evidence modules in `model/temporal/realize/Modules.scala`: a request base, a described status and history evidence. The three realizations now use them. A scratch lift of model/ir shows every `realizations` section equal to the baseline's once positions are stripped.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- **Request base.** `RequestBase(role, "namespace" -> workerNamespace, "activity_id" -> run)` extends `Addressee`, so `rpc(calls, M) { … }` and the kit's `await(evidence, calls)(until) { … }` accept it. The kit's `await` now takes `Addressee` instead of `Role`.
  - The lifter writes the base's fields first, in base order, then the call's own fields.
  - A base field is resolved by protobuf name: the request's own field, or the one message field that holds a field of that name (`workflow_id` resolves to `execution.workflow_id` for the history and describe calls).
  - A call that assigns a base field overrides it. If the override's value equals the base's, the lifter refuses it.
- **Described status.** `DescribedStatus(calls, METHOD, Field(_.getInfo), Field(_.activityId), Field(_.status))(fact -> status, …)` provides:
  - `.evidence`: every listed fact's evidence, in table order;
  - `described(fact)`: one fact's evidence;
  - `described.await(fact)`: a poll named `await-<status>` after the status value's suffix past `_STATUS_` (`ACTIVITY_EXECUTION_STATUS_TIMED_OUT` gives `await-timed-out`);
  - `.table`: a StatusTable.
  - The `activityStatus` and `operationStatus` vals the Describable capabilities read are kept as `described.table`, so the System files are untouched.
- **History evidence.** `HistoryEvidence(key = "scheduled_event_id", factPrefix = "nexusOperation")(HistoryKind(fact, _.attributes.x), …)`.
  - `.evidence` gives one exhaustive kind per entry. A kind is named after its fact without the prefix, counts in `historySource` (= `sourceId("history")`), and its operation path is `attributes<x>.scheduled_event_id`.
  - The nexus caller writes `closes = historyKinds.evidence` and `evidence = Vector(scheduled) ++ historyKinds.evidence :+ pending`.
- **Lifter (Realizations.scala, section "The kit's evidence modules").** It writes each module's declarations directly as the core IR records, by name:
  - `itemsOf` expands a module's `.evidence` into marked entries;
  - `declaration`/`textOfBound` write an entry's Evidence or its id;
  - `commandName`/`commandValue` handle `described.await(fact)`;
  - `rpcValue`/`pollValue` apply a request base.

### Decisions (owner unavailable)
- **Evidence order stays as it was, so the activity lists per-fact `described(fact)` in its evidence.** The lowering keeps the IR's evidence order in the Case program (tools/umpire/lower/realization.go:97). In the activity the status kinds are not contiguous (`statusCancelRequested` sits between paused and completed), so a single `described.evidence` there would reorder evidence and change existing Case bytes. Nexus standalone also uses `described(fact)`, because only `terminated` of its four table entries is evidence.
- **Await ids come from the status value, not the fact.** The status value gives today's command ids for both features. Fact names would not, because nexus standalone's facts are `nexusOperation*`.
- **The request base's operation field is named by its protobuf field name.** No common Scala type spans the request messages. The lifter checks every name against each call's request descriptor at lift time.
- **"Lint: an override equal to the base" is implemented as a lifter refusal at the call's line,** not as an umpire-lint finding.
- `factPrefix` is the declared constant that names history kinds without free strings.

### Line counts
After .2: activity 296, nexus workflow 357, nexus standalone 88. Baseline was 333 / 514 / 101.

### Declared IR delta (batch regeneration)
- Source positions only, in `realizations`. No IDs change, no Cases are added, and no carrier metadata changes.
- The lifter fixture `rejects.txt` gains four lines: ScriptRejects.scala:120, 136, 147 and 155.

### Tests
- `mise exec -- scala-cli test model/irgen`: 94 passed, plus Overlap, which first failed on copied fixture lines and passed after the fixture was rewritten with named arguments. The test was re-run alone with `--test-only umpire.irgen.Overlap`.
- New refusals:
  - `baseOverride` (ScriptRejects.scala:120);
  - `awaitUnlisted` (:136);
  - `describedFactTwice` (:147);
  - `historyKeyless` (:155).
  - Each was checked by lifting the root directly.
- `mise exec -- scala-cli test model/project.scala model/umpire model/temporal`: 27 passed.
- `--check-syntax` and `--check-comments`: clean.
- A scratch lift of model IR: every `realizations` section is equal with positions stripped.

### For later tasks
- The module markers in the lifter work as follows: a marked entry is a Bound whose env holds `Symbol.noSymbol -> module`, and `entryEvidence` writes it. Module class methods are never followed: they are vocabulary members, and their bodies are runtime stand-ins.
- `evidenceId("scheduled")`, `sourceId("scheduled")`, `sourceId("describe")` and `answeredAs("statusScheduledAgain", …)` remain for fn-133.3.
- `HistoryKind` and `DescribedStatus` facts are `Fact = AnyRef`. fn-133.6 should type them against the machine.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 4b7ef01a84
- Tests: mise exec -- scala-cli test model/irgen, mise exec -- scala-cli test model/irgen --test-only umpire.irgen.Overlap, mise exec -- scala-cli test model/project.scala model/umpire model/temporal, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, scratch lift --ir of model IR, realizations equal with positions stripped
- PRs: