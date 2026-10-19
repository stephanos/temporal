---
satisfies: [R1, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.1 Unstub ringpop in the slim guest and record the module size delta

## Description
Change temporal/membership_gomad.go and its test to the gomad tag only so the gomad_wasm_slim guest links the default ringpop membership module while the native build keeps the stub. Keep provider_cloud_gomad.go and flag_sql_wasm_slim.go excluded. Rebuild the slim one-box guest with backend.TemporalGuestBuildTags() and record the module byte delta against the fn-182 closeout sizes. Confirm testcore, wasi.Config, the helper protocol and the network and clock models are untouched.

**Size:** S
**Files:** temporal/membership_gomad.go, temporal/membership_gomad_test.go, tools/gomad/internal/backends/wasm/README.md (slim exclusion list)

## Acceptance
- [ ] Slim guest build links ringpop; native gomad build still returns the stub error
- [ ] Cloud archivers and SQL driver trim remain excluded
- [ ] Slim module byte delta recorded against fn-182 closeout sizes
- [ ] No diff under testcore, wasi config, helper protocol, network or clock model

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
