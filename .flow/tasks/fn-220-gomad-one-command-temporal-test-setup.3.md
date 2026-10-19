---
satisfies: [R4]
---
# fn-220-gomad-one-command-temporal-test-setup.3 Document the one-command workflow

## Description
Update internal/backends/wasm/CLI.md and tools/gomad/README.md (the paragraph at README lines 150-172) to describe the one-command workflow and remove the statement that arbitrary Temporal tests lack a setup command. State that support is limited to the named test and that qualification is a separate spec.

**Size:** S
**Files:** tools/gomad/README.md, tools/gomad/internal/backends/wasm/CLI.md

## Acceptance
- [ ] CLI.md and README describe the install and explore commands for a Temporal test package
- [ ] No claim of support beyond TestFrontendSystemInfo; qualification named as separate

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
