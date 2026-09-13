---
satisfies: [R5, R6]
---
# fn-87-tighten-the-testpilot-protocol-glossary.8 Presence as optional fields, Deadline bound oneof, capture type and one named-value message

## Description
Apply R5's remaining presence changes and the Contract/correlated half of R6: single-arm oneofs become proto3 `optional`; `Deadline` gets `violation_state_id` plus a `bound` oneof; the capture type becomes `SingularType` restricted at preparation; one named-value message replaces the three binding shapes; `CorrelatedContract.version` is removed. The Program/value half of R6 (opaque handle, `natural_value`, wire `EntrypointKind`, scalar kinds) is .9, split because it touches the Driver-contract leaf and type admission instead of the evaluator.

**Size:** M
**Files:** `proto/.../v1/{run,contract,correlated}.proto`, `api/testpilot/v1/*`, `common/testing/testpilot/internal/verification/{prepare.go,prepare_test.go,evaluator.go,captures.go,correlated_prepare.go,correlated.go}`, `common/testing/testpilot/internal/execution/{projection.go,dataflow.go,evidence_lift_test.go}`, `model/Testpilot/{Authoring,Correlated}.lean`, `model/Umpire/Case/Correlated.lean`, `model/Temporal/Case/Evidence.lean`, Producers setting deadlines, fixtures, mapping, `common/testing/testpilot/internal/verification/README.md:19-22`
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, common/testing/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Temporal/**, tests/testcore/testpilot/**]

### Approach
- Probe first: confirm the Lean protobuf elaborator accepts proto3 `optional` (synthetic oneofs) and that `make proto` (`lint-api`) passes on a scratch message. If Lean rejects it, keep single-arm oneofs and record the reason; the Deadline oneof is unaffected.
- R5 presence: `RunDiagnostic.support` → `optional int64 supporting_event_sequence`; `Run.evaluation_failure` → `optional int64 evaluation_failure_sequence` (JSON names unchanged; `protocol_test.go:28-31` already asserts presence). Empty marker messages stay where they select a oneof arm (`RunReference`, the presence path selector), per Edge Cases.
- R5 Deadline: `Deadline { string violation_state_id = 1; oneof bound { int64 rule_events = 2; int64 elapsed_milliseconds = 3; } }` (spec API Contracts; `ContractDeadline` → `Deadline`). Replace the "exactly one positive" check at `verification/prepare.go:267-278` with: no bound set rejects at preparation, located at the rule; a set bound that is not positive still rejects (EVD-21 requires a positive bound; the oneof removes only the two-bounds case); a missing `violation_state_id` and a deadline on a `SAFETY` rule keep rejecting as today (unit tests if not already pinned). Update the evaluator comment at `evaluator.go:301`. EVD-21's approved text is not edited.
- R6 capture type: `ContractCapture.type` becomes `SingularType`; preparation rejects `any` and `opaque_handle` (until .9 removes it) with a located error naming the capture.
- R6 named value: the evidence side (`CorrelatedBinding` with a string value, `CorrelatedEvidenceField` with a `Value`) becomes one `NamedValue { string field_id = 1; Value value = 2; }`. The lift side (`CorrelatedEvidenceBinding`: a path or a literal) is an expression over the projected value after .6, so it becomes `NamedValue`-shaped with an `Expression` in place of the `Value` (`NamedExpression { field_id; Expression value }`), whose literal arm is a `Value`. That leaves one named-value shape per side instead of three unrelated ones; record this reading of R6 in the done summary and the spec's decision note. `CorrelatedBinding.value` was a string: its values become `text_value`, which grows `proto.Size`; if any `max_event_bytes` check moves, record it as a finding and adjust only with a declared loosened-bound entry.
- R6 version: remove `CorrelatedContract.version` and the admission that checks it (the Case carries `FormatVersion`); `preparation_error_test.go:82-84` pins "unsupported correlated capability version" and changes with it.
- Duplicate `field_id` within one binding list rejects at preparation (unit test).
- Lean: Authoring constructors for deadlines (`Contract.deadline` or equivalent) take a bound sum; `Testpilot.Correlated` decodes `NamedValue`; Producers and `Umpire/Case/Correlated.lean` stop writing `version`.
- Mapping steps: deadline shape (drop zero-valued bound, move the positive one under `bound`), capture type unwrap, named-value shapes, `version` drop. Retire `ContractDeadline`, `ContractCaptureType`, `CorrelatedBinding`, `CorrelatedEvidenceField`.

### Investigation targets
**Required** (read before coding):
- `common/testing/testpilot/internal/verification/prepare.go:255-290` — deadline check
- `common/testing/testpilot/internal/verification/captures.go` — capture typing
- `common/testing/testpilot/internal/verification/correlated_prepare.go:40-80` — version and limits admission
- `common/testing/testpilot/internal/execution/dataflow.go:339-464` — evidence bindings
- `model/Testpilot/Correlated.lean:200-240` — decoder

**Optional:**
- `.plans/UMPIRE4_SPEC.md` EVD-21

### Key context
- `correlated.json`, the typed Nexus and worker-outage Cases are the Verdict pins (deadlines and captures).

## Acceptance
- [ ] single-arm presence oneofs are `optional`; `Deadline` has `violation_state_id` and a `bound` oneof; a deadline with no bound rejects at preparation (unit test) and a non-positive bound still rejects (unit test)
- [ ] capture type is `SingularType`; a capture typed outside scalar, enum or message rejects at preparation naming the capture (unit test)
- [ ] one named-value message replaces the three binding shapes (final shape recorded); `CorrelatedContract.version` removed
- [ ] equivalence test passes with declared steps; `expected.json`, correlated expectations and live Verdicts unchanged; retired tokens added
- [ ] `make umpire-check-regression` exit 0 with nine live identities; `make lint-model` 163; `make lint-code` no new issues


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
