# D26 clock documentation source checkpoint

CLI, Tutorial and Architecture now describe the integrated shared process clock,
not the superseded `time.Now`-only offset. Forward draws advance `faketime` by
1–1024 nanoseconds on a separate seed-derived stream. Monotonic durations,
timers and simulation time read that clock; runtime-internal reads do not tick.
The strict idle-jump boundary, private synctest precedence, next-check timer
delivery and coarse timestamp ties remain explicit.

The conductor inspected the exact four patch hunks and their runtime/patch
source basis. The source proof and independent audit accompany this checkpoint.
No native workload or replay qualification is inferred from this documentation.
R26's transport reproducer, late-deadline fixture, seeded workloads and full
both-platform gates remain open under the existing task acceptance criteria.

The writer applied the hunks without replacing task 19's concurrent World text.
The index uses the isolated HEAD-plus-D26 Architecture file so that this commit
contains only clock documentation. Its complete SHA-256 is retained in
`handover.json`; task 19's working-tree World changes remain unstaged.

The three-file predecessor and final archives remain local, with their hashes
recorded in `handover.json`; they are not needed as committed duplicate snapshots.
The exact patch and compact source audit are committed instead. Documentation
and diff checks apply here; this introduces no runtime source or generated input
change and does not warrant a broad test retry. No push or Flow completion is
included. MILESTONES item 5 governs this separate verified-progress commit.

The documentation source landed as `108a881807`. An artifact-count preflight
expected eight files rather than the actual seven and stopped before receipt
staging; the subsequent commit command still ran. The follow-up receipt commit
records this omission and carries the source audit, exact hashes and Flow
description without rewriting history or including another task's source.
