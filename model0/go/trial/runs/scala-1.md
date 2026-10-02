# Scala run 1

- Wall time: 264 s. Check runs: 1. Final: pass (re-verified on the saved diff).
- Diff: scala-1.diff (666 lines).
- Deviations the agent reported: the framework's class and `~>` extensions stop at three inputs, so the Model file adds a local four-input extension and keeps `start` a true four-input action; the timer is named `heartbeatTimeout`, because an action and a timer both named `heartbeat` would share row keys (the agent believed nothing checks this; the table build does reject it, with "two steps bind the action class"); two views skip the comparison with the unchanged Go goldens because their line counts differ.
- Pins updated: protocol state and end counts doubled, action count 22 to 32, the `at` helper gains the heartbeat field, four refinement keys gain a segment; goldens re-rendered.
