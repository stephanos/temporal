# Reproducible generated-source differential spike

This retained experiment has passing complete graph, mutation, refusal, shrinking
and process-failure controls. The verdict and measured limits are in
`model/design/generated-differential.md`. No production crate or gate is registered.

Run from the repository root with the existing pinned toolchain and a TLC 1.7.4 jar.
The backend verifies the jar's SHA-256 before use. The destination must be new.
Execute mode rejects a missing `--tlc-jar` path
before preparing that destination, even if a previous copied jar exists. Source-only
preparation does not require a jar.

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
BASH_ENV and sets two compiler jobs. Cargo runs with the framework as its working
directory and reads that framework's mise pins. The copied tool metadata resolves
Java from the task tool root; its trusted-path setting does not override the
framework configuration used by Cargo. The runner records source hashes, output,
timings and Java argv sampled through Linux `/proc`. JVM sampling is Linux-only;
on other platforms empty observations cannot establish zero launches, so use
separate launch evidence such as the retained TLC execution logs.
Each compile has a 120 s spike-only ceiling; runtime has 45 s per selected case.
The runner cancels on less than 30 GiB free disk. Required-backend execution must
share the conductor's JVM allocation with other active lanes.

Each generated Model has two enum fields, a reachable count bound of one or two,
an optional two-value Finite wrapper, guarded arithmetic, rejection, facts and an
invariant. Its declarative starts have one or two members. Compiled and interpreted
checks complete independently. TLC parses the complete printed source independently.
Complete snapshots include starts, states, action/choice/successor/outcome/facts and
claim verdicts. A case selector lets nested emitted modules share one TLC process
without constructing expectations from native states. Batch schemas must match,
and this family's claims must verify by construction. The batch reports
Verified-or-fail; refutable-claim families need per-claim operators or individual
checks in the maintained task .2 harness.

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

Paired and shrinking measurements used the exact historical adapters retained in
those run directories. Only `checked-batch` ran the current adapters. The difference
is the equivalent nested-if to if-let-chain cleanup; `evidence/adapters-provenance.json`
and `adapters-provenance.patch` identify every run and the complete delta. The
follow-up changes execute-mode Python preflight and documentation only. The previous
committed archive and its manifest remain byte-for-byte under
`evidence/history/4285914b1b.tar.gz` and `4285914b1b-manifest.json` inside the current
archive, preserving the earlier top-level probe and measurements as well.
