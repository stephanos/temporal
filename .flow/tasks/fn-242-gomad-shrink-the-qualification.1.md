---
satisfies: [R1]
---
# fn-242-gomad-shrink-the-qualification.1 Obligation inventory: name the failure every subgate, reader and receipt check catches

## Description
Analysis only; may start before the starting point because it changes no cycle-qualified code. For each of the 145 required-final.json subgates, each reader/receiver module in rust/driver (receiving.py, final_profiles.py, final_retained.py, core_*.py, cell_reader.py, onebox_receiving.py, reach.py, terminal/cost/identity readers) and each receipt check: record the concrete failure it detects, the test or mutation that proves it catches that failure, or the deletion reason (re-proves another check, bookkeeping only). Also derive the stage table the verb must reproduce: for all 96 stages the exact argv, bounds, cores/memory/locks/scratch declarations and isolation class, from driver.py and adapters.py. Preserve the original raw argv table unchanged. Add an explicit old-Python-endpoint to surviving-Rust-endpoint mapping for orchestration entry points, generated configuration bindings and retained producers, including compose.py, configure.py, static_stages.py, primary_stages.py and every referenced workload_tools entry point. Leaf tool/guest commands and their arguments remain byte-for-byte unchanged; only named orchestration endpoint substitutions are permitted. Each substitution names its original contract and catching execution/mutation proof. Task .9 owns fresh candidate composition/preparation; .10 owns retained producer adapters; .7 cannot delete an endpoint before its replacement passes. Artifacts: .flow/artifacts/fn242-obligation-inventory.md plus a JSON sidecar and .flow/artifacts/fn242-stage-table.json. Touches: .flow/artifacts only.

Inventory the three launcher-policy failures for the Rust port and name a catching regression for each. Historical reference only: /home/agent/wt/evidence/fn232-split-closure/launch-current-source-v1.py:342 catches BaseException around Popen and can erase a charge after fork followed by KeyboardInterrupt; :98 can re-enroll with create=True when a fixed journal binding is missing despite older started outputs; :973 checks only the12GiB inner floor and omits the45GiB HOST/20GiB overlay recheck immediately before reservation. Keep ambiguous spawn charged, refuse missing/mismatched bindings after initial enrollment, and test all three floors at the reservation boundary. Do not patch the historical Python launcher. Implementation starts after the narrow fn-232.30 checkpoint; analysis here may prepare before it.

## Acceptance
- [ ] Every subgate, reader module and receipt check has one inventory row with failure, proof (test/mutation) or deletion reason
- [ ] No row is marked keep without a named catching test or mutation
- [ ] Stage table lists all 96 stages with argv, bounds, resources and isolation class, derived from the driver, not retyped
- [ ] Endpoint inventory covers composition/preparation and every retained producer, assigns .9/.10 ownership and maps every permitted Rust orchestration substitution; unchanged leaf argv and all original obligations remain exact
- [ ] Inventory sizes .3/.4/.9/.10 against one work iteration and requires any oversized coherent surface to be split through Flow before implementation, with cutover waiting for every successor
- [ ] No code changes
- [ ] Inventory rows name catching regressions for ambiguous spawn charging, missing/mismatched journal enrollment and all three dispatch resource floors; original stage/obligation coverage is preserved

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
