---
satisfies: [R1]
---
# fn-242-gomad-shrink-the-qualification.1 Obligation inventory: name the failure every subgate, reader and receipt check catches

## Description
Analysis only; may start before the starting point because it changes no cycle-qualified code. For each of the 145 required-final.json subgates, each reader/receiver module in rust/driver (receiving.py, final_profiles.py, final_retained.py, core_*.py, cell_reader.py, onebox_receiving.py, reach.py, terminal/cost/identity readers) and each receipt check: record the concrete failure it detects, the test or mutation that proves it catches that failure, or the deletion reason (re-proves another check, bookkeeping only). Also derive the stage table the verb must reproduce: for all 96 stages the exact argv, bounds, cores/memory/locks/scratch declarations and isolation class, from driver.py and adapters.py. Artifacts: .flow/artifacts/fn242-obligation-inventory.md plus a JSON sidecar and .flow/artifacts/fn242-stage-table.json. Touches: .flow/artifacts only.

## Acceptance
- [ ] Every subgate, reader module and receipt check has one inventory row with failure, proof (test/mutation) or deletion reason
- [ ] No row is marked keep without a named catching test or mutation
- [ ] Stage table lists all 96 stages with argv, bounds, resources and isolation class, derived from the driver, not retyped
- [ ] No code changes

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
