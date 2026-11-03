# Retired lifecycle checkpoint

`original/` contains all 15 activity expectation files reconstructed from Git commit
`af0ebafc571d33651fdf2c008d4415ad72467a0d`.
`.flow/artifacts/fn-227-parameterized-models-declared-worlds/original-artifacts.sha256`
retains their historical hashes. They remain as provenance; no test reads them as a gate.
The old lift refusal is intentionally retained.

The former `checkpoint.rs` (fn-227 R4, fn-237 R3) projected a one-attempt World with
atomic cancellation, one-tick schedule-to-start and start-to-close and no other timeouts
onto these six states and eleven edges, with a changed-old-edge negative control. Its
last committed version is in Git history before fn-253.10.

The owner retired the control on 2026-10-30 (fn-253 Decision Context, R6): the activity
World no longer has a cancellation-mode flag, so the atomic comparison World is gone.
A projection from the cooperative Model would have to account for every pending edge
(pending Cancel, Heartbeat, Complete, Fail, local and global expiry, rejection and
acknowledgement), including the corrected pending Failed/TimedOut outcomes, which the
old graph never had. No total state map makes the two graphs equal: a pending Cancel
followed by acknowledgement resembles one old atomic Cancel edge, but the intervening
pending states and their other exits have no old counterpart. The extension the control
guarded (retries and deadlines over the original lifecycle) is complete, and the
cooperative laws, mutants and the `started_cancel_is_pending` law now pin the behaviour.
