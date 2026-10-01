# fn-105.16 (D16) review - raw codex bridge on the working-tree diff (commits forbidden)

Reviewer: gpt-5.6-sol at high reasoning effort, `codex exec -s read-only`. Scope: the report docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md, the two D16 entries in .plans/GOMAD_MILESTONES.md, the docs/research/gomad/README.md index row, and the fn105-d16-* evidence.

## Round 1 - NEEDS_WORK

- **BLOCKER** — [Option A](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md:235) cannot fix the retained fixture. At `BURN=700000`, the client and child timers are already armed for 3.3584 s and 3.4337 s. Absorbing the offset when the scheduler later advances to the earliest timer does not change those deadlines, so the client still fires first and the stated [regression criterion](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md:252) cannot pass. Remove Option A or redesign it to prevent/absorb the lead before deadlines are constructed.

- **BLOCKER** — [Option B](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md:225) leaves `context` expiration inconsistent. `WithDeadlineCause` computes `Until` once and arms a fixed timer; subsequent `time.Now` draws would reduce the proposed offset-aware `Until` without retiming that timer. Enough post-creation reads can therefore make `Until(deadline) <= 0`—and make gRPC reject or shorten propagation—while `ctx.Done()` remains open. The proposal needs timers and deadline observations on consistent semantics, plus a reproducer that burns reads after creating the context.

- **SHOULD-FIX** — [The D16 milestone](/Users/stephan/Workspace/temporal/gomad/.plans/GOMAD_MILESTONES.md:187) says all five failing seeds fail “repeatably,” but only seed 5 was rerun. Remove “repeatably” or retain reruns for seeds 12, 20, 21, and 24.

- **SHOULD-FIX** — [The report](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md:212) says the other six forward suites “qualify today,” but no D16 artifact establishes that. Cite retained qualification evidence or weaken this to the manifest expectation and note that D16 did not run them.

- **NIT** — [The native trace table](/Users/stephan/Workspace/temporal/gomad/docs/research/gomad/GOMAD_D16_FORWARD_CLOCK_POLL_DEADLINE.md:118) labels 998 ms as “child before client.” The trace shows approximately 999.8 ms between the child and client deadlines; 998.5–998.8 ms is the margin when the response reaches the client. Correct the label/value and clarify the matching milestone wording.

The causal diagnosis, cited source lines, forward/strict fixture arithmetic, skip retention, seed-11 non-reproduction disclosure, linux/amd64 disclosure, and “nothing fixed” status otherwise check out. R16’s proposed-correction requirement is not yet satisfied because neither correction is internally sound as stated.

VERDICT: NEEDS_WORK
## Author response to round 1

- Option A blocker: accepted. It is now option 3, labelled partial, and the report states that the retained fixture would still fail under it. The fixture criterion no longer names it.
- Option B blocker: accepted. It is now option 2, labelled partial, with the residual stated (deadline passes on the user-visible clock before ctx.Done by the draws made during the context lifetime, about 10 ms in the seed 5 trace) and a second-fixture criterion it fails by construction.
- New proposed correction, option 1: one clock (add the draws to the virtual clock itself, as the fn-103 spec described). Its feasibility is stated as NOT established, because fn-103's first implementation did this and broke the simulation time transport; that failure was not reproduced here. Sources: .flow/specs/fn-103-gomad-seeded-virtual-clock-ticks.md (Edge Cases) and .flow/tasks/fn-103-gomad-seeded-virtual-clock-ticks.1.md (Done summary).
- "repeatably": seeds 12, 20, 21, 24 were rerun; stdout and stderr digests equal the first runs (fn105-d16-evidence.json, forward_rerun_failing_seeds).
- "qualify today": weakened to the manifest expectation; D16 did not run the other suites.
- Native margin: corrected to 999.8 ms between deadlines, with 998.5 to 998.8 ms left when the response reaches the client.

## Round 2 - SHIP

No BLOCKER, SHOULD-FIX, or NIT findings. Round 1 issues are resolved; causal claims, arithmetic, source references, proposed-correction limits, acceptance/R16 coverage, retained skip, seed-11 non-reproduction, and linux/amd64 disclosure all check out.

VERDICT: SHIP