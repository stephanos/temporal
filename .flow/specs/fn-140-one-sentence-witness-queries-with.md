# One-sentence witness Queries with explicit live expectations

> HTML render lens (local): open `.flow/artifacts/fn-140-one-sentence-witness-queries-with/spec.html` — regenerable, markdown is the record. <!-- flow-next:artifact-link -->

## Conversation Evidence

> user (turn 1, part 1): "look at temporal/model/temporal/features/activity/standalone/system/"
> user (turn 1, part 2): "on a high-level, abstract level"
> user (turn 1, part 3): "can we take a step back and look at the clarity, readability, transparency of the model DSL"
> user (turn 1, part 4): "maybe let's zoom in on the pieces that are unique to it compared to temporal/model/temporal/features/activity/standalone/product/Product.scala"
> user (turn 2): "pick the highest value yield item and let's discuss it"
> user (turn 3): "yes, write flow next spec to captire this"
> user (turn 4, selected): "Rename once"
> user (turn 4, selected): "Delete hand-written"

## Goal & Context
<!-- scope: business -->
<!-- Goal & Context: 10% [user], 75% [paraphrase], 15% [inferred] -->

The owner asked for a review of the "clarity, readability, transparency of the model DSL" in the standalone activity's System level, and then for "the highest value yield item". That item is the way a Model author states "this path ends like this, and a real server should show it".

Today that statement takes three declarations in two sections. The standalone activity's System machine has nine Properties, eight Scenarios and nine Queries, and only one Property and one Scenario there serve two of those Queries. A reader meets `completes`, `completed` and `completion` and has to work out that they are one claim. Most of those Properties restate the effect of the action they name.

The same sentence, `query find|verify <Property> in <Scenario> limits <Limits>`, also covers three different acts. A `verify` proves an invariant on every trace. A `find` without an expectation exhibits a path in the model. A `find` with `.expect(…)` drives a real server down the path and judges the Run. Whether a Query generates a live Case depends only on whether `.expect` is present, and nothing at the Query says so. The generated Cases of the standalone activity are exactly the Queries that carry an expectation.

An expectation states what the judge concludes, never why. Thirteen expectations across the Models are `inconclusive`. Each records a known gap as a golden. The explanation, where one exists, sits in a comment far from the Query, and for the activity's retry it is written nowhere.

This spec gives the path-and-outcome claim one sentence of its own, makes the live switch a word the reader sees, and puts each non-satisfied expectation's reason on the expectation. The invariant form keeps the Property, Scenario and Query triple, because there a few Properties run over many Scenarios and several designs.

## Architecture & Data Models
<!-- scope: technical -->

**A witness is one declaration.** [paraphrase] A witness names a path, what its last step records, and optionally what a live Run of it is expected to show. The author writes the path's classes in order inside the declaration. No separate Property or Scenario is written for it.

**It is surface over the existing IR.** [paraphrase] The IR generator lifts a witness to the declarations the triple form produces today: a pinned Scenario of the listed classes, a same-step Property about the path's last class, and a `find` Query over the two. [paraphrase] The generated Property and Scenario take the Query's name, which the owner accepted as a one-time rename, so one name identifies the claim in the source and in the IR. [paraphrase] The IR gains no new kind of declaration, and the Go reader, the checker and the lowering read a lifted witness as they read a hand-written triple.

**The claim is a recorded fact, with an optional state condition.** [paraphrase] `.records(<fact>)` says the last step records the fact. [inferred] `.ends(<state predicate>)` adds a condition on the state after the last step, for a claim the fact alone does not carry, such as the attempt count a retried completion ends with. The generated Property holds of a step when the step records the fact and the predicate, where written, holds of its state.

**Limits follow the path.** [paraphrase] A witness's depth bound and schedule length are the number of classes it lists. [inferred] The search bound is one framework default for every derived Limits. `.limits(<Limits>)` replaces the derived value for a witness that needs another.

