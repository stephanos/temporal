# Retained lifecycle checkpoint

`original/` contains all 15 activity expectation files reconstructed from Git commit
`af0ebafc571d33651fdf2c008d4415ad72467a0d`, verified against
`.flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256`.
`manifest.json` retains their original paths and hashes; `verify.py` checks those hashes.
The source remains in Git at that pin; this tree contains no executable second Model.
The old lift refusal is intentionally retained, even though current source lifting admits activity.

`checkpoint.rs` parses the independent old states and rows. Its comparison World selects
maximum_attempts=1, retains both direct deadline actions, and keeps cancellation atomic.
The state projection keeps phase and removes the constant attempt=1. The action projection
maps Poll(1) to Poll and both RespondFailed retryable assignments to RespondFailed; all
other action names remain exact. Facts project identically: both graphs have empty facts.
Outcomes and destinations remain exact. Canonical sets deduplicate payload assignments
that have the same projected source/action/outcome/facts/destination; this deliberately
compares semantic relationships rather than catalog evaluation counts or BFS IDs.

The test compares exact starts, all six reachable states, and all eleven projected edges,
including ScheduleToStart and StartToClose to TimedOut and four terminal Cancel
Rejected(NotFound) loops. The old report independently retains starts=1, reached=6,
emitted=11, evaluated=36, refused=25 and depth=2. A mutation changes the old
ScheduleToStart destination and fails this same graph comparison.

This is fn-227.3's one-attempt/direct-deadline checkpoint. fn-237.3 must state and verify
the final Tick-era projection with its final compatibility World; this proof does not
establish that later relation.
