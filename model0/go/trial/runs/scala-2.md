# Scala run 2

- Wall time: 292 s. Check runs: 1. Final: pass (re-verified on the saved diff).
- Diff: scala-2.diff.
- Deviations: a local four-input extension keeps start a true four-input action; timer named heartbeatTimeout; "started attempt" read as every held attempt; two views exempted from the Go line comparison because their line counts differ, the diagram's three lines listed; the product machine unchanged.
- Pins updated: state and end counts doubled, action count 22 to 32, the at helper gains the heartbeat field, four refinement keys gain a segment; goldens re-rendered.
