# Seven-task readiness and parallel scheduling amendment

Applied under explicit conductor authority after source join `523d131d2cf992b04a43b4d595ea0f1a434f44b6` (model tree `8469b19c58647dc8785b5318ad7cfd20cfa0f8d2`). No source/build/index/commit/MILESTONES/review or task claim/completion operations were performed. Whole-spec fn247 admission remains unchanged.

## Authority and scope

[Direct owner] "can we paralleize more aggressively?" and "can we take more risks, too?" permit isolated concurrent implementation, ordinary central merge/rework and coordinated breaking interfaces under R18. Preserve scientific oracle independence, semantics, process/security safety and current integrity. [Direct conductor decision] Adopt six independent source lanes plus one final integration task, retaining all six existing task IDs and R1–R4 responsibilities; R5 belongs to final integration once.

Tasks .1–.6 now have no intra-spec prerequisites. Existing .4/.5/.6→.1 edges described snapshot reacceptance contention rather than runtime inputs, so only those three obsolete edges were removed. Newly allocated task **fn-248-off-the-shelf-crates-and-consolidation.7** depends on all six existing tasks. Original titles and all seven todo statuses were preserved. The new task ID came from `flowctl task create --spec fn-248-off-the-shelf-crates-and-consolidation --from-json .flow/tmp/fn248-readiness/new-task-create.json --json`; it was not guessed. The installed allocator scans existing task IDs under its allocation lock and refuses overwrites, so the stale `next_task:1` did not replace an existing task. All template placeholders were replaced with the actual returned ID before canonical body publication.

Each source lane now has Size/Files/Touches, source-backed focused commands and meaningful error/boundary controls. Private worktrees/indexes/targets permit overlapping source/Cargo/codegen edits; conductor-owned integration reconciles them and reviews/reaccepts current snapshots and necessary guest publication once. No lane-local result certifies the joined tool. Final .7 consumes the existing canonical `UMPIRE_APALACHE=require make model-validate` pipeline: one complete required model-check and its existing benchmark, without per-lane full-gate or cargo-insta workspace repetitions.

## Golden gate and native correctness boundaries

[Source-supported, conductor accepted] Plain text insta snapshots replace binary snapshot filename decoding. Extend existing-pass ownership/pending checks with explicit Rust/Go/live/external readers and named snapshots. Mere payload reads and skipped-backend ownership are not execution evidence. Keep orphan/unowned/pending and skipped-incomplete failures, historical-original semantic comparisons and mutation controls; do not impose a blanket cargo-insta run or installed CLI prerequisite. A local cargo-insta1.49.0 binary is availability evidence only.

The task contracts explicitly retain independent Go/live producers and require a narrow reviewed candidate-update interface. They keep small snapshot readers outside wasm backend dependency closures, require coordinated native prost-types Any wrappers with canonical URL and Malformed/Unsupported admission, account for generated WKT paths shared with the codegen lane, and preserve ITF strict duplicate/null/budget/schema/typed-replay controls. Darling0.24/syn3/R18 availability remains an implementation admission proof, not a claimed resolved package. Breaking compatibility does not waive these correctness obligations.

## Fresh guards and bounded metadata fallback

Before allocation, all seven existing Markdown bodies were re-read with Flow CLI and matched committed preparation basis `2c3c308ee2` after EOF normalization; every visible tracked metadata field matched its committed basis, with no foreign differences. Fresh snapshots and per-body/metadata SHA-256 guards are retained in the machine receipt. Every body was checked again immediately before publication and read back afterward. Flow CLI owned task creation and every parent/task Markdown update. Unknown metadata was preserved by its installed writers; no tracked task definition was reconstructed from cached display output.

The installed CLI exposes dependency addition but no removal/replacement. Conductor explicitly authorized the installed module's `canonicalize_task_for_write` plus `atomic_write_json` for exactly .4/.5/.6 removing [.1]→[]. Each fallback asserted the fresh CLI dependency, canonical task ID and complete expected tracked definition immediately before writing; it preserved every field except `depends_on` and `updated_at`, then verified the written definition and CLI output. No installed CLI patch or new installed tool was created. Exact expected-definition hashes and before/after edges are recorded in `parallel-readiness-amendment.json`.

## Actual verification

Preparation static checks passed: original titles/IDs, declared scopes/focused checks, synchronized R1–R5 rows and no R5 source-lane frontmatter claims. After application, fresh CLI state had seven tasks; an independent in-memory dependency traversal found no cycle or missing edge, all original tasks had empty dependency lists, .7 depended on precisely .1–.6 and the whole fn247 prerequisite was unchanged. Actual `flowctl validate --spec fn-248-off-the-shelf-crates-and-consolidation --json` and the same command with `--coverage` both returned valid, seven tasks, no errors and no warnings. Fresh task Markdown confirmed R5 belongs only to .7 and no allocation placeholder remained.

These are planning-state/DAG/coverage results, not Rust/Go runtime or build results. No primary review verdict or implementation readiness beyond the checked plan state is claimed. Root owns subsequent review, source admission, lifecycle, staging and commits.

Machine receipt: `parallel-readiness-amendment.json` beside this file. Detailed preparation audit: `readiness-audit.md`. Exact applied bodies/logs: `.flow/tmp/fn248-readiness/{applied/,allocation.json,applied-payloads.json,apply-log.json,fresh-before-apply/}`.
