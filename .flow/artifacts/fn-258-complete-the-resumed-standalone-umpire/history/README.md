# Current-source history checkpoint

Frozen Temporal source: `ac60daaf47d2513b41eeff612f0e2758d33e186d`.
Filtered checkpoint: `a92db0610f13c9f623b526f2aa721abda69aa84d`.
Bundle ref: `refs/heads/umpire-current` in `current-history.bundle`.

The independent verifier compares all 7,261 available source snapshots: 627 retained
commits preserve raw authors, committers, dates, encoding, messages, trees and retained
parent frontiers; 6,634 pruned commits are projection-neutral. The current active
projection exactly matches the source manifest's 1,189 files, including LICENSE;
30 historical support files remain under `.history-support/`. The source archive
identity is bound by the manifest and checked unchanged. The original 7,149 map rows
are identical and the original `cf8bec4d6231302df54b57fccdc5bdb9d11552fc`
export is an ancestor. Thus the 46 additional retained commits preserve integrated
source history beyond the original export.

`reproduction/` records a fresh successful filter invocation; its commit map is byte
identical to `export/commit-map`. The initial run's original retained-tip guard refused
after rewriting because the frozen source tip itself is projection-neutral. New-only
tooling accepts that zero map row only alongside complete final projection equality
and independent neutrality checks, exact exported-graph/nonzero-map equality,
and a unique ordered source retained frontier equal to the checkpoint. `initial-refusal.json` and
`verification-tip-guard-refusal.json` retain the failed attempts. Original fn-257
scripts, inventory and export evidence were not modified.

`checkpoint-negative-controls.json` reproduces old-verifier false passes for an
unmapped identical-tree tip and a tip with a foreign second parent; both corrected
checks reject each graph. `history-provenance-verification.json` independently
reproduces every declared ancestry definition and representative history at ac60.

`negative-controls.json` proves file corruption, omitted source map coverage and a
synthetic extra sibling parent are rejected. The synthetic parent retains identical
active trees and raw metadata. `isolation-controls.json` proves source and registered
worktree filtering are refused before report creation. `preservation-comparison.json`
compares all registered source worktrees, excluding the declared migration artifacts.
The independent successful verifier also fingerprints source refs, index, registration,
and tracked/nonignored bytes before and after its invocation.

`bundle-proof.json` records strict Git integrity and bundle verification after cleanup
of clone-only stale shallow metadata, origin, reflogs and unreachable monorepo objects.
Recover independently with:

```sh
git clone --branch umpire-current current-history.bundle /chosen/new/checkout
```

Recommended integration: fetch this bundle into the standalone repository, then make
a merge commit whose first parent is the current portable standalone tip and second
parent is `a92db0610f13c9f623b526f2aa721abda69aa84d`, retaining the first
parent's exact active tree (`git merge -s ours --no-ff`). Check the merge tree equals
the previous portable tree, both parents are reachable, and the old export remains
an ancestor. No standalone refs were changed by this lane.

The supplied Temporal source is shallow before all selected Rust/host/tool births;
unavailable earlier history, including pre-cutoff LICENSE history, is not recovered.
No builds, runtime tests, remote writes or source deletion occurred.

Review correction: `review-before/` retains superseded evidence and the old verifier
strictly as the regression fixture. Canonical `export/` and `reproduction/` reports
were regenerated from two fresh isolated clones with the corrected filter and map,
including actual preflight hashes. The exported checkpoint/map/bundle are unchanged.
Representative commands, records, counts, hashes and blob identities are recomputed
at the frozen current source; no c88 definition commands are represented as current.
Source-refusal control now uses an external report path and records its full argv,
so it reaches the original-repository identity guard. Earlier verification sessions
may observe different source refs between invocations; preservation claims compare
before/after within each invocation and the declared preservation capture window.

The final rebinding also rebuilds current_source_map from all 1,189 manifest
records and updates the algorithm inventory count. Original investigation objects
and capture-time gap statements are explicitly nested under c88-bound original
input provenance. Intermediate review-fix reports retain their actual inputs in
review-before/checkpoint-fix; canonical reports use the final rebound map.
