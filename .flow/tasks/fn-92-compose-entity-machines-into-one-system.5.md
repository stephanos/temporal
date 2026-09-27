---
satisfies: [R5, R7, R8, R9, R10]
---
# fn-92-compose-entity-machines-into-one-system.5 The nexusCaller composition: handlerReply never fires while the worker is stopped

## Description
Declare the `nexusCaller` composition over the unchanged protocol machine and the worker machine, verify the cross-entity claim, pin its reachable count and kernel times, and confirm every caller fixture and golden is byte-identical.

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Caller/Model.lean` (add `compose nexusCaller`, its Property, Scenario, Limits, `verify` Query beside the existing sets), `model/Temporal/Feature/Nexus/Caller/Tests.lean`
**Touches:** [model/Temporal/Feature/Nexus/Caller/Model.lean, model/Temporal/Feature/Nexus/Caller/Tests.lean, model/AUTHORING.md]

### Approach
- Members `operation: nexusProtocol`, `worker: polling` (the handler's worker); `sync: workerStop: operation.workerStop ∥ worker.workerStop`, `handlerReply: operation.handlerReply ∥ worker.serve`; `starts:`/`ends:` over the structure; Property: `handlerReply` never fires while `worker.phase == stopped`; `verify` Query with Limits that complete on `reference`. No set and no case over the composition; the functional, canary, and exploratory sets are untouched.
- Pin the reachable count and the outcome; record kernel seconds of the agreement check and law proof and elaboration seconds; if fn-88 adopted its backend, record the outcome on both backends.
- Confirm byte identity of the seven caller fixtures, the canary fixture, `CallerExploratoryCoverage.json`, and the replay bridge goldens.
- The new `import Temporal.Feature.Worker.Model` line sits inside the `-- authoring: header` region that `model/AUTHORING.md` §0 quotes; re-quote that region in AUTHORING.md in this task so the Go drift test (`tools/umpire/authoring/drift_test.go`) stays green before task .6.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Caller/Model.lean` (protocol machine, `protocolWorkerStepStop` stutter row, the sets), `Caller/Tests.lean:109`
- task .1's `Worker/Model.lean`; task .3's `workerOutage` as the template

### Key context
- No `workerResume` in this composition; the operation's timers keep every state unstuck after a stop.

## Acceptance
- [ ] `nexusCaller` elaborates; reachable count pinned and below the bound; the claim verifies and is pinned
- [ ] Kernel and elaboration seconds recorded
- [ ] Caller fixtures, canary fixture, exploration golden, replay goldens byte-identical; the seven caller Queries unchanged
- [ ] AUTHORING header region re-quoted; the Go authoring drift test passes
- [ ] `make umpire-check-regression` passes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
