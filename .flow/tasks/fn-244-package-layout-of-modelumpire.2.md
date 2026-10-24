---
satisfies: [R2]
---
# fn-244-package-layout-of-modelumpire.2 Go module split into host, runner and gen; Kettle Go under reference/kettle

## Description
After fn-228.6, before fn-228.7. Split the Go module umpire.local/model: the wasm Host moves from the root package to hosts/go/host, the Runner to hosts/go/runner, generated protobuf packages to hosts/go/gen/...; the proto-go freshness check covers exactly gen/ with no exception. Move Kettle's Go server and Kettle-specific host tests under reference/kettle/ as their own Go module with a replace directive. Update all importers (tools/umpire/cmd/umpire, common/testing/testpilot/temporal/act, tools/canary, model/temporal Go), the fn-228 spec's import path and the module map.
## Acceptance
- [ ] host, runner and gen packages; freshness check without exceptions
- [ ] Kettle Go server and tests under reference/kettle
- [ ] Importers updated; Go tests, live slice and default model gate pass
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
