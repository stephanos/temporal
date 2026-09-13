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
A liveness `Deadline` now holds `violation_state_id` and a `bound` oneof. A capture is typed by `SingularType`, restricted at preparation. One named-value message per side replaces the three binding shapes, and `CorrelatedContract.version` is gone. No `expected.json`, no correlated `expected`/`incomplete` value and no live Verdict changed. The oracle passes with five declared steps, and the regression gate passed with 9 live identities.

**Decisions** (recorded in the fn-87 Planning decisions as "Presence, deadlines, captures and named values (decided in .8)")
- **proto3 `optional` not adopted (R5).**
  - The probe showed the Lean elaborator supports it. `make proto` fails, though, because the pinned `protoc-gen-go-helpers` v1.63.5 rejects proto3 optional fields.
  - So `Run.evaluation_failure` and `RunDiagnostic.support` stay single-arm oneofs, as the task's fallback allowed.
  - The failed generation deleted the CHASM generated files. I restored them from git before the rerun.
- **Final named-value shape (R6).**
  - Evidence side: `NamedValue { field_id, Value value }`. A scope value must be a non-empty `text_value`.
  - Lift side: `NamedExpression { field_id, Expression value }`. It is admitted only as a text literal or `path(projected_value, p)`, and it runs the evidence-lift context check.
  - Both live in `correlated.proto`.
- **Errors.**
  - Deadline, category `malformed`:
    - no bound: `contract.rules[<rule>].deadline`
    - non-positive bound: `...deadline.rule_events` or `...deadline.elapsed_milliseconds`
  - Capture, category `malformed`: `any`, `opaque_handle` or no type rejects at `contract.rules[<rule>].captures[<capture>].type`.
  - Correlated `version`: a Case that still writes it fails strict ProtoJSON decode naming the field.
- **Lean.**
  - `Contract.deadline (bound : Deadline.bound_Type) violationStateId` replaces `deadline`/`deadlineEvents`.
  - The capture constructors are gone in favor of `Types.scalar`/`enumeration`/`messageType`.
  - `Program.evidencePath` and `Program.evidenceLiteral` replace the binding constructors.
  - `Testpilot.Correlated` drops the version check and rejects a non-text scope value.
- **Bound finding.** `max_event_bytes` counts rune lengths, so it did not move. The encoded evidence a lift charges as runtime work grows by 2 bytes per scope value.

**Tests** (each asserts the exact located error)
- `TestPrepareLocatesDeadlineBounds` covers:
  - no bound, and no deadline
  - zero and negative bounds
  - missing violation state
  - a deadline on a safety rule
  - both bounds admitted
- `TestPrepareLocatesCaptureTypesOutsideScalarEnumOrMessage`
- `TestCorrelatedEvidenceScopeValuesAreText`: confirmed red with the text check loosened.
- `TestCorrelatedContractCarriesNoVersion`
- `TestEvidenceLiftBindingRejectsReferencesOutsideItsContext` and `TestEvidenceLiftLiteralBindingRequiresText`
- New lift rejection cases: a path over another operand, and a non-path expression. The duplicate `field_id` case was already pinned.
- `case_schema_test` gains a crossed-bound decode case.
- Lean `#guard`s cover the bound sum and the non-text scope rejection.

**Oracle, fixtures and vocabulary**
- New oracle steps: the deadline check plus its rename, the capture-type rename, the `version` drop (checked equal to 1), scope text wrapping into `NamedValue`, and lift bindings rewritten to `NamedExpression`.
- `TestDeclaredDeadlineVersionAndNamedValueSteps` covers each step and its refusal cases. A mutation check on the binding step made the oracle fail.
- Regenerated through `make umpire-gen-case-runtime-conformance`: `correlated.json`, `typed-nexus-case.json` and `async-nexus-case.json`. Their only changes are the `version` drop and the wrapped scope and lift values; `worker-outage-case.json` is byte-identical.
- The retired-vocabulary gate and `protocol_test` gain `ContractDeadline`, `ContractCaptureType`, `CorrelatedBinding`, `CorrelatedEvidenceField` and `CorrelatedEvidenceBinding`, with positive and negative test lines.
- The READMEs are updated.

**Outside declared Touches**
- `tools/umpire/internal/retiredvocabulary/check.go` and `check_test.go`: the retired tokens.
- `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go`: its crossed-oneof test names the removed type.
- `.flow` spec: the decision note, plus flowctl review bookkeeping.

**Gates**
- Baseline was green: the oracle ran, and the regression receipt at 02d09609 was honored.
- `make umpire-check-regression`:
  - Run 1, uncommitted tree: `TestTestpilotAsyncNexusCase` ended INCONCLUSIVE, the known flake (c).
  - Run 2, at 187ff94033: exit 0, 9 identities.
  - Run 3, at HEAD 6448006489: exit 0, 9 identities. Receipt written.
- `lint-code`: 161 after `go clean -cache`, none in changed files. `lint-model`: 163. Both are the baselines.

**Follow-ups (not built)**
- Reviewer P3: export one located-error constructor from `ir`. It would replace the four 256-byte path-truncation copies, including the new `verification.invalidAt`.
- Pre-existing: a duplicate or invalid evidence binding `field_id` still reports at the node path, not at the binding site.

stage: impl-review - ran (claude backend, SHIP on first round; two P3s applied in follow-up commit 6448006489, one deferred)
## Evidence
- Commits: 187ff940334bb90fe9f89d5d63b58ed83f0f5be0, 64480064894a94bb24fd2702d45db75c90e298b4
- Tests: go test -count=1 -tags test_dep ./common/testing/testpilot/internal/protocolmigration/, go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tools/umpire/..., make proto, make umpire-check-testpilot-protocol umpire-check-testpilot-authoring umpire-check-case-runtime-conformance, make umpire-check-retired-vocabulary, CC=/usr/bin/cc TMPDIR=$(cd "${TMPDIR:-/tmp}" && pwd -P) make umpire-check-regression (run 1 at pre-commit tree: failed TestTestpilotAsyncNexusCase INCONCLUSIVE, known flake c; run 2 at 187ff94033: exit 0, 9 identities; run 3 at 6448006489: exit 0, 9 identities), go clean -cache && make lint-code GOLANGCI_LINT_FIX=false (161, baseline), make lint-code-fast GOLANGCI_LINT_FIX=false after review follow-up (161, none in changed files), make lint-model (163, baseline)
- PRs: