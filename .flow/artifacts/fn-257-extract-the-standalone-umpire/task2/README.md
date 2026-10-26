# Verified local history export

Fixed source: `c88fdc550694020eb1c6a2e09577813d906c9ffe`.
Export: `cf8bec4d6231302df54b57fccdc5bdb9d11552fc`.

`checkpoint.json` identifies the live disposable checkouts and the committed recovery
bundle. Recover an independent checkout without the Temporal source repository:

```sh
git clone --branch umpire export-history.bundle /chosen/new/checkout
```

The export contains 1,178 active files (1,177 framework files and LICENSE) and 30
historical support files under `.history-support/`. Remove those support copies in
the separate portable-layout commit, retaining their history. No module identity,
protocol, semantic golden or Model adaptation is part of this checkpoint.

`verification-final.json` independently compares every available original snapshot:
581 retained commits preserve raw author/committer/date/message/encoding metadata;
6,568 pruned commits are projection-neutral. Both sibling archive moves, old host
paths and native-tool paths are included. The verifier reads raw Git trees/objects;
it does not import the filter's projection implementation.

`reproduction-and-preservation.json` records identical export commits/maps from two
fresh clones and unchanged refs, indexes and product states across all 36 source
worktrees, including fingerprints of 56 dirty paths. Root migration evidence/docs
are the declared exceptions; ignored cache bytes are not inventoried or modified.
`negative-controls.json` records rejection of the original repository and its
registered worktree, a corrupted exported file, and a tampered source commit map.
The file mutation was restored exactly. `bundle-recovery.json` proves clean recovery.

The supplied source is shallow before 2026-02-05. Its cutoff precedes every recorded
Rust/host/tool birth and contains no selected Rust prefixes; the clone has the same
available ancestry. Earlier history, including pre-cutoff LICENSE ancestry, is
unavailable and is not claimed recovered. See `source-history-boundary.json`.
The rewritten history is complete within that available input. Clone-only cleanup
removed stale shallow metadata, origin, reflogs and unreachable monorepo objects;
both exports pass strict Git integrity checks and use one 3.25 MiB object pack.

The tools use Python's standard library and the separately installed, pinned
`git-filter-repo 2.47.0`; no workspace dependency was added. Tool hashes and clone
commands are recorded in `tool-provenance.json` and each export's `preflight.json`.
`export-history.py --help`, `verify-export.py --help` and
`capture-preservation.py --help` describe their explicit inputs. Reports must remain
outside the clone being filtered. The root HEAD baseline in the recorded run is
`8bcf4c9fa6`; later evidence commits legitimately advance that metadata-only root
HEAD, so a new verifier invocation must supply its actual current root baseline.

The initial source-shallow guard refusal and the verifier's unrelated-Temporal-
submodule scan failure are retained. The corrected tools produced the successful
independent proof; neither failure is represented as a successful check.

No Rust/Go build, runtime test, generator freshness or consumer proof ran. Those
remain required under task 5 when the owner validation hold ends. Destination
contents/permissions remain unknown (HTTP 401); nothing was published or removed
from Temporal.
