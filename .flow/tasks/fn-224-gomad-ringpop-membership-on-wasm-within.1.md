---
satisfies: [R1, R4]
---
# fn-224-gomad-ringpop-membership-on-wasm-within.1 Unstub ringpop in the slim guest and record the module size delta

## Description
Change temporal/membership_gomad.go and its test to the gomad tag only and remove !gomad_wasm_slim from temporal/membership.go. The slim guest must select exactly one real membership implementation; retain the legacy gomad-tag stub through source-selection checks only, without reviving native execution. Keep provider_cloud_gomad.go and flag_sql_wasm_slim.go excluded. Rebuild the slim one-box guest with backend.TemporalGuestBuildTags() and record the module byte delta against a freshly prepared matching baseline in the same host/window with the same compiler/configuration and only the unstubbing change; retain source/module identities for both. fn-182 sizes are historical context only. Confirm testcore, wasi.Config, the helper protocol and the network and clock models are untouched.

**Size:** S
**Files:** temporal/membership.go, temporal/membership_gomad.go, temporal/membership_gomad_test.go, tools/gomad/internal/backends/wasm/README.md (slim exclusion list)

**Touches:** tools/gomad/Makefile, tools/gomad/internal/backends/wasm/backend/integration_test.go, temporal/membership.go, temporal/membership_gomad.go, temporal/membership_gomad_test.go, tools/gomad/internal/backends/wasm/README.md (slim exclusion list)

## Acceptance
- [ ] Update tools/gomad/Makefile slim test selection and backend/integration_test.go TestSlimGuestClosureCutsOnlyOptionalProviderSeams for positive ringpop retention/source-selection; preserve cloud/SQL exclusion and missing-test detection. The old ringpop-refusal test cannot remain mandatory under slim tags after unstubbing.
- [ ] Slim guest build links ringpop; legacy gomad-tag source selection retains the stub and slim selection includes exactly one real implementation
- [ ] Cloud archivers and SQL driver trim remain excluded
- [ ] Slim module byte delta recorded against the matched current baseline preparation; fn-182 closeout sizes are historical context only
- [ ] No diff under testcore, wasi config, helper protocol, network or clock model

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