**`live` is the switch.** [paraphrase] `.live(<expectation>)` marks a Query as one that generates a live Case and states what its Run is expected to show. A Query without it is checked in the model and generates no Case. [inferred] Both forms use the same word. The triple form's `.expect(…)` becomes `.live(…)`, so `.expect` no longer exists on a Query.

**A reason travels with the expectation.** [paraphrase] An expectation whose Property outcome or Contract verdict is not satisfied states why, as text on the expectation value. [inferred] The same value is what a witness's `.live`, a triple-form Query's `.live` and a capability's expectation binding take, so the rule holds in all three places without a new parameter on any of them. A monitor expectation inside a full Run expectation follows the same rule. [inferred] The reason stays in the source. No IR field, Case or assessment carries or reads it, so adding reasons changes no generated byte.

**The triple form stays for invariants.** [paraphrase] `query verify` and its Properties and Scenarios are unchanged. A pinned `find` stays writable as a triple where another declaration also reads its Property or its Scenario.

## API Contracts
<!-- scope: technical -->

- [paraphrase] **Witness**, named after its `val`: `witness(<class>, …).records(<fact>)`, followed by any of `.ends(<state predicate>)`, `.limits(<Limits>)`, `.explore(<space>)` and `.live(<expectation>)`.
- [inferred] **Named witness**, where no `val` names it, such as an item of a list a function builds for several designs: `witness("<name>")(<class>, …)`, then the same members.
- [inferred] **On another machine or a composition**: `m.witness(…)` and `c.witness(…)`, as `m.scenario` and `c.scenario` are written today. A composition's witness lists composed steps in the forms a composition's Scenario takes, and `.records(<member selector>, <fact>)` names a member's fact.
- [paraphrase] **Core form of a witness named `n`**: Scenario `n` pinned to the listed classes; Property `n`, a same-step Property about the last listed class that holds when the step records the fact and the `.ends` predicate holds; Query `n`, a `find` of Property `n` in Scenario `n` under the derived or given Limits, with the expectation `.live` states.
- [paraphrase] **Expectations**: `satisfied` takes no reason. `inconclusive(<reason id>, because = "<text>")` and every other constructor of a non-satisfied outcome take a non-empty reason text.
- [inferred] **Removed surface**: `.expect(…)` on a Query.

## Edge Cases & Constraints
<!-- scope: technical -->

- [inferred] A same-step Property holds of every step of its class on a trace. A witness whose last class also occurs earlier in its path would therefore claim the fact of both steps. The IR generator refuses such a witness and the author writes the triple.
- [inferred] A generated Property or Scenario that shares a name with a hand-written one of the same family would share its Definition ID. The IR generator refuses the pair, naming both positions.
- [paraphrase] Query names do not change, so Case file names, receipts and accepted findings keyed by Query name stay stable. The Definition IDs and fingerprints of the migrated Properties and Scenarios change once.
- [inferred] A derived Limits may differ from the hand-chosen one a migrated Query had, where the old bound exceeded the path's length. The regeneration shows each such change, and the Query's answer must not change with it.
- [inferred] Whether the live assessment's explanations depend on the phase condition today's Properties state beside the fact is established by the first conversion. Where dropping the condition changes a Case's assessment, the witness keeps it through `.ends`.
- [inferred] fn-134 ("Capabilities own their properties") changes how a machine declares capabilities and where their generated Queries are bounded. The duplicate check in R6 reads the Queries a machine's capabilities generate in the shape fn-134 leaves.
- [inferred] fn-139 ("Actor-grouped rules, per-RPC actions, shared rejections") renames the activity's actions. Whichever of the two specs lands second rewrites the class names inside the witnesses or the triples it finds.

## Acceptance Criteria
<!-- scope: both -->

- **R1:** [paraphrase] A Model author declares a path-and-outcome claim as one witness: the path's classes in order, `.records(<fact>)`, and optionally `.ends(<state predicate>)`. The IR generator lifts it to a pinned Scenario, a same-step Property about the last listed class and a `find` Query, each named after the witness. A machine, a derived machine and a composition each declare witnesses, and a witness outside a `val` takes an explicit name. Errors: a witness with no class is refused; one with no `.records` is refused; a fact of another machine's fact type does not compile; a witness whose last class occurs earlier in its path is refused, naming the class and both positions in the path; a witness neither a `val` nor an explicit name names is refused; a generated name that collides with a hand-written Property or Scenario of the same family is refused, naming both.
- **R2:** [paraphrase] A witness that states no Limits is bounded by the length of its path, and `.limits(<Limits>)` replaces that bound. Errors: a `.limits` whose step or schedule bound is shorter than the path is refused by the IR generator, naming the witness and both numbers.
- **R3:** [paraphrase] A Query generates a live Case exactly when it states `.live(<expectation>)`, on a witness and on a triple-form Query alike, and `.expect` no longer exists on a Query. The set of generated Cases is the set of Queries that state `.live`, plus those a capability's expectation binding generates. Errors: no error surface beyond R4's, and the existing admission errors for an incomplete Run expectation.
- **R4:** [paraphrase] Every expectation whose Property outcome or Contract verdict is not satisfied, and every non-satisfied monitor expectation inside it, carries a non-empty reason text on the expectation, wherever it is declared: a witness, a triple-form Query or a capability's expectation binding. Every such expectation in the Models carries one. A reason that the Models' comments and the Case's recorded assessment do not establish is asked of the owner and never invented. Errors: the IR generator refuses a non-satisfied expectation with a missing or empty reason, naming the Query or the capability binding; a reason on a satisfied expectation does not compile.
- **R5:** [paraphrase] In the standalone activity's System, every `find` Query over a pinned Scenario whose Property and Scenario no other declaration reads is a witness, and its `properties` section holds only Properties that a `verify` Query or another declaration reads. [inferred] The same holds in every other Model, and the model lint flags a pinned `find` written as a triple whose Property and Scenario nothing else reads. Every migrated Query keeps its name, its answer and its Case file name, and every live Case keeps its expected assessment. Errors: the regeneration's diff holds only the Definition IDs and fingerprints of the migrated Properties and Scenarios, the references to them, witness-core rewrites of migrated Property functions with equal truth values on every applicable model row, source positions explicitly mapped to edited declaration spans, derived Limits and R6's removal; any other difference stops the regeneration.
- **R6:** [paraphrase] The hand-written `terminate` Query of the standalone activity's System, which repeats the capability-generated `terminateSettles` Query, is deleted with its Property, its Scenario and its Case. Errors: a witness with the same pinned path and the same recorded fact as a Query a capability of its machine generates is refused by the IR generator, naming both.
- **R7:** [inferred] The Models' author documentation states when to write a witness and when to write a `query verify`, gives the witness's core form, and states that `.live` is what generates a Case. The layout template Model shows one witness. Errors: no error surface.

## Boundaries
<!-- scope: business -->

- [paraphrase] The IR keeps its shape. No new declaration kind, Query form or "last step only" Property semantics is added, and the Go reader, checker and lowering are not changed for witnesses.
- [paraphrase] The Property, Scenario and Query triple stays the form for invariants. This spec does not change `query verify`, the functions that build one list of Queries for several designs, or the claims types those functions return.
- [inferred] Queries whose stated Property only gives a path's monitors something to watch keep their present form. A direct way to ask a monitor over a path is a separate change.
- [inferred] A capability's path and expectation bindings keep their shape. Only R4's reason rule reaches them.
- [inferred] Expectation reasons are not carried into the IR, the Case manifest or an assessment.
- [inferred] Cross-level fact identity, composed Scenario steps, the monitor declaration form and the realization's evidence helpers, which the same review named, are not part of this spec.

## Decision Context
<!-- scope: both -->

[paraphrase] The triple form is paid for everywhere and earned only where the relation is many-to-many. In the activity's System it is close to one Property, one Scenario and one Query per claim. In the history record and its compositions a few invariants run over many paths and several designs, and there the separation does its job. So the witness is added beside the triple and does not replace it.

