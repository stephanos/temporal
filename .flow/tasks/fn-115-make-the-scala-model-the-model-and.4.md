---
satisfies: [R5, R6, R7, R16, R17, R18]
---
# fn-115-make-the-scala-model-the-model-and.4 Promote producer-neutral runtime helpers and live orchestration

## Description
Promote producer-neutral runtime helpers and live orchestration. Implements R5, R6, R7, R16, R17, R18 using the reviewed parent contracts.

**Size:** M
**Files:** common/testing/testpilot helper destinations from the map; Temporal binding helpers; live importers of evaluation/recordedrun/publish/binding/replay/campaign
**Touches:** [common/testing/testpilot/**, tools/canary/**, tests/**, model/scalav2/**, .plans/umpire-migration-*.json, .gitignore]

### Approach
- Preserve the frozen obsolete TestTestpilotOwnsCaseProtocolAndRuntime test, but transfer its meaningful dependency restrictions/negative checks into live ownership tests with only the six map-approved Driver edges. Record the original failure; transitional Quick excludes exactly this old test until task 6 archives regression. All other old tests remain enabled.
- Preserve replay’s existing bounded teardown through the neutral Temporal binding adapter. Omit the Model-specific campaign/bridge_live_test.go and replay/bridge_live_test.go from the neutral copy; keep it unchanged in original tools while task 6 transfers its claims to IR bridge integration tests.
- Keep copied fixture bytes discoverable by Git: add narrow new-home testdata exceptions to .gitignore as necessary and verify every new fixture path with git check-ignore. Do not defer this essential copy requirement until final ignore reconciliation.
- Copy the existing functional Run/Report signature encoder and its tests into neutral `common/testing/testpilot/recordedrun` ownership. Preserve the actual encoder-to-command-parser contract check. Keep old functional definitions available to the frozen old command until task 6 switches the retained live command; do not add a runtime dependency on functional test wiring.
- Copy each map-selected neutral helper into its final runtime or Temporal-driver home while preserving original tooling bytes for the archive. Move shared casefile ownership with its callers; preserve parsing/admission, passivity, immutable identity and exclusive publication semantics.
- Give live campaign/replay protocols the map's chosen producer-neutral home. Change live callers to those interfaces, eliminating forwarding-only recorded-run aliases while keeping actual exclusive-write behavior.
- Keep the current command paths executable during this phase. Preserve original old-tools source bytes rather than cleaning their future archival copies; update the manifest if a live command must be prepared for the later namespace swap.
- Check production and test import graphs, capability/preflight behavior and affected canary/functional compatibility. No model imports may enter Testpilot.

### Investigation targets
**Required** (current paths at planning time; follow the recorded move map after relocation):
- `tools/umpire/replay/recorded.go:21`
- `tools/umpire/recordedrun`
- `tools/umpire/evaluation`
- `tools/umpire/binding`
- `model/scalav2/explore/bridge.go:13`
- `tests/testpilot_scala_canary_test.go`

### Quick commands
CC=/usr/bin/clang mise exec -- go test -tags test_dep -skip '^TestTestpilotOwnsCaseProtocolAndRuntime$' ./common/testing/testpilot/... ./tools/canary/... ./tools/umpire/... ./model/scalav2/...; make lint-code-fast

### Execution constraints
Preserve the authorized uncommitted baseline and comments except the explicit R25 historical-attribution change. No staging, commits, worktrees or recursive deletion. Once task 2 exists, run the complete golden verification after every task. Resolve task-1 map choices before using projected destination names; capture any change in the map and downstream task briefs before work.
## Acceptance
- [ ] Live consumers use the map's neutral helper/orchestration homes without added model knowledge or changed compatibility decisions.
- [ ] Archive-original hashes remain unchanged and no live consumer starts importing an archive.
- [ ] Passive evaluation, recorded identity, exclusive publication, preflight and affected consumers pass along with the semantic goldens.

## Done summary
Bookkeeping 2026-10-18: parent spec already closed

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: