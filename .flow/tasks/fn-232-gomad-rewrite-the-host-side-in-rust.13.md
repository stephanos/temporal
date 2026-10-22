---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.13 Qualification cycle on the Rust binary and Go host retirement

**Touches:** tools/gomad/cmd/gomad, tools/gomad/internal/runner, tools/gomad/internal/evidence, tools/gomad/internal/cli, tools/gomad/Makefile, .github/workflows/gomad.yml, tools/gomad/internal/archcheck/**, tools/gomad/ARCHITECTURE.md, tools/gomad/CLI.md, MILESTONES.md

## Description
Frozen Go baseline owner/input index: /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a/README.md. Preserve its independently copied launcher/helper, source archive and receipts/tool pins for fresh equivalence; it supplies no qualification claim.
fn-228 R7 receiving acceptance: retain the exact deletion-free replay ledger at .flow/artifacts/fn228-replay-identity-ledger-2026-10-20.md and use the independent receiver index /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/FINAL_CUSTODY.md. The verified root inputs.sha256 digest is 8ff5fd82a1b0442c44004c73b90bd0f82806bbb40424f9732a64fc991519afd8; combined/inputs.sha256 is 2d853237dfa74d71e22ff4ad81c027c1a8efdcef6e6e5ec460eb01cc0443fe1a. Receiver-owned handover/, r7-fresh/, source/ and bin/ preserve the exact deletion-free pair c36dddcb3e29498d36ed207d2a5b9f4013b32cb2 → 12cdc816727a232eb7d0b1ff26cc3cfe951527ea, helper b25439140f0c850a82f99471c9fd48ac864569a75e407a76850873f6ece055a1 → ab4059583f7d3ca265cdbce2de064cd2e54b744b9943cf38e2addf45e1825193, and WASI ModelSHA256 73c1a4f3f64dedd4c18dad47849f6dcc8bc04d08e27067dedd096acde05e2e60 → 18a88761bf147ca7436ffe0782e6a4c6f9ddf816e782bd42a90ea41a492987b7 from the four measured WASI input changes. Full old/new source manifests and exact per-artifact BuildKey/Choice/tape/failure values remain unmodified in that custody. The earlier fresh own-artifact comparison passed with zero branching; separate 407-terminal/full-divergence comparisons and 59-fixture/nonzero prefix-oracle checks do not replace final qualification. These are Go-lineage historical inputs, not artifacts to migrate or cross-admit under Rust.

Native-deletion receiver custody: /home/agent/wt/evidence/fn232/fn223-deletion-inputs-2026-10-20/README.md, authenticated inputs.sha256 digest 0bd49511d13e37e48afa0072cb908ea321e2fe25eba55f199429c1420a6d7692. This independent copy preserves the deletion-only pair 18e5e7cac713b899b794cb23b9e4db863d913b8b → f76f6263b13eede69276224e8b89ac8e33e03d21 and ledger checkpoint 40351165d587ba1ec9f4fdee081d9ce61b6a76bb. Its shared-cluster/transitive rotation causes remain separate from fn-228's replay-source/model/helper changes; copying evidence supplies no qualification pass.

Historical Go attempt receiver index: /home/agent/wt/evidence/fn232/fn219-go-attempts-2026-10-20/README.md preserves failed/interrupted/unrun attempts with commands, receipts and logs; it is not qualification success evidence.
Run one full qualification cycle with the Rust binary alone under the one-hour driver and retain the reports; then remove the Go host-side packages and binary from the default build, keep the Go overlay, models, generator and corpora, update archcheck to forbid default-set imports of removed packages, switch Make and CI to the Rust binary, and rewrite ARCHITECTURE.md's module map and CLI.md to the Rust host. Close out with the identity ledger. Before fn-228 closes, record this task's acceptance of its pending R7 final runtime/workload/replay qualification, and inventory the frozen pre-fn-228 Go executable/helper/source/preparation/configuration identities under fn-232 ownership. This task also receives pending fn-219 R5, fn-227 R4, fn-188 identity-triggered qualification gates and fn-223 R3/R4's full changed/integrated-candidate runtime/workload/retained-replay qualification and complete behavioral test-wasm reach. Before fn-223 source acceptance closes, record receiving acceptance, a complete gate/subgate/fixture mapping from its QUALIFICATION.md/test-wasm obligations and custody of its deletion-base/candidate source inventories and identity ledger. Preserve deletion-only cluster-glue/transitive causes separately from fn-228/Rust-host identity changes. These transferred gates remain pending until the final frozen Rust cycle supplies actual bound evidence; this creates no additional full Go qualification cycle and waives no behavioral coverage. Preserve their failed/interrupted/unrun Go evidence as history; no transferred gate is reported passed before this frozen Rust candidate supplies its actual reports and exits.

Closed-fn-219 host tools now belong to this open owner at `/home/agent/wt/evidence/fn232/fn219-host-tools-2026-10-20/README.md`. Its independent copies and receiver-path `host-tools.sha256` verify. Future qualification must use that receiver as `RETAINED_FREEZE` with fresh candidate bindings; original configs/manifests remain historical and are not silently repointed. This transfer is not a qualification pass.

Completed combined receipts and probes are independently retained in that receiver's combined/ append. Its historical measured source is 0561e32514a2a2cf64d1a2840315e8f030ae9799: helper ab4059583f7d3ca265cdbce2de064cd2e54b744b9943cf38e2addf45e1825193, WASI model 18a88761bf147ca7436ffe0782e6a4c6f9ddf816e782bd42a90ea41a492987b7, shared cluster 891c336f2b0a943b4a36e7f4568b744ac41857d7a7190d298b2fb76d3f0ca003 and composed WASM cluster a25134c081e544f944ed54c7ae65b5de8e4e3d36570842f87720e28e6885c619, with eight controlled preparation/binding comparisons. Initial and final non-Flow source inventory digests agree at b846af88cd68a20e32e0e2e05849d329547c55f7a479a7b1b17bf2e0fb35bbe9. Those source-admission receipts are historical, not proof of current review SHIP or a final Rust qualification pass; the separate narrow-fix append and any later candidate need their own evidence. FINAL_CUSTODY.md supersedes only the unchanged initial README/integration-pending snapshot's missing-combined-receipts statement. The earlier zero-choice R7 campaigns are not combined-candidate campaigns. Preserve both contexts, exact executable/source identities, manifests and raw logs; no donor path must remain necessary for future acceptance.

Verify-only-fix receiver custody: /home/agent/wt/evidence/fn232/fn228-replay-inputs-2026-10-20/verify-only-fix/README.md, verified inputs.sha256 digest c4d8d75b1b31e9b446547726b8993d5e09ea36a22733ce088c29c07a91c406c3. This completed independent append preserves the narrow workflow-routing and verify-only replay fixes, their original RED/GREEN/whole-Runner/read-only-lint receipts and exact source patches; it discharges the earlier pending narrow-fix custody note without relabeling the deletion-free R7 or combined 0561 observations. Final source 07ebfdb4eec9e6cfde5535fdda84ad25b3bbb220 has non-Flow inventory f2dacdd826eee853450e0b60c3477bd5d4b9e3fd94756ade6137f9a06b7602ba: 8,263 paths and 226,349,334 bytes. Exactly four non-Flow paths change from 0561, with no additions/removals: .github/workflows/gomad.yml, tools/gomad/internal/archcheck/integration_test.go and tools/gomad/internal/runner/replay_operation{,_test}.go. All 71 helper source/pin entries and WASI/cluster/model/supporting inputs remain byte-identical, so the existing measured helper/model bindings remain valid for those unchanged inputs, not for the changed full-source inventory. No new helper/model build, fixed CLI/R7 campaign or full qualification is claimed. Final Rust runtime, mandatory workload and retained-replay qualification remains PENDING under R12/.13 and the unchanged 3,600-second limit.

This task explicitly accepts the outstanding fn-228 R7 final frozen Rust runtime, mandatory workload and retained-replay gates under R12 and the unchanged 3,600-second limit. All remain PENDING until actual final commands, numeric exits and complete candidate/source/helper/runtime/module/model/compiler/engine/host bindings are retained. The extracted library, Rust host and later fixes require separately measured actual identities; neither the deletion-free nor historical combined binding is asserted current by assumption.

Pending fn-185.1/R4 receiving obligation: this task's frozen Rust qualification window must supply the actual runtime-cost-equivalent release-behavior check and qualified-workload preservation comparisons for choice, WASI, network, stdout and stderr, with original assertions, inputs and bounds intact. Record candidate/source/helper/model/module/compiler/runtime identities, exact comparison commands, raw results and numeric exits under the unchanged 3,600-second limit. Preserve the older f172 raw early-proof observations as historical evidence, not proof for the current or final Rust candidate; never migrate artifacts, normalize identity-bound digests or relabel old results. This is a pending receiving/evidence dependency, not a transfer that discharges fn-185.1/R4: R4 remains OPEN in fn-185.1 until the actual receiving proof exists and is reconciled there, and this task cannot report that obligation passed without it. No new Go-only profiling run or second Go full-qualification cycle is authorized; no existing release-preservation, dependency/lockfile or qualified-workload requirement is waived.

### Single Cargo workspace at Go host retirement

Owner decision 2026-10-21: when the Go host retires, move `gomad-wasmhost` (`tools/gomad/internal/backends/wasm/wasmhost`) into the `tools/gomad/rust/` workspace with one `Cargo.lock` and one shared target. Its standalone `gomad-wasmhost` binary exists only for the Go host; remove it or keep it as a fixture-only target that the release exclusion check covers. Reconcile the two lockfiles without changing resolved dependency versions unless separately reviewed, and record the resulting helper/implementation identity rotation in the closeout ledger. Make, CI and docs build through the single workspace.

### fn-235 R5 rotated inspection-contract receiver

Owner decision 2026-10-21 ("yes" to the conductor's proposal): fn-235.2 completes with its measurement, attribution, flat-cost, fresh-replay and corpus evidence, and this task receives R5's remaining obligation: qualify the rotated inspection contract on the frozen Rust candidate. The declared rotations are `gomad.inspection-state-sha256/v2`, the source-bound helper implementation identity and installation `ModelSHA256` `d6ca1cda…` → `e5541190…` (`.flow/artifacts/fn235-commitment-cost-2026-10-21.md`, sections "Declared rotations" and "Identity and replay"). The final cycle must bind the actual current helper/model identities, run the long three-node ringpop scenario (`testdata/ringpop`, 120 s) to success with fresh retained exact replay, and keep the canonical World, network and choice records unchanged apart from the declared rotation fields. Record commands, numeric exits and identities in the closeout ledger and reconcile them into the fn-235 artifact. The obligation stays PENDING until those receipts exist; it is not waived.

### fn-223 full-module live-behavior receiver

In addition to the full qualification/test-wasm transfer, accept the following exact 39 actual live-guest roots from the deletion batch's former unselected full-module test command. Use the receiver-owned /home/agent/wt/evidence/fn232/fn223-deletion-inputs-2026-10-20/producer-evidence/live-gate-names.txt and the disjoint nine-source list, selectors and original failed command/log/classification alongside it. These independent copies remain under fn-232 custody after donor cleanup; producer absolute paths inside historical receipts are provenance, not a dependency on a closed donor directory. The nine source/admission/preparation roots remain required Go source checks in fn-223.1 and are not transferred. Record a one-to-one behavioral/subtest/fixture/bounds mapping to the final Rust lanes, even if tests are renamed or regrouped; no blanket unsupported/skip disposition, weakened assertion or budget increase. Keep the original Go failure and invalidated probe-race observations as history; this receiver remains pending, not a Go execution or qualification pass.

- `TestInspectionRetainedArtifactWithoutSourceOrCompiler`
- `TestInspectionRetainedClusterNetworkStoppedPrefix`
- `TestInspectionRetainedClusterTargetsRestartedIncarnation`
- `TestIntegratedIntentionalFailureObservedReplay`
- `TestReplayRejectsChangedArtifactIdentityBeforeGuest`
- `TestObservedCampaignResumeRetainsBackendAndOrdinals`
- `TestIntegratedCooperativeFailureArtifactDetachedReplay`
- `TestWASMChoiceCampaignCanonicalRoundsAndDetachedReplay`
- `TestWASMChoiceCampaignInterruptedRoundRecoveryNoDuplicates`
- `TestWASMChoiceCampaignFuelCapacityIsNotApplicationFailure`
- `TestWASMCooperativeCampaignWithExplicitEnvironment`
- `TestWASMMinimizeEssentialPrefixDetachedReplayRepeatAndResume`
- `TestScenarioPreservesChoiceCapacityBeforeDeclaredMarker`
- `TestScenarioPreservesUnsupportedBeforeDeclaredMarker`
- `TestScenarioPreservesChoiceCapacityBeforeRetention`
- `TestSoftAssertionActualPassingGuestArtifactReplay`
- `TestSoftAssertionChoiceCampaignReportsCommittedRounds`
- `TestProviderDetachedOracleCanarySourceLessReplay`
- `TestProviderClusterStartsOnlyRequestedIncarnations`
- `TestProviderMandatoryCleanupReapsWithoutLifecycleActions`
- `TestProviderStopReapsGuestAndPeerContinues`
- `TestProviderGroupedAcknowledgmentCrashSourceLessReplay`
- `TestProviderActualCrashDurableRestart`
- `TestProviderIndependentGuestsLifecycleReplay`
- `TestProviderCompressedIndependentGuestsLifecycleReplay`
- `TestProviderUnboundRevocationOfSplitStorageNodeRestarts`
- `TestSoftAssertionActualClusterGuestAndRetainedReplay`
- `TestSoftAssertionActualClusterOutputCapacityRemainsInfrastructureFailure`
- `TestCooperativeInspectionEveryDecision`
- `TestCooperativeGoGuestForcesExactReplay`
- `TestCooperativeRuntimeProbes`
- `TestBrokerCrashRecoveryFreshGuest`
- `TestClusterOutputBoundaryParity`
- `TestClusterBrokerDurableRestartParity`
- `TestClusterShutdownLiveNodesEvidenceOrder`
- `TestBrokerIndependentGuestStreamsReplay`
- `TestHelperModelPreservesGoOwnerEvidence`
- `TestNamespaceOwnerFilesystemParity`
- `TestBrokerCrashRetainsInterruptedRuntimePrefix`

### Darwin-only stale identity baseline (recorded 2026-10-21, worker D)

`tools/gomad/internal/runner/testdata/diagnostic-identity-choices-{native,portable}.json` (and the shared `diagnostic-identity-choices.json`) still name `gomad-choice-trace/v3`. The fn-230 R12 v4 repair did not refresh them. Their identity assertion is `TestDiagnosticsOffPreservesExistingIdentities`, which skips everywhere except darwin/arm64, so no Linux host can regenerate or verify them. `TestExternalBackendTransportPreservesPortableNativeRecordSemantics` reads the shared file and passes because it does not compare the profile name. The baseline is recorded here as a darwin/arm64-only stale baseline, not regenerated or faked. Before retirement, regenerate it on darwin/arm64 if the Go runner tests survive. Otherwise remove it together with the Go host packages.

## Acceptance
The admitted/shipped release binary excludes fault and deterministic-fixture features and rejects their fixture-only controls; retain build-feature evidence. Every verb uses task .4's declared golden view with independent canonical record/hash and raw ledger-integrity checks, never normalization of behavior or evidence fields. Qualification reports retained for every cell on the Rust binary; default build contains no Go host packages (archcheck test); CI source wiring validated and mandatory local commands pass, with no CI run dispatched; docs updated; every public verb's local parity/golden lane passes before harness job removal, and goldens remain as regressions; runtime/workload/retained-replay exits and final identities recorded under the unchanged one-hour qualification limit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
