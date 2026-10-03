# fn-114 task 12: select-poll reduction (darwin/arm64, 2026-10-02)

Toolchain build key `2008ea81459afbd1ee019beb4a231968c1af8a24f4b46d4a8426b3188b1a9b87`
(task 11's, unchanged by this task; `toolchain-build-key.txt`). linux/amd64
was not run: no native host in this session.

## Soundness check (`search-reproduction.json`)

`TestRuntimeSearchFixtures` explored each shape of the select-readiness fixture
to exhaustion twice under seed 1: in full, and with every select-poll decision
whose select recorded fewer than two ready cases left unexpanded. A shape is
sound when both reach the same outcomes and deadlocks. The explorer lists a
shape as a no-op only when it is sound and the reduced run skipped something;
the fixture fails if a listed shape is not.

| shape | readiness | full executions | reduced executions | skipped alternatives | outcomes | sound |
| --- | --- | ---: | ---: | ---: | --- | --- |
| blocking-zero-ready | ready 0 | 196 | 96 | 96 | first | yes |
| blocking-one-ready | ready 1 | 68 | 32 | 32 | first | yes |
| blocking-two-ready | ready 2 | 68 | 68 | 0 | first, second | equal, nothing reduced |
| nonblocking-default | ready 0, default | 68 | 32 | 32 | default | yes |
| timer-channel | ready 0, timer | 196 | 96 | 96 | timer | yes |
| closed-channel | ready 1, closed | 68 | 32 | 32 | closed | yes |
| nil-channel | ready 1, nil | 68 | 32 | 32 | first | yes |
| timer-channel-due | ready 1, timer | 200 | 96 | 96 | timer | yes |
| repeated-channel | ready 2, repeated | 68 | 68 | 0 | again, first | equal, nothing reduced |

No deadlock in any run. The full counts for the seven task 2 shapes
(196/68/68/68/196/68/68) equal the baseline task 2 retained under
`runtime-reproduction/final-approved/` on build 6b775117 and task 11's on build
2008ea81; `timer-channel-due` and `repeated-channel` equal task 11's 200 and 68.
Every select polls two cases, so the explorer's list
(`choice.NoOpSelectShapes`) carries the seven sound shapes keyed on two polled
cases; a select that polls more cases, or whose readiness carries another flag
combination, stays expanded until a fixture shape covers it. The fixture holds
its proven shapes and that list to the same set before exploring, so a shape
added to either side alone fails it. The timer sections of the fixture are
unchanged from task 11.

The reduced-versus-full comparison was shown to catch an unsound listing: with
`blocking-two-ready` marked as a no-op and the reduction widened to fewer than
three ready cases, the fixture failed with `reduced exploration reached
[blocking-two-ready first], not [blocking-two-ready first blocking-two-ready
second]` (`red-first.txt`). The run took 54 s; the 1,633 execution directories
and the per-case command log stayed in the session scratch directory.

## Signal suite measurement (`signal-seed11.json`)

`gomad explore --seeds 11 --choices --choice-bytes=64MiB` on `./tests`
`-test.run=^TestSignalWorkflowTestSuiteChasm$` with the manifest's build tags
and schema mount, success retained (`signal-seed11-explore-result.json`; the
163 MB artifact stayed in scratch). The counts below come from projecting the
retained trace through `choice.ProjectReplayPlan` and the explorer's own
`noOpSelectPolls` (a temporary test in the engine package, not committed).

| | before | after |
| --- | ---: | ---: |
| trace bytes | 8,262,720 | 8,262,720 |
| trace records | 86,070 | 86,070 |
| branching decisions (replay plan) | 57,696 | 57,696 |
| decisions the explorer expands | 57,696 | 42,918 |
| alternatives a full expansion of the trace would run | 426,802 | 412,024 |

The trace does not change: recording is task 11's. The 14,778 decisions no
longer expanded are select-poll decisions of the seven listed shapes, 55.2% of
the 26,797 select-poll decisions and 25.6% of all decisions, but 3.5% of the
alternatives because the 30,899 runnable decisions average 12.8 alternatives.
The retained D14 report on build 8d28bd44 (`fn105-d14-qualification-seed11.json`)
had 86,243 records and 57,801 decisions (30,936 runnable, 26,865 select-poll).

Select-poll decisions by readiness: 21,352 known below two (25,539
alternatives), 1,337 unknown (the select blocked and never resumed), 4,108
known at two or more. Of the 21,352, the list covers 14,778; the 6,574 left
expanded are 3,002 two-case selects with a flag combination no fixture shape
has (nil and closed with one ready: 2,919; timer and closed with one ready: 81;
nil and timer with none ready: 2) and 3,572 poll steps of three- to six-case
selects (21,740 of the trace's 23,721 selects poll two cases; 1,523 poll three,
31 four, 217 five, 210 six). These are the candidates for new fixture shapes.

## Decision: no-op decisions stay in the Choice Trace

The listed no-op records are 1,418,688 of 8,262,720 trace bytes (17.2%); every
ready-below-two poll record would be 2,049,792 (24.8%). That saving has no
evidence of bringing any of the eight D15 overflow suites under the 64 MiB cap.
Dropping needs a different recording point: readiness is known after `sellock`,
after the poll draws were taken and recorded, so the runtime would have to
withhold a select's poll records until its result, and a replaying runtime
would have to take poll order from the seed before it can know whether the tape
holds that select, a new choice-wire identity and replay rule coordinated with
fn-110 and fn-112. The records are also what pins a forced prefix and a
divergence to the exact poll step. Kept; no follow-up task.