[paraphrase] For a live Case the Property is nominal. The pause-and-resume Case asks a Property about completion that says nothing of pausing. The Run conforming to the path, with each step's facts evidenced, is what tests the pause. The lowering already reduces such a Property to the fact it names. That is why a witness claims a fact and treats a state condition as the exception.

[paraphrase] A capability binding already states an action, a fact, a path and an expectation in one place, and generates the same three declarations. Turning every path-and-outcome claim into a capability was considered. [inferred] It was rejected because a capability is a protocol several entities share, and a feature's own path is not one.

[user] "Rename once". [paraphrase] The owner accepted a single regeneration that changes the Definition IDs of the migrated Properties and Scenarios. The alternative kept today's names by having the witness take explicit Property and Scenario names, which would leave three names per claim in the source.

[user] "Delete hand-written". [paraphrase] The owner chose to remove the hand-written `terminate` and to refuse a witness that repeats a capability-generated Query. The alternative kept both as a pin that the two forms agree, at the cost of running the same server path twice.

[inferred] The reason sits on the expectation value so that one rule covers a witness, a triple-form Query and a capability binding. A `because` parameter on `.live` was considered and would have left capability bindings without the rule.

[inferred] The form is named `witness` because the semantics document already calls the trace a `find` returns a witness. fn-137 uses "type witness" for a different thing, in generic capability code that Model authors do not read beside their Queries.

[inferred] fn-138 ("Retries and Deadline capabilities") keeps the hand-written timeout Properties beside the Deadline Properties and leaves retiring them to the owner. Under R5 the Queries that read those Properties become witnesses, so the Properties stop being separate declarations whichever way that decision goes.

[inferred] fn-135 ("effect { } and is { } blocks for effects and predicates") derives a status fact from the phase a step enters. After it, the fact a witness records and the phase it ends in are one statement, which supports claiming the fact alone.

The completed DSL supplies the witness's core declarations and capability expansion. Planning uses the existing Query-returning discovery path and source-aware admission, with no new IR or Go witness machinery. A composition's member selector retains that member's fact type. The derived search budget is 512, the established live-query search budget; the path supplies the step and schedule bounds, and an explicit Limits value remains the override.

The migration inventories inbound readers before converting each pinned find. Shared triples, invariant Queries and monitor-only claims keep their existing form. The current custom-start Scenarios feed verify Queries, and the current eligible Models need no additional witness starts or total surface. The existing assessment tests establish the retry explanation, including its full terminal-state condition; implementation checks that evidence again against the completed activity batch rather than carrying an obsolete explanation forward.

Broad generated API drift verification and new CI coverage remain outside this plan, per the declined decision. The existing focused fixture and gate checks supply the required verification.

Maintainability (plan review): duplication - none identified; structure - witness folding uses one cohesive helper behind existing claim registration, keeping new construction logic out of the dispatch.

## Early proof point

Task fn-140-one-sentence-witness-queries-with.4 proves the first Activity conversion preserves its Query answers and expected assessments, including the retry's full-state condition, while removing only the duplicate terminate claim. If that equivalence fails, resolve the witness's state condition and the unexplained regeneration difference before expanding the migration in task .6.

## Execution

Implementation starts after the activity batch closes at its shared regeneration, review and live-run boundary. The three foundation tasks serialize their shared declaration and fixture surfaces. Tasks .4 and .5 are disjoint parallel candidates; .4 owns the Activity proof regeneration while .5 writes author documentation and the template. Task .6 joins them and owns the final regeneration, managed fixture checks and integration gates. Fault declarations follow this spec, and the generator restructuring consumes the later settled schema baseline.

The approved fn-142 then fn-143 preparation may run alongside the activity batch in isolated worktrees with clean integrations. Before implementation, re-anchor all current framework and shared-package paths to those completed moves without expanding any task's scope.

## Quick commands

