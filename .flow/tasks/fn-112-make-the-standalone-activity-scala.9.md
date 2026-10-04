---
satisfies: [R12, R13, R16]
---
# fn-112-make-the-standalone-activity-scala.9 Extract the shared Temporal realization kit and script helpers

Touches: [model/umpire/realize/**, model/lifter/Realizations.scala, model/lifter/Syntax.scala, model/lifter/test/**, model/lifter/testdata/**, model/temporal/realize/**, model/temporal/standaloneactivity/Realization.scala, model/temporal/nexuscaller/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, model/ir/**, model/cases/**]

## Description
Create the shared Temporal-specific kit and rewrite activity scripts through value references and typed Item-mode helpers.

**Size:** M
**Files:** model/temporal/realize/** (core helpers, plus `model/temporal/realize/Syntax.scala` for the kit's sugar), standaloneactivity Realization.scala, nexuscaller integration, umpire.realize helpers and lifter Realizations.scala/`Syntax.scala`/tests.

### Approach
- Move only the identical roles, environment bindings and correlation window shared by activity and Nexus into temporal.realize; parameterize feature-specific values.
- Add general script/perform/onPath/always helpers that lower to existing Item fields and preserve order, optional modes and declaration/reference IDs. Consult fn-118's inventory/interface decision so the shared kit exposes the typed method/read-condition seam future hints need, without adding hint fields, derived waits or Program changes here.
- Typed request assignments use `:=` (spec API Contracts, realization script helpers; Decision Context "fn-117's `:=` promise"). `rpc(role, method) { … }` and `poll(…) { … }` open a scope (a context function with the method's request type `Req` fixed) whose `field` method takes a `Req => V` selector, so a line reads `field(_.namespace) := environment(workerNamespace)`, `field(_.activityId) := run`. `field(…)` is the contract: a bare lambda as the receiver of an extension `:=` has no expected type in Scala, so `_.namespace := …` cannot type-check. This is the same operator and meaning as fn-112.5's named input (`@targetName("set")`, "this named slot receives this value"); do not give it a second reading or add another symbol. The lifter reads `Req` from the enclosing `rpc`/`poll` call rather than from a `Field` type argument and lowers to the existing assignment record. fn-117's shipped `Assignment.typed(Field[Req, V](_.x), operand)` leaves the author surface; keep `Assignment.typed` only if the kit needs it internally.
- **Core and sugar (spec Architecture; Decision Context "Core and sugar").** `rpc`, `poll`, `perform`, `onPath`, `always`, `script`, `command` and the kit's roles and bindings are core (realization instructions the IR carries). `field(…) :=` is sugar for the kit's core assignment record and lives in `model/temporal/realize/Syntax.scala`, documented with the core form it stands for; core kit files import nothing from it; its matching lives in `model/lifter/Syntax.scala` (reached from `Realizations.scala` through one hook) and lowers to the same assignment record, with one fixture declaring an assignment both ways and proving the records equal.
- The five `awaitStatus(fact, enumValue)` pairs (`Realization.scala:288-303, 381-471, 681`) become one declared fact-to-status table beside the realization (`Describable`'s status map in `.plans/SEMANTIC_PROTOCOLS.md`; spec API Contracts "Realization script helpers"), consulted by the kit's await helper so a call site writes `awaitStatus(ProtocolFact.statusPaused)` and no pair is repeated. The table is Scala-side only: the lowered poll, its condition, interval and evidence are unchanged, so every Case stays byte-identical. fn-118's seam (`await(evidence, role)(assign, until)`) is kept; the table supplies `until`. `fn-122-capabilities-and-their-laws` reads this table as its `Describable` capability; this task declares no capability.
- Rewrite standalone activity realization with typed fn117 API selections, fact values and declaration references; migrate Nexus only enough to consume the shared kit.
- Add positive/refusal fixtures for unknown references and invalid helper combinations, keeping fn118 behavior hints out. For `:=` in scripts: one lifting fixture and one refusal fixture (a selector of a request type other than the enclosing method's does not compile; record that compiler refusal, per the settled R16 rule).
- Add to `model/README.md`'s "Writing a Model" the one-line operator legend (the three symbols `~>` binds an action to its step function, `->` pairs a key with its value, `:=` gives a named slot a value, each with one meaning; everything else is a word) and one paragraph on core and sugar: what is core, what is sugar, that sugar lives in the `Syntax.scala` files of the DSL, the kit and the lifter, each form documented with its core spelling and proven IR-equal by a fixture, and that core files import no sugar. This task touches README already and lands the last sugar form.
## Acceptance
- [ ] Shared roles, bindings and window are written once, documented and consumed by activity and Nexus with unchanged lifted values.
- [ ] Standalone Realization.scala contains no direct Item constructor, fact-name string or same-realization ID/reference duplication.
- [ ] Request fields are written `field(_.name) := operand` inside `rpc`/`poll` with the request type taken from the scope; Realization.scala contains no `Assignment.typed` and no `Field[Req, V](...)` on the author surface, and `:=` has the same `@targetName("set")` definition and meaning as the named-input form.
- [ ] `field(…) :=` lives in `model/temporal/realize/Syntax.scala`, its matching in `model/lifter/Syntax.scala`; core kit files import no `Syntax.scala`; one lifting fixture proves IR equality with the core assignment record and one refusal fixture (selector of another request type) is recorded; no new IR field.
- [ ] Awaited statuses come from one declared fact-to-status table; `awaitStatus` takes the fact alone and no `(fact, enumValue)` pair is written at a call site; the lowered polls are unchanged.
- [ ] `model/README.md` "Writing a Model" lists the three symbols with their one meaning each, in one line, and carries the one-paragraph core/sugar rule.
- [ ] Script order/modes, realization IDs, evidence catalogs and all Cases equal task 1.
- [ ] The helper interface accommodates fn-118's inventoried typed API and read-condition use, while this task introduces no hint-driven wait or Case Program delta.
- [ ] Focused lifter/model/Nexus tests and lint-model pass; no behavior hint or Go SDK work is introduced.
## Done summary
Extracted the shared Temporal realization kit and the script helpers, and rewrote the standalone
activity realization with them. Commits 75022999cf, 35579c985d, bb039cfbc7. Lifted IR differs only in
positions; Definition IDs, script order/modes, evidence catalogs, Query answers and Case bytes are
unchanged (OriginalBaseline passes, original.json untouched).

**Framework (core, model/umpire/realize/Scripts.scala)**: `script(id, activation)(items*)`,
`perform(step -> command, …)`, `onPath(classes*)(command)`, `always(command)`,
`command(instruction, after, timeoutMs, regardless, closes)`, request scopes `rpc(role, method) { … }`,
`poll(evidence, role, until, intervalMs) { … }`, `call.setting { … }` (appends fields, keeps the
command name), `statusTable(fact -> value, …)` (settled spelling of the sketch), `type Fact`.
Declarations are referred to by value: role/script/actuator/command/evidence params widened to
`String | X`. `umpire.realize.Control` is renamed `Actuator`, so a Model's own `Control` needs no
`Control as _` (IR message unchanged).

**Kit (model/temporal/realize/Kit.scala)**: roles `workflowService`, `caseWorker`, `taskQueue`,
`handlerTaskQueue`, `nexusEndpoint`, bindings and operands, `correlation(entity)`, `correlated`,
`temporalRealization(machine, operation, roles, scripts, evidence, …)(using Family)`, `controller`,
`await(evidence, role)(until) { … }` with the one 250 ms interval, deadlines `deadlineSeconds`/
`unreachedDeadlineSeconds`, `evidenceId`/`sourceId`/`runRecord`, `answered`/`answeredAs`/`delivered`.
Sugar `field(_.name) := operand` in kit `Syntax.scala`: `RequestField` is an `umpire.Slot`, so `:=` is
the same `@targetName("set")` definition; core form `Assignment.typed(Field[Req, V](_.name), operand)`.
Activity and Nexus both build through `temporalRealization`; Nexus also polls through `await`.

**Lifter**: commands named after their val in kebab case (`Command(id, …)` keeps its id); script
helpers written by name; request scopes read with `Req` from the scope the call opened; facts by
enum case or case companion; status table looked up at lift time; `requestAssignment` hook in
lifter/Syntax.scala; a realization is placed at its val. Refusals: unnamed command, empty
perform/onPath, unlisted or doubly listed status, non-field line in a scope; compiler refusal of a
foreign selector (typedInvalid Invalid.scala:169:26).

**Decisions**: `await` lives in the kit (its body carries the interval the lifter reads; framework
bodies are not reducible), signature `await(evidence, role)(until) { assign }` instead of fn-118's
`(assign, until)`; `temporalRealization` takes `roles` and the entity as `operation`. Positions stay
where a record is written (kit helper bodies), Go tests accept the kit file; golden merges gained
file entries (both Realization.scala files and realize/ compare as one name). The lost-response
release keeps a written-out `Command("release-dispatch", …)` because two realizations share that id.

**Metrics**: standaloneactivity 2449 lines / 181 literals -> 1994 / 98; Realization.scala 862 / 94 ->
407 / 11; kit 288 / 26; standalone+taskqueue+kit 2700 / 149 (was standalone+taskqueue 2867 / 206).
Task 10 still needs -394 lines and -38 literals (left: 25 computed names, 21 declared, 17 ids, 9
prose, 8 repeated, 8 composition keys, 7 own, 3 evidence).

**Review**: claude-opus-5-5 high via `--spec claude:claude-opus-5-5:high`; writer and reviewer are the
same family (Opus). Round 1 SHIP with two P3s (written-out Command around rpc; Nexus 250s), both fixed
in bb039cfbc7; round 2 SHIP. Deferred P3/FYI: kit-written evidence is placed at the kit body, not the
call site; `Fact = AnyRef` is loose (lifter refuses non-facts only where it can tell);
heldDelivery's await reads `status(ProtocolFact.statusPaused)` beside `status(AdmissionFact…)` (same
id); `closes` names attemptAdmitted by its derived id (the evidence and release refer to each other).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 75022999cf, 35579c985d, bb039cfbc7
- Tests: make umpire-gen-model MODEL_GATE_ARGS=--skip-go-checks (exit 0), go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline|MigrationGoldens|MigrationProjection|IRInventory' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), go test -tags test_dep -count=1 -p 2 -timeout 40m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 232 s), go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/internal/golden ./tools/umpire/lower ./tools/umpire/model ./tools/umpire/export ./tools/umpire/conformance (exit 0, after review fixes), make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0)
- PRs: