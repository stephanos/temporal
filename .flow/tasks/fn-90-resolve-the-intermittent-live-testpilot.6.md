---
satisfies: [R6, R7]
---
# fn-90-resolve-the-intermittent-live-testpilot.6 Resolve or close the async-Nexus INCONCLUSIVE Run

## Description
Resolve (3), the async-Nexus Run that ends INCONCLUSIVE or INCOMPLETE, at the path fn-90.3's
signatures name, or close it as not reproduced (R6; R7 only if the cause is external).

**Size:** S if not reproduced; M for one path's fix. If signatures show both (3a) and (3b), fix the more frequent one here and create a follow-up task for the other with `flowctl task create --spec fn-90-resolve-the-intermittent-live-testpilot` (dep on this task; fn-90.7 then depends on it too).
**Files:** (3a): `model/Temporal/Case/Realization/Nexus.lean` (declared `timeoutMilliseconds`), regenerated fixtures. (3b): `model/Temporal/Feature/Nexus/Caller/Model.lean` or the Program realization in `model/Temporal/Case/Realization/Nexus.lean`, regenerated fixtures. Quarantine: `tests/testcore/testpilot/quarantine.go` and `quarantine_test.go` (created by fn-90.5 if it needed one; otherwise here).
**Touches:** [tests/testcore/testpilot/quarantine.go, tests/testcore/testpilot/quarantine_test.go, model/Temporal/Case/Realization/Nexus.lean, model/Temporal/Feature/Nexus/Caller/**, tests/testcore/testpilot/testdata/**, common/testing/testpilot/testdata/case-runtime-conformance/**, tests/testpilot_nexus_caller_case_test.go]

### Approach
- Not reproduced in fn-90.3 on every (3) identity: close with that evidence. No quarantine.
- Path (3a), Run INCOMPLETE from an exhausted instruction timeout: the bound is Case-owned. Correct the declared value in the Nexus realization (`model/Temporal/Case/Realization/Nexus.lean:333-370,442-464`; `respond-async`, `finish-workflow` and the other 5000 ms bindings) to cover the p99 fn-90.3 extracted, with a stated margin. Regenerate with `make umpire-gen-case-runtime-conformance`; check with `make umpire-check-case-runtime-conformance`. Never override it in Go or the test.
- Path (3b), Run COMPLETED with an unresolved rule on the completion-before-start path: first confirm on both switch values that the server records the synthesized started event with the same shape (CHASM `chasm/lib/nexusoperation/operation.go:268-298`; hsm `service/history/hsm/nexusoperations/completion.go:122-155`). Then pick one:
  - the async Model admits the synthesized started event as product behaviour (upstream temporal#6821), or
  - the Program orders the completion after the start is observed (`await-completion-authority` at `Nexus.lean:442` gains an observed-start precondition).
  Record which, and why, in the spec's Decision Context via `flowctl spec set-plan`. Prefer the Model option when the Case's claim is about the server's async protocol, the Program option when the claim is about the handler's normal path.
- Before any Lean edit: `flowctl show fn-88-veil-concrete-checker-as-the-umpire`. If fn-88 is open, block this task naming fn-88 and the change (spec Edge Cases, fn-88 overlap) rather than landing uncoordinated.
- Error cases to keep proving: the worker-outage Case still closes as its Contract says; the forged-completion control (`tests/testpilot_nexus_control_case_test.go:30`) still yields VIOLATED; `CheckSwitchAgreement` still fails on a hsm/chasm divergence.
- Cause outside the repository: upstream issue plus an R7 quarantine for exactly that signature in `tests/testcore/testpilot/quarantine.go` / `quarantine_test.go` (add an entry if fn-90.5 created them, else create them as fn-90.5 describes).
- Afterwards rerun the fn-90.3 loops 3 to 6 at their counts.

### Investigation targets
**Required:**
- fn-90.3 receipt (signatures and latencies for (3))
- fn-90.3 Run-capture latency figures (p99 for the bound)
- `common/testing/testpilot/internal/execution/recorder.go:319-329,377-400` (3a)
- `common/testing/testpilot/internal/verification/correlated.go:645-670` (3b)
- `model/Temporal/Case/Realization/Nexus.lean:330-470`
- `model/Temporal/Feature/Nexus/Caller/Model.lean`

**Optional:**
- `tests/testpilot_nexus_caller_case_test.go:124-210` (switch-value runs)

### Key context
- SEM-16: no retry in Testpilot, the Driver or the evaluator. QLF-05: no evaluator weakening.
- Do not edit `tools/umpire/cmd/umpire-gen-*` or `model/Temporal/API/**` (other sessions).
## Acceptance
- [ ] Either "not reproduced on the successor" per identity, or the named path fixed at its source with regenerated fixtures and the Decision Context updated for (3b).
- [ ] After a fix: loops 3 to 6 from fn-90.3 show zero failures at full count; controls and switch agreement still pass.
- [ ] Any quarantine matches the spec's contract and its offline unit test passes; otherwise "no quarantine needed".
- [ ] `make lint-code-fast` clean; `make umpire-check-case-runtime-conformance` clean if fixtures changed; `make lint-model` (with `LEAN_NUM_THREADS=1`) clean if Lean changed.
## Done summary
Closed failure (3), the async-Nexus Run that ended INCONCLUSIVE or INCOMPLETE, as not reproduced on the successor. The static analysis of paths (3a) and (3b) found no live defect, so there is no source change, no timeout change and no quarantine.

Evidence from fn-90.3 (commits 71559d5f45, 376943907d, receipt ed55fb0646), at b9bb1a58ad, process mode unless noted, 95% CI Clopper-Pearson:
- Retired TestTestpilotAsyncNexusCase, successor TestTestpilotNexusCallerAsyncCompletion (hsm and chasm, 4 Runs each): 0/200, CI 0.00%-1.83%; in-process `-count=4`: 0/52, CI 0.00%-6.85%.
- Retired TestTestpilotAsyncNexusCaseRunsFromItsFixtureNameAlone, successor TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone: 0/200 and 0/52.
- TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone (identity unchanged): 0/200.
- The umpire-run test, which drives the same fixture: 0/50, no kind-(3) signature.
- All 3118 captured Runs closed RUN_DISPOSITION_COMPLETED with VERDICT_STATUS_SATISFIED. No signature was observed, so no root cause is claimed.

Confirmation run at 08de7fb687 in the shared tree: `go test -v -tags 'test_dep integration' -count=5 -run '^(TestTestpilotNexusCallerAsyncCompletion|TestTestpilotNexusCallerCaseRunsFromItsFixtureNameAlone|TestTestpilotWorkerOutageCaseLeavesAnotherQueueAlone)$' ./tests` gave 5/5 PASS per identity, both switch subtests included, and no TESTPILOT-SIGNATURE line. Load was 5.96 at start and 7.77 at end. The harness (`make umpire-repeat-run`) refused to count: fn-90.4's concurrent edit to tests/testcore/onebox.go changed its input fingerprint before the first iteration. The tree also held fn-90.4's uncommitted tests/testpilot_umpire_run_test.go edit, which these three tests do not use.

Path (3a), Run INCOMPLETE from an exhausted instruction bound:
- The recorder closes INCOMPLETE on an execution error or a failed step (common/testing/testpilot/internal/execution/recorder.go:271-282, 319-329, 357-360, 397-400).
- The two 5000 ms bindings on the async path are `respond-async` (model/Temporal/Case/Realization/Nexus.lean:336) and `finish-workflow` (Nexus.lean:463-464). The fixture carries them (tests/testcore/testpilot/testdata/nexusCallerTests-asyncCompletion-case.json, `respond-async` limits at 569-571). Every other instruction there takes the Profile default of 10000 ms (common/testing/testpilot/temporal/profile.go:85).
- The worker driver on the successor does not read either 5000 ms bound. The workflow interpreter runs Finish with no timeout (common/testing/testpilot/temporal/worker/interpreter.go:99-103). The handler interpreter returns the reply with no timeout (interpreter.go:239-247 into typed.go:112-123). The only worker reads of an instruction's TimeoutMilliseconds are the schedule's default schedule-to-close (typed.go:68) and the workflow await (interpreter.go:119). The scheduler bounds only controller nodes (common/testing/testpilot/internal/execution/scheduler.go:525). So exhausting those two bounds cannot be the INCOMPLETE cause on the successor.
- The bounds that do apply are 10000 ms each, on controller windows whose p99 fn-90.3 measured: await-scheduled 279 ms, await-completion-authority 1 ms, complete-nexus-operation 6 ms, await-close 15 ms, whole Run 317 ms (max 345 ms), at host loads of about 5 to 25.

Timeout-sizing note: no timeout changed, because there is no failing data to size one from. The largest enclosing window, 345 ms, is about 7% of the 5000 ms bound and 3.5% of the 10000 ms default. Two gaps stay open for whoever next sizes a bound:
- Run capture does not record the two bounded steps' own start and end events. A bounded-step p99 needs Run capture to record workflow and handler instruction events, or a Driver-side timing field.
- The worker driver ignores the Case-declared 5000 ms on Finish and NexusHandlerReply. That is a Case-authority gap to raise as a follow-up spec, not a cause of (3).

Path (3b), Run COMPLETED with an unresolved rule on the completion-before-start path:
- The race exists. The handler publishes the completion authority before it returns the async start response (typed.go:118-123, publish at interpreter.go:299), so the controller can complete the operation before the server records the start.
- Both switch values record the same history shape. CHASM applies the started transition first when started is still possible (chasm/lib/nexusoperation/operation.go:279-287). It writes NEXUS_OPERATION_STARTED with scheduled_event_id, operation_token and request_id, and dates it with the callback's start time (chasm/lib/workflow/nexus_methods.go:60-72). hsm calls fabricateStartedEventIfMissing before the completion (service/history/hsm/nexusoperations/completion.go:210-213). That function writes the same attributes, plus the deprecated operation_id, with the same event time (completion.go:126-164). In both, the started event takes the next event id, ahead of the completion event.
- The Case cannot tell the synthesized event from a normal one. Its evidence reads only the attribute field and scheduled_event_id (fixture 596-628). Testpilot never reads EventTime (`grep -rn "EventTime\|event_time" common/testing/testpilot`, non-generated sources: no match). So the backdated start time cannot reorder evidence.
- The correlated monitor answers INCONCLUSIVE only when the Run is incomplete (path 3a), the evidence stream is empty, accepted evidence is unprocessed, or an obligation is pending (common/testing/testpilot/internal/verification/correlated.go:644-666). None of these follows from the synthesized started event.
- The Model already admits this server behaviour. `protocolCompleteStep` prefixes nexusOperationStarted when the operation has not started (model/Temporal/Feature/Nexus/Caller/Model.lean:347-361). The Model option in the Approach is therefore already in place, and neither the Model nor the Program (`await-completion-authority` at Nexus.lean:442) needs a change. There is no fix, so the Decision Context needs no entry.

No Model, Producer, realization or fixture changed. The fn-88 coordination clause therefore does not apply, and the worker-outage, forged-completion and switch-agreement controls are unaffected. No quarantine needed (R7): nothing points to an external cause.

Follow-ups, not built here: the Run-capture field for workflow and handler instruction events, and worker enforcement of Case-declared timeouts on Finish and NexusHandlerReply.

Baseline: green, with CC=/usr/bin/clang. `go test -tags test_dep ./tools/umpire/cmd/umpire-repeat/...` and `go vet -tags 'test_dep integration' ./tests` both passed.

GATE_SKIPPED:unittest:docs-only - task commits touch only the task .md receipt
GATE_SKIPPED:live:not-applicable - no source change; the 5x confirmation run is recorded above

stage: impl-review - ran [claude: SHIP first pass, P3 monitor-condition wording fixed after]

## Evidence
- Commits:
- Tests:
- PRs:
