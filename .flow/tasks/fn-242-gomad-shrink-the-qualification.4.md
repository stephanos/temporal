---
satisfies: [R2]
---
# fn-242-gomad-shrink-the-qualification.4 Port the kept final readers: profiles, retained, terminal, reach, onebox support, cost

## Description
Same contract as .3 for final_profiles.py, final_retained.py, terminal_module_receiving.py, reach.py (test-wasm reach map reader, including the fn-232.14 report-versus-refuse switch), onebox_receiving.py (fn222 two-cell support list), receiving_cost.py, executed_divergence.py, identity_harness.py and the subgate obligation closer for required-final.json. Touches: tools/gomad/rust/gomad/src/cycle/final/**.

## Acceptance
- [ ] Every kept final check has a Rust implementation and a mutation fixture
- [ ] Reach reader keeps the report-until-fn-232.14-closes behavior and the refuse-after behavior behind one switch
- [ ] Support-list production and validation reproduce the fn222 two-cell gate exactly
- [ ] Deleted checks each cite their .1 deletion reason

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
