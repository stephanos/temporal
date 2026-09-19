---
satisfies: [R17]
---
# fn-83-author-a-live-case-from-a-model-file.17 Reduce case template arguments to the ones a test depends on

## Description
Reduce a `case` block's template arguments to the ones a test depends on, and make the template reject evidence its Program cannot produce (R17). After .15 and .16 the Nexus success block reads:

```lean
case asyncNexus
  realizes: completion
  template: nexusOperation
  service: "umpire.case.service"
  operation: "complete"
  responds: async
  evidence:
    awaitStart ← nexusOperationStarted
    awaitSuccess ← nexusOperationCompleted
```

`service` and `operation` are names the generated workflow and the in-worker handler only have to agree on; nothing in the model, the Contract or the Go tests reads them. `responds` changes the Program and the recorded history. The `workflow` template's `type` is the same kind of agreed name: the Testpilot worker registers whatever the Program names, and no Go test refers to it. After this task:

```lean
case asyncNexus
  realizes: completion
  template: nexusOperation
  responds: async
  evidence:
    awaitStart ← nexusOperationStarted
    awaitSuccess ← nexusOperationCompleted
```

**Size:** S
**Files:** `model/Temporal/Case/Template/NexusOperation.lean` and `Workflow.lean` (defaults, per-form evidence sources), `model/Temporal/Case/Template.lean`, the Temporal `case` command (template keys), `model/Temporal/Case/Tests/Template.lean`, `model/Temporal/Feature/Nexus/Success/Model.lean`, `Tests.lean`, `.flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md` and `.6.md` (their `case` blocks)
**Touches:** [model/Temporal/Case/**, model/Temporal/Feature/Nexus/Success/**, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.5.md, .flow/tasks/fn-83-author-a-live-case-from-a-model-file.6.md]

### Approach
- **Defaults.** `nexusOperation` takes no service or operation argument; it uses today's values (`umpire.case.service`, `complete`) so the async-Nexus Program is byte-identical. `workflow` takes no type argument; pick a default and record whether it moves the worker-outage fixture (it is re-authored in .5, which regenerates it anyway). Do not keep the removed keys as optional overrides.
- **Template keys.** Each template declares the keys it accepts (`nexusOperation`: `responds: sync | async`; `workflow`: none). The `case` command rejects a key the chosen template does not declare, a missing required key, and an unknown value, each in place and listing the template's keys or values. An unknown template name lists the declared templates, read from one list rather than a literal message.
- **Evidence per response form.** A template's admitted evidence kinds depend on its form. Confirm in the server's Nexus history handling (under `service/history`) which events a synchronous and an asynchronous completion record. If a synchronous completion records no `NexusOperationStarted`, `responds: sync` admits only `nexusOperationCompleted`, and an `awaitStart ← nexusOperationStarted` line under `sync` rejects at that line listing the admitted kinds, instead of compiling a clause no Run can confirm. Record the server citation in the receipt.
- **Bytes.** The async-Nexus fixture is byte-identical. The sync form has no checked-in fixture until .6.

### Investigation targets
**Required:**
- `model/Temporal/Case/Template/NexusOperation.lean` — `nexusOperation`, `asyncProgram`, `syncProgram`, `sources`
- `model/Temporal/Case/Template/Workflow.lean` — `workflow`, its `workflowType` uses
- the Temporal `case` command — `templateTerm`, the evidence-kind check
- `model/Temporal/Case/Tests/Template.lean` — hook and template pins
- server Nexus history recording under `service/history` (started and completed events for sync and async operations)

### Key context
- Depends on .16 so it edits the respelled `case` grammar with the fixture slot already gone. .14 follows it so compile-time diagnostics cover template keys.

## Acceptance
- [ ] `nexusOperation` accepts only `responds:`; `workflow` accepts no argument; neither keeps the removed names as optional overrides
- [ ] An undeclared key, a missing required key, an unknown value and an unknown template each reject in place listing what the template declares, pinned by `#guard_msgs`
- [ ] Admitted evidence kinds depend on the response form as confirmed in the server source (citation in the receipt); an evidence line the chosen form cannot produce rejects at that line, pinned by `#guard_msgs`
- [ ] `Model.lean` uses the reduced block; the async-Nexus fixture is byte-identical
- [ ] `.5` and `.6` `case` blocks use the reduced template forms
- [ ] `cd model && lake build`, `make umpire-check-case-runtime-conformance` pass


## Done summary
Blocked:
Blocked 2026-09-10; superseded by fn-85 ("Model side effects as typed actions and run query sets").

The per-Case `case` block (one Query, one hand-picked whole-Program template, per-Case evidence lines) is replaced by:

- **Side effects in the Model.** Entities with identity; actions a party performs, with input classes, examples, an optional schema and results; machines that keep each entity's state and rows; observations that confirm rows. Request fields that decide the outcome are Model behavior, not binding detail.
- **A Temporal realization** in `Temporal.Case` that binds actions, results, observations, timers, setup parameters, switches and parties to RPCs, Testpilot instructions, history events and dynamic config. The Producer assembles Program and Contract from a Query's path; whole-Program templates and the `case` command are removed.
- **Query sets per purpose.** A set binds each party to `driven` or `observed`; a functional set compiles to one Case per Query; canary and exploratory sets are admitted for fn-70/fn-29 and fn-33.

fn-85's final task closes this task as superseded and names where its concern went. Design record: `model/Temporal/Feature/Nexus/DESIGN.md`.
## Evidence
- Commits:
- Tests:
- PRs:
