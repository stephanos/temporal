# Go run 3

- Wall time: 212 s. Check runs: 1. Final: pass (re-verified on the saved diff).
- Diff: go-3.diff.
- Deviations: the same AttemptTimeouts struct for the missing four-input action and timer name heartbeatTimeout; "started attempt" read as every held attempt; the product machine unchanged; the heartbeat not synced with the worker's polling.
- Pins updated: state and end counts doubled, action count 22 to 32, four refinement keys gain a segment, comments updated.
