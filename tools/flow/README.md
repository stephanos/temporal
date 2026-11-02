# Local Flow patches

These patches modify the installed Flow scripts, not Umpire's runtime. Stage
copies of the current installation before applying `validation-review.patch`
from its `scripts` directory with `patch --fuzz=0 -p1 --forward`. A failed or
ambiguous hunk requires rebasing and rerunning the controls. Preserve current
retirement and default-disabled behavior while rebasing.

This workspace deliberately keeps its tracker unversioned at the workspace root
and selects explicit runtime state. Leave native dependency admission unchanged
here: the legacy `committed-dependencies.patch` requires tracker blobs in Git HEAD
and is incompatible with that storage model. Its retained controls concern tracked
repositories only and do not qualify this workspace. Product source commits and
predecessor closure evidence remain independently required by the workspace rules.

## Spec completion

This project enables `specClose.validationReviewRequired`; other repositories
default to false. Before normal spec completion, spend about a minute assessing existing
validation and review costs. Reuse existing evidence; rough estimates and unknown
costs with reasons are acceptable. Record whether gates can be sped up while
preserving coverage. A concrete reason to retain them is a valid decision.

Write a v1 JSON file under the repository (scratch under `.flow/tmp` is valid):

```json
{
  "schema_version": 1,
  "spec_id": "fn-123-example",
  "gates": [
    {
      "command": "python3 tools/flow/test_validation_review.py",
      "mode": "isolated temporary Git repositories",
      "cost": {"kind": "estimated", "seconds": 20},
      "coverage": "Report validation, publication, refusal and retry controls",
      "evidence": ["Existing task test log; estimate rounded from that run"]
    },
    {
      "command": "flowctl codex impl-review fn-123-example.1",
      "mode": "primary review, configured model and effort",
      "cost": {"kind": "unknown", "reason": "No elapsed-time receipt available"},
      "coverage": "Independent correctness and integration assessment",
      "evidence": []
    }
  ],
  "speedup_decisions": [
    {
      "opportunity": "Repeated full-suite executions",
      "decision": "retain focused controls",
      "rationale": "Each isolated harness checks a distinct boundary; no safe savings found",
      "coverage_preserved": "All success, refusal, retry and integration controls remain"
    }
  ]
}
```

`cost.kind` is `measured`, `estimated` (finite, nonnegative `seconds`) or
`unknown` (nonempty `reason`). Each gate requires nonempty `command`, `mode`,
`coverage`, and an `evidence` string array (empty if unavailable). An empty
`gates` array requires a nonempty `no_validation_reason`. Every decision requires
nonempty `opportunity`, `decision`, `rationale`, and `coverage_preserved` strings.
Optional source basis and Flow follow-up references remain assessment content;
close does not verify evidence trees, create tasks or execute recommendations.
Limits: 256 KiB, 128 entries per collection and 32 levels of nesting.

Run `flowctl spec close <id> --validation-review-file <repo-file> --json`.
The exact validated bytes are copied to
`.flow/artifacts/<id>/validation-cost-review-<sha256>.json`; the spec JSON records
`validation_review` with `path`, `sha256`, and `schema_version`. Retain every
returned `modified_paths` entry, including the companion `.gitattributes`. Commit
those paths with normal closure changes only when the tracker is Git-tracked;
this workspace retains them in its unversioned evidence store. The two attribute
rules preserve report and attribute
bytes through Git text, filter, encoding and ident conversions. Close checks the
effective attributes locally before lifecycle writes; conflicting higher-priority
attributes or a differing existing companion file refuse without overwriting
user policy. A plain workspace has no Git conversion engine; it retains the same
exact file bytes/hash/reference without requiring tracker commits. Ambiguous or
failed Git discovery refuses rather than assuming plain storage. No conversion
command is executed. The input may
be uncommitted or staged; there is no preliminary report commit or clean-HEAD
requirement. For make-pr's automatic close, put the input at
`.flow/artifacts/<id>/validation-cost-review.json`; preflight forwards it only
for enabled new closures, then commits the content-addressed copy with closure.

Report/config/path/publication errors refuse before close's lifecycle writes.
Explicit retirement retains native retired/not-required semantics and needs no
completion report. Combining retirement with a report refuses before runtime
changes; retirement supplies no completion evidence.
Already-done calls preserve their saved report and existing task checks; reopen
requires an explicit input again. Retry revalidates input and includes an identical
previously published artifact in `modified_paths`. An artifact can remain after
a later task/spec write fails; existing partial task publication has no rollback
transaction. The report attests structure and exact bytes, not human freshness.

## Staged verification and installation

Coordinate a narrow lease with other writers of the installed scripts. Running
processes in other repositories continue using their loaded code. No global idle
state or process shutdown is needed.

Prepare a task-owned staging directory with `scripts/` copies, sibling
`templates/` copied from the current installation, and immutable
`flowctl.py.baseline` / `make-pr-preflight.sh.baseline` copies of the current
scripts before patching. Record baseline hashes. Apply patches only to staged
copies. The legacy dependency patch is not installed for this unversioned
workspace. Run:

```bash
FLOWCTL_PY=<stage>/scripts/flowctl.py python3 tools/flow/test_validation_review.py
FLOWCTL_PY=<stage>/scripts/flowctl.py MAKE_PR_PREFLIGHT=<stage>/scripts/make-pr-preflight.sh python3 tools/flow/test_validation_preflight.py
```

The controls use disposable Git repositories, plain workspaces with explicit
external runtime state, and stub preflight network/PR boundaries. The retained
legacy dependency suite tests tracked-storage policy only. No Rust/Go/model
gates are needed for these tooling changes.

Under the installation lease run:

```bash
python3 tools/flow/install_validation_review.py <stage> <installed-scripts> <new-backup-directory>
```

The installer checks both baselines before starting, retains exclusive recovery
copies, and checks each target again immediately before atomic per-file
replacement. It installs CLI first and preflight second. A mismatch stops without
overwriting the changed file: restage CURRENT copies, rebase patches and retest.
For a new installation, enable this project's config last, after both files pass
their installed checks. When repairing an already-enabled policy, retain the
opt-in and finish both files plus installed controls before attempting closure.

Replacement is atomic per file, not across files. If only CLI installs, do not
attempt project closure; for a new installation keep the opt-in off. Inspect the
returned error and backups, then restage/retest and finish preflight installation.
To restore a target, first verify its current
hash matches your installed replacement, then atomically replace it from the
known backup under the lease. Preserve another writer's changes. Keep backups
until the installation is accepted; repository patches reproduce the changes
after future reinstalls.
