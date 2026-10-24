# Activity policy verification

Task .3 implements the independently approved deadline and cancellation contract. Offline checks cover atomic and cooperative Worlds; live constructors keep the admitted timeoutless atomic policy until downstream binding work. The parent spec and live cancellation obligations remain open.

## Recorded baseline

Pre-edit base is `88f3f2098e434cdcd3078f8afe792ae1e64b0fe7`. Normal gate checks required fresh runs after the external MILESTONES commit invalidated the .flow-only handoff. Both focused suites passed. Full baseline exits were zero for `make model-check` (444 s), `UMPIRE_BACKENDS=require make model-check` (619 s), and `make model-time` (351 s wall). Timing reported gate 343.896 s and inner loop 6.471 s; the 60 s gate soft target was missed.

## Selected complete Worlds

Each row has complete compiled/interpreted/source-TLC agreement and every registered claim Verified. The timeouts column is ScheduleToStart, StartToClose, ScheduleToClose, Heartbeat. A dash means absent. Positive rows replay zero false claims; separate controls below supply counterexamples.

| Maximum | Cooperative | Timeouts | States | Edges |
| --- | --- | --- | --- | --- |
| 1 | false | -, -, -, - | 5 | 10 |
| 3 | true | -, -, -, - | 18 | 84 |
| 1 | false | 1, -, -, - | 6 | 12 |
| 1 | false | -, 1, -, - | 6 | 12 |
| 1 | false | -, -, 1, - | 6 | 13 |
| 1 | false | -, -, -, 1 | 6 | 12 |
| 3 | true | 3, -, -, - | 27 | 132 |
| 3 | true | -, 3, -, - | 33 | 219 |
| 3 | true | -, -, 3, - | 39 | 264 |
| 3 | true | -, -, -, 3 | 33 | 219 |
| 1 | false | 1, -, 1, - | 6 | 13 |
| 3 | true | -, 1, 1, 1 | 21 | 96 |
| 3 | true | -, 1, -, 1 | 21 | 93 |

The default atomic fixture selects maximum three, one-tick waiting/running deadlines and absent global/heartbeat deadlines. Its native graph has 18 states, 63 emitted edges, 216 evaluations, 153 disabled evaluations and depth six without a cut. Its 17 registered claims verify. The retained suite uses Cases(3 * 7), with 21 original atomic closing/retry witnesses. Pending premises are exercised independently at every cooperative attempt, including both caps.

## Countdown bound and availability

For a selected global budget B, the registered bounds/arming laws give 1<=R<=B in every live state. The universal nonreplenishment law includes Poll, retry, Heartbeat, Cancel and rejections. Tick strictly decreases R while live and expires globally at R=1. Terminal permanence and the Tick before-state law exclude later Ticks. After k actual Ticks before closure, R<=B-k; positivity permits at most B-1 live-after Ticks and the first closing Tick adds at most one. Therefore every arbitrary interleaving has at most B actual Ticks. No history counter, clamp, exploration cut, Tick-only scope or all-transition bound supplies this argument.

Native complete graphs check maxima 1/2/3 crossed with B=1/2/3. Required TLC checks the law bundle at (maximum,B)=(1,1),(2,2),(3,3). Source-only replenishment and skipped-expiry controls refute the unchanged laws and disagree with healthy native results on the same edge. Removing Tick is checked separately through exact availability and independent state-relation disagreement; the numerical upper bound alone would be vacuous.

## Fairness and boundary controls

The selected exact fair lists are Poll(1)+RespondCompleted, Poll(1,2)+RespondCompleted, and Poll(1,2,3)+RespondCompleted. All catalog actions remain allowed. Typed catalog assignments, exact ordinals and one emitted WF obligation per value are checked independently. Timeoutless closure verifies for maxima 1/2/3. Missing completion fairness, omitted current Poll, stale fair payload and existential-union WF expose real heartbeat or stale-Poll lassos. The heartbeat-specific negative strengthens fairness with Heartbeat while removing completion; it keeps every action allowed.

Eight real TLC false-claim witnesses cover both failure booleans at every pending attempt for maxima one and three. Two more retain exact Canceled StartToClose and Heartbeat facts. Source-only cap, nonretryable, heartbeat-closes and global-priority policy mutations refute the original claims. A source-only failure-scope boolean mutation separately refutes the original pending claim; typed metadata checks both original concrete payload ordinals. Wrong/missing expiry facts and an ordinary fact leak fail framework replay; empty/duplicate forged vectors fail exact replay. Traces check full state, typed payload/catalog ordinal, choice, outcome, facts and actual cycle back edges. A separate evidence-only harness imports the committed activity library and lifts its actual source with false scoped witness claims. It replays current ACK at attempts one/three and stale ACK in scheduled/pending states, including exact Accepted/Canceled or Rejected(NotFound) self-loop results and empty facts. All four cases passed with execution exit zero in 10.952 s. Its argv, existing library/source/lock hashes and typed traces are retained; this establishes no live SUT ACK proof.

