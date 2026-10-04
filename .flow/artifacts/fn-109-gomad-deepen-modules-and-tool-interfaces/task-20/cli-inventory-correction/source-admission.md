# CLI inventory correction admission

Resume task 20's documentation ownership for the nine existing registered
public flags absent from CLI.md, identified by task 21's committed preservation
audit. The predecessor checkpoint is
`5b8261599475f0cef21b6c7d23d272fc9439b26f`; the tracked tree and index are clean.
The historical audit, measurements and earlier task-20 review stay immutable.

Document `--env`, `--io-ro-mount`, `--max-bytes`, `--min-free-bytes`,
`--observed`, `--prune-qualified-artifacts`, `--terminate-grace`,
`--toolchain-root` and `--world-transition-limit` at their actual command
boundaries, with source-verified defaults, syntax and limits. Keep their current
registrations, parser behavior and all existing shell examples unchanged.
Reconcile current documentation evidence without attributing this repair to
the earlier frozen review. Go-interface/protocol reconciliation remains with
its implementation owners, outside this documentation correction.

The explicit dependency/status override admits source progress only under
MILESTONES immediate-delivery item 4 and the task-21 owner handback. Task 19's
formal review/native gates, R18/R19 final acceptance and inherited D5 closure
remain open. Task 20's earlier three-draw SHIP qualifies only its recorded
source; obtain a fresh review for this correction. The original task acceptance
and dependencies are unchanged.

Use the existing gomad branch, one documentation writer and parallel read-only
scouts where useful. Root owns lifecycle, review, precise staging and the
per-task progress commit. Preserve unrelated untracked artifacts and .turbo.
No push, worktree, stash or history rewrite is authorized. The actual host is
Linux aarch64 with the patched Go executable absent; document checks and pinned
stock-Go tests are developmental, not either native qualification.
