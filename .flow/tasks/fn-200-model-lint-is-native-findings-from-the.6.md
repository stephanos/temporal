---
satisfies: [R6]
---
# fn-200-model-lint-is-native-findings-from-the.6 Samples lint in the framework check; delete the IR lint and acceptance files

## Description
Run native lint over the six samples after their native checks, with gaps in source; delete umpire-lint, the umpire-check-lint target, every .lint.json, and the IR lint package's static and graph kinds (keep only what the modality decision retains).

**Files:** model/framework/check/FrameworkLoop.scala, model/framework/samples/*.scala, Makefile, tools/umpire/lint/, tools/umpire/cmd/umpire-lint/, model/**/*.lint.json.

**Quick commands:** `make check-framework`; `go build ./...`

## Acceptance
- Six samples pass with their gaps in source.
- No .lint.json remains; `make umpire-check-lint` no longer exists.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
