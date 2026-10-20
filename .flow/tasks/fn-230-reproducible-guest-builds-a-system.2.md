---
satisfies: [R2, R3, R4]
---
# fn-230-reproducible-guest-builds-a-system.2 After .1: commit the Temporal guest with go:embed; freshness step in model-check; umpire-guest target; docs

## Description
Commit model/temporal/hosts/go/umpire_temporal_guest.wasm built by guest.sh; export Guest() []byte via go:embed; remove UMPIRE_TEMPORAL_WASM and Cargo target path logic from the Temporal host tests and slice targets (umpire-check-model-slice, umbrella) so they use the embedded guest. Add the guest.sh --check step to make model-check (skip with warning outside CI when the toolchain is missing, fail in CI) and make umpire-guest to refresh. Add .gitattributes for *.wasm binary. Update MIGRATE.md, README.md and the hosts README with the decision. Verify go test in the Temporal host module on a shell with Rust removed from PATH, a stale-guest negative control, and the full gate with model-time.

## Acceptance
- [ ] Temporal guest committed and embedded; no env var or target path in Go tests or slice targets
- [ ] go test passes without Rust on PATH
- [ ] Stale guest fails model-check with the summary; umpire-guest refreshes
- [ ] MIGRATE, README, hosts README, .gitattributes updated

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
