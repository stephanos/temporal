# Reproducible generated-source differential spike

This retained experiment has passing complete graph, mutation, refusal, shrinking
and process-failure controls. The verdict and measured limits are in
`model/design/generated-differential.md`. No production crate or gate is registered.

Run from the repository root with the existing pinned toolchain and a TLC 1.7.4 jar.
The backend verifies the jar's SHA-256 before use. The destination must be new.

```sh
unset BASH_ENV
mkdir -p .flow/tmp/fn255-spike/evidence/tmp
TMPDIR="$PWD/.flow/tmp/fn255-spike/evidence/tmp" \
  bash .flow/tmp/fn255-spike/run.sh "$PWD/.flow/tmp/fn255-spike/evidence/reproduced" \
  --seed 255003 --cases 2 --batch-tlc --execute --shrink-controls \
  --tlc-jar /absolute/path/tla2tools-1.7.4.jar
python3 .flow/tmp/fn255-spike/check-runner.py
```

Omit `--batch-tlc` for individual TLC checks; use the same seed and count to compare
identical sources. Omit `--execute` to print source only. Counts are restricted to
one or two Models. Dependencies build offline into the task-owned `evidence/target`;
subsequent crates and shrink candidates reuse it. Provision dependencies before
admitting gate timing. The runner disables quick/backend cache reads, removes
BASH_ENV, sets two compiler jobs, and trusts only its copied task tool metadata.
It records actual source hashes, stdout/stderr, timings and sampled Java argv.
Each compile has a 120 s spike-only ceiling; runtime has 45 s per selected case.
The runner cancels on less than 30 GiB free disk. Required-backend execution must
share the conductor's JVM allocation with other active lanes.

Each generated Model has two enum fields, a reachable count bound of one or two,
an optional two-value Finite wrapper, guarded arithmetic, rejection, facts and an
invariant. Its declarative starts have one or two members. Compiled and interpreted
checks complete independently. TLC parses the complete printed source independently.
Complete snapshots include starts, states, action/choice/successor/outcome/facts and
claim verdicts. A case selector lets nested emitted modules share one TLC process
without constructing expectations from native states.

The complete extraction proof is limited to this generated family. Refuse is enabled
in every state, so the probe excludes synthetic completion and has no deadlocks.
All successful graphs satisfy 36 states, 144 emitted edges and 290 augmented
assignments per Model. The raw assignment check occurs after bounded dump decode.
Optional tree-only construction, dropped-guard mutation and general deadlock handling
remain outside this spike. Matching actual cap refusals fail, and both fact-only and
outcome-only mutations are caught while reached states and claim verdicts stay equal.
Three source reductions recompile and retain all mutation controls; they make no
global source-minimality claim. Native-only reduction launches no TLC.

The evidence archive preserves sources and the final measured run directories while
excluding targets, jars and scratch. Verify `archive-manifest.json`, then restore
into a clean scratch repository checkout with:

```sh
tar -xzf .flow/tmp/fn255-spike/evidence/reproducible-spike.tar.gz \
  -C /absolute/path/to/checkout
```

Archive entries begin with `.flow/tmp/fn255-spike/`. Restore with an empty destination
or first inspect the paths to preserve local work. Runtime manifests contain the
original absolute checkout paths as evidence; `run.sh` regenerates current paths for
fresh runs. The archive manifest hashes every member and the archive itself.
