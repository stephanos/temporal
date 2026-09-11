---
satisfies: [R6, R2]
---
# fn-83-author-a-live-case-from-a-model-file.6 Translate the upstream synchronous Nexus completion test

## Description
Re-author `TestNexusOperationSyncCompletion` as a Model file using only the six commands and its `enum` (or `inductive`) declarations, and no `Setup`, on the `nexusOperation ... responds sync` template (R6, R2). Write `COVERAGE.md` beside it mapping every upstream assertion to a Property clause, a Contract rule, or a Known Gap, and add the one live Go test. The commit is the spec's two-authored-files measurement, so keep every other change out of it. Depends on .5 so the aggregator import and the syntax are final.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Sync/Model.lean` (new; final path a task decision), `model/Temporal/Feature/Nexus/Sync/COVERAGE.md` (new), `tests/testpilot_sync_nexus_case_test.go` (new), generated `tests/testcore/testpilot/testdata/sync-nexus-case.json`, plus the aggregator import if `Temporal.lean` needs one (land that import in a separate preparatory commit so the measurement commit stays at two authored files)
**Touches:** [model/Temporal/Feature/Nexus/Sync/**, tests/testpilot_sync_nexus_case_test.go, tests/testcore/testpilot/testdata/sync-nexus-case.json, model/Temporal.lean]

### Approach
- Model: `scheduled → succeeded` on one Action `awaitResult`, outcome `completed`, no Fact that only mirrors a state; Property requiring state, outcome, and fact; Scenario `actions exactly [result: awaitResult]`; `limits`; `find` Query; `case ... fixture "sync-nexus" as nexusOperation "service" "operation" responds sync evidence awaitResult ← history nexusOperationCompleted`.
- Insert `-- authoring: <name>` marker comments around each block; .8's tutorial quotes these regions.
- COVERAGE.md rows: workflow result equals the handler value (Property clause via the finish outcome, or Known Gap if the Contract cannot read the finish value); completed history event present (carried); the handler link on the completed event (Known Gap unless the template can attach links); the admin-service `DescribeMutableState` sub-state-machine assertion (Known Gap, kind white-box, with one sentence on why no black-box Run can observe it); and a row recording that the upstream handler is an external Nexus server while this handler runs in the Testpilot worker. Every Known Gap goes into `authoredKnownGaps` on the Query so it lands in Provenance.
- Do not weaken the vacuity rule for this translation; record instead.
- The live test is `runCase(t, env, "sync-nexus")` plus Verdict assertions, following the outage test's shape.

### Investigation targets
**Required:**
- `tests/nexus_workflow_test.go:497-566` — the upstream test and its four assertions
- `model/Temporal/Feature/Nexus3/Nexus.lean:51-93` — the success Model to pattern after (respelled by fn-82)
- `model/Temporal/Feature/Nexus/EVIDENCE.md` — the existing coverage-record shape (`COVERAGE.md` after fn-82)
- `tests/testpilot_worker_outage_case_test.go:31-41` — the live-test shape to copy

**Optional:**
- `model/Umpire/Case/Compiler.lean` — how `knownGaps` reach Provenance
## Acceptance
- [ ] The Model file uses no Lean beyond the six commands and `inductive` declarations; `lake build` passes
- [ ] `COVERAGE.md` maps all four upstream assertions plus the handler-placement row; the mutable-state row is a white-box Known Gap present in the fixture's Provenance
- [ ] The measurement commit touches exactly the Model file and the live test as authored files (fixture generated)
- [ ] The live test passes under `make umpire-check-live-tests`
## Done summary
Blocked:
Blocked 2026-09-10; superseded by fn-85 ("Model side effects as typed actions and run query sets").

The per-Case `case` block (one Query, one hand-picked whole-Program template, per-Case evidence lines) is replaced by:

- **Side effects in the Model.** Entities with structured state; actions with a kind (`call`, `command`, `reply`), a party, a schema, input classes with representatives, and result classes; observations that confirm steps; machines of step rows. Request fields that decide the outcome are Model behavior, not binding detail.
- **A Temporal Realization** that binds actions, result classes, observations, setup parameters and parties to RPCs, workflow commands, handler replies, history events and dynamic config. The Producer assembles Program and Contract from the witness; whole-Program templates and the `case` command are removed.
- **Query sets per purpose.** A set binds each party to test or environment; a functional set compiles to one Case per Query; canary and exploratory sets are admitted for fn-70/fn-29 and fn-33.

fn-85's final task closes this task as superseded and names where its concern went. Design record: `model/Temporal/Feature/Nexus/DESIGN.md`.
## Evidence
- Commits:
- Tests:
- PRs:

## The white-box Known Gap representation (decided by .11)

**No new kind.** `UMPIRE4_SPEC.md` enumerates the four Known Gap kinds normatively ("A missing or
unsupported Capability, input, interpretation, or claim"), so a `whiteBox` kind would need a GOV-02
amendment for something the existing vocabulary already carries.

The admin-service mutable-state assertion is `interpretation`: the Run records what the Case can
observe, and nothing in it interprets the server's internal mutable state. Write it as

```
gap interpretation "white-box-mutable-state"
  detail "The upstream assertion reads mutable state through the admin service; a Case observes only
    what its Program records, so no black-box Run can establish it."
```

which produces the code `<family>.known-gap.white-box-mutable-state`. The detail is the place that
says "white-box", and `COVERAGE.md` names the same code in its row.