The independent stale-acknowledgement policy uses NotFound for a live mismatched attempt token. It follows the explicit current-attempt identity precondition and invalid-token domain rationale, separate from stale Poll and repeated pending Client Cancel. It was chosen before any deployment comparison. Current unsolicited and terminal worker acknowledgements are disabled. No server status supplied this choice.

## Retained compatibility and boundaries

The final compatibility World has maximum one, atomic cancellation, one-tick waiting/running budgets and no global/heartbeat deadline. Projection checks every complete state and maps Tick through its exact singleton expiry fact. Only an Accepted, empty-fact, unchanged Started Heartbeat is erased. The exact graph retains one start, six states and eleven old edges; changing an old destination fails the same relation. All 15 original hashes, original manifest, Python verifier and fn-227 hash authority remain exact.

The current guest was refreshed normally to SHA256 `eaf6efd637e0e33b1a9f1e02b24cd8595c1f73706b5695fcde86151e884515f0`. Live maximum_attempts is request-derived; all live timeout defaults remain absent and cancellation remains atomic. No heartbeat/acknowledgement program row, cooperative admission or observation reconstruction is added. The exact unheld cancellation refusal remains. Historical cancellation evidence and the original guest references remain unchanged. The three authorized Go consumer tests first failed with obsolete 18/6 counts against actual 15/5 timeoutless graphs, then passed with exact state/action sets and allocation/publication equality assertions retained. The full Go host module passed.

## Execution receipts and limitations

The first progress-registration control failed with empty old registrations before implementation and passed afterward. A source-admission probe then rejected a claim else-if; the same priority obligations expressed with admitted boolean implications passed. The probe source was removed and both observations retained.

Lane receipts retain actual exits. Native policy has 12 tests; all 13 positive required TLC selections passed in a 385.38 s run whose overall exit was 101 because four test-harness errors remained. Corrected backend controls subsequently passed 5/5 in 39.94 s. Required proof passed 7/7 in 146.10 s; realization passed 91 tests. Missing lane-local TLC jars and mismatched test property metadata are recorded as inconclusive or failing observations and receive no pass credit. The integrated native Temporal suite passed 33 tests with one inherited ignored rebuild control.

Fresh parallel correctness and standards audits found zero correctness defects and one documentation size issue. Moving detailed policy text into the feature guide resolved it; follow-up found zero remaining or new findings.

Final canonical gates all exited zero. The stable-head default run took 441.325 s, required-backend run 960.602 s, and timing run 373.928 s wall. Both default and required logs contain 722 Rust passes, zero failures and four inherited ignored tests, plus active host checks. Timing reported a 367.163 s gate, missing the 60 s soft target, and 6.566 s inner loop, meeting 10 s. The first default run also passed, but its 455.374 s wrapper wall included a deferred receipt coordinator wait after Make finished. Only the coordinator was stopped; Make continued. Its receipt later became invalid after external MILESTONES metadata changes, so the stable-head default was rerun normally.

The refreshed-guest live slice passed in 98.727 s and gaps passed in 362.102 s, including their mandatory parent assertions. The broader archived Scala/IR Go umbrella remains inherited red; these results are scoped to the active gates. No cooperative or timeout live admission is claimed.

All 863 unrelated protected files remain byte-exact. Root authorized only the exact authored expansion mirror, two existing test dependency-list edges in Cargo.lock, and the three exact Go test consumers. All four task worktrees and branches were removed normally after recovery patches, source archives and hashes were verified. Shared and uncertain caches were preserved.

Primary implementation review completed with actual dispatcher and finalizer exits zero and SHIP. All three explicit codex:gpt-6.1-sol:high draws returned SHIP with zero findings and no unaddressed requirements. This is a same-family review. The reviewer read-only Cargo attempts were blocked before compilation; their cached checks and the committed actual gate execution evidence are distinguished in the retained reports. Final gate checks honor all three green receipts at 1ef0402c. The durable execution archive retains all task logs, initial failures, exact typed traces, timing and recovery manifests. The separate ACK source, pinned input/library hashes and exact trace text remain directly readable beside this record. Native ACK binary remains in the owned ignored task scratch with its recorded SHA256; no proof-binary cleanup was performed.
