# Go run 2

- Wall time: 219 s. Check runs: 1. Final: pass (re-verified on the saved diff).
- Diff: go-2.diff.
- Deviations: the same four-input workaround (an AttemptTimeouts struct as start's third input) and timer name heartbeatTimeout as run 1; the timer fires only on started (run 1 used every held attempt); the product machine left unchanged so the Lean parity dumps still hold.
- Pins updated: state and end counts doubled, action count 22 to 32, four refinement keys gain a segment, state builders set Heartbeat.
