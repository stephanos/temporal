---
satisfies: [R2]
---
# fn-249-gomad-retire-the-go-reference-models.3 Delete the test-only Go reference models

## Description
Delete Go model packages marked convert or obsolete, with their tests, `protocolgen` targets for deleted families, Makefile `go test` lines and archcheck/determinism inventory rows. Keep `runtimesource`, `capture`, guest-boundary wire generation and the toolchain overlay.

## Acceptance
- [ ] No test-only Go model package remains; `make -C tools/gomad generate validate` and the remaining Go tests pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
