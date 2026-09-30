# Results: the Go model layer

Run on 2026-09-30 against commit `d08d20140` plus the uncommitted `model/` tree. Plan:
[`.plans/UMPIRE_GO.md`](../../.plans/UMPIRE_GO.md). The Scala implementation built afterwards is
reported in [`model/scala/RESULTS.md`](../scala/RESULTS.md); the tables below show all three sides.

## Outcome

The Go model layer reproduces every answer the Lean model gives for the Nexus caller, emits the same
seven Case files byte for byte, and has an edit loop under three seconds against Lean's three
minutes. Three of the four conditions of the plan's decision rule hold. The fourth, the agent trial,
cannot be decided, because the Lean side of the trial cannot run: the activity protocol machine is
already past the Lean elaborator's 256-state bound. Under the rule, the result is "in between" and
goes to a human with the trade stated below.

## Parity with Lean

| What | Result |
| --- | --- |
| Translated Nexus pins | All pass |
| Translated activity pins | All pass |
| Nexus tables, Definition IDs, refinement rows, witnesses, 889 targets | Equal |
| Target, Scenario, Property and Query canonical strings | Byte-equal; 22 fingerprints equal |
| `nexusCallerTests-*-case.json` fixtures | Seven of seven byte-identical, and each prepares under its derived Profile |
| Activity parity | Product machine only: Lean refuses the 288-state protocol machine |

Two documented divergences from Lean, both searches rather than answers:

- **Product-state count.** The free-path search over `terminalIsFinal` visits 111 product states in
  Go and 171 in Veil, because Lean keeps a fired bit per lowered clause group. Both verify.
- **`pausedIsNotDispatched`.** Go verifies it; Lean's clause lowering rejects it as claiming
  nothing, because it fixes no state, outcome or fact.

## Decision rule

| Condition | Result |
| --- | --- |
| Pin, row-level and Case parity complete | Holds for the Nexus caller; the activity is limited by Lean, not Go |
| Go edit loop under ten seconds | Holds: 2.7 s to test, 10.6 s through the whole gate with lint |
| Every corpus mistake caught no later than `go test`, near the author's line | Holds after one fix; test-time messages name the declaration and the rule, not a line |
| The agent trial takes no more check runs in Go than in Lean | Not decidable: Lean cannot run the task; Go took one check run per trial |

The corpus condition first failed: Go never caught two Properties declared with one name. The
framework now records every Property and Scenario name per machine, and `umpire.Check` reports the
second declaration.

## Error corpus

The twelve mistakes of `corpus/cases.json`, applied one at a time by `corpus/run.py` and reverted
after each. Seconds run from applying the edit to the first failure. These runs overlapped with
builds of another side, so the seconds are indicative.

| # | Mistake | Lean | Go | Scala |
| --- | --- | --- | --- | --- |
| 1 | A step names an undeclared action | compile, 7.6 s | compile, 0.3 s | compile, 0.6 s |
| 2 | A step's input type disagrees with the action's input | compile, 4.7 s | compile, 0.4 s | compile, 0.6 s |
| 3 | A Scenario passes an input of the wrong type | compile, 174 s | compile, 0.3 s | compile, 0.7 s |
| 4 | A match or switch misses a case | compile, 30 s | lint, 3.4 s | compile, 0.8 s |
| 5 | A step moves the attempt count past its bound | pins, 596 s | test, 8.2 s | test, 3.0 s |
| 6 | A recorded fact has no evidence line | pins, 581 s | test, 7.5 s | compile, 0.9 s |
| 7 | A protocol row has no product counterpart under the map | compile, 29 s | test, 7.3 s | test, 3.9 s |
| 8 | A verify Query pairs a product claim with a protocol path and no refinement | compile, 180 s | compile, 0.3 s | compile, 0.9 s |
| 9 | A Query's claim is unreachable on its path | compile, 188 s | test, 7.3 s | test, 3.7 s |
| 10 | A machine has a stuck state | compile, 4.5 s | test, 6.2 s | test, 3.7 s |
| 11 | A canary set names a path with a silent step | compile, 202 s | test, 7.1 s | test, 3.7 s |
| 12 | Two declarations share a model name | compile, 200 s | test, 6.3 s | test, 3.4 s |

- **Lean catches ten at compile time,** because its elaborator runs the Queries, the canary rule and
  the stuck-state check while the file compiles. Its two misses cost a full pins build, about ten
  minutes each.
- **Go catches four at compile time, one by lint, seven by test,** every one within nine seconds.
  The compile-time ones are the typed pairings: step inputs, Scenario inputs, and the refined
  `VerifyRefined`.
- **The plan's hypotheses were optimistic for Lean** on cases 5 and 6, which it expected at compile
  time, and accurate for Go.

## Loop, size and dependencies

Measured by `measure.sh`, one side at a time; results in `results/`.

| Metric | Lean | Go |
| --- | --- | --- |
| Cold build and test | 20 min for the whole workspace after the move | 22.4 s with an empty build cache |
| Warm, nothing changed | 1.4 s | 3.4 s |
| Edit the Nexus Model file, rebuild and test | 191 s | 2.7 s |
| Edit, reaching the pins | about 590 s | same |
| Full gate after an edit | not measured | 10.6 s |

| Lines | Lean | Go |
| --- | --- | --- |
| Framework production | 57,088 authored in `model/lean`, 40,733 generated | 4,935 |
| Models production | 873 for the Nexus caller and worker | 2,194 for all three Models |
| Nexus realization | in the framework count | 385 |
| Tests | 41,101 | 1,411 |

The Lean edit that reaches the pins is the corpus's semantic edit; a comment edit leaves the Model's
compiled output unchanged, so the pins do not rebuild.

Dependencies: no new Go module. Three tools run outside `go.mod`, from `UMPIRE_GO_TOOLS`:
golangci-lint 2.13.1, `exhaustive`, and `go-check-sumtype`. Both checkers must run with
`-default-signifies-exhaustive=false`, and `go-check-sumtype` only checks an interface whose
declaration carries a `//sumtype:decl` line; the first runs of both passed vacuously until those were
set.

## Agent trial

Six runs of the heartbeat task in [`trial/`](trial/README.md), three per side, each a fresh agent
session. Lean could not run: its activity protocol is already past the elaborator's 256-state bound.

| Side | Wall time | Check runs | Reviewer |
| --- | --- | --- | --- |
| Go | 188, 219, 212 s | 1, 1, 1 | changes requested on all three |
| Scala | 264, 292, 280 s | 1, 1, 1 | accepted with a follow-up on all three |

Every run passed on its first check. The Go runs were sent back because the framework has no
four-input action, so each folded two deadlines into one struct input; the Scala runs declared the
missing arity beside the Model and kept the action's real shape.

## Findings on the Lean side

- **The elaborator's state bound.** `Umpire.Command.elaborationBound` is 256, and the activity
  protocol machine has 288 states, so Lean refuses the Model the plan ports.
- **No activity RPC schema is reachable** from the roots in `Temporal/Case/Schema.lean`.
- **The vacuity rule** rejects `pausedIsNotDispatched`, a claim Go and Scala verify.
- **The move to `model/lean/`** left two `protoc` paths in `Testpilot/Protocol.lean` pointing one
  level too shallow; the first full rebuild failed until they were fixed.
- **A comment edit costs three minutes** because the Model file re-elaborates whole; the pins cost
  another six to seven when the Model's compiled output changes.

## Recommendation

Go meets the plan's bar on parity, loop and error coverage, and the trial condition is not decidable.
The trade for a human under GOV-02:

- **For Go:** one language with the server and Testpilot, direct use of the server's protobuf types,
  a loop measured in seconds, and a framework a Go reader follows without new concepts.
- **Against Go:** more of Lean's checks move from compile time to `go test`, the framework needs two
  extra linters to recover exhaustiveness, and nothing proves a claim for every state.
- **The Scala experiment** matches Go on parity and loop, catches two more mistakes at compile time,
  and adds proofs. It costs a JVM toolchain the team does not otherwise use.
