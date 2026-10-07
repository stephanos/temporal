---
satisfies: [R3, R11, R14]
---
# fn-133-lean-typed-realizations.1 Literal and call scopes: proto[T] { ... }, response reads, literal helpers

## Description
**Batch:** DSL batch (see MILESTONES.md, DSL batch). Do not run `make umpire-gen-model`, regenerate fixtures or Cases, or run the full gates in this task; any IR proof or comparison below is checked at the batch's single regeneration against the batch baseline (the tree at fn-132's close), not against a snapshot taken by this task. Framework and lifter fixtures and munit tests still run here. Commit the task on its own.
Part A, R3, R11, R14 (fields). Paths are fn-132's.

- **`proto[T] { field(_.x) := … }`.** A protobuf literal scope modelled on the `rpc` request scope: nested messages, enums, maps, roles and payloads, with types inferred. The lifter lowers it to the same IR as `Proto[T](ProtoField.typed(…))`, which stays the core form.
- **Response reads in the call scope.** A `read(field, cardinality) into (targets…)` line inside `rpc(…) { … }`, and a way for one call to extend another call's request, so the Nexus caller's `history` extends `awaitClose` and drops the core `Instruction.rpc` form.
- **Request, response and error in one place.** Like a `stamp` handler, which saw the request and then the response or its error (`common/testing/stamp/mdl_router.go`), a call declares in its scope what it answers for each outcome. Example: `answers(Outcome.accepted -> ok, Outcome.notFound -> "not_found")`. Evidence then confirms the outcome, not only success. Today `answered` guards on `succeeded` alone. The Run already records the gRPC code (`InstructionOutcome.protocol_code`, `run.proto`), so Testpilot does not change. Today's realizations declare only `ok`, so the IR projection stays identical. fn-128.2's rejection rows (`failedPrecondition`, `invalidArgument`) bind their codes here.
- **Helpers in `model/temporal/realize`:** `applicationFailure(type, message, retryable)`, `jsonPayload(text)`, `finish(text)` and `duration(seconds)`.
- **Rewrite the three realizations with them.** Drop `Field[T, V]` type spellings where the scope infers them.

One lifter fixture per new form proves it lifts to the core form's IR. A field assigned twice in one literal scope gets a refusal fixture.
## Acceptance
- [ ] No feature file uses `ProtoField.typed`, `Instruction.rpc`, `Assignment.typed`, `ResponseRead.typed`, `ProtoValue.Text`, `Operand.Literal` or a hand-built `Failure`/`Payload`/`Duration` literal.
- [ ] Lifter fixtures show each new form lifts to its core form's IR; the double-assignment refusal is refused at its line.
- [ ] A before/after projection of `model/ir` and `model/cases` is identical.
- [ ] Line counts before and after are recorded; the spec's Verification gates pass.

## Done summary
Added the kit's protobuf literal scope `proto[M] { field(_.x) := v }`, response reads in a call's scope (`read(path, cardinality).into(targets…)`), `extended` (a call of its own that extends another call's request), message values for request fields, and the literal helpers `applicationFailure`, `jsonPayload`, `finish` and `duration`. The three realizations are rewritten with them. A lift of model/ir in the lane (scratch output, not written back) shows every `realizations` section equal to the baseline's once positions are stripped.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: IMPLEMENTER claude-opus-5-5 at high

### What changed
- `model/temporal/realize/Syntax.scala` (sugar, each with `Core form:`):
  - `proto[M] { … }`: lines are `field(_.x) := v`, a nested message `field(_.m) { … }`, or a passed-in `ProtoScope[M] ?=> Unit` applied in place.
  - The type class `ProtoValueOf[V, X]` decides which values a field takes: its own type (String, Boolean, Long, Int widened, enum), `TypedProto[V]`, a role (role_id), a `Name` (named), a String for bytes (utf8), `Map[String, String]` for a map of bytes, or a core `TypedProtoValue`.
  - `field` now finds its scope through a `FieldScope` type class, so it works in request scopes and literal scopes. Overloading on the using clause is ambiguous in Scala 3.
  - `RequestField` has member `:=` overloads, one for an operand and one for a `TypedProto`. The message form assigns each field the message sets, at `<path>.<field>`. So `field(_.getStartToCloseTimeout) := duration(2)` lifts as `start_to_close_timeout.seconds := 2`.
  - `read(path, cardinality).into(observed | Target …)`: an `Observed` written by value stands for `Target.Observe(id)`. The lifter refuses a read whose path is not a field of the call's response type, and a read inside a `readUntil` scope.
- `model/umpire/realize/Scripts.scala`: `call.extended { … }`. Like `withFields`, it appends assignments and reads, but the result is a command of its own, named after its val.
- `model/temporal/realize/Kit.scala`: `duration(seconds)`, `jsonPayload(text)`, `applicationFailure(type, message, retryable)` and `finish(text)`. They are written in core form and appended at the end of the file, so no earlier kit line moves.
- Lifter (`model/irgen/Syntax.scala` hooks `protoLiteral`, `responseRead` and `requestAssignment`, which now returns a List; `Realizations.scala`):
  - `reduce` no longer follows the kit's sugar defs (`vocabularyMember` covers `temporal.realize.Syntax$package$`).
  - A flag may be negated (`!retryable`).
  - `rpcValue`/`scoped` carry reads, and `extended` is handled like `withFields`.
  - A literal that sets one field twice is refused at its second line.
- `model/check/SyntaxRule.scala`: `"proto"` appended to `sugarNames`. `read`/`into` are not added, because `Recorded.read` is an object member and the rule would flag it.

### Decisions (owner unavailable; please record in the spec's Decision Context)
- **`answers(Outcome -> code)` not built.** fn-139.8 replaces per-realization answer strings with one shared Rejection→code table checked by conformance. A per-call `answers` line would duplicate that table. Today's realizations declare only `ok`, so building it would change nothing, and it has no IR field. The bullet is deferred to fn-139.8. None of the task's acceptance criteria name it.
- **History calls.** `history` cannot literally extend `awaitClose`: `awaitClose` sets `historyEventFilterType` and `history` does not. Both now extend one shared request, `historyRead`, which sets namespace, workflow id, page size and `waitNewEvent`. The IR is identical.
- **`historyEvents` keeps `Field[GetWorkflowExecutionHistoryResponse, Seq[HistoryEvent]]`.** It is a standalone val with no scope to infer the types from, and both `Recorded.read` and the `read` line use it. Dropping the `.map(e => e)` would change the IR path from `history.events[*]` to `history.events`. `historyKind`'s parameter types stay as well; fn-133.2 removes that helper.
- The activity's `attemptFailure` now takes `retryable` (part of fn-133.5's R8 list).

### Line counts (R10)
- Before (baseline):
  - activity `system/Realization.scala`: 333
  - nexus workflow: 514
  - nexus standalone: 101
- After: 320 / 387 / 100.

### Declared IR delta (batch regeneration)
- Source positions only, in `realizations`. No IDs change, no Cases are added, and no carrier metadata changes.
- No kit line before the end of Kit.scala moved.
- The lifter fixture `rejects.txt` gains one line (`ScriptRejects.scala:110`).

### Tests
- `mise exec -- scala-cli test model/irgen`: 95 passed, 0 failed. This includes the new test "proto, read, extended and a message for a request field lift as their core forms" (`lifts/Literals.scala`, sugared vs core realizations equal apart from positions, ids and names) and the `literalTwice` refusal (ScriptRejects.scala:110).
- `scala-cli run model/check -- --check-syntax` and `--check-comments`: clean.
- scalafmt `--check` over model sources, irgen and check: clean.
- A scratch lift of every model IR file (`lift --ir … /tmp/laneE-ir`), compared with model/ir: every `realizations` section is equal with positions stripped.
- Not run, per batch rules: umpire-gen-model, the full gates, scalafix lint-model.

### For later tasks
- In a request scope, `field(_.x) := …` is now a member of `RequestField`, not umpire's `Slot` extension. A file whose only use of `umpire.*` was `:=` must drop that import (nexus standalone did).
- Kit sugar lives in `temporal.realize.Syntax$package$` and is lowered by hooks in irgen/Syntax.scala, never followed. Kit helpers in Kit.scala are followed, with parameters bound. Kit core files must not name anything Syntax.scala defines; the syntax rule checks every name in it, givens too.
- Feature files still import `temporal.realize.{deadline as _, *}` (activity) and `worker as process`. fn-133.5 removes those collisions. The kit's `deadline`/`unreachedDeadline` operands are now unused by realizations (only lift fixtures use them), so .5 can delete or rename them.
- The README does not yet describe `proto`, `read`, `extended` or the helpers; fn-133.7 docs should add them.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: cd400f2a22
- Tests: mise exec -- scala-cli test model/irgen, scala-cli run model/check -- --check-syntax, scala-cli run model/check -- --check-comments, scala-cli fmt --check model sources, scratch lift --ir of model IR, realizations equal with positions stripped
- PRs: