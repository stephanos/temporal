---
satisfies: [R1]
---
# fn-223-gomad-delete-the-native-backend.1 Remove the nested native module and prove the default set builds

## Description
Confirm fn-219 is closed and its archcheck string check passes. On a throwaway branch remove internal/backends/native and both its require and replace declarations in tools/gomad/go.mod, run go build ./... && go vet ./... on the default set; if anything outside the known seam, fixtures and glue fails, stop and report that fn-219 missed a consumer. Then apply the removal for real, delete the gomad_native CLI seam (cmd/gomad/internal/cli/native_enabled*.go, native_disabled.go and tests) collapsing to the WASM-only launcher path, delete internal/backends/testfixture/native.go, internal/evidence/record/testfixture/native.go, every *_native_test.go under internal/runner and cmd/gomad/internal/cli, and internal/runner/internal/exploration/integration_test.go if native-only. Run go mod tidy and retain its output.

**Size:** M
**Files:** tools/gomad/go.mod, tools/gomad/go.sum, tools/gomad/internal/backends/native/** (removal), tools/gomad/cmd/gomad/internal/cli/**, tagged fixtures and tests

## Acceptance
- [ ] internal/backends/native and both its require and replace are gone; go mod tidy output retained
- [ ] go build ./... && go vet ./... && go test -tags test_dep ./... pass from tools/gomad with no selection
- [ ] Existing WASM CLI fixtures pass byte-for-byte; a native installation header is refused with the existing unregistered-backend status

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
