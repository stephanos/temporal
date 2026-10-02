# Runtime reproduction on the unchanged toolchain

The authoritative capture is `final-approved/`, on native darwin/arm64 using
build key `6b775117cc6b13d04c2d00926818e102edb540f8c74ece2794b5e6f3cd2c19ee`.
The fixture source snapshots, binary hashes, exact child commands and exits,
compressed raw trace backings, terminal frames, and forced prefixes are retained.
Linux execution is unverified. No runtime or overlay source was edited or rebuilt.

## C2: changed failure premise, confirmed identity instability

Callbacks A and B immediately execute distinct identifying selects using channels
prepared before either timer is armed. Each timer creator arms a one-second
virtual deadline. Both creators block or exit and main blocks before time can
advance, leaving the two callbacks as the only runnable alternatives when due.
A first-marker adjacent two-way Runnable digest must match two consecutive
parentless `/v1` identities; subsequent associations must use that observed pair.
Callbacks perform no allocation, yield, or blocking operation before the marker.
The buffered marker send cannot reorder marker labels. Missing adjacent Runnable
choices produce no inferred identity. This is a fixture invariant, not a general
inference from trace adjacency.

The seed-6 parent maps A/B to parentless ordinals 3/2. Seed 16 maps them to 2/3.
Valid same-seed alternatives at decision 3/rank 2 and decision 4/rank 1 also map
them to 2/3, and succeed. All 16 one-decision alternative prefixes of this one
seed-6 parent succeed; this does not claim all programs' prefixes succeed.
`final-approved/c2-associations.json` provides select sites, adjacent Runnable
record ordinals and IDs, parentless ordinals, and trace hashes for those four runs.
The source assertion derives the swapped associations and enumerates the parent's
actual alternatives; these counts and ordinals are dated native evidence.

`BuildRankPrefix` changes the final forced decision and truncates the later
suffix. Once that decision is applied, a new later alternative set is not compared
with an old suffix. Thus task 2's predicted valid swapped-prefix alternative-set
failure is changed: stable IDs remain task 5's obligation, while its supported
same-seed prefix success criterion is a preservation check.

A complete seed-6 prefix executed under seed 16 exits 125 with a select-site
mismatch at decision 8. It is a cross-seed experiment; normal replay binds the
recorded seed. Early child experiments also produced a cross-seed alternative-set
mismatch, but their callback association yielded before the marker and was not
sound. `attempts.json` marks these early captures exploratory and excluded from
acceptance. All attempt binaries and raw trace contents remain retained; none is
substituted for the final source-bound capture.

## E3: exhaustive unreduced baseline

Initial readiness is fixture-controlled. Buffered and closed channels are
prepared before the select. The zero-ready and timer shapes become ready only
when virtual time advances by one second, after the polling goroutine blocks;
that eventual completion does not change the readiness count of its initial poll.
Nil cases are disabled. Each shape has one poll decision and one result observation.
Readiness is inferred from this controlled program state, not read from a runtime
field: task 11 has not added that field yet.

| Stable shape | Initial ready cases | Poll decisions | Polls with fewer than two ready | Executions to exhaustion | Outcomes |
| --- | ---: | ---: | ---: | ---: | --- |
| blocking-zero-ready | 0 | 1 | 1 | 196 | blocking-zero-ready first |
| blocking-one-ready | 1 | 1 | 1 | 68 | blocking-one-ready first |
| blocking-two-ready | 2 | 1 | 0 | 68 | blocking-two-ready first, blocking-two-ready second |
| nonblocking-default | 0 | 1 | 1 | 68 | nonblocking-default default |
| timer-channel | 0 | 1 | 1 | 196 | timer-channel timer |
| closed-channel | 1 | 1 | 1 | 68 | closed-channel closed |
| nil-channel | 1 | 1 | 1 | 68 | nil-channel first |

The seven frontiers exhausted after 732 executions, with eight shape/outcome
pairs and an empty deadlock set. Every recorded decision is expanded, including
runtime-owned Runnable decisions and the no-op select polls; no reduction or
start/depth exclusion is used. Prefixes are deduplicated by their canonical SHA.
Seed 1 is fixed. The 2,048-execution-per-shape and 32-decision-per-execution
limits fail the check when reached; they never stand in for exhaustion. Every
shape records `frontier_exhausted`. These finite independent shape programs form
the task-12 unreduced baseline, not a proof of reduction soundness for arbitrary
select programs or repeated-channel shapes.

`final-approved/search-reproduction.json` retains the exact configuration,
per-shape counters, outcomes, deadlocks, and stopping reason.
`final-approved/cases.json` records all 783 expected-passing commands; the one
cross-seed experiment intentionally exits 125. `commands.json` records executable
argv, working directory, controlled seed/choice environment, timeout, and exit.

## Replayable retained inputs

Raw trace files are stored as lossless `trace.gz`, including their complete
one-MiB backing. `compressed-traces.json` records uncompressed bytes and SHA-256;
archives were decompressed and compared before removing task-owned raw duplicates.
The `terminal` and `tape` files remain uncompressed. To decode a retained trace,
decompress its backing, read the big-endian next offset at header bytes 24–31,
and call `choice.DecodeTrace(backing[64:next], terminal, 1<<20)`. Source snapshots
and target binaries stay beside the authoritative trace directories.

The top-level test command was:

```sh
env -u GOMADSEED -u GOMAD3_CHILD_SEED -u GOROOT \
  GOMAD3_RUNTIME_REPRODUCTION_DIR="$PWD/.flow/artifacts/fn-114-gomad-correct-search-path-defects-and/runtime-reproduction/final-approved" \
  tools/gomad3/.toolchain/bin/go -C tools/gomad3 test -tags test_dep \
  ./internal/gomadtool/conformance -count=1 -v
```

Use a new retained directory for a new capture; existing evidence is not rewritten.
Fresh host/runtime gate results and root-lint limitations are recorded in the
parent task's `task-2/` completion evidence.
