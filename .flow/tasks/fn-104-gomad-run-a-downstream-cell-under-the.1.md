---
satisfies: [R1]
---
# fn-104-gomad-run-a-downstream-cell-under-the.1 Add --working-dir to explore, qualify, and analyze and resolve adapters and the schema mount from the module root

## Description
C1. explore/qualify/analyze take the target module from os.Getwd(); adapter selection reads <cwd>/go.mod directly (deterministicio/adapter_registry.go). Add --working-dir (absolute, clean, must be a module root; invalid input otherwise), select adapters from the resolved module root, and let a read-only mount source name the server module directory (local replace or module cache) so a downstream module can mount the server schema. Document the forced build environment (GOWORK=off, GOFLAGS cleared, GOENV=off, -mod=readonly) and its consequences in the README.

## Acceptance
- a go-test target in a module outside this repository that replaces go.temporal.io/server with a local path prepares and analyzes with --working-dir
- relative, missing, or non-module-root working directories and unresolvable mount sources are invalid input with a named reason
- README documents the forced environment, vendoring, and private-module limits


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
