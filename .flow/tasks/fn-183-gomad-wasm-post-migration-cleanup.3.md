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

## Done summary
Verified final Rust/reference/standalone/stock/native-source ownership, recovery parity, generated drift controls and all five explicitly retained lifecycle/callback/storage-evidence regressions. Current broad affected packages pass; unchanged lower-level Rust/WASI/model/generator/runtime inputs retain their original full-qualification identities. Final docs distinguish historical results, current candidate and deferred native execution. Independent integrated source review is SHIP at 18882a5f2a. The separately isolated fn-180 snapshot repair and one-box R6 failure are not relabeled as qualified.

Candidate, exact commands, numeric exits, manifests, identities and review are consolidated in `.flow/artifacts/fn180-current-batch-2026-10-16.md`. Spec completion review follows before spec closure; no PR, push, CI or native qualification is claimed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 406e6bdf87, 86f25ef560, 18882a5f2a
- Tests: GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact GOMAD_WASM_CONFORMANCE_HELPER=/tmp/gomad-fn180-conformance-helper-compact GOMAD_STOCK_GO=/Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/878c9b2935e2c9e6/go/bin/go GOFLAGS=-p=2 go test -tags test_dep ./internal/backends/wasm/backend ./internal/backends/wasm/cluster ./internal/backends/wasm/command ./internal/backends/wasm/installation -count=1, GOMAD_WASM_TEST_HELPER=/tmp/gomad-fn180-release-helper-compact go test -p=2 -tags test_dep -v -count=1 -timeout=15m -run '^TestInstalledOperatorTypedScenario(KillResume|DetachedReplay)$' ./internal/backends/wasm/command, go test -tags test_dep ./cmd/tools/lintcode -count=1, make lint-code-fast GOLANGCI_LINT_BASE_REV=HEAD GOLANGCI_LINT=/tmp/gomad-host-tools/golangci-lint ERRORTYPE=/tmp/gomad-host-tools/errortype GOLANGCI_LINT_FIX=false
- PRs: