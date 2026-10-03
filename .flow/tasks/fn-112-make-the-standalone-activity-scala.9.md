---
satisfies: [R12, R13, R16]
---
# fn-112-make-the-standalone-activity-scala.9 Extract the shared Temporal realization kit and script helpers

Touches: [model/umpire/realize/**, model/lifter/Realizations.scala, model/lifter/test/**, model/lifter/testdata/**, model/temporal/realize/**, model/temporal/standaloneactivity/Realization.scala, model/temporal/nexuscaller/**, model/README.md, model/SEMANTICS.md, .plans/UMPIRE_MODULES.md, model/ir/**, model/cases/**]

## Description
Create the shared Temporal-specific kit and rewrite activity scripts through value references and typed Item-mode helpers.

**Size:** M
**Files:** model/temporal/realize/**, standaloneactivity Realization.scala, nexuscaller integration, umpire.realize helpers and lifter Realizations.scala/tests.

### Approach
- Move only the identical roles, environment bindings and correlation window shared by activity and Nexus into temporal.realize; parameterize feature-specific values.
- Add general script/perform/onPath/always helpers that lower to existing Item fields and preserve order, optional modes and declaration/reference IDs. Consult fn-118's inventory/interface decision so the shared kit exposes the typed method/read-condition seam future hints need, without adding hint fields, derived waits or Program changes here.
- Typed request assignments use `:=` (spec API Contracts, realization script helpers; Decision Context "fn-117's `:=` promise"). `rpc(role, method) { … }` and `poll(…) { … }` open a scope (a context function with the method's request type `Req` fixed) in which `_.field := operand` takes a `Req => V` selector on the left, e.g. `_.namespace := environment(workerNamespace)`, `_.activityId := run`. This is the same operator and meaning as fn-112.5's named input (`@targetName("set")`, "this named slot receives this value"); do not give it a second reading or add another symbol. The lifter reads `Req` from the enclosing `rpc`/`poll` call rather than from a `Field` type argument, receives the lambda through the extension rather than `Field.apply`, and lowers to the existing assignment record. If the context-function form proves awkward in TASTy, fall back to `field(_.namespace) := operand` with `Req` still inferred from the scope, never to a symbol with another meaning. fn-117's shipped `Assignment.typed(Field[Req, V](_.x), operand)` leaves the author surface; keep `Assignment.typed` only if the kit needs it internally.
- Rewrite standalone activity realization with typed fn117 API selections, fact values and declaration references; migrate Nexus only enough to consume the shared kit.
- Add positive/refusal fixtures for unknown references and invalid helper combinations, keeping fn118 behavior hints out. For `:=` in scripts: one lifting fixture and one refusal fixture (a selector of a request type other than the enclosing method's does not compile; record that compiler refusal, per the settled R16 rule).
- Add the one-line operator legend to `model/README.md`'s "Writing a Model": the three symbols `~>` (binds an action to its step function), `->` (pairs a key with its value) and `:=` (gives a named slot a value), each with one meaning; everything else is a word. This task touches README already and lands the last use of `:=`.
## Acceptance
- [ ] Shared roles, bindings and window are written once, documented and consumed by activity and Nexus with unchanged lifted values.
- [ ] Standalone Realization.scala contains no direct Item constructor, fact-name string or same-realization ID/reference duplication.
- [ ] Request fields are written `_.field := operand` inside `rpc`/`poll` with the request type taken from the scope; Realization.scala contains no `Assignment.typed` and no `Field[Req, V](...)` on the author surface, and `:=` has the same `@targetName("set")` definition and meaning as the named-input form.
- [ ] The typed `:=` request assignment has one lifting fixture and one refusal fixture (selector of another request type), and the lifter lowers it to the existing assignment record with no new IR field.
- [ ] `model/README.md` "Writing a Model" lists the three symbols `~>`, `->` and `:=` with their one meaning each, in one line.
- [ ] Script order/modes, realization IDs, evidence catalogs and all Cases equal task 1.
- [ ] The helper interface accommodates fn-118's inventoried typed API and read-condition use, while this task introduces no hint-driven wait or Case Program delta.
- [ ] Focused lifter/model/Nexus tests and lint-model pass; no behavior hint or Go SDK work is introduced.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
