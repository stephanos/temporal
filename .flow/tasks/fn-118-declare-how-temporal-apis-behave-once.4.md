---
satisfies: [R3, R7]
---
# fn-118-declare-how-temporal-apis-behave-once.4 Derive Case waiting from hints in the lowering and refuse undeclared visibility

## Description
The lowering reads the hints and emits a bounded wait for a read after an eventually visible write, one read after a write visible at once, and refuses a read after a write with no declared visibility.

**Size:** M
**Files:** `tools/umpire/lower/realization.go` (commands :495-532, poll :822-853), `lower/internal/producer/build.go:89-104`, lowering tests and fixtures.
**Touches:** [tools/umpire/lower/**, tools/umpire/model/**, model/irgen/testdata/**]

### Approach
- For each path, find what each read waits on in path order across scripts, using task 1's classification: a preceding write (look up its visibility: at once -> task 3's read-once form, eventually -> bounded condition wait) or an asynchronous cause (bounded condition wait with the cause kind's declared bound). Emit task 3's instruction fields with the bound and hint position.
- Refusal names both methods (R3 errors), located at the read's source. A read that waits for an asynchronous cause whose kind has no declared bound is refused the same way, naming the cause kind and the read.
- R7: for each adopted hint, a test removes it from a fixture IR and asserts the affected Case is refused at lowering.
- Realizations still carry explicit polls until task 5; the lowering accepts them, and the final rule (also after task 5) is: an explicit poll is accepted and exempts its read from R3's refusal only when it carries a recorded reason, listed under R4's errors clause; without a reason it is refused. Existing Cases are unchanged here.

### Investigation targets
**Required:**
- `tools/umpire/lower/realization.go:480-860`
- `tools/umpire/lower/internal/producer/build.go`
**Optional:**
- `tools/umpire/lower/migration_golden_test.go`

### Quick commands
```bash
go test -count=1 -tags test_dep ./tools/umpire/lower/...
make umpire-check-cases
```

### Execution constraints
- Existing checked-in Cases unchanged in this task; Program deltas arrive in task 5.
## Acceptance
- [ ] Eventually visible write -> bounded condition wait; visible-at-once write -> single read (fixtures for both).
- [ ] Read after write with no declared visibility is refused naming both methods.
- [ ] Each adopted hint has a removal test that makes lowering refuse the affected Case.
- [ ] Existing Cases unchanged; lowering tests pass.
## Done summary
Behavior phase, lowering half (R3, R7): the lowering derives each read's wait from the declared API behavior and refuses a read a declaration is missing for. Existing Cases unchanged.

What changed
- `tools/umpire/lower/waits.go` (new). Classifies each command: a read is a Poll or a GET call that reads its response; a write is a POST call or a worker script's command, by the cause kind of the script's activation (activity answer, workflow task, handler reply). A NexusCompletion is a write no hint can name. Driver controls and waits are neither. Per script, a read's window runs from the last read to the step its evidence confirms (the producer's confirmations), in path order across scripts.
  - Waiting: if the read's own script performed that step with a write, that write's visibility decides: at once reads once (`ReadEvidence.once`), eventually polls within the visibility's bound. Otherwise the read polls within the sum of the cause bounds in its window, plus the eventual visibility of the write that performs the step. A timer contributes `deadline.<class>` and then `cause.timer`. The interval is the smallest of the hints'. The node's `wait_hints` carry each hint's id and Scala position, and its timeout is their sum.
  - Refusals, reported at the read's position:
    - a write with no declared visibility, naming both methods (or the cause kind and the read method);
    - a step no command performs and no server step declares;
    - a cause kind with no bound;
    - a call that reads after an eventually visible write.
  - POST/GET checks (moved here from fn-118.2): a visibility's write must be bound to POST and its read to GET, each refused at the hint. A command calling a method bound to neither is refused.
- `producer.ReadOnce` and `producer.WaitWithin` (build.go). The adapter applies the derived waits to placed and performed nodes.
- Reader: a poll's interval 0 means "derive", where it used to be refused. A derived poll that writes a deadline is refused. `ACause` is exported for diagnostics.
- Inventory: a hint or server step a wait reads is `in-case` and lists the instructions that read it. One no wait reads stays `unread`.
- Docs: SEMANTICS.md, README.md, `.plans/API_BEHAVIOR_HINTS.md` ("As built by task 4"), and the spec's API Contracts.

Fixtures and tests (`waits_test.go`). The fixture is the checked-in IR with every poll's interval cleared, which is task 5's form. Results, all at 250 ms, and every derived Case is prepared unchanged by Testpilot:
- `await-paused` (2 Cases) and `await-terminated` (3 Cases) read once.
- `await-completed` / `await-failed` wait delivery + answer = 5 s.
- `await-timed-out` waits deadline + slack = 5 s.
- Every Nexus `await-scheduled` waits one workflow task = 5 s.
- `pending-attempts` waits handler reply + eventual visibility = 7 s.

Further tests:
- An eventually visible own write (Terminate made eventual in the fixture) gives a bounded poll.
- R7: one removal test for each of the 15 adopted hints (10 visibilities, 5 cause bounds), each refusing its Case. Removing `cause.delivery` or `cause.timer` is refused by the reader at its server step first, so the removal of those two server steps is tested as well.
- Location of a refusal, POST/GET bindings, calls that read, a closing derived poll, explicit polls unchanged, and no behavior.

Decisions, and why
- Only a poll with no interval derives. An explicit poll keeps its interval and checks nothing, so Case bytes are unchanged. The "recorded reason" for an explicit poll is left to fn-118.5: the IR has no field for it, and requiring one now would refuse every existing Case.
- Calls that read are checked only when the realization declares a behavior. The historical IR (migration and original baselines) declares none, and its non-closing `history` read must lower as before.
- A refusal is an error, not a gap, like the other path errors. It comes before gaps.
- A NexusCompletion before a checked read is refused, because no hint can name it. The plan called this "none needed", but refusing finds a missing hint at lowering rather than as a flaky test. No path does this today.
- A closing poll with no interval reads once. This came from a reviewer FYI.

For fn-118.5
- activity-retry's path takes the Model's `backoff` timer step, which no server step declares, so its derived read is refused. Declared as a delivery, it waits 3 deliveries + 2 answers = 13 s.
- The activity `cancel` and `cancelRequest` Queries are unsupported today. They read after RequestCancelActivityExecution, which no visibility names. With derived polls their refusal is an error that comes before their gaps, so task 5 must either declare that pair or keep their polls explicit.

Review: claude-opus-5-5 at high via `--spec claude:claude-opus-5-5:high`. Writer and reviewer are the same family (Opus). Round 1 was SHIP with 2 P3s, both applied in 31dc3bcb (plus a lint fix in ad7abccf): a span struct for the window's indices, and a no-behavior assertion. The FYI on closing derived polls was fixed too. The FYI on `httpRule` silently returning nil was left as is, because the extension is always linked through the workflowservice imports.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: eb6bfb0fca, 31dc3bcbd8, ad7abccf3e
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB go test -tags test_dep -count=1 -p 2 -timeout 40m ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 0, 280 s, at eb6bfb0 before the P3 follow-ups), go test -tags test_dep -count=1 -p 2 ./tools/umpire/lower/... ./tools/umpire/model/... (exit 0, 334 s, at 31dc3bcb), go test -tags test_dep -count=1 -p 2 -run OriginalBaseline ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0), make umpire-check-cases (exit 0; checked-in Cases unchanged), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main make lint-code-fast (exit 0, final), go test -run 'TestReadsWait|TestEachAdopted|...' ./tools/umpire/lower (exit 0)
- PRs: