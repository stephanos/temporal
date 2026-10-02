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
All remaining model/umpire targets from the map; CC=/usr/bin/clang mise exec -- go build ./...; CC=/usr/bin/clang mise exec -- go vet ./...; make lint-code; make lint-scala

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Import and export-surface rules cover production and tests, name offending files, and reject archive dependencies.
- [ ] Every remaining model/umpire target succeeds; existing CI selects live nonempty suites and production canary build remains valid.
- [ ] Main-module build/vet/lint exclude archives; ignored-path checks admit future fixtures and gates leave no extra drift.
- [ ] No file or filename inside model contains prohibited historical references; the external check catches new ones while source coordinates and goldens stay fixed.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
