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
- [x] Update tools/gomad/Makefile slim test selection and backend/integration_test.go TestSlimGuestClosureCutsOnlyOptionalProviderSeams for positive ringpop retention/source-selection; preserve cloud/SQL exclusion and missing-test detection. The old ringpop-refusal test cannot remain mandatory under slim tags after unstubbing.
- [x] Slim guest build links ringpop; legacy gomad-tag source selection retains the stub and slim selection includes exactly one real implementation
- [x] Cloud archivers and SQL driver trim remain excluded
- [x] Slim module byte delta recorded against the matched current baseline preparation; fn-182 closeout sizes are historical context only
- [x] No diff under testcore, wasi config, helper protocol, network or clock model

## Done summary
Ringpop membership links into the gomad_wasm_slim guest: temporal/membership.go selects !gomad, so the slim guest selects exactly one real implementation, and membership_gomad.go and its test keep the legacy gomad-tag stub as a source-selection check. Cloud archivers and the SQL driver trim stay excluded; the Makefile slim selection and TestSlimGuestClosureCutsOnlyOptionalProviderSeams assert positive ringpop retention.

Size (R1): the matched preparation (same gomad, compiler and runtime; only the unstub differs) is 311,082,625 B → 314,279,810 B (+3,197,185 B, +1.01%) in .flow/artifacts/fn224-unstub-2026-10-21.md. The gomad-tip one-box module is 311,040,589 B without fn-224 and 314,171,975 B with it (+1.01%).

One-box cells with ringpop linked (fn-224 tip ab7625c647, seed 0, 4 GiB, 2^50 fuel):
- frontend-system-info (./tests/gomadfunctional TestFrontendSystemInfo): success with the leaf PASS line.
- mandatory smoke (./tests TestWorkflowTypeEncodingSuite/TestPlainASCII/Succeeds): success with the leaf PASS line.
- Neither hit a capacity refusal, so no per-workload tag is needed.

Gates on the merge candidate 87fcc5e24e exit 0: slim refusals, the gomad and default membership tests, lintcode tests, TestWASMArchitecture, slim closure, scenario preflight, lint-code-fast, gofmt. The R4 diff is empty under testcore, wasi, model, toolchain, wasmhost and rust.

Independent re-review: SHIP-WITH-FIXES, .1 done; the fixes are applied (fn224-ringpop-scenario artifact, "Merge with gomad"). Merged into gomad as 2256f61d7a.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 3925d83499, ab7625c647, 87fcc5e24e, 502e7ce811, 2256f61d7a
- Tests: candidate2/gates.sh on 87fcc5e24e: all exit 0, onebox-premerge2.sh: frontend-system-info and smoke success with leaf PASS, evidence/fn224/review2/codex2.out: SHIP-WITH-FIXES, .1 done
- PRs: