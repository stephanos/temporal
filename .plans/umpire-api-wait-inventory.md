# Existing API waits: fn-118 source inventory

Snapshot: 2026-10-03, current typed fn-117 worktree. This records the waits used by
the existing Cases and their execution path. It is preparation for fn-118 R1,
not proof that its complete inventory or server-behavior validation is finished.
Flow remains the record for that work.

## Authored waits

Three poll declarations produce 17 poll instructions in 15 of the 16 checked-in
Cases: eight activity polls and nine Nexus polls. Each declaration uses a 250 ms
interval. None sets a poll command timeout; the current local and canary Profiles
supply 10,000 ms and one instruction attempt.

| Declaration / read | Condition and producer | Current source |
| --- | --- | --- |
| Activity `awaitStatus` / `DescribeActivityExecution` | `info.status` equals the requested status after pause, worker completion/failure/cancel, terminate, or deadline expiry | `model/temporal/standaloneactivity/Realization.scala:65`, `:287`, `:370`, `:426`, `:466` |
| Nexus `awaitScheduled` / `GetWorkflowExecutionHistory` | A scheduled Nexus event appears after a workflow starts and its WorkflowCommand schedules the operation | `model/temporal/nexuscaller/Realization.scala:96`, `:264`, `:387`, `:464` |
| Nexus `pendingAttempts` / `DescribeWorkflowExecution` | A pending Nexus operation has attempt 1 after a retryable handler answer | `model/temporal/nexuscaller/Realization.scala:111`, `:275`, `:467`, `:593` |

The activity pause script stops the worker until unpause: an already delivered
attempt can beat the pause. The held-delivery race uses the same read helper.
These are authoring constraints, not independent proof of the server's visibility
contract (`standaloneactivity/Realization.scala:305`, `:670`).

Nexus `awaitClose` is a `GetWorkflowExecutionHistory` call with `waitNewEvent=true`
and the close-event filter, followed by a separate history read for evidence
(`nexuscaller/Realization.scala:185`, `:225`, `:482`). Its blocking behavior is
asserted by the realization and still needs its server basis recorded. The
forged-completion control's `DescribeWorkflowExecution` read is one-shot, without
a poll (`:669`).

The two authored 5,000 ms limits are on Nexus handler `NexusReply` and
`finish-workflow`, not on the three polls (`nexuscaller/Realization.scala:416`,
`:531`). Their purpose must be distinguished from visibility bounds before hints
are chosen.

## Lowering and execution

- Typed calls use `MethodDescriptor[Req, Rsp]`; polls use an `EvidenceRef`.
  The lifter validates the generated unary method and message descriptors
  (`model/umpire/realize/Realize.scala:63`, `:433`, `:465`;
  `model/lifter/Realizations.scala:123`).
- Lowering copies only a positive explicit timeout and preserves the poll
  interval (`tools/umpire/lower/realization.go:494`, `:823`).
- Timeouts resolve from the Case or Profile. Runtime validates a positive
  interval no greater than that timeout (`common/testing/testpilot/internal/execution/dataflow.go:232`,
  `evidence.go:251`). Local Profile defaults are at
  `common/testing/testpilot/temporal/profile.go:80`, `:145`; canary repeats the
  policy at `tools/canary/casebinding/casebinding.go:100`.
- The scheduler passes the interval to the session, which repeats successful
  reads whose predicate is false. Empty or unmatched reads are unsatisfied.
  A gRPC error ends the poll immediately; it does not mean "not yet"
  (`execution/scheduler.go:843`, `execution/evidence.go:312`,
  `temporal/server/session.go:121`, `:138`). Both scheduler and session impose
  context deadlines (`execution/scheduler.go:649`, `temporal/server/session.go:193`).

No currently used not-yet error is established by this code. Such a hint stays a
candidate unless the complete inventory and server evidence justify adopting it.

## Decisions the inventory must support

A method pair alone may not identify a visibility relationship: one describe
method serves several conditions and causes, while history serves both a scheduled
event poll and a blocking close read. Producers also include worker answers,
workflow commands, handler replies and timers, not just service RPCs. The final
hint representation must cover the existing conditions without pretending those
producers are service methods.

Before changing the schema, establish the server basis for each adopted hint,
account for Driver-internal waits and retries, distinguish API visibility from
the 5-second local instruction limits, and determine how hint position and
Profile scaling reach failure reports. Do not substitute this snapshot for that
remaining R1 work or change Case Programs during fn-112's structural freeze.

## Source fingerprints

SHA-256 at the snapshot above:

| Source | SHA-256 |
| --- | --- |
| Activity realization | `a258bea4b7ed6c8ba0099f23caabc255d6fd71f306adee8fdd44c4e39c343040` |
| Nexus realization | `68bb1d69ed223580645828c1f97c59b7d28ef8b710e4208cdb35c7809d634cd3` |
| Go realization lowering | `6e60f4be551223dde7462b64c6e92996d235496856c519d4e7a4167ee32da459` |
| Local Temporal Profile | `6d018a64de1874f5336beab3f1359bbde1b05a807edebfc1e5de44998ddf9e4c` |
| Temporal server session | `92580973811d76b0a05aaf0512d6ece38ae351356e29bda7948becf2062b2016` |
| Evidence execution | `3db7eb89ebe284c98c2877da9c8546f10ea24a733af73941dfc4d4ec013a9024` |
