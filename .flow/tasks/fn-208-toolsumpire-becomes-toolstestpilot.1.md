---
satisfies: [R1]
---
# fn-208-toolsumpire-becomes-toolstestpilot.1 Delete the Model-reading Go packages and commands

## Description
Delete tools/umpire/{ir,interp,internal/engine,check,realization,lower,explore,lint,export,internal/runtimecel} and the Model-reading half of conformance, plus umpire-gen-cases, umpire-ir-bridge, umpire-lint, umpire-check-samples and the generated Go bindings of the IR proto; `go build ./...` passes; no gate references them.

**Files:** tools/umpire/**, api/umpire/**, Makefile, tests/testpilot_exploration_test.go.

**Quick commands:** `go build ./...`; `go vet ./...`

## Acceptance
- Build and vet pass; `grep -rn tools/umpire --include=*.go` shows only tools/testpilot after task 2.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
