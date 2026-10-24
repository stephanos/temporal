---
satisfies: [R12, R13, R14]
---
# fn-232-gomad-rewrite-the-host-side-in-rust.13 Qualification cycle on the Rust binary and Go host retirement

**Touches:** tools/gomad/cmd/gomad, tools/gomad/internal/runner, tools/gomad/internal/evidence, tools/gomad/internal/cli, tools/gomad/Makefile, .github/workflows/gomad.yml, tools/gomad/internal/archcheck/**, tools/gomad/ARCHITECTURE.md, tools/gomad/CLI.md, MILESTONES.md

## Description
Current implementation and verification map:
[Rust qualification design](../artifacts/fn232-rust-qualification-design-2026-10-22.md).
It maps every transferred live root and the remaining release-executable gates;
the map records no qualification pass and does not replace those gates.

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

### Go versus Rust code-size closeout

Owner request 2026-10-22: after the rewrite and Go host deletion, compare the
Go and Rust implementations by size. This task owns the comparison alongside
retirement; no separate deletion task is needed. Retain reproducible commands,
counter/version, exact baseline and final revisions, and explicit path lists.
Report file counts, source bytes and code lines, separating production code,
tests and generated code. Compare the removed Go host with its Rust replacement;
show the preexisting linked Rust helper and retained Go overlay/models/generator/
corpora separately so shared code is not counted as a new port. Include supporting
languages and the net repository change after deletion. Record measurements in
this task's existing closeout ledger.

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

### Darwin-only stale identity baseline (resolved 2026-10-23)

The darwin/arm64-only `gomad-choice-trace/v3` baselines (`tools/gomad/internal/runner/testdata/diagnostic-identity-choices{,-native,-portable}.json`) were removed with the Go runner and `TestDiagnosticsOffPreservesExistingIdentities` in b64f3dcb89; they were never regenerated or faked. Branch `fn232-retirement-v2` removes the orphaned sibling `diagnostic-identity-plain{,-native,-portable}.json` baselines of the same retired test. No Go runner test survives, so no darwin regeneration is owed.

### Closeout ledger: Go versus Rust code size (pre-freeze row, 2026-10-23)

Counter `gomad-source-size/v1` (sha256 e9e4a58c…), baseline 1d7b37f00a (frozen Go baseline), candidate tree 2185b5d140 (`fn232-retirement-v2`). Full table, commands and path-group rules: `.flow/artifacts/fn232-retirement/source-size-2026-10-23.md`. Code = physical lexical code lines.

| Group | Baseline files / bytes / code | Candidate files / bytes / code |
| --- | --- | --- |
| Go host retired by fn-232 (prod 51,329 + tests 53,341 code) | 432 / 4,755,828 / 104,670 | 6 / 8,243 / 336 (cmd/gomad testdata fixtures) |
| Rust host outside the engine (`rust/gomad`, `rust/driver`, workload tools): Rust prod 84,230 + tests 22,701, Python 19,715 + 11,588, other | 0 | 415 / 6,372,871 / 138,368 |
| Linked Rust engine (preexisting helper relocated and extended) | 83 / 1,710,919 / 43,336 | 152 / 2,647,488 / 67,221 |
| Go WASI reference (moved to internal/model/wasi) | 85 / 1,286,015 / 30,346 | 74 / 858,916 / 21,209 |
| Retained Go models/oracles/toolchain/support | 328 / 2,754,813 / 68,628 | 305 / 2,246,307 / 56,963 |
| Go native backend (removed by fn-223, not this port) | 558 / 4,289,404 / 101,564 | 0 |
| Whole repository (all tracked blobs) | 9,938 files / 238,487,870 bytes | 9,827 files / 271,660,908 bytes |

Headline: the retired Go host (104,670 code lines) is replaced by 106,931 Rust lines (mostly the `gomad` crate) plus 31,303 Python driver/tool lines; the linked engine grew by 23,885 lines over the preexisting helper. Final row: rerun on the frozen candidate.

## Acceptance
The admitted/shipped release binary excludes fault and deterministic-fixture features and rejects their fixture-only controls; retain build-feature evidence. Every verb uses task .4's declared golden view with independent canonical record/hash and raw ledger-integrity checks, never normalization of behavior or evidence fields. Qualification reports retained for every cell on the Rust binary; default build contains no Go host packages (archcheck test); CI source wiring validated and mandatory local commands pass, with no CI run dispatched; docs updated; every public verb's local parity/golden lane passes before harness job removal, and goldens remain as regressions; runtime/workload/retained-replay exits and final identities recorded under the unchanged one-hour qualification limit.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:


### Received from fn-232.8 (2026-10-22, merge f16f53e59d)
Candidate-v6 implementation identities and binary hashes are in `.flow/artifacts/fn232.8-identity-handoff-2026-10-22.txt`. Key values:
- helper `GOMAD_LINKED_IMPLEMENTATION_SHA256` 54773ee7…;
- fixture-role host 09017ee1…;
- runtime 4a82aea2…;
- Go WASI 5fddc30f….
The release-role host identity differs, because the host identity hashes the cargo features, so measure it from gomad-release. Two Go `wasm/backend` inspection tests fail on the pre-existing baseline; they are on this task's transferred live-gate list.

### Provisional transferred-live-corpus development, 2026-10-22

With .10 accepted and .11 implementing the declared preparation/retention interface, develop the exact 39 transferred live-root ports in an isolated worktree under the milestone provisional-interface batch policy. This starts executable corpus development only; .13 lifecycle/acceptance still waits its predecessors, and no retirement, full frozen cycle or qualification pass is authorized by this note. Preserve every original variant, fixture, assertion and bound from the existing mapping. Integrate these ports before the final identity freeze; test names or source mapping alone cannot earn a pass. Root owns scheduling of expensive guest gates and final review.

The same provisional-interface batch also develops the Rust serial/parallel qualification driver and concrete adapters in a separate worker worktree. Every original stage, bound and the 3,600-second end-to-end ceiling remain required; missing receiving endpoints fail closed as unexecuted. This supplies no frozen-cycle acceptance or permission to retire source.

### Worker A checkpoint, 2026-10-23 (resume of the parked final cycle)

Development and provisioning only. No final qualification pass is claimed.

- **Installation blocker resolved.** The external recipe compared the captured mode string `"0644"` with `int(mode)`, which is decimal. Production capture was correct. `provision-v3.py` parses the value as octal. All six final installations plus discovery now exit 0. Evidence: `.build-cache/fn232-final-installations-v1/provision-v3.{py,stdout}`. The v1 and v2 recipes and their outputs are retained.
- **Builds on 13ecf9f405.** `ordinary-host-test` built and was read independently (exit 0). The native release build, release stage and diagnostic measurement roles all exit 0 under `.build-cache/fn232-final-native-v1`. The remaining roles were cancelled because the candidate must be re-frozen (see below). Their receipts are in `/home/agent/evidence/fn232-final-roles-resume-controller-v2/`.
- **The candidate cannot pass the 146-obligation reader.** No `gomad.original-test-wasm-reach/v1` map exists for its 752 callables. Of those, 253 dynamic sites are unresolved, and 156 measurement-sweep callables have no receiving equivalence. In `qualification-map.json`, 41 of 72 roots still list gaps. The original `TestDeterminismExecutedDivergenceArtifacts/handoff` predicate also fails on the original Go baseline (see `.flow/artifacts/fn232-core-original-handoff-causal-check.json`). The 2026-10-19 scheduler estimate is 3,569 s of 3,600 s before any added Rust obligations.
- **fn-233 interception fixes** are on branch `fn233-shim-artifacts`:
  - `ca584bbe95`: a wrapped `go test` dropped `--artifacts`/`--seeds`.
  - `01429c4441`: a guest `go build` refused the shim's own `-overlay`, and the wrapper run identity was missing for caller launchers.
  - `6a901adddc` and `28fa92eecb`: harness changes and goldens.
- **fn-233 evidence on fixed binary 12ae1f83…** All 81 shim cases pass in two generations, and the comparison writes the golden (`/home/agent/evidence/fn233-final-shim-v3`). All six Make stages pass in two byte-identical generations, and the three positive artifacts replay exactly (`/home/agent/evidence/fn233-final-make-v2`). For R5, a run with no explicit installation passes, and the derived configuration equals the `install` configuration (`/home/agent/evidence/fn233-final-r5-v1`).
- **Open finding: identity depends on cache location.** Go test source inventories record `-compiled` files under the caller's GOCACHE path, so the runtime/configuration identity differs between GOCACHE locations. The second generation reused the first generation's cache path, with a fresh empty cache. The first cache is retained as `shim-cache-first.gen1`.

### Worker A cycle-closure batch, 2026-10-23 (branch fn232-cycle-closure)

Development evidence only; no frozen cycle.

- **Ruling 4:** `8cdaa8db61` records cache-resident compiled sources under a content-addressed `/$GOCACHE/<shard>/<digest>-d` key, so a guest identity no longer depends on the caller's cache path. Regression: `campaign::prepare::inventory::tests`.
- **Timing blockers found and fixed:**
  - `e154f663d3`: generated durable-cell campaigns no longer set the reserved TZ.
  - `63c8f3d5a3`: qualification replays use the installation compile cache; samples carry `cache_hit`, so the 4 GiB warm ceiling is exact on cache-hit windows. The accounting corpus gains two cases, and the golden was regenerated twice.
  - `ce0a868119`: cluster evidence is bounded per incarnation, not by the 64 MiB oracle evidence bound.
  - `ed04d3625d`: seed scenarios no longer journal cluster choice features. Go recorded them only for choice coverage.
- **Timing (item e)**, on the branch release binary under host load 8–10. Evidence: `.build-cache/fn232-timing-v{1..4}`.
  - Onebox success cell: 300 s, exit 0. It was 562 s and failed its budget before the fixes; the Go estimate was 247 s.
  - Multi-instance success cell: 290 s, exit 0. It failed twice before the fixes; the Go estimate was 729 s.
  - Not yet measured in Rust: temporal-inspection (Go estimate 1,296 s), ordinary (1,151 s) and storage (404 s). They need the composed primary-stage inputs.
- **Reach map (items b and c):** `tools/gomad/testdata/parity/test-wasm-reach-map.json`. Of its 901 entries, 504 are mapped (stage subgate, live root, retained Go test, final obligation or Rust test with correlation of at least 0.35), 54 manual, 79 weak (correlation below 0.35, marked in `basis`), 131 collapsed named subtests and 104 retired with reasons. 29 are explicit port gaps, which refuse completion. The 253 dynamic sites resolve at their root. The 41 qualification-map roots with gaps are dispositioned in this map, and 9 of them remain port gaps. `qualification-map.json` stays source correlation only.
- **Still open:**
  - Item d: compose bindings for live_corpus families, final_profiles, final_retained, stress, public_verbs, private_observations, terminal_controls_reference, primary and fn222.
  - Port the 29 gaps.
  - Run the dry run after the corpus fix lands.

### Worker B notes (2026-10-23): helper corpus fix and parked retirement-gap branch

- Corpus root cause, found by first-parent bisect of `helper_corpus` (b64f3dcb89 good, gomad bad): cae886df69 moved two linked-only decisions into the shared helper. Those were the `sock_accept`/`sock_shutdown` namespace refusal and the conversion of an exit with an unfinished runtime trace to `invalid`. The reference transport forwards sockets to its remote host and leaves trace collection to it. Fix 807520bc96 adds `State::helper_owned`, set only by linked standalone and cluster execution. No corpus expectation changed.
- The default-lane failures masked a second fault: 25 diagnostic, fuel-off and guest-profile sidecars still carried pre-6aca443134 `source_sha256` values for the eight storage_context cases. Commit 24f1403f80 rebinds the hash only and leaves every expected block unchanged.
- `make validate` (also run inside test-wasm) was red on two retained Go tests whose fixture paths b64f3dcb89 had relocated. Commit e4d3e68f42 fixes the paths only.
- Gate on branch `fn232-corpus-fix` (e4d3e68f42, rebased on 10763834f0, stock Go 1.27.1 first on PATH): `make -C tools/gomad test-harness test-rust-engine validate-wasm-determinism` exited 0 on 24f1403f80. On e4d3e68f42, `make validate` and `make test-wasm` both exited 0.
- Parked branch `fn232-retirement-v2`, rebased on the corpus fix. It adds the archcheck `retired-host-import` finding and its test, rewrites the ARCHITECTURE.md module map, makes small CLI.md/README edits, removes the orphaned `diagnostic-identity-plain*` baselines, resolves the darwin note and records the pre-freeze size row above.
  - Resolved (worker C): the pin in `GO_SCOPE_SOURCES` is `internal/archcheck/defaultset_test.go`, the source of the skipped `TestNoDefaultSourceNamesNestedTree`. The new `retired-host-import` test moved to `internal/archcheck/retired_test.go`, so the pinned file is byte-identical to gomad and the pin stays unchanged.
  - Verified: Go build/vet exits 0 and `make build` exits 0.
  - Already on gomad before this branch: CI/Make on cargo and the single workspace.
- Worktrees kept: `/home/agent/wt/fn232-corpus-fix` (merge source) and `/home/agent/wt/fn232-retirement-v2` (parked). Cargo target kept: `/home/agent/.cache/codex-build/fn232-corpus-target` (seeded from a copy of fn188-gates-target). Size evidence kept: `/home/agent/wt/evidence/fn232/source-size-2026-10-23/`.

### Acceptance amended by owner (2026-10-24, opportunistic merge)

This task closes on one green run of the full 96-stage cycle on current `gomad`. A formal freeze and the 13-role receipt rebuild are not required, and evidence binds to the commit that ran it. Remaining test-wasm reach-map `gap:`/`partial:` entries move to fn-232.14; the reach check reports them instead of refusing. Timing and performance results are recorded but not judged, per the performance-deferral decision. Everything else in this task's acceptance stands: the cycle's correctness, replay, determinism, memory and capacity results, the archcheck, the docs, and the identity and size ledger.

### Worker B notes (2026-10-23): item d, dry-run composition (branch fn232-composition)

- **Dry run (rule 2/3).** `rust/driver/dry_run.py` builds role executables from the working tree with a shared incremental target and records SHA plus compiled host/helper identities (`gomad.dry-run-build/v1`). Readers accept a record only under `dry_run: true` (or `GOMAD_DRIVER_DRY_RUN=1` for preparation scripts); a normal freeze refuses it, this describes the earlier formal-freeze route, superseded by the owner amendment above. compose/configure leave unbound groups in `dry_run_unexecuted`; their final readers still refuse.
- **Collapses.** fn-185 R4 `qualified.{workflow,activity}` bind only a request recipe; the module is the same-cycle `terminal-module` output (identical `./tests` module, three tags, no arguments) and `gomad-measure --qualified-request` regenerates the request in-cycle. `private_observations` is the per-stage switch (`true`); its receipts are cycle outputs.
- **New minimal producers.** `gomad-measure --stress-inputs` (stress spec inputs; the assembler installs with `stress-release`, configure binds `SOURCE_MANIFEST` at freeze), `diagnostic-helper-stage` (the diagnostic admission terminal-exit requires; only release builds could be staged), `--qualified-request REPO runtime` (runtime recipe template).
- **installed_cluster_failure.py** runs in the `operator` stage: evidence IDs `stage/operator/TestInstalledOperatorClusterFailureEndsPromptly` and `stage/operator/TestInstalledOperatorAsymmetricGuestFailureEndsPromptly`.
- **Bugs found and fixed:** stale `seed` in `live-inspection-fixture.py`; freeze-time stage preflights guarded a not-yet-written config (`guard` skips the digest only while freezing); stress reader expected role `stress` with `stress-release` features; live runtime/inspection `--runtime-identity` is the runtime implementation SHA, not the overlay identity.
- **Later on this branch:** worker C's prepared-metadata producer (groups_live.py) resolves the last live family; stress leaves the cycle (ruling 7 revised: 145 final IDs, 7 unbound sections, still 96 stages; launch it after the freeze with `stress-window.py`); fn222 runs beside temporal-inspection in one stage (dag.tsv 6 cores / 9 GiB); an overrun that otherwise passes exits 0 with `performance.status: deferred-perf`; `live_corpus.test_wasm_reach` names the static Go/Rust test stages.
- **Dry-run freeze proof:** head 4d6075fdc8 froze with no unexecuted group (`/home/agent/evidence/fn232-composition-v1/dryrun-v4/freeze/driver.json`, sha256 23c71ee9…). It seals 165,578 files and one `check_files` over them takes ~220 s, and `guard` repeats that per stage reader, which is wall time the dry run will show.
- Provisioning scripts: `/home/agent/evidence/fn232-composition-v1/{assemble.py,groups.py,groups_r4.py,rebuild.sh}`; run `rebuild.sh TAG` then `GOMAD_DRIVER_DRY_RUN=1 assemble.py --candidate CHECKOUT --tag TAG --r4 --out FRESH`.
- **Handover (2026-10-24, stop).** Branch `fn232-composition` is gate-able on current gomad. The reach reader reports gaps (`live.REACH_GAPS_BLOCK = False`; fn-232.14 flips it). Run the cycle on a clean checkout of the branch head:
  1. `/home/agent/evidence/fn232-composition-v1/rebuild.sh TAG` (roles, native, installations and inspection fixture from the checked-out head; REPO is fixed to `/home/agent/wt/fn232-composition` in the script).
  2. `GOMAD_DRIVER_DRY_RUN=1 /usr/bin/python3.14 /home/agent/evidence/fn232-composition-v1/assemble.py --candidate /home/agent/wt/fn232-composition --tag TAG --r4 --out FRESH` (writes `FRESH/freeze/driver.json`; about 1.5 h, mostly file hashing).
  3. `cd /home/agent/wt/fn232-composition && /usr/bin/python3.14 tools/gomad/rust/driver/driver.py parallel --config FRESH/freeze/driver.json`.
  4. After the cycle, stress: `/usr/bin/python3.14 /home/agent/evidence/fn232-composition-v1/stress-window.py --window FRESH/stress/window.json --frozen-config FRESH/freeze/driver.json --root FRESH_STRESS`.
- **Open:** never run end to end; `test_wasm_reach` was added after the v4 freeze proof; the 3 slim-refusal and 4 `stockgo` Go roots are not run by any static stage (reach reports them); one `check_files` over the 165k sealed files takes ~220 s per `guard`; `linked_cluster::...native_host_termination_preserves_watchdog_and_parent_cancellation_status` failed once under load (20 ms cancellation race).

### Worker A handover, 2026-10-24 (branch fn232-flake-class, head below)

Stopped on owner instruction. No cycle run was started.

- **Merged or gated:** fn232-gap-ports (28 of 29 reach gaps ported or retired with reasons; builder and volume `/original-make/` selections have distinct ordinary run IDs), fn232-lint-flake (strict clippy in all 13 rust-lint modes), fn232-shim-etxtbsy.
- **This branch:**
  - `eb773cd5ac`: fn-227.1's four inventory rows cite the Rust map-order tests (R1, R2).
  - `45e831a79e`: removes the allocation-probe wires and refuses retaining them inside the checkout.
  - `ebaed60313`: `crate::fork_order` orders child creation against lock release (retention leases, store/pin/children locks, wrapper guards); `make_executable` for test fixtures; exact assertions for the capacity, launcher-pipe and wasmhost pre-launch-termination tests (core_lifecycle reader accepts the never-created form).
  - `ffe9d0eea7`: removes the 93 derive-tests scratch files it accidentally committed, moves four test scratch roots to the system temp directory, and ignores `rust/*/target/`.
- **Loop evidence** (`/home/agent/wt/evidence/fn232-flake-class-loop/`, load 30–50):
  - gomad lib: 12/200 failed. None were the fixed classes. 7 were timeouts (`cli::run_guest::admission_tests::{same_driver_launches_have_distinct_campaigns_and_record_commitments, structured_all_pass_reports_both_authenticated_outcomes_then_summary}`, `evidence::store::publish::tests::{target_link_collision_releases_exclusive_guard_before_reuse_read, module_reader_serializes_cross_store_link_retirement_and_staging_cleanup}`, `measurement::cancellation_tests::cancellation_while_publication_lock_is_held_returns_before_lock_release`), which are deferred-perf under the quiet-host ruling. Runs 21, 24, 25, 112, 168 and 169 ended with no failure section; dmesg shows OOM kills only of other sessions' processes, so the cause is unknown.
  - wasmhost lib: 1/200 failed (`live_cleanup_reaps_a_noncooperative_store_at_the_original_five_second_grace`, timing).
  - allocation_receiving: invalid. Source files were edited during the loop, which trips the measurement source-closure tests.
- **Next steps:**
  1. Rerun the gomad lib loop on a committed, unedited tree, recording each run's exit status and signal, to classify the silent ends (OOM, backstop or crash).
  2. Rerun the 50-run allocation_receiving loop the same way.
  3. When B reports a resolving composition, run the full cycle with `driver.py parallel` on gomad head. Record every stage process's exit status and signal; treat a silent end as a real failure until shown otherwise; record timing-only failures as deferred-perf. Include the Rust sweep on a real guest and fn-233's launcher, shim and Make lanes.
  4. Map `TestInstalledOperatorAsymmetricGuestFailureEndsPromptly` once `installed_cluster_failure` is wired into a stage.
- **Retained:** worktree `/home/agent/wt/fn232-cycle-closure` (branches fn232-cycle-closure, fn232-gap-ports, fn232-lint-flake, fn232-shim-etxtbsy, fn232-flake-class); cargo target `/Users/stephan/Workspace/skunkworks/gomad/.build-cache/fn232-cycle-closure-target`; gate logs under `/home/agent/wt/evidence/fn232-*-gate/`. No background processes remain.


### Current composition handover (2026-10-24)

This supersedes the earlier stop/launch instructions above; their commands, failures and
SHAs remain historical provenance. fn-232.13 is still in_progress and no full 96-stage
cycle has run. The current reviewed composition is `4e4b8b51a5b65d76868712b588b2ed1067ee82cd`,
with `gomad` base `318427522b009f5a27eeeb3c7c17b211b0ce2c34`. Its 567-test driver suite
and full mandatory worker merge gate exited 0. Independent re-review returned SHIP with
no remaining findings. The conductor's full merge gate exited 0 on the same commit, and
`gomad` was fast-forwarded to that reviewed composition. Its first cold-cache attempt failed
on missing offline Go modules after cache cleanup; declared dependency prefetch resolved
that environment failure without changing the source. Numeric gate exit and raw log:
`/home/agent/wt/evidence/fn232-resume-composition/root-ready-gate.{exit.json,log}`.
Full evidence and retained launch risks:
`/home/agent/wt/evidence/fn232-resume-composition/{summary.md,evidence.json}`.

The fix retains the operator and artifact lifecycle producer/readers, passes sealed driver
configuration to installed/public-parity children, restores real private-capture source and
binary pins, validates each role's actual compiler/features/current implementation source,
and retains shared counterfactual context. Original build_head provenance is distinct from
current execution custody. Source-matching role reuse is permitted; a formal 13-role rebuild
is not required. Partial dry-run bindings execute their owned stages while final collection
still refuses missing groups. Named unexecuted slim roots report fn-232.14-owned reach gaps
through REACH_GAPS_BLOCK; reenabling completeness restores refusal.

After the conductor's gate and merge, assemble a fresh identity-bound configuration for the
actual current `gomad` candidate and run `tools/gomad/rust/driver/driver.py parallel --config
CURRENT_DRIVER_CONFIG`. Preserve all stage exits/signals and raw candidate/source/compiler/
helper/runtime/module bindings. Timing is measured and deferred; correctness, replay,
determinism, identities, memory and capacity remain strict. A green source gate is not the
cycle. Rerun the retained source-size counter on the final integrated candidate and append
its row without replacing the historical comparison. Stress, fn-233 rerun and fn-221 sweep
follow the cycle under MILESTONES.

Owner-authorized history cleanup removed oversized generated `.wire` blobs from all 11
local refs without pushing or changing the main/composition trees. Mapping and backup:
`/home/agent/wt/evidence/fn232-history-cleanup/applied.json`. Current remaining branch inputs:
sweep `911833fb481f8f51d3af6dd20699fc435032c78a`; reach `1f85fbfcf95604757a6742d54a934e13395c2bc4`.

Launch controller: `/home/agent/wt/evidence/fn232-resume-launch/launch.py --execute --tag resume-4e4-cycle1`. Its preflight validates 57 retained inputs; fresh role records are authenticated before assembly. The source must remain unchanged throughout preparation and execution. This handover records readiness, not a cycle pass.


### Remaining closure split (2026-10-24)

Append three narrowly scoped fix tasks; preserve fn-232.1 through .12 as done and retain all existing fn-232.13 acceptance and historical evidence. Task creation order is .15 static Go selector alignment (S; R11/R12), .16 counterfactual absence/configuration handling (M; R12/R14), and .17 shared broad-harness readiness (M; R12).

Execution wave one is .15 and .17. Task .16 depends on .17 because both change `broad_stages.py`. Task .13 depends on all three fixes and remains the actual green 96-stage cycle plus unchanged correctness, replay, determinism, identity, memory, capacity, documentation, size and retained-evidence obligations under the owner's existing performance deferral. Focused fix tests cannot substitute for that cycle.

Task .14 explicitly follows .13; encode that dependency while preserving its existing reach-closure scope. Until .14, absent counterfactual reach may be reported only as explicit unexecuted, uncredited reach with `REACH_GAPS_BLOCK=false`; declared malformed or failed bindings remain strict refusals. These tasks neither close the reach gaps nor claim a cycle pass.


### Closure fixes integrated (2026-10-24)

Tasks .15, .16 and .17 are implemented, independently reviewed SHIP, integrated into gomad and verified done in Flow. Their focused regressions and full mandatory gates pass; the final combined gate on b15d9b5e0eb0c044612c8fb28db9a701aeea25de passed all 583 driver tests. Evidence and review receipts: /home/agent/wt/evidence/fn232-split-closure/.

The next actual cycle uses launch-next.py in that evidence directory, with an explicit current clean HEAD and fresh output tag. It authenticates reusable semantic inputs without changing their original build provenance and genuinely regenerates the R4 bridge-helper receipt for the executed commit. C4 remains a failed validate attempt with zero passing stage receipts. This task remains open until the actual green 96-stage cycle and all existing receiving/size/evidence reconciliation obligations are satisfied.
