# Task 25 source progress

Lifecycle fault resolution retains its stop/crash and restart behavior while the affected package's pinned exhaustive lint passes. `controller.go` replaces only the redundant inner enum switch with `if action.Kind != FaultRestart` and its `else`; both condition expressions, target assignments, outer switch, validation order and public surface remain unchanged.

Task `fn-109-gomad-deepen-modules-and-tool-interfaces.25` remains `in_progress`. Base and current HEAD are `ff7da7419bbc8321dd4004e58249bb149ffd31df`; the commit range is empty. Root owns Git, lifecycle, independent review and the source-progress commit. This frozen source is ready for one fresh independent review; the worker issues no review verdict.

Tier: session (jev-unavailable(no_key))
stage: impl-review - skipped(policy: conductor-deferred; root owns review)

## Evidence

[`evidence.json`](evidence.json) points to exact argv, cwd, controlled environment, tool/source hashes, start/end, elapsed seconds, terminal exits and before/after stability in each receipt. `old-source.sha256`, `draft-characterization-source.sha256`, `characterization-source.sha256` and `final-source.sha256` each describe one actual frozen revision. They cover gomad3sim and lintcode source, root module inputs/config/Makefile, inspected generator inputs and the three pinned executables. They intentionally omit concurrent root Flow/MILESTONES metadata and do not claim a whole-repository source freeze.

The actual unfiltered package linter fails before production edits with the five missing network enum cases at `controller.go:159:3` (`lint-red`, exit 1). The identical package command passes after the repair (`lint-green`, exit 0, 0 issues). Both commands use the existing v2.13.0 binary, unchanged config, `test_dep` and `--fix=false`, with no diff filter or download.

The new tests run against old production source (`characterization-old`, exit 0) and final source (`characterization-final`, exit 0). `TestResolveFaultLifecycleStates` covers all three lifecycle kinds across all six declared states with nil/nonnil operations, literal admission tables, exact sentinel identity/zero failures, immutable node state and complete successful realizations. Three literal canonical JSON vectors independently supply the identity oracle; expectations do not call controller identity helpers. `TestResolveFaultLifecycleTargets` covers explicit/missing nodes, Match.Node, stop/current-versus-next incarnation, deterministic candidates, prior targets, first matching prior, missing-prior rejection and existing explicit-node fallback. `TestResolveFaultOuterControls` covers all five network kinds and unknown-kind rejection, existing endpoint checks and detached group slices. Candidate slices are also checked for detachment.

The initial draft characterization (`characterization-before`, exit 1) had two incorrect candidate literals and is retained honestly. Independent SHA-256 arithmetic for seed 19, ordinal 4, ID `candidate` gives `05be18e2d62338092e1b58cdc5e4b5c577e19973557ab854bd9c697856f17bfd`, first little-endian uint64 664320350860393989, modulo 2 equal to 1. Correcting the fixture to select `stopped` made old-source characterization pass before the production repair. This draft failure is not the lint regression proof; `lint-red` supplies that proof.

Final ordinary gomad3sim tests pass with `-count=1 -tags test_dep -v` (`package-tests`, exit 0, 199 RUN entries, no skips, reported package time 0.012 s). Unfiltered package errortype passes (`errortype`, exit 0). All lintcode contracts run with the actual pinned `LINT_POLICY_GOLANGCI` (`helper-contracts`, exit 0, 8.620 s); real-tool policy tests are executed. Stock Linux arm64 tests exclude `gomad3_toolchain` cases and supply developmental evidence only. Zero-second receipt durations are whole-second clock resolution, not zero-test selections.

The real required root fast gate runs with base `951c5516e9e7b3066e7e069adda9565cfd68844c`, existing tags, `LOCALBIN=/tmp/fn109-lint-tools.ZdNe1t50` and `GOLANGCI_LINT_FIX=false` (`root-fast`, Make exit 2, 75 s). Its 108-package ordinary root and tagged integration scopes complete with 0 golangci issues and their Make recipes finish, including errortype. Its automatic 55-package nested Gomad scope then fails on 419 findings; nested errortype and later dispatch scopes remain unreached. No separate unchanged broad Gomad command was rerun. [`findings-comparison.json`](findings-comparison.json) proves all path/line/column/message/linter/owner entries exactly match task24's inventory, across 31 owners. The retained [exact findings](../task-24/gomad-findings.json) and [source owners](../task-24/gomad-finding-owners.json) remain authoritative.

Generator inspection found neither changed file in VERSION_INPUTS, BOUNDARY_INPUTS, COMPATIBILITY_INPUTS, protocol source identity lists, schemas/templates or overlay outputs. `controller.go` has no go:generate directive and changes no protocol layout or generated contract. `make -C tools/gomad3 validate` is unnecessary for this bounded branch rewrite and was not rerun. Final gofmt output and scoped whitespace checks are empty; source/config/module/Make/helper preservation checks pass.

## Remaining acceptance

Original R18/R19, task21, formal green-tree review, workload/default proof and both native darwin/arm64 and linux/amd64 gates remain open. Task24's unchanged `^.git` reporting limitation still suppresses `.github` findings. Root's final review and commit remain required before another implementation task.

Defect route:
- prior fixes: task24 review and local controller history read; memory search returned no matching bug. PR/tracker checks unchecked under the conductor's admitted bounded owner and prohibition on external writes.
- diagnosis: actual pinned lint reproduces missing cases in the inner three-kind subset; literal outer controls and lifecycle tables pass on old source; replacing only that redundant switch clears the actual finding.
- introduced by: skipped; no known-good lint revision supplied, and no worktree/bisect authorized.
- base: actual package lint exit 1 and corrected behavioral characterization exit 0 on old controller; head: identical lint exit 0 and unchanged behavioral characterization exit 0.
- live: no live application surface; actual linter invocation is the failing boundary.

BLOCKED: SCOPE_EXCEEDED
Task: fn-109-gomad-deepen-modules-and-tool-interfaces.25
Summary: Required root-fast advances past the repaired controller and fails on the exact retained 419 nested findings outside this task's Touches.
Impact: Task21 final qualification, formal green-tree review and original R19 remain incomplete; both native gates remain separately open.
Suggested resolution: Root reviews and commits this coherent source progress, then assigns bounded fixes using task24/gomad-finding-owners.json and obtains native qualification without changing rules or comparisons.

Delegated agents: 0. Live command handles: none. No worker Git or Flow lifecycle mutation occurred.
