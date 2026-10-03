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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
