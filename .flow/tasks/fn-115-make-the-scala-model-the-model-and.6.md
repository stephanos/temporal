---
satisfies: [R2, R4, R5, R6, R7, R8, R9, R10, R11, R14]
---
# fn-115-make-the-scala-model-the-model-and.6 Move the live model and tooling and isolate the frozen archives

## Description
Move the live model and tooling and isolate the frozen archives. Implements R2, R4, R5, R6, R7, R8, R9, R10, R11, R14 using the reviewed parent contracts.

**Size:** M
**Files:** model and tools live/archive trees; affected imports and source paths; essential Make/CI/ignore references; archive module metadata
**Touches:** [model/**, model0/**, tools/umpire/**, tools/umpire0/**, common/testing/testpilot/**, tests/**, tools/canary/**, Makefile, .github/workflows/**, .gitignore, .plans/umpire-migration-*.json]

### Approach
- Remove the task-4 transitional test-command exception by excluding archived regression from the main module. Run final live checks without the obsolete TestTestpilotOwnsCaseProtocolAndRuntime skip; preserve its surviving ownership claims in the live Testpilot tests and final architecture gate.
- Before archiving original campaign/bridge_live_test.go, replay/bridge_live_test.go and the old functional bridge helper, transfer their actual admission/identity/crossed-subject/reduction/proposal protocol claims into the live IR bridge integration tests. Task 4 intentionally leaves both Model-specific bridge tests in original tools instead of copying a Lean executable dependency into Testpilot. Require executed tests, no Lake or skip-only success.
- Switch the retained `umpire-repeat` signature compatibility test to the promoted recordedrun encoder and migrate functional callers before removing the old functional signature definitions. Preserve the real encoder/parser cross-check and all signature claims; do not permit live tooling imports of functional cluster wiring.
- Perform the map's namespace swap as one coherent mechanical phase: move original old tools aside, populate the new live tools tree from the prepared reader/lower/conformance/export/explore and retained commands, and update all affected live importers together.
- Move the Scala project/IR/Cases/specimens/docs into their final model locations; archive the other model trees. Use guarded filesystem moves and source manifests, not git staging or recursive deletion. Preserve frozen original bytes, with only the documented archive-root go.mod/README additions.
- Apply the closed source path/derived-hash migration; keep line/column positions and stable namespaces. Update fixture root resolution, bridge defaults, generated source links and existing source-confinement checks together.
- Repair essential active command/build/CI references immediately, retaining temporary orchestration scripts only until task 10 replaces them. Keep the live pinned external-tool launcher out of the archive. Add archive import/build exclusion checks before accepting the new layout.
- Run the golden and affected Go/model gates from the new paths; no stable intermediate result may depend on an archive. Capture separate move/import evidence and defer cleanup to following tasks.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `model/scalav2/run.sh`
- `model/scalav2/goir/isolation_test.go:39`
- `model/scalav2/goir/testpilot/generated.go:219`
- `model/scalav2/explore/trace.go:62`
- `tools/umpire/cmd/umpire-ir-bridge/main.go:35`
- `Makefile:1182`

### Quick commands
The exact renamed model check/generation commands from task 1; CC=/usr/bin/clang mise exec -- go test -tags test_dep ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/...; make lint-scala; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] model has no Go files, no scalav2 subtree remains, live tools occupy the reviewed destinations, and archive modules are excluded from main-module traversal.
- [ ] Every live importer resolves outside archives; golden comparison accepts only the exact recorded source transformation and derived hashes.
- [ ] Frozen archive bytes and unrelated owner/index state are preserved; move/import changes are independently inspectable.
- [ ] New-layout model/generation and affected consumer gates pass; source links and missing-fixture diagnostics resolve correctly.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
