# Scala run 3

- Wall time: 280 s. Check runs: 1. Final: pass (re-verified on the saved diff).
- Diff: scala-3.diff.
- Deviations: a local four-input extension keeps start a four-input action; timer named heartbeatTimeout; heartbeat and its timer only on started, as the task states; two views exempted from the Go line comparison while their line counts differ; the product machine unchanged.
- Pins updated: state and end counts doubled, action count 22 to 32, the at helper gains the field, refinement keys gain a segment; goldens re-rendered.
