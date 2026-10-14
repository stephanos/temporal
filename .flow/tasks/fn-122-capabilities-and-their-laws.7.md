---
satisfies: [R10]
---
# fn-122-capabilities-and-their-laws.7 Declare Pausable on the fn-119 workflow example once it exists

## Description
The third instantiating machine for `pausedIsNotDispatched`. When fn-119's example workflow Model exists, it declares `Pausable`, `Pollable`, `Closable`, `Terminable` and `Describable` and receives the laws without listing them.

**Cross-spec entry gate:** start only after fn-119.4 (the example Model in the gate) is done. If fn-119.4 has not landed when task 6 closes, block this task with that reason (R10 errors); never drop it silently.

**Size:** S
**Files:** `model/examples/activityworkflow/{Model,Properties,Realization}.scala` (the capabilities declarations and the workflow's named status sets); `model/ir/<example>.json` + `.laws.json`; `model/cases/**` (new generated Cases); `tools/umpire/internal/golden/original.json` (by name).
**Touches:** [model/examples/**, model/ir/**, model/cases/**, tools/umpire/internal/golden/original.json]

### Approach
- Workflow pause semantics cited from `service/history/api/pauseworkflow` (or the handler fn-119 used); `pausedIsNotDispatched` for a workflow means no workflow task is created while paused, declared through `Pollable(dispatch = workflowTask, running = …)` beside `Pausable`.
- Any waiver gets a reason and lands in the sidecar, then the accepted-findings file (task 5).
- No Go file names the example (fn-119 R6's check stays green).

### Quick commands
```bash
make umpire-gen-model && make umpire-check-model && make umpire-check-live-tests
```
## Acceptance
- [ ] The example Model declares `Pausable`, `Pollable`, `Closable`, `Terminable` and `Describable`, and its generated laws pass the gate and run live.
- [ ] `pausedIsNotDispatched` lists three instantiating machines in the catalog test.
- [ ] The example's existing Cases are byte-identical; new Cases and the sidecar are allow-listed by name; fn-119's no-Go-file check passes.
## Done summary
Blocked:
Blocked: deferred by the owner on 2026-10-04 as not needed for the code deliverable (the DSL and its execution). It needs fn-119's workflow example, which is deferred.
## Evidence
- Commits:
- Tests:
- PRs:
