---
satisfies: [R1, R2]
---
# fn-98-gomad-f4-close-the-tests-capability.2 Close the ./tests closure on darwin/arm64

## Description
Run closure analysis of `go-test ./tests` with tags disable_grpc_modules,gomad,test_dep; add a darwin counterpart of temporal-functional-tests-linux-amd64 with exact facts if needed.

## Acceptance
- zero unsupported_target findings on darwin

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
