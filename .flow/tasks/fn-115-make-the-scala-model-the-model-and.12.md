---
satisfies: [R2, R5, R7, R8, R9, R10, R14, R17, R18, R20, R25]
---
# fn-115-make-the-scala-model-the-model-and.12 Enforce the final import graph and reconcile build, export and archive rules

## Description
Enforce the final import graph and reconcile build, export and archive rules. Implements R2, R5, R7, R8, R9, R10, R14, R17, R18, R20, R25 using the reviewed parent contracts.

**Size:** M
**Files:** architectural import/gate tests; export tool runner; Make/CI/ignore configuration; model historical-reference cleanup
**Touches:** [tools/umpire/**, common/testing/testpilot/**, tools/canary/**, model/**, Makefile, .github/workflows/umpire.yml, .gitignore, .plans/UMPIRE_MODULES.md]

### Approach
- Enforce tools/umpire/internal/golden as test support only, with IR/protobuf/stdlib imports and no production importer. Permit exploration, conformance, runtime and recordedrun imports only in the lowerer external test package for complete artifact checks; do not widen lowerer production edges. Preserve narrow Git visibility exceptions for both golden fixture trees.
- Complete executable production and test import-graph rules from the map, including reader independence, runtime independence, export and exploration allowances and archive exclusion. Require actual callers for every live tooling package/command.
- Move the export shell runner's live pinned-tool behavior into explicit opt-in Go tests; archive-only commands and variables are deleted. Repair the existing Umpire CI selector to run live nonempty suites; retain canary-build and do not add a JVM job.
- Reconcile ignore rules against final paths and prove a newly generated fixture is discoverable. Verify the model gate and ordinary Go tests introduce no additional generated/untracked drift.
- Remove only historical attributions inside model while preserving explanations and lifted source line/column positions. Put the literal-bearing vocabulary check in Go tooling outside model and invoke it from the gate.
- Check all remaining model/umpire Make targets, main-module archive exclusion and public surface inventory. Apply any essential repair that earlier tasks already required immediately, not by leaving it deferred until here.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/backends/run.sh`
- `model/scalav2/backends/README.md`
- `model/scalav2/goir/isolation_test.go`
- `Makefile`
- `.github/workflows/umpire.yml`
- `.gitignore`

### Quick commands
All remaining model/umpire targets from the map; CC=/usr/bin/clang mise exec -- go build ./...; CC=/usr/bin/clang mise exec -- go vet ./...; make lint-code; make lint-model

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Import and export-surface rules cover production and tests, name offending files, and reject archive dependencies.
- [ ] Every remaining model/umpire target succeeds; existing CI selects live nonempty suites and production canary build remains valid.
- [ ] Main-module build/vet/lint exclude archives; ignored-path checks admit future fixtures and gates leave no extra drift.
- [ ] No file or filename inside model contains prohibited historical references; the external check catches new ones while source coordinates and goldens stay fixed.

## Done summary
Enforced the final import graph as Go tests that name the offending file and import: Testpilot imports nothing under `tools/umpire`, the reader nothing of Testpilot, export only the reader, the golden helper is test support only, the lowerer reaches exploration, conformance and recorded Runs only from its external test package, and no live file imports an archive. A new test requires every `tools/umpire` package to have a live importer or a Make/CI runner. `tools/umpire/export/run.sh` is replaced by opt-in Go tests (`UMPIRE_BACKENDS=require`). Makefile, CI and ignore rules match the final layout; the branch-only blanket `testdata/` ignore rule is gone. Nothing under `model/` mentions the retired front end (60 lines in 18 files before), with source positions unchanged, and the gate's first step is a Go vocabulary check that fails on a new mention.

Defect repairs: the dead campaign integration test is removed (its claim lives in the Scala exploration test), a test-code data race on the Run ID is fixed, `make lint-api` passes through `proto/api-linter.yaml` path entries, and `make lint-code` passes using the Go-only patch shared with `lint-code-fast`.

Build, vet with live tags, the full Go suite (48 packages, 5,100 tests), model gate, Case/fixture/canary checks, the live test selection (81 identities) and all linters pass; IR, Cases, fixtures and the 1,411 goldens are byte-identical. Independent review (Claude Fable, fresh context): NEEDS_WORK in round 1, SHIP in round 2. Recorded, not fixed: `make umpire-check-backends` needs P and .NET and is deferred by the owner; plain `go vet -tags test_dep ./...` reports 11 findings in three files identical to `main`; the canary policy `workflowPath` names the workflow the owner deleted and fails closed, an owner decision. Handover: .flow/tmp/fn115-12-summary.md; evidence: .flow/tmp/fn115-12-evidence.json; reviews: .flow/tmp/fn115-12-review/. No agent commits.
## Evidence
- Commits:
- Tests: go test -count=1 -json -tags 'test_dep canary_harness' ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... ./tests/testcore/testpilot/..., make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks, make umpire-check-live-tests, make lint-model, make lint-api, Independent review round 2 SHIP (claude-fable-5-1); .flow/tmp/fn115-12-review/round2-review.md
- PRs: