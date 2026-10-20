---
satisfies: [R1]
---
# fn-223-gomad-delete-the-native-backend.1 Remove the nested native module and prove the default set builds

## Description
Confirm fn-219 is closed and its archcheck string check passes. On a throwaway branch remove internal/backends/native and both its require and replace declarations in tools/gomad/go.mod, run go build ./... && go vet ./... on the default set; if anything outside the known seam, fixtures and glue fails, stop and report that fn-219 missed a consumer. Then apply the removal for real, delete the gomad_native CLI seam (cmd/gomad/internal/cli/native_enabled*.go, native_disabled.go and tests) collapsing to the WASM-only launcher path, delete internal/backends/testfixture/native.go, internal/evidence/record/testfixture/native.go, every *_native_test.go under internal/runner and cmd/gomad/internal/cli, and internal/runner/internal/exploration/integration_test.go if native-only. Retire its obsolete TestSimulationGateSelectsProcessNetworkHandles consumer in internal/archcheck/integration_test.go before the full-module test gate. Run go mod tidy and retain its output, preserving the root server requirement used by dynamic WASM fixture modules. Redirect the four native-relative fixture paths in tools/gomad/internal/backends/wasm/backend/integration_test.go and the one read in tools/gomad/internal/backends/wasm/command/command_test.go to the promoted ../testdata/io_failure_module fixture only after proving byte identity (expected source-file SHA-256 28f9ea5f31fa073209210bf05079bb9de4f13409e4b31bed9829b8f0a8034513). Retire only native routing/ownership/allowlist readers and matching fixtures in cmd/tools/lintcode/**; run focused router tests and keep retained routing protections. Implement the coordinated .1-.3 batch before claiming final acceptance.

**Size:** M
**Files:** tools/gomad/go.mod, tools/gomad/go.sum, tools/gomad/internal/backends/native/** (removal), tools/gomad/cmd/gomad/internal/cli/**, tagged fixtures and tests, tools/gomad/internal/archcheck/integration_test.go, tools/gomad/internal/backends/wasm/backend/integration_test.go, tools/gomad/internal/backends/wasm/command/command_test.go, cmd/tools/lintcode/**, tools/gomad/internal/backends/wasm/layout_test.go (relative-path dependency guard)
**Touches:** tools/gomad/go.{mod,sum}, tools/gomad/internal/backends/native/**, tools/gomad/cmd/gomad/internal/cli/**, native-tagged fixtures/tests, tools/gomad/internal/archcheck/integration_test.go, tools/gomad/internal/backends/wasm/backend/integration_test.go, tools/gomad/internal/backends/wasm/command/command_test.go, cmd/tools/lintcode/** (native router/allowlist retirement and matching focused tests only), tools/gomad/internal/backends/wasm/layout_test.go (focused regression rejecting executable native-relative fixture paths).

### Exact source-test lane

Retain the original failed unselected tests.command/tests.log and classification history. Use the full module package universe for the ordinary test_dep lane with only the exact anchored union from /home/agent/wt/evidence/fn223/ordinary-skip-selector.txt excluded there. This union is the following nine source roots plus the 39 live roots named in fn-232.13, not 48 transferred gates. Run these nine positively with the exact source-run-selector.txt in ./internal/backends/wasm/backend, explicit authenticated pinned Go and fn-232 baseline-helper paths/identities, and all existing assertions/subtests/bounds intact:

- `TestProvenanceRejectsChangedIdentityBeforeExecution`
- `TestProviderRejectsEmptyModelIdentity`
- `TestTestSourceChangesInvalidateCompilationCache`
- `TestBuildSelectionChangesInvalidateCompilationCache`
- `TestSlimGuestClosureCutsOnlyOptionalProviderSeams`
- `TestAssemblyHeaderChangesInvalidateCompilationCache`
- `TestStockPreparationDisablesAmbientPGO`
- `TestStockPreparationClearsAmbientCompilerSettings`
- `TestInterruptedCachePublicationIsRebuilt`

These are required provenance/admission/cache/build-selection/slim-go-list/preparation checks, not guest-execution exemptions. Keep positive selected/matched package-qualified inventories, all subtest results and numeric exits. Prove the two lists are disjoint and the union exactly 48 with no accidental same-name package matches or unclassified omissions; all other source roots stay in the ordinary full-package lane. Only the 39 actual live-guest roots transfer to fn-232 R12/.13 with complete behavior/fixture mapping. The two temporary-probe inventory races require clean frozen reruns; the real tidy/server-fixture regression requires its repair. No unselected full-module or qualification pass is claimed.
## Exact source-test lane

Retain the original failed unselected tests.command/tests.log and classification history. Use the full module package universe for the ordinary test_dep lane with only the exact anchored union from /home/agent/wt/evidence/fn223/ordinary-skip-selector.txt excluded there. This union is the following nine source roots plus the 39 live roots named in fn-232.13, not 48 transferred gates. Run these nine positively with the exact source-run-selector.txt in ./internal/backends/wasm/backend, explicit authenticated pinned Go and fn-232 baseline-helper paths/identities, and all existing assertions/subtests/bounds intact:

- `TestProvenanceRejectsChangedIdentityBeforeExecution`
- `TestProviderRejectsEmptyModelIdentity`
- `TestTestSourceChangesInvalidateCompilationCache`
- `TestBuildSelectionChangesInvalidateCompilationCache`
- `TestSlimGuestClosureCutsOnlyOptionalProviderSeams`
- `TestAssemblyHeaderChangesInvalidateCompilationCache`
- `TestStockPreparationDisablesAmbientPGO`
- `TestStockPreparationClearsAmbientCompilerSettings`
- `TestInterruptedCachePublicationIsRebuilt`

These are required provenance/admission/cache/build-selection/slim-go-list/preparation checks, not guest-execution exemptions. Keep positive selected/matched package-qualified inventories, all subtest results and numeric exits. Prove the two lists are disjoint and the union exactly 48 with no accidental same-name package matches or unclassified omissions; all other source roots stay in the ordinary full-package lane. Only the 39 actual live-guest roots transfer to fn-232 R12/.13 with complete behavior/fixture mapping. The two temporary-probe inventory races require clean frozen reruns; the real tidy/server-fixture regression requires its repair. No unselected full-module or qualification pass is claimed.

## Acceptance
- [ ] internal/backends/native and both its require and replace are gone; go mod tidy output retained
- [ ] Unselected go build ./... && go vet ./... pass; the full-package ordinary test_dep lane minus exactly 48 anchored roots and the separate positive nine-source lane both pass with required pinned inputs, selected/matched inventories and numeric exits; no source gate is transferred or skipped
- [ ] Exact 39-root live behavior transfer is accepted by fn-232 R12/.13 and remains pending; preserve the failed unselected run and all subsequent repair/rerun dispositions without relabeling a full-module pass
- [ ] Five WASM fixture references resolve to byte-identical promoted data; focused cmd/tools/lintcode router tests and retained source-routing checks pass
- [ ] Existing WASM CLI fixtures pass byte-for-byte; a native installation header is refused with the existing unregistered-backend status

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
