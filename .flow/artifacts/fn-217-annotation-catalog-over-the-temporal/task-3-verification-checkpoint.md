# Activity catalog verification checkpoint

This records source `55a27ddd024a20ea94c40f0209eeb7d6408794a4`, integrated
through root `53d51ac71cdd52365a86e4e029ee741a47b8ad07`. It does not establish
task completion: formal review subsequently retained a nested failure-field
coverage finding, and the fix requires fresh verification.

All commands below terminated with exit 0, observed through their original
foreground handles. Worker commands used the lane-specific target
`/home/agent/.cache/codex-build/fn2173-activity`; the independent root gate used
`/home/agent/.cache/codex-build/umpire-integrated`.

| Check | Evidence |
| --- | --- |
| Focused realization tests and Clippy | 25 native tests and three supplemental compile-fail controls; warnings denied. |
| `make umpire-check-model-slice` | Six original scenarios and five annotation controls; 42.835s wall. |
| `make model-check` in worker | Passed; 74.863s wall. |
| `make model-check UMPIRE_BACKENDS=require` | Passed; 154.634s wall. |
| Independent integrated `make model-check` | Original root session 79632 returned exit 0; Temporal Go-host step 33.273s. |
| Actual `make model-time` | Exact activity source/crate edit loop; full warm gate 41.444s, inner loop 10.548s, overall command 52.184s. |

Default/required correctness gates overlapped the root gate; their wall times
are not budget measurements. Timing started only after root session 79632
terminated. Both timing soft targets were missed, not passed or waived.
Temporal Go-host startup dominated the warm gate (32.578s), followed by native
tests (6.149s). Source inspection identified repeated guest compilation in
fresh Host instances as a contributor; no isolated compilation time was measured.

A bounded repeat touched the same `temporal/features/activity/api.rs` and ran
the same `cargo test -p umpire-temporal --quiet`, preserving all controls. It
passed in 10.123s; the new supplemental admission harness took 6.74s, including
a nested generated-API/realization build reporting 3.50s. This investigates the
repeated inner overrun without replacing the official timing or relaxing tests.

The required gate selected all 12 admitted generic TLC fixture consumers and
their existing goldens; 21 real semantic-control transcripts were retained.
The pinned 2,274,532-byte TLC jar SHA-256 was
`936a262061c914694dfd669a543be24573c45d5aa0ff20a8b96b23d01e050e88`.
Temporal activity source remains unadmitted by the source lifter and retains
its existing refusal; no Temporal activity TLC execution is claimed.

Detailed active evidence remains under `.flow/tmp/fn2173-*`, including final
live/default/required/model-time logs, inner profile and TLC proof. The worker
handover lists exact commands and source commits. Existing Model, lift, wire
and coverage goldens were unchanged at this checkpoint.