```bash
python3 /home/agent/.codex/scripts/flowctl.py validate --spec fn-140-one-sentence-witness-queries-with --coverage --json
make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks
make umpire-check-cases umpire-check-fixtures canary-check-case
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | [paraphrase] A Model author declares a path-and-outcome claim as one witness: the path's classes in order, `.records(<fact>)`, and optionally `.ends(<state predicate>)`. The IR generator lifts it to a pinned Scenario, a same-step Property about the last listed class and a `find` Query, each named after the witness. A machine, a derived machine and a composition each declare witnesses, and a witness outside a `val` takes an explicit name. Errors: a witness with no class is refused; one with no `.records` is refused; a fact of another machine's fact type does not compile; a witness whose last class occurs earlier in its path is refused, naming the class and both positions in the path; a witness neither a `val` nor an explicit name names is refused; a generated name that collides with a hand-written Property or Scenario of the same family is refused, naming both. | fn-140-one-sentence-witness-queries-with.1, fn-140-one-sentence-witness-queries-with.2 | — |
| R2 | [paraphrase] A witness that states no Limits is bounded by the length of its path, and `.limits(<Limits>)` replaces that bound. Errors: a `.limits` whose step or schedule bound is shorter than the path is refused by the IR generator, naming the witness and both numbers. | fn-140-one-sentence-witness-queries-with.1, fn-140-one-sentence-witness-queries-with.2 | — |
| R3 | [paraphrase] A Query generates a live Case exactly when it states `.live(<expectation>)`, on a witness and on a triple-form Query alike, and `.expect` no longer exists on a Query. The set of generated Cases is the set of Queries that state `.live`, plus those a capability's expectation binding generates. Errors: no error surface beyond R4's, and the existing admission errors for an incomplete Run expectation. | fn-140-one-sentence-witness-queries-with.3 | — |
| R4 | [paraphrase] Every expectation whose Property outcome or Contract verdict is not satisfied, and every non-satisfied monitor expectation inside it, carries a non-empty reason text on the expectation, wherever it is declared: a witness, a triple-form Query or a capability's expectation binding. Every such expectation in the Models carries one. A reason that the Models' comments and the Case's recorded assessment do not establish is asked of the owner and never invented. Errors: the IR generator refuses a non-satisfied expectation with a missing or empty reason, naming the Query or the capability binding; a reason on a satisfied expectation does not compile. | fn-140-one-sentence-witness-queries-with.3 | — |
| R5 | [paraphrase] In the standalone activity's System, every `find` Query over a pinned Scenario whose Property and Scenario no other declaration reads is a witness, and its `properties` section holds only Properties that a `verify` Query or another declaration reads. [inferred] The same holds in every other Model, and the model lint flags a pinned `find` written as a triple whose Property and Scenario nothing else reads. Every migrated Query keeps its name, its answer and its Case file name, and every live Case keeps its expected assessment. Errors: the regeneration's diff holds only the Definition IDs and fingerprints of the migrated Properties and Scenarios, the references to them, witness-core rewrites of migrated Property functions with equal truth values on every applicable model row, source positions explicitly mapped to edited declaration spans, derived Limits and R6's removal; any other difference stops the regeneration. | fn-140-one-sentence-witness-queries-with.4, fn-140-one-sentence-witness-queries-with.6 | — |
| R6 | [paraphrase] The hand-written `terminate` Query of the standalone activity's System, which repeats the capability-generated `terminateSettles` Query, is deleted with its Property, its Scenario and its Case. Errors: a witness with the same pinned path and the same recorded fact as a Query a capability of its machine generates is refused by the IR generator, naming both. | fn-140-one-sentence-witness-queries-with.2, fn-140-one-sentence-witness-queries-with.4 | — |
| R7 | [inferred] The Models' author documentation states when to write a witness and when to write a `query verify`, gives the witness's core form, and states that `.live` is what generates a Case. The layout template Model shows one witness. Errors: no error surface. | fn-140-one-sentence-witness-queries-with.5, fn-140-one-sentence-witness-queries-with.6 | — |
