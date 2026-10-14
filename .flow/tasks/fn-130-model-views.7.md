---
satisfies: [R1, R2, R3, R4, R5]
---
# fn-130-model-views.7 Publish the complete view tree and close freshness, cost and preservation gates

## Description
Complete managed publication and the proven foreground gate join, then close the full spec with docs and exact preservation/cost evidence.

**Size:** M
**Files:** `tools/umpire/cmd/umpire-render/**`, `tools/umpire/render/generated*`, `tools/umpire/lint/lint.go` and focused sharing tests as proven in .1, `model/check/{Gate,Tools}.scala`, `model/check/test/Gate.test.scala`, `Makefile`, `.gitattributes`, `model/views/**`, existing reader/Model/module docs
**Touches:** [tools/umpire/cmd/umpire-render/**, tools/umpire/render/generated*, tools/umpire/lint/lint.go, tools/umpire/lint/*sharing*test.go, model/check/Gate.scala, model/check/Tools.scala, model/check/test/Gate.test.scala, Makefile, .gitattributes, model/views/**, tools/umpire/README.md, model/README.md, .plans/UMPIRE_MODULES.md]

### Approach
- Finish check/update and whole selected-tree staging/publication using the established command pattern. Protect symlink/path containment, unrelated selected scopes and old managed output on compile/write failure; avoid importing the Case-specific lower.SyncCases helper.
- Integrate the proven narrow foreground lint/render sharing with exactly the existing lint/check scopes, independent replay and gate failures. Retain normal Case-generation overlap and execute view freshness outside goChecks. No native lifetime repair is smuggled into this join.
- Generate every production view from the re-anchored fresh source baseline. Compare IR/Case bytes, table/check/receipt/replay identities and source inventory to the frozen pre-view baseline; include metadata-only changes, missing/stale/orphan controls and two-run bytes. Verify every machine against the independent projection inventory, including nested phase, queue fields, close policy, hidden-only self-loops and attributed full-state fallback; retain enabled-branch explanations without another evaluator pass.
- Mark SVGs generated, document text diffs/tooltips/projections/inferred uncertainty and commands, and measure full equivalent added elapsed gate cost against R5 without reduced scope or tuning. Preserve cold/warm and separate full-command wall receipts.

### Investigation targets
**Required:**
- `tools/umpire/lower/generated.go:440` - publication behavior reference only.
- `model/check/Gate.scala:392` - ordinary Cases/lint ordering and overlap.
- `model/check/Tools.scala:88` - foreground process ownership.
- `model/check/test/Gate.test.scala:790` - failed-lift/stale-Case gate controls.
- `tools/umpire/ir/ownership_test.go:252` - live command ownership.

### Quick commands
```bash
mise exec -- go test -count=1 -tags test_dep ./tools/umpire/render ./tools/umpire/cmd/umpire-render
```
Final gates run serially under `/tmp/umpire-heavy-gates.lock`: canonical `mise exec -- go test -tags test_dep -p 2 -timeout 30m -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...` with separate wall/exit/output, `make umpire-gen-model` with full artifact diff review, unchanged no-update `make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks`, `make umpire-check-cases`, applicable Scala gate tests/format/lint, Go changed-package lint, goldens and dependency/ownership checks. Record equivalent complete baseline/candidate added-cost measurements, then configured independent implementation and completion review. No partial gate or subset supplies closure credit.

## Acceptance
- [ ] All seven original view families and independently frozen full inputs produce the complete stable production tree; all IR/Case/receipt/replay preservation pins hold.
- [ ] Read-only freshness and safe selected-tree publication pass missing/extra/stale/symlink/failure controls, including under --skip-go-checks.
- [ ] Equivalent complete gate measurements meet R5 with unchanged coverage/concurrency/replay; all canonical full gates pass and docs/generated attributes reflect actual observed behavior.
- [ ] Configured independent review and completion review close the unchanged R1-R5 contract; unavailable capacity or failed measurements remain blocking.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
