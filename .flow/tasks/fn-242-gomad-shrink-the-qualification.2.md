---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.2 Rust cycle verb: stage DAG, resource scheduler, serial and parallel dispatch

## Description
Add a gomad verb (name decided here, documented in CLI.md) that loads the qualification config and its sha256, builds the 96-stage DAG and dispatches stage processes with argv identical to the retired driver, in serial (predecessor edge between rows) and parallel mode with the original resource scheduler: core and memory caps, locks, scratch declarations, the 8 GiB live-memory guard, heavy/budgeted isolation, exclusive timing stages, twelve-worker busy-boundary groups reaped before return, and the 3,600-second wall limit with the infrastructure exit. Retain command exits/signals and candidate identities per stage. Golden: a dry-run plan for all 96 stages compared byte-for-byte with the .1 stage table; mutation fixtures for each scheduler bound. No Python. Touches: tools/gomad/rust/gomad/src/cycle/** (new), cli, CLI.md.

## Acceptance
- [ ] Dry-run plan golden equals the .1 stage table for all 96 stages (argv, bounds, resources, isolation)
- [ ] Serial and parallel modes dispatch the same stage processes; parallel honors caps, locks, scratch, live-memory guard and exclusive stages, each with a mutation fixture
- [ ] 3,600 s limit exits with the infrastructure classification and retains partial exits
- [ ] Per-stage exits/signals and identities retained; no Python added

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
