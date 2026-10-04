# Task 26 source progress

CLI consumers now delegate seed cardinality, coverage/probe validation, trace capacity, choice-coverage dependency and absent guided coverage defaulting to the existing Runner seams at their original validation points. Typed-error translation retains CLI messages, and the CLI retains presence checks and enabled-zero trace rejection. Runner production is unchanged.

Status is `SOURCE_PROGRESS_ONLY`; Flow remains `in_progress`. The conductor owns the independent source review, progress commit and lifecycle decisions. This handover claims no review verdict or native qualification.

Tier: session (jev-unavailable(no_key))

stage: impl-review - skipped(policy: CONDUCTOR-DEFERRED - conductor owns review; qualification remains red)

## Source and regression evidence

`evidence.json` indexes the exact commands, environment, tool SHA-256 values, source freezes, raw logs, start/end times, elapsed times and terminal exit codes. Every command has stable sources before and after. The baseline freeze covers the nested Gomad tree and root lint config because architecture discovery inspects the whole module. Later freezes reference that immutable baseline plus changed-file deltas. Root-owned Flow, inventory and MILESTONES edits are excluded.

- `ownership-red.receipt.json` records exit 1 against pre-edit CLI SHA-256 `234c3d5ce203142d1976db95efaa7f667ab8143fbc2ea25ef6abfdf4be76cdcb`. The ownership test rejects actual CLI cardinality/probe/range/dependency checks and requires checked Runner calls in their consuming functions. `ownership-green.receipt.json` and the final focused CLI run pass it. Architecture assertions accompany behavioral tests and do not establish byte preservation themselves.
- `focused-cli.receipt.json` passes 34 top-level tests, including literal error order, plan route/default/output, typed error messages, guide defaults, explicit-zero/irrelevant flags, argv/environment/tags and writer statuses. `old-cli-behavior.receipt.json` passes the same 33 behavioral tests using the saved CLI bytes from base `984fa118347ebc7b39b7080dd5b9e95e941a00d4`. The ownership test is excluded from that overlay selection because it parses on-disk source. `old-cli-provenance.json` and the receipt bind the exact overlay mapping and stable saved bytes.
- New literal `--choice-bytes=0` rows retain byte-size parser rejection before semantic validation. The direct resolver's new enabled-zero row protects the CLI guard against adopting Runner's disabled-zero convention. Draft full-CLI receipts retain the mistaken new expectations and their actual failure; only those new oracles were corrected. Existing assertions remain unchanged.
- `focused-runner.receipt.json` passes normalization, typed errors through wrapping, shared seed parsing, configuration/error cases, canonical options/legacy characterization and the portable-plan strategy rejection. The existing Darwin-only diagnostic identity golden remains skipped; its native proof is open.
- `final-boundary.receipt.json` passes `TestPackageArchitecture` and both retained external-consumer compilation tests. The existing `testdata/runnerconsumer` fixture now compiles all eight public R6 additions, including both typed error methods through the error interface. This proves accessibility and claims no unknown consumer migration.
- `format-check.receipt.json` and `affected-vet.receipt.json` pass.

## Qualification remains open

Actual unfiltered pinned CLI golangci exits 1 before and after with the same 54 findings (53 errcheck, one staticcheck). `lint-delta.json` pairs exact diagnostics and source excerpts, retaining both locations and file owners; introduced and resolved finding counts are both zero. The actual errortype command exits 0 before and after. No filtering, suppression, error discard, lint config, pin or comparison change was made.

`final-cli.receipt.json` retains the `./cmd/gomad/...` failure because end-to-end `TestMain` executes absent `.toolchain/bin/go`. `affected-cli.receipt.json` retains the final complete internal CLI run's three environment failures: real readonly analysis cannot start that launcher, and two doctor availability tests reject `linux/arm64`. The failing doctor/analysis test bodies and TestMain inputs have unchanged hashes in `evidence.json`. Focused developmental results do not turn this full-package gate green.

`final-runner.receipt.json` retains expanded portable-plan failures on unsupported `linux/arm64`. Native planning, all-platform fixed-identity proof, complete host/runtime/default/workload gates, original task 5 and its predecessors, R6/R18/R19, final task 21 and formal full-green-tree review remain open. This task did not rerun the unchanged broad 419-finding/root-fast gates.

The Makefile's `VERSION_INPUTS`, `BOUNDARY_INPUTS` and `COMPATIBILITY_INPUTS` were inspected before production edits. These CLI and fixture changes touch none of those generator inputs, shared runtime/protocol sources, source templates or generated outputs, so this bounded correction does not require an additional validate run. The retained source delta confirms all generated/fixed-identity inputs are unchanged. Native/full validation obligations remain open in their existing owners.

## Defect route

- Prior fixes: task 5's historical summary and the caller-repair scout were read; current source reproduces the actual ownership gap. Memory search/read found the separate projection-drift warning, which this correction does not touch. PR/tracker/network checks were not run under the conductor's no-network scope.
- Diagnosis: the pre-edit AST regression fails on the existing CLI's duplicated rules; CLI/Runner behavior characterization passes before the delegation change. The remaining defect is ownership, with semantic behavior retained by the old-program overlay and final literal tests.
- Introduced by: unresolved mixed-WIP attribution. No known-good bisectable source revision was supplied; no bisect or worktree was used.
- Base/head: ownership regression exits 1 before and 0 after; the same literal behavioral selection passes against saved base CLI bytes and final CLI source. The reproduction was not committed separately because the conductor alone owns commits.
- Live: library/CLI test evidence only. The patched launcher and qualified native host are unavailable.

No worker commit, Flow lifecycle mutation, formal review, bridge, delegate, push or checkout operation ran. No commands remain live. The conductor must review and commit this verified progress before admitting another source writer.
