---
satisfies: [R2]
---
# fn-244-package-layout-of-modelumpire.2 Go Runner out of the generated protobuf tree

## Description
After fn-228.6, before fn-228.7. Move the hand-written runner package out of hosts/go/umpire/*, drop the proto-go --check exception, update importers (tools/umpire/cmd/umpire, testpilot/temporal/act, canary), the fn-228 spec's import path and the module map.

## Acceptance
- [ ] Runner outside the generated tree; freshness check has no exception
- [ ] Importers updated; Go tests, slice and default model gate pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
