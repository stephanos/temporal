---
satisfies: [R5]
---
# fn-64-umpire-case-runtime.20 Record Host-close failure in cleanup outcome

## Description
Resolve the verified completion-review close-only failure: session.Close errors or cooperative timeout must set cleanup failed and link the host_close_failed diagnostic while preserving fixed Run disposition and Verdict.

## Acceptance
Add regression tests for close-only error and cooperative timeout with diagnostic linkage; preserve ordinary disposition and satisfied or violated Verdict. Focused tagged execution tests, relevant race tests, full regression, formatting and task-relative lint pass. Official staged-tree implementation review returns SHIP.

## Done summary
Host-close errors and cooperative timeouts now set cleanup failed and retain the host_close_failed diagnostic reference without changing fixed Run disposition or Verdict. Extended TestRunTerminalPrecedence covers close-only errors/timeouts for satisfied and violated outcomes and combined cleanup/close diagnostic linkage.

Baseline: green (focused execution/verification tagged suite). The TDD regression failed on succeeded cleanup and missing close diagnostic linkage before the fix. Verification passed: focused tagged execution/verification, relevant race tests, full make umpire-check-regression, make fmt-imports, and full task-relative make lint-code with GOLANGCI_LINT_FIX=false. The aggregate regression gate matched the exact inherited live-failure identity set; this does not claim those inherited tests pass. Task-relative lint reported zero new issues; the repository retains its inherited lint backlog. CC=/usr/bin/clang used consistently. Logs: /tmp/fn64-task20-{baseline,red,green,race,regression,fmt,lint,review}.log. No gate was skipped; receipt writing was unavailable because the worktree differs from HEAD.

No commits or pushes: user owns commits. Base HEAD remains 7774fdc7ac751ac959816c9829516ce54af57194; original staged tree ed17d358925ebf4a0a10ef0c5d1c9a29ad469dbd preserved. Official staged-tree review snapshot: .flow/tmp/fn64-task20-review-snapshot.json; reviewed tree b22a20a09ea3a031fadd72ed00383f5bb6cc12a1. Receipt: /tmp/impl-review-receipt-fn-64-umpire-case-runtime.20.json. The reviewer returned SHIP with no findings; its attempted test run was sandbox-blocked, while host verification above passed.

stage: impl-review - ran (codex:gpt-6-astra:medium; SHIP; immutable staged-tree adapter)
## Evidence
- Commits:
- Tests: baseline: green (focused execution/verification), TDD_RED: close-only error/timeout cleanup status and combined diagnostic linkage, CC=/usr/bin/clang go test -count=1 -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., CC=/usr/bin/clang go test -count=1 -race -tags test_dep ./tools/umpire/internal/execution/... ./tools/umpire/verification/..., CC=/usr/bin/clang make umpire-check-regression (exact inherited live-failure identities matched), CC=/usr/bin/clang make fmt-imports, CC=/usr/bin/clang make lint-code GOLANGCI_LINT_BASE_REV=7774fdc7ac751ac959816c9829516ce54af57194 GOLANGCI_LINT_FIX=false, REVIEW_SHIP: staged tree b22a20a09ea3a031fadd72ed00383f5bb6cc12a1; /tmp/impl-review-receipt-fn-64-umpire-case-runtime.20.json
- PRs: