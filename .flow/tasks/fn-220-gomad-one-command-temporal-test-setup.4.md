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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
