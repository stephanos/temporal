---
satisfies: [R1]
---
# fn-95-gomad-f1-restore-the-checkout-on.1 Build the go1.27.1 toolchain on darwin/arm64 and pass gomad doctor

## Description
Run `make gomad3` with GOROOT on a stock go1.27.1; confirm `tools/gomad3/.bin/gomad doctor` reports the runner, all adapters, and the artifact store available. Repair any darwin build failure.

## Acceptance
- `make gomad3` exits 0 on darwin/arm64
- `gomad doctor` reports runner available

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
