---
satisfies: [R3]
---
# fn-220-gomad-one-command-temporal-test-setup.4 Restore the pure WASI setup boundary without changing configuration bytes

## Description
The integrated `make -C tools/gomad validate` at ef48c836b0 fails `TestWASMArchitecture` for both darwin/arm64 and linux/amd64: pure `wasi/temporal.go` imports `path/filepath` and `time`. Its schema capture also performs host filesystem effects through capture. Restore the established pure/host boundary without weakening PureImports, exempting the file, replacing complete-policy tests with normalized comparisons, or duplicating policy values. Keep complete policy assembly pure and make host schema capture an explicit host-owned step shared by consumers. Account for white-box WASI integration-test import cycles when choosing the smallest shared owner. Preserve all four final policy configurations byte-for-byte, actual schema bytes/mapping/default limits, errors, CLI grammar/defaults and existing manifest/config identities. Migrate production and fixture callers together. The failing existing architecture test supplies the regression's red evidence. Run both-platform architecture, complete policy and caller tests, affected lint, generation validation and shared default validate; retain actual before/after byte comparisons. No guest capability, model semantics, canonical artifact format, protocol or native change.

**Files:** tools/gomad/internal/backends/wasm shared Temporal policy/capture owner and its callers/tests; architecture regression coverage if needed; existing fn220 evidence.

## Acceptance
- [ ] Existing both-platform WASM architecture gate passes without import allowlist/exemption weakening
- [ ] Host filesystem capture no longer runs from the pure WASI policy owner
- [ ] One complete policy assembly and one shared schema-capture definition preserve independent legacy expectations and byte-identical complete final configurations
- [ ] All production/fixture/qualification callers migrate without import cycles or post-hoc policy overrides
- [ ] CLI/config/error behavior and recorded schema mappings/bytes remain unchanged
- [ ] Focused tests, affected lint and default generate/validate pass, with retained red/green and byte comparison evidence


## Done summary
Separated shared host schema capture into `internal/temporalinput` and kept complete four-policy assembly in pure WASI, consuming captured inputs. Migrated installation, inspection, qualification, measurement and white-box fixtures without import cycles or policy overrides. Preserved complete actual-schema configurations and full installation bytes, capture error behavior, schema mappings/defaults and CLI contracts. No architecture allowlist or exemption changed.

Existing both-platform architecture failure supplies red evidence; repaired architecture, full-policy/caller tests, independent historical measurement/error tests, affected lint and generation checks pass. Root integrated default validation passes after the separate Make helper-publication prerequisite correction. Logs and byte-comparison hashes are in `.flow/artifacts/gomad-integration-checkpoint-2026-10-20.md` and `/home/agent/wt/evidence/integration-2026-10-20/boundary-worker.md`.

Fresh current-helper install/discover/named TestFrontendSystemInfo execution/exact replay all exit 0; a further retained replay after removing the source checkout also exits 0. Full provenance and numeric exits are in `.flow/artifacts/fn220-setup-proof-2026-10-20.md` and `/home/agent/wt/evidence/fn220/current-setup.JLtluL/report.md`. This is not combined-candidate qualification.

stage: impl-review - ran (model: codex:gpt-6.1-sol:high)

Official three-draw implementation review finalized SHIP with zero findings. Receipt `/tmp/impl-review-receipt-657da2bc4466-fn-220-gomad-one-command-temporal-test-setup.4.json`, RID `a63de3f014fc407ba3fdc5b1533da87f`, binds `3cbd8b3cf7..c20b280e2b`. All correctness/contracts/integration draws passed; reviewer and writer share a model family. Reviewers traced source and retained passing logs/byte comparisons; the read-only review sandbox could not start fresh Go tests. Root's separate executed tests supply verification.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: f108a425bf4dfee0c616ae17812c4afd0032f9f8, c20b280e2b4e63759f34c1abfb07a4487d93e79a
- Tests: Existing TestWASMArchitecture before repair: exit 1 on both platform source sets (boundary-red.log), Both-platform architecture, complete Temporal policy and caller tests: exit 0 (boundary-focused.log, boundary-cli-full.log, boundary-measure.log), Affected staged lint: exit 0, zero issues (boundary-lint-staged.log), make generate validate-protocol: exit 0, no generated diff (boundary-generate.log), Actual schema complete policy configurations and full installation manifests before/after: cmp exit 0, Root focused command/installation/CLI/measure/temporalinput tests: exit 0, make -C tools/gomad validate: exit 0 (validate-published-helper.log), Fresh current-helper install/discover/explore/replay: all exit 0; exact TestFrontendSystemInfo PASS and choice replay, Retained replay after removing source checkout: exit 0, exact choice replay
- PRs: