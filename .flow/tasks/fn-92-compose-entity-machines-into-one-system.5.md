---
satisfies: [R5, R6, R7, R8, R9, R10, R14]
---
# fn-92-compose-entity-machines-into-one-system.5 The nexusCaller composition: handlerReply never fires while the worker is stopped

## Description
Declare `handlerWorker`, the worker machine restricted to stop and serve, and the `nexusCaller` composition over the unchanged protocol machine and it; verify the cross-entity claim on `veil` with a field-addressed requirement, pin its reachable count, kernel times, and differential line, and confirm every caller fixture and golden is byte-identical (R5, R6, R7, R8, R9, R10, R14).

**Size:** M
**Files:** `model/Temporal/Feature/Nexus/Caller/Model.lean` (add `machine handlerWorker from: polling restrict: [workerStop, serve]`, `compose nexusCaller`, its Property, Scenario, Limits, `verify` Query under a new `-- authoring:` marker after the existing regions), `model/Temporal/Feature/Nexus/Caller/Tests.lean`, `model/AUTHORING.md` (header region re-quoted, new block for the new marker), `model/TemporalModelTests/SearchDifferential.lean` (expected line; the Caller module is imported at :2)
**Touches:** [model/Temporal/Feature/Nexus/Caller/Model.lean, model/Temporal/Feature/Nexus/Caller/Tests.lean, model/AUTHORING.md, model/TemporalModelTests/SearchDifferential.lean]

### Approach
- `machine handlerWorker from: polling restrict: [workerStop, serve]` (task .4's key), declared in the caller module as the caller's view of the handler's worker; its catalog is pinned to hold no `workerResume`, because an unsynchronized member action would otherwise stay executable and admit a stop, resume, reply path.
- Members `operation: nexusProtocol` (`Caller/Model.lean:406-432`), `worker: handlerWorker`; `sync: workerStop: operation.workerStop ∥ worker.workerStop`, `handlerReply: operation.handlerReply ∥ worker.serve` (`handlerReply` is classed by `input: reply: Reply`, `:84-92`; `serve` is classless and matches every class); `starts:`/`ends:` over member-qualified values; Property `when: handlerReply` bare, which covers every class (`Syntax.lean:393`), with `holds` reading `step.state.worker.phase == .polling`, a field-addressed requirement task .7 makes fixable while the operation member varies; a `scenario` declaration and `limits` under which both backends answer verified within limits; `verify` Query. No set and no case over the composition; the functional, canary, and exploratory sets are untouched.
- Pin the reachable count, the outcome, and the line `Temporal.Feature.Nexus.Caller.<query>: veil default, …` in `SearchDifferential.lean:30-104`; record kernel seconds of the agreement check and law proof, predicate-enumeration seconds of the Property, and elaboration seconds.
- Confirm byte identity of the seven caller fixtures, the canary fixture, `Caller/Fixtures/CallerExploratoryCoverage.json`, and the replay bridge goldens; the seven existing differential lines for Caller stay byte-identical.
- Drift: the composition sits under its own marker (a name no Caller or Worker region uses) placed so no existing region's bytes change; the `import Temporal.Feature.Worker.Model` line sits inside the `-- authoring: header` region `model/AUTHORING.md:28-59` quotes, so re-quote that region and add the new block in this task, so `go test -tags test_dep ./tools/umpire/authoring/...` stays green before task .6 adds the marker-to-file map.

### Investigation targets
**Required:**
- `model/Temporal/Feature/Nexus/Caller/Model.lean:84-111, 343-345, 406-432, 453-494, 596-626`; `Caller/Tests.lean:105-114`
- task .1's `Worker/Model.lean`; task .3's `workerOutage` as the template; task .4's `restrict:`; task .7's field requirement and its fixture Property
- `tools/umpire/authoring/authoring.go:18-24`, `drift_test.go`; `model/AUTHORING.md:1-59`

### Key context
- No `workerResume` in this composition, by the `restrict:`; the operation's timers keep every state unstuck after a stop.
- Bound estimate: 158 reachable operation states × 2 worker phases over about 25 action classes, about 8k evaluations, under `enumerationBound` 16384.
- fn-88.7 also edits `model/AUTHORING.md` (§8, lines 612-686); this task lands after fn-88 closes and rebases on it.

### Quick commands
```bash
cd model && lake build Temporal.Feature.Nexus.Caller.Tests TemporalModelTests.SearchDifferential
make umpire-check-goldens && make canary-check-case && make umpire-check-case-runtime-conformance
go test -tags test_dep ./tools/umpire/authoring/...
make umpire-check-regression
```
## Acceptance
- [ ] `handlerWorker` derives from `polling` with a catalog pinned to hold `workerStop` and `serve` only; `nexusCaller` elaborates over it; reachable count pinned and below the bound; the bare-trigger, field-addressed claim verifies and is pinned; its Temporal differential line reads `veil default` with both backends verified within limits
- [ ] Kernel, predicate-enumeration, and elaboration seconds recorded
- [ ] Caller fixtures, canary fixture, exploration golden, replay goldens byte-identical; the seven caller Queries and their differential lines unchanged
- [ ] The composition sits under its own AUTHORING marker with its block; the header region re-quoted; the Go authoring drift test passes
- [ ] `make umpire-check-regression` passes
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
