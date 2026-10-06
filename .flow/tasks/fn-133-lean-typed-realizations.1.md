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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
