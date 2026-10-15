---
satisfies: [R4, R5]
---
# fn-183-gomad-wasm-post-migration-cleanup.3 Verify cleanup parity and refresh execution-owner documentation

## Description
Validate the integrated cleanup and document final owners/identity changes for R4/R5.

**Size:** M
**Files:** tools/gomad/internal/backends/wasm/wasi/*test.go, cluster/provider_test.go, README.md, QUALIFICATION.md, tools/gomad/ARCHITECTURE.md, SPEC.md
**Touches:** [tools/gomad/internal/backends/wasm/wasi/*test.go, tools/gomad/internal/backends/wasm/cluster/*test.go, tools/gomad/internal/backends/wasm/README.md, tools/gomad/internal/backends/wasm/QUALIFICATION.md, tools/gomad/ARCHITECTURE.md, tools/gomad/SPEC.md]

### Approach
- Run namespace/volume/network/World differential and generated-source drift guards, plus actual recovery/restart and retained replay against the integrated candidate. Reuse current reference fixtures rather than adding another oracle implementation.
- Explicitly retain TestStorageEvidenceLargeEnumerationOwnerParity, TestStorageEvidenceEscapedMetadataOwnerParity, TestStorageEvidenceStreamAdmissionKeepsIndependentBounds, TestProviderIndependentGuestsLifecycleReplay and TestRetainedLifecycleRejectsHostCallbacksBeforeReplay.
- Confirm same-schema evidence equivalence and intended helper/source identity rotations separately; retain historical results under historical identities.
- Update ownership/capability documentation and the retained report with removed responsibilities, exact gates and current candidate. Native execution remains deferred; portable native-source checks remain required.
- Re-anchor regressions to final fn-176 full-context admission (operations/digest, dependency closure, chunked transport, bounds, valid context versus actual pending mismatch) and final fn-178 client (delayed/never-started nodes, holds/Finish, restarted evidence), rather than stale intermediate fixtures.
- Exercise detached current-format replay plus interrupted campaign recovery/resume with source/compiler/mounts/preparation cache unavailable. Keep Go oracles/generated projections/drift guards independent, stock/default/native source controls intact and every shipped capability represented after removals.
- Include authenticated malformed-image controls and valid partial-persistence orphan/durable-over-capacity cases, reserved inode/rebuild overflow, independent image/state/history/context limits and nil/default-byte controls. Document expected implementation-identity rotation separately from unchanged semantic evidence.

### Investigation targets
**Required:**
- tools/gomad/internal/backends/wasm/wasi/volume_test.go
- tools/gomad/internal/backends/wasm/cluster/provider_test.go
- tools/gomad/internal/backends/wasm/cluster/retained_admission_test.go
- tools/gomad/internal/backends/wasm/README.md
- tools/gomad/internal/backends/wasm/QUALIFICATION.md
- tools/gomad/ARCHITECTURE.md

### Quick commands
`make -C tools/gomad test-wasm GOMAD3_STOCK_GO=/absolute/pinned/go1.27.1`; run `make -C tools/gomad validate` when identity/generated/protocol inputs changed, plus affected portable/source checks.

## Acceptance
- [ ] All retained differential models and drift guards run with no lost reference coverage
- [ ] Actual lifecycle/recovery/retained replay gates preserve outcomes, evidence semantics and failure classes
- [ ] Independent storage-evidence bounds and callback refusal regressions remain covered
- [ ] Final-owner docs and candidate-bound results are current; historical identities and native deferrals remain explicit
- [ ] Source-less replay/recovery/resume, final context validation and delayed-start/restart lifecycle controls preserve all shipped capabilities
- [ ] Valid orphan/durable capacity behavior and rejected aliases/counters/paths remain distinct; no implementation-identity rotation is misreported as semantic parity loss or historical qualification

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
