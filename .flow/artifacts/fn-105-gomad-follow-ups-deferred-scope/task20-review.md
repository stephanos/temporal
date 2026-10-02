# fn-105.20 review (raw codex bridge, gpt-5.6-sol at high, read-only, working-tree diff; commits forbidden)

## Round 1 - 2026-10-01 - NEEDS_WORK

1. [blocker] [GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:244](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:244) claims Variant T “removes the dependence on incidental elapsed time in every mode.” The client-derived deadline is indeed no earlier than the server deadline, so expected rejections are deterministic. However, all other heartbeats are expected to succeed without proving they precede the server deadline: `OriginalScheduledTime` starts before the poll returns, while `chainDeadline` starts afterward. Sufficient poll/RPC delay can therefore wrongly reject an expected acceptance; retained evidence shows only a ~0.93 s margin on a lightly loaded host, and the 20 s run timeout leaves roughly 1.7 s overall. Resolve by providing an accepted-side semantic bound or controlled server clock, or explicitly retain this as a residual timing limitation and stop presenting T as time-independent acceptance of R20.

2. [should-fix] [fn105-d20-variant-t-test.diff.txt:31](/Users/stephan/Workspace/temporal/gomad/.flow/artifacts/fn-105-gomad-follow-ups-deferred-scope/fn105-d20-variant-t-test.diff.txt:31) adds `s.Equal(expectTimeout, isNotFound, ...)`. This violates the repository’s `require` rule and continues down the branch selected by the unexpected result after the assertion fails. Use a fatal `s.Require().Equal(...)` before branching.

3. [should-fix] [GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:140](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:140) says, “The clock advances only in the test's own `time.Sleep`.” The strict history shows the recovery poll advancing virtual time by 3.002 s while waiting for the start-to-close timer. Restrict the statement to active heartbeat chains or acknowledge timer-driven recovery advancement.

4. [should-fix] [GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:266](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:266) says Variant P changes behavior “at an instant a wall clock does not reach,” and line 272 says every option is sufficient alone. Equality is possible on discrete wall clocks; it was merely not observed here. Moreover, P was not qualified under forward and no option was tested on linux/amd64. Limit these claims to the tested darwin scope.

5. [nit] Two semantic citations are incomplete. [Line 72](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:72) cites only the `RealTimeSource.Now` implementation, not the handler/provider wiring; add `api.go:89` and `common/resource/fx.go:152-153`. [Line 87](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:87) cites code that merely sets `scheduleWorkflowTask = true`; the actual scheduling is at `timer_queue_active_task_executor.go:477,1013-1027`.

6. [nit] [GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:45](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D20_HEARTBEAT_TIMEOUT_COUNTING.md:45) says the evidence index “records every command.” It records parameterized command templates and run summaries, not every literal invocation. Correct the claim or retain an exact invocation log.

7. [nit] [.plans/GOMAD_MILESTONES.md:212](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:212) says strict produced one rejection “on all 16 seeds tried.” Evidence includes leaf seed 17 in addition to seeds 1–16, so 17 distinct strict leaf seeds were tried. Correct the count or name the 1–16 sweep explicitly.

| Acceptance | Status | Basis |
|---|---|---|
| Reproduce native/Gomad behavior with retained histories and timing | MET | Counts, deadlines, event numbers, margins, pass/fail totals, seeds 11/17, and qualification reports match the retained evidence. |
| Establish timeout/reset semantics and cause | MET | Source supports strict `After`, inherited/reset `OriginalScheduledTime`, inline task timeout, and the test-timing cause; the equality-intent limitation is disclosed. |
| Provide a time-independent semantic correction | NOT MET | Variant T fixes rejection timing but retains an unbounded accepted-heartbeat timing assumption. |
| Record cause, owners, next actions, and qualification criteria as open work | MET | Report and milestone retain explicit owners, steps, gates, alternatives, and limitations. |
| Keep skip and preserve timeout/recovery assertions | MET | Manifest skip remains; count, 47-event history, two recoveries, and completion remain asserted. |

VERDICT: NEEDS_WORK
## Worker response to round 1

1. Accepted. The report no longer presents variant T as time-independent. The summary, "The proposed condition" (new list of the three remaining timing assumptions, including that chainDeadline is read after the poll returns so poll latency counts against the accepted-side margin, and that removing the assumption needs a controllable server clock or a rejection-driven loop, neither built), the "Native comparison" paragraph on the opposite bound, the recommendation, and the milestone row now state that T removes the need for latency in the two rejections and keeps the existing latency ceiling (about 0.93 s per chain) for accepted heartbeats. The limitation is retained explicitly, as the finding allowed.
2. Rejected with evidence. In this suite `s.Equal` IS fatal: `parallelsuite.Suite` embeds `*require.Assertions` and sets `s.Assertions = require.New(assertT)` (common/testing/parallelsuite/suite.go:59, 95; the type comment says "require-style (fail-fast) assertions"). The retained failing runs confirm it: the unchanged test stops at `s.Equal(2, hbTimeout)` (tests/workflow_task_test.go:110) and never reaches the history assertion. `s.Require().Equal` would be equivalent; the existing file uses `s.Equal`/`s.NoError` throughout. The report now states this next to the per-heartbeat assertion.
3. Fixed: the sentence is restricted to within a chain and names the 3.002 s recovery advance.
4. Fixed: option P's risk now reads "at the exact deadline instant, which no native run here reached", and the closing sentence is limited to the darwin/arm64 runs, noting P was not qualified under forward and nothing ran on linux/amd64.
5. Fixed: added api.go:89 and common/resource/fx.go:152-153 for the time source, and timer_queue_active_task_executor.go:477, 1013-1027 for the scheduling.
6. Fixed: "records the command templates and every run with its seeds and outcome".
7. Fixed: the milestone row says "every leaf seed tried (1 to 17)"; the report summary says "leaf seeds 1 to 17, and seeds 11 and 17 in the whole suite".

## Round 2 - 2026-10-01 - SHIP

No findings. All round-1 findings are resolved.

| Acceptance | Status | Basis |
|---|---|---|
| Native/Gomad reproduction and evidence | MET | Seeds 11/17, histories, deadlines, counts, event numbers, and elapsed times match retained artifacts. |
| Semantics and cause | MET | Source confirms strict `After`, inherited/reset deadlines, inline-task recovery, and a test timing assumption—not a demonstrated defect. |
| Semantic correction | MET | Variant T deterministically places required rejections after the deadline while preserving history and recovery; residual accepted-side timing limits are explicit. |
| Owners, actions, and qualification criteria | MET | Corrections remain open, with owners, next steps, gates, and platform limits recorded. |
| Skip and assertion strength | MET | The tracked skip remains; timeout count, exact outcomes, 47-event history, recoveries, and completion remain asserted. |

Review note: the configured flow-next backend could not launch because the read-only sandbox denied its temporary file; verification was performed directly against source and artifacts.

VERDICT: SHIP