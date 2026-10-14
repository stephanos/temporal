---
satisfies: [R10]
---
# fn-85-model-side-effects-as-typed-actions-and.9 One observation declaration per Case; the pendingAttempts read source

## Description
A Case declares each observation once with its source (a history event kind, a Run Event kind, or a read such as `DescribeWorkflowExecution`), its correlation key path and the fields it exposes; Program waits and Contract rules refer to it by name (R10, second half). The Producer emits the declarations from the Model's evidence and the realization's catalog, which gives `pendingAttempts` its read source: a bounded poll of one RPC projected by the pending operation's `scheduled_event_id`.

**Size:** M
**Files:** `proto/internal/temporal/server/api/testpilot/v1/program.proto` (observation declaration with a source oneof: history event, Run Event, read), `contract.proto` and `correlated.proto` (rules reference observations by name; no repeated key paths), `api/testpilot/v1/*`, `model/Testpilot/Authoring.lean`, `model/Umpire/Case/Producer.lean` and `Projection/Declaration.lean` (emit one declaration per observation), `model/Temporal/Case/Realization/Nexus.lean` (catalog: history events keyed by `scheduled_event_id`; `pendingAttempts` read binding), `model/Temporal/Case/EventKind.lean` and a sibling for reads (the catalog hook .16 installed answers history event kinds and Run Event kinds; a read source is its third answer), `common/testing/testpilot/internal/execution/{prepare,dataflow,program,response_read}.go` (waits resolve a declaration; a read source polls with a bound; there is no `projection.go`, the projection lives in `dataflow.go`, `program.go` and `verification/correlated*.go`), `common/testing/testpilot/internal/verification/prepare.go` (rules resolve declarations; duplicate or undeclared rejects), `common/testing/testpilot/temporal/server/session.go` (the read RPC), conformance corpus (one case per observation source; undeclared and duplicate rejections)
**Touches:** [proto/internal/temporal/server/api/testpilot/**, api/testpilot/**, model/Testpilot/**, model/Umpire/Case/**, model/Temporal/Case/**, common/testing/testpilot/**]

### Approach
- The protocol-migration oracle is retired in `.1`, so no declared mapping step is needed here; the
  conformance `expected.json` pins are the Verdict net.
- Extension checklist again: a new Run Event payload is not needed; the read source is a new observation source kind, so the checklist's "expression reference" and "conformance class" rows apply.
- A read observation is an instruction the Program performs (poll `DescribeWorkflowExecution` until the projected field satisfies the wait or the bound expires) whose result feeds the declared observation; the Contract reads it by name. No wait-for-duration instruction; timeouts in Queries 6 and 7 are observed through history events (task .11).
- Rejections: a reference to an undeclared observation; the same source and key path declared twice.
- The response read (fn-87's `ResponseRead`) stays for slot-bound reads; an observation declaration is the Contract-facing form.
- Adjusted 2026-09-19 after .16, .4 and .7 landed: since .16 an `evidence:` line's observed name
  resolves against a catalog hook the platform installs (`Temporal.Case.EventKind` answers from the
  generated `HistoryEvent` attributes oneof, the Testpilot Run Event kinds are the second source) or
  against an `observation` the Model declares, so the read source is a third resolver answer beside
  `EventKind`, not a second module of the same shape. The Producer still takes a per-Case evidence
  map (`Umpire.Case.Producer.EvidenceMapping`, built by the `evidence` lines of the
  `case … realizes <set>` block .7 added) on top of the machine's own `evidence:` lines; once the
  declarations are emitted from the Model's evidence and the catalog, those `case` lines duplicate
  the machine's and should go here, or in .11 with the `fixture` form -- the receipt says which.
  `.4` recorded that a projection rule's confirmed steps are a sequence, one result per evidence
  kind on the seven Queries; a path that takes one kind to two results needs this task's
  declarations to say which, so a declaration carries the outcome it confirms where the kind alone
  is ambiguous. The `.4` conformance family (`correlated.json`, a structured `attempts` field) is the
  corpus to extend with the read source. `Realization` gained `setup` and `switches` in .5; the
  catalog and the `pendingAttempts` read binding go beside them.

### Investigation targets
**Required:**
- `model/Temporal/Case/EventKind.lean:16-76` — the history catalog to generalize
- `model/Umpire/Case/Projection/Declaration.lean` and `Producer.lean:423-460,561-640` — `projectionDeclaration`, `resolveEvidence`, `produce`
- `common/testing/testpilot/internal/execution/dataflow.go`, `response_read.go` and `program.go:121-176` — reads and sinks
- `common/testing/testpilot/internal/verification/prepare.go` and `correlated_prepare.go` — how rules resolve observations today
- `tests/nexus_workflow_test.go:579-700` — `TestNexusOperationRetriesAfterHTTPFault` reads `pending_nexus_operations[].attempt`
- `model/Umpire/Command/Syntax.lean:1383-1440,1965-2000` — the `evidence:`/`unobservable:` keys and the catalog resolver the `machine` command calls (.16)

**Optional:**
- `common/testing/testpilot/temporal/server/session.go:72` — `InvokeRPC`

### Key context
- Spec Decision Context: "Program and Contract declaring observations separately" was rejected because the same event kind, key path and fields were written twice and could drift.

## Acceptance
- [x] every fixture declares each observation once; Program waits and Contract rules refer to declarations by name; a Case that declares the same source and key path twice, or references an undeclared observation, rejects at preparation with an existing category
- [x] `pendingAttempts` is a read observation whose source is `DescribeWorkflowExecution`'s `pending_nexus_operations.attempt` keyed by `scheduled_event_id`, declared once; a conformance case reads it through the Temporal Driver
- [x] a Driver conformance case exists per observation source (history event, Run Event, read)
- [x] fixtures regenerate with the diff listed; `make umpire-check-regression` exit 0


## Done summary

### The declaration

`Program.evidence` (program.proto) declares each kind of correlated evidence once: its identity, the
source identity its ordinals count in, the recorded data it is read from as a `source` oneof --
`HistoryEventSource { attributes_field }`, `RunEventSource { kind }` or `ReadSource { method, path }`
-- the Run coordinates that scope it (`NamedValue` literals), the path of its operation key and the
fields it exposes (`EvidenceFieldDeclaration { field_id, path }`). A `CorrelatedEvidenceRule` may
name a declaration (`evidence_id`, field 7) and spell nothing else; a `ReadEvidence` instruction
(arm 12, controller only) names a read declaration, an endpoint role, request assignments, a
boolean `until` over one element of the declared path and a poll interval. The correlated Contract's
projection rules already name kinds, and the Case compiler's local-name pass now renames the
declarations, the by-name rules and the read instructions with everything else, so the kind, the
source, the key path and the fields are written once: the regenerated async-Nexus fixture's two lift
rules are `evidenceId: evidence.started` / `evidence.completed` and its Program carries the two
declarations the Contract's projection rules name.

### Admission and the runtime

`internal/execution/evidence.go` binds the declarations before the instructions: each identity
once, each source and operation key path once (`malformed` at `program.evidence[i]`), a history arm
the `HistoryEvent` attributes oneof carries (`unknown`), a Run Event kind that carries a payload
(`unsupported`), a read method the catalog knows whose path ends in repeated messages
(`type_mismatch`), every path typed against the recorded value; `ProgramView.Evidence()` exposes
them. A by-name rule requires a history declaration and a history-event read (`unknown` at
`...rules[i].evidence_id` for an undeclared kind, `unsupported` for a read or Run Event
declaration, `malformed` for a rule that also spells) and takes the declaration's guard
(`present(attributes<arm>)`), scope, key and fields. A `ReadEvidence` node binds its assignments
like an RPC, its `until` in the evidence-lift context, and synthesizes one `EMIT_EACH` response read
of the declared path whose lift is the declaration with `until` as its guard, so every element the
condition selects is lifted; its poll interval must be positive and within the node's timeout. The
scheduler dispatches it through a new `Session.PollRPC` (interval, predicate) and `readSatisfied`
answers the predicate; a Run Event declaration is lifted by `scheduler.liftRunEvents` as the event
is recorded, into the Program's one `CorrelatedEvidence` Observation, ordinals dense per source.
Verification checks, when a Program declares anything, that every projection rule kind, every field
it reads and every source is declared (`unknown`). `DeriveProfile` authorizes the declaration's
method on the poll's role and the `ReadEvidence` Opcode.

### The Drivers

The server Session's `PollRPC` is one effect: it repeats the declaration's RPC on the endpoint at the
interval until the predicate accepts a response, returning that response, the first failed poll's
outcome, or `TIMED_OUT` when the instruction's timeout ends it; the instruction's attempt and
identity are counted once (`TestPollRPCRepeatsTheReadUntilSatisfied`,
`TestPollRPCRefusesWhatTheDeclarationDoesNotAdmit`, on a synthetic pending service since the
server package may not name Nexus). The worker Session refuses polls, the composite routes them to
the controller Session, and every test Session implements the method.

### Lean

`Testpilot.Authoring` gains `Program.historyEvidenceDeclaration`, `runEventEvidenceDeclaration`,
`readEvidenceDeclaration`, `evidenceScope`, `evidenceField`, `declaredEvidenceRule` and the
`readEvidence` instruction; `Program.make` takes `evidence`. The Producer's `EvidenceSource` carries
a `RecordedSource` (`historyEvent`, `runEvent`, `read`) and `fields`, and `assembleProgram` emits one
declaration per admitted kind the resolved rules read; `Temporal.Case.Evidence.target` lifts the
history kinds among them by name. `Temporal.Case.ReadKind` is the third catalog answer beside
`EventKind` and the Run Event kinds: `pendingAttempts` bound to `DescribeWorkflowExecution`,
`pending_nexus_operations`, key `scheduled_event_id`, field `attempts` from `attempt`; the Nexus
template's `pendingAttemptsSource` and `pendingAttemptsNode` are built from that binding, and the
template's sources now carry it beside the two history kinds.

### Conformance

Five corpus entries: `satisfied/history-evidence` (a history read lifting by name),
`satisfied/run-event-evidence` (an injected fault lifted as it is recorded; the facade now realizes
faults) and `satisfied/read-evidence` (a `ReadEvidence` poll of `DescribeWorkflowExecution`'s
`pending_nexus_operations` until an attempt is above one, the facade answering attempt 2), each
satisfied by the event that carries the lifted evidence; `static-preparation-rejection/
undeclared-evidence` (`unknown` at the rule's `evidence_id`) and `duplicate-evidence` (`malformed`
at `program.evidence[1]`). The facade catalog carries the Testpilot protocol beside the service.

**The regenerated fixtures.** `async-nexus-case.json` and `nexusSuccessTests-completion-case.json`
change identically (120 lines each): the history read's two lift rules collapse to
`evidenceId: evidence.started` and `evidenceId: evidence.completed`; the Program gains `evidence`
with the two history declarations (`evidenceSource: history`, `historyEvent.attributesField`,
`scope: [run = <fixture>]`, `operation: attributes<arm>.scheduled_event_id`); and the provenance
`localNames` rows reorder because the declarations are visited first. Nothing else changes.

**The `case` block's evidence lines** stay: they are what selects each Case's Actions and resolve
through the realization's catalog into the declarations, so dropping them is the `fixture` form's
job in `.11`, as the task's adjustment anticipated. `pendingAttempts` is declared by the template's
catalog binding and emitted only when a Case's evidence line names it; the seven Queries' Cases
that need it come with `.10`/`.11`.

## Evidence
- Commits: 7d80947
- Tests: `go test -count=1 -tags test_dep ./common/testing/testpilot/... ./tests/testcore/testpilot/... ./tools/umpire/...`; `make umpire-check-testpilot-protocol`; `make umpire-check-testpilot-authoring`; `make umpire-check-case-runtime-conformance`; `make umpire-check-goldens`; `make umpire-check-inventory`; `make umpire-check-retired-vocabulary`; `LEAN_NUM_THREADS=1 make lint-model` (baseline warnings only); `GOLANGCI_LINT_BASE_REV=10c3da6 make lint-code-fast` (0 issues); `CC=/usr/bin/cc TMPDIR=$(cd /tmp && pwd -P) make umpire-check-regression` (exit 0, 11 passing live identities)
- PRs:
