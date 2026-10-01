---
satisfies: [R2, R3, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.15 Admit and interpret the portable finite semantic IR

## Description
**Touches:** [model/scalav2/goir/**]

Extract IR admission and primitive finite interpretation from task 3. Task 14 owns generic checker algorithms; task 3 binds authored claims and receipts after both lanes finish. Do not depend on unintegrated task 14 APIs.

**Size:** M
**Files:** existing load.go/eval.go/machine.go, reusable interpreter helpers, and focused admission/interpretation tests.

### Approach
- Admit the task-2 IR with located version/reference/type/duplicate/bounds diagnostics. Reject malformed or unsupported executable constructs explicitly.
- Interpret finite values, bounded FIFO/unordered channels and their declared delivery/loss/duplicate choices, nondeterministic transitions, and declared/undeclared semantic holes through one evaluator.
- Preserve disabled-empty transitions separately from reachable holes, source identity, and all supported legacy Nexus tables/results/fingerprints.
- Expose reusable evaluation/metadata for task 3 to bind monitors, assumptions, composition, claims and queries; no partial consumer may silently report a query success while ignoring those declarations.
- Bound catalog/table work before allocating a large finite product. Exercise a tenfold finite-input probe: complete under sufficient ceilings or return an explicit resource-limit result with actual counts under insufficient ceilings.
- Use ordinary source-derived fixtures from task 2 for positive semantics and small generic malformed-IR controls for admission; never hand-construct feature IR or edit feature policy.

### Investigation targets
**Required:** model/scalav2/goir/load.go; model/scalav2/goir/eval.go; model/scalav2/goir/machine.go; model/scalav2/goir/diagnostics_test.go; model/scalav2/goir/parity_test.go; proto/internal/temporal/server/api/modelir/v1/ir.proto; model/scalav2/SEMANTICS.md; model/scalav2/lifter/testdata/.
**Optional:** existing generic Go table/claims APIs, read-only.

### Quick commands
`mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/...`; scoped Go IR lint. Record any inherited repository-wide lint failure separately.
## Acceptance
- [ ] Unknown versions, invalid/crossed references and types, duplicate declarations, malformed bounds and unsupported executable constructs produce located admission errors.
- [ ] Source-derived bounded-channel/presence/choice fixtures evaluate correctly; disabled actions and semantic holes remain distinct; legacy table/result/identity pins pass.
- [ ] Catalog/table construction completes within sufficient ceilings or reports explicit resource exhaustion with actual work counts, including the tenfold input probe.
- [ ] The interpreter exposes enough typed evaluation and declaration metadata for task 3 without silently claiming unchecked monitor/property/progress results; focused Go gates pass.


## Done summary
The finite IR consumer now admits typed declarations, evaluates channels and holes, and bounds catalog, class and table work before allocation. Task3 binds authored claims and receipts.

The independent native rereview returned SHIP. Final integrated Scala-to-Go, focused Go tests and scoped lint passed. Tested source and all original comments are retained; the user owns commits.

stage: implement - ran (model: claude-opus-5-5; CLI --model opus --effort high; owner session 42fe243d-e20b-46a4-833a-1a829b424d92)
stage: impl-review - ran (codex:gpt-5.6-sol:high; first-round three-axis fanout and same-primary fix rereview; SHIP)
stage: wave-join - ran (2/2 returned; guarded uncommitted copying; final integrated gates rc0; clones retained because they contain uncommitted work)
Tier: session (jev-unavailable(no_key)); retained pinned opus/high.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: GOFLAGS=-tags=test_dep make umpire-check-scala (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/scala-wave-foundations-final.log), mise exec -- go test -tags test_dep -count=1 ./model/scalav2/... ./model/go/... (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/go-wave-foundations-final.log), GOFLAGS=-tags=test_dep mise exec -- make lint-code 'LINT_CODE_TARGETS=./model/go/umpire ./model/scalav2/goir' GOLANGCI_LINT=.flow/tmp/fn-107/task2-tools/golangci-lint-v2.13.1 ERRORTYPE=.flow/tmp/fn-107/task2-host-tools/errortype GOLANGCI_LINT_FIX=false (rc0; /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/parallel-integrated-gates/lint-wave-foundations-final.log), MISE_TRUSTED_CONFIG_PATHS=/private/tmp/fn-107-parallel-20260930T204609Z/15/temporal mise exec -- go test -tags test_dep ./model/scalav2/... ./model/go/..., MISE_TRUSTED_CONFIG_PATHS=/private/tmp/fn-107-parallel-20260930T204609Z/15/temporal GOFLAGS=-tags=test_dep mise exec -- make lint-code LINT_CODE_TARGETS=./model/scalav2/goir GOLANGCI_LINT=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task2-tools/golangci-lint-v2.13.1 ERRORTYPE=/Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task2-host-tools/errortype GOLANGCI_LINT_FIX=false, gofmt -l model/scalav2/goir, Actual independent native rereview SHIP; task15-final-review-receipt.json; task15-review-r2 immutable source artifact, Unchanged reviewed source, user HEAD and raw index guarded at bbc7dab1a4f9f0760f8c9316e9bfc5d9701334b7; task15-post-ship-guards.json; worker evidence at /Users/stephan/Workspace/skunkworks/umpire/temporal/.flow/tmp/fn-107/task15-reviewfix-r1-evidence.json
- PRs: