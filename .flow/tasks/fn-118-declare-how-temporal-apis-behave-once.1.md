---
satisfies: [R1]
---
# fn-118-declare-how-temporal-apis-behave-once.1 Inventory every wait and settle the hint-aware helper interface for the shared kit

## Description
Interface phase. Produce the committed inventory (R1) and decide the typed method / read-condition seam that fn-112.9's shared kit must leave open. Changes no realization, no schema and no Case.

**Cross-spec entry gate:** fn-117 is closed (spec dependency). Run alongside fn-112's showcase and finish **before fn-112.9 finalizes the shared-kit interface**; fn-112.9 reads this task's interface section. fn-112.9 has no flowctl dependency on this task (cross-spec edges cannot be recorded, and fn-112 files are not edited here), so the conductor must hold fn-112.9 until this task is done. Do not edit fn-112 files or `model/temporal/realize/**`; the interface decision is recorded here and in the spec, and fn-112.9 implements it.

**Size:** M
**Files:** new `.plans/API_BEHAVIOR_HINTS.md` (inventory + interface decision), spec `## API Contracts` and `## Architecture & Data Models` via `flowctl spec set-plan`.
**Touches:** [.plans/API_BEHAVIOR_HINTS.md, .flow/specs/fn-118-declare-how-temporal-apis-behave-once.md]

### Approach
- List every wait, poll, retry, interval and timeout with file:line, its value, the API fact it rests on (server code path or documented behavior) and the proposed hint. Known starting points:
  - realizations: `standaloneactivity/Realization.scala:288-303` `awaitStatus` (250 ms; callers :381-681; pause read-back comment :309-314); `nexuscaller/Realization.scala:287-293` and `:297-307` polls (250 ms), `:417` and `:538` `timeoutMs = 5000`, `:469` `AwaitLearned`, `:523` `AwaitCommand`; DSL defaults `model/umpire/realize/Realize.scala:375,439,477`.
  - lowering: `tools/umpire/lower/realization.go:495-532,822-853`, `lower/internal/producer/build.go:89-104`; validator `tools/umpire/model/validate_realization.go:636,812`.
  - Testpilot: `common/testing/testpilot/temporal/profile.go:84,96` (10000 ms instruction default, 30000 ms total), `temporal/server/session.go:130-211`, `temporal/worker/interpreter.go:117,166`, `internal/execution/evidence.go:277-281`; `temporal/provision/provision.go:59` is infrastructure (list, out of scope).
  - server facts: `chasm/lib/activity/handler.go:190,377-410`, `statemachine.go:219-231`.
  A wait nobody can explain is listed as unexplained for the owner.
- Record before-numbers for R8: polls issued per existing Case and the total declared wait budget, by a reproducible command over `model/cases`.
- Resolve the three questions formerly parked in the spec, from the inventory: method-to-method vs write-to-field visibility; bound on the hint scaled by Profile vs bound on the Profile; whether a Testpilot IR field is needed for the hint position or `CaseProvenance.sources` (`testpilot/v1/case.proto:21,57`) suffices. Record each answer in the spec's Architecture/API Contracts.
- Classify every wait into one of three kinds (spec Architecture, amended at planning): (a) a read after a write, governed by visibility (at once / eventually); (b) a wait for an asynchronous cause on the path - another script's performed command (worker `Finish`/`AttemptFailure`/`AttemptCanceled`, workflow `WorkflowCommand`/`Finish`, handler `NexusReply`), a server timer such as a realization-set deadline, a server retry, or a workflow task - governed by a declared wait bound for that cause kind (a timer-caused wait takes the realization's deadline plus a declared slack); (c) an instruction timeout on a performed command (the two `timeoutMs = 5000`), governed by a declared per-command-kind bound or kept explicit with a reason under R4's errors clause. Expect most of today's polls to be kind (b); the code indicates Pause->Describe is visible at once (`handler.go:202,406`, `statemachine.go:221-229`) - confirm it.
- Map each non-RPC performed command to the API method it realizes (e.g. a worker `Finish` -> RespondActivityTaskCompleted) or to a cause kind, and order writes before reads in path order across scripts (the path's `when` guards), not by script order.
- R3 refuses any read after a write with no declared visibility, so the inventory also enumerates every write->read pair on every existing realization path, including pairs read once today (they need a visible-at-once declaration or the final hints-only lowering refuses existing Cases). Decide how the lowering tells a write from a read from realization structure that exists today (performed commands vs evidence reads/polls), without adopting the read-only candidate hint.
- Not-yet errors: adopt only if a Case uses one, and then define the lowering refusal its R7 removal test asserts (e.g. a condition wait that tolerates an error code the API does not declare as not-yet is refused). If no refusal can be defined, do not adopt it and record why.
- Decide which hints are adopted (only those an existing Case needs) and the exact IR fields; write the helper interface fn-112.9 needs (e.g. read helpers take the typed `MethodDescriptor` and a condition, not a raw poll) as signatures only.

### Investigation targets
**Required:**
- `model/umpire/realize/Realize.scala:63-126,329-480`
- `model/temporal/standaloneactivity/Realization.scala:280-320`
- `model/temporal/nexuscaller/Realization.scala:280-310,410-545`
- `.flow/tasks/fn-112-make-the-standalone-activity-scala.9.md`
**Optional:**
- `common/testing/testpilot/temporal/profile.go:80-100`
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:900-1010`

### Quick commands
```bash
grep -rnE 'intervalMs|timeoutMs|TypedPoll|Poll\(' model/temporal model/umpire/realize
grep -rn 'TimeoutMilliseconds\|PollInterval' tools/umpire/lower common/testing/testpilot | head
```

### Execution constraints
- Read-only for code; only the inventory doc and spec change.
## Acceptance
- [ ] `.plans/API_BEHAVIOR_HINTS.md` lists every wait/poll/retry/interval/timeout in realizations, lowering and the Testpilot Temporal Driver with file:line, value, the API fact, and the proposed hint; unexplained waits are flagged for the owner.
- [ ] Before-counts of polls and declared wait budget for existing Cases are recorded with their command.
- [ ] Every wait is classified as visibility, asynchronous cause or instruction timeout, with the cause and its bound source named; non-RPC commands are mapped to methods or cause kinds; cross-script order is by path.
- [ ] Every write->read pair on existing realization paths is listed with its visibility (at once / eventually) and the write/read classification rule is stated.
- [ ] The three formerly parked questions are answered in the spec; adopted hints and exact IR fields are named.
- [ ] The helper interface for fn-112.9 is recorded as signatures, with no realization, schema or Case change.
## Done summary
Inventory and interface for fn-118 R1, with no realization, schema or Case change.

What changed
- New `.plans/API_BEHAVIOR_HINTS.md`. It lists every wait, poll, interval and timeout (W-1..W-20 in the realizations, plus the DSL, the lowering and the Testpilot Temporal Driver), each with file:line, its value, the server fact it rests on and the proposed hint.
- It classifies each wait as visibility, asynchronous cause or instruction timeout.
- It maps every non-RPC command to its API method or cause kind, and lists every write->read pair on the 16 existing Cases, ordered by path across scripts.
- It records the before-numbers with a reproducible jq command: 17 polls, at most 697 poll RPCs, 52 waits, 440,000 ms declared wait budget, 80,000 ms of it explicit.
- The spec's Architecture and API Contracts now hold the three settled questions, the adopted hints, the exact Umpire and Testpilot IR fields, and the fn-112.9 seam.

Decisions, and why
- Visibility is method to method: every write commits its whole effect in one transaction and every read reads that execution's mutable state.
- Bounds sit with the hint, and a Profile only scales them by a factor. Server latencies differ per cause; environment slowness does not.
- The Testpilot IR gains `InstructionNode.wait_hints` and `ReadEvidence.once`. Provenance cannot carry the hint's position: Testpilot never reads it, its rows have no instruction key, and they are written at line 1.
- Write/read classification uses the API's own `google.api.http` GET/POST binding instead of the read-only candidate.
- Adopted: visibility (8 declarations, one of them eventual: handler reply -> DescribeWorkflowExecution) and cause bounds (delivery, activityAnswer, workflowTask, handlerReply, timer), plus ServerStep declarations.
- Not adopted: not-yet errors (no Case tolerates an error, so no refusal test exists), the per-command instruction timeout, a retry cause, and the other candidates.
- Driver awaits and `await-close` keep the Profile default.
- Projected effect: three polls become single reads, and 14 polls remain.

For the owner
- The two `timeoutMs = 5000` are inert: the Driver never reads them. Recommend deleting them in fn-118.5.
- The Nexus schedule-to-close timer comes from the Profile's default instruction timeout (`worker/typed.go:68`).
- `wait_new_event` on the closing `history` read is inert.
- `pending-attempts` relies on the retry backoff outlasting its interval, and on the default HSM attempt counting.
- `.plans/umpire-api-wait-inventory.md` is superseded and can be deleted.

Review: claude-fable-5-1 (high), round 1, SHIP. I applied its two P3 notes: the full Case-set hash plus the totals command, and the read-once preparation rule.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 48deb9b6f7, 797ba0c2d2
- Tests: jq -c -f wait-budget.jq model/cases/*-case.json (exit 0; 17 polls, 440000 ms; Case set sha256 4fa0f35ab786b1e54c1f8de86cd12c551646ac1770cd66a0e8811dbf71a799cf), go run (scratch) HTTP-binding read of go.temporal.io/api WorkflowService descriptors (exit 0), flowctl claude impl-review --spec claude:claude-fable-5-1:high (VERDICT=SHIP, round 1)
- PRs: