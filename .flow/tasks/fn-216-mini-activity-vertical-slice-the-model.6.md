---
satisfies: [R1, R2]
---
# fn-216-mini-activity-vertical-slice-the-model.6 Temporal envelopes, realization and guest over the real API closure

## Description
Complete R1's realization and R2's system binding using the independent Model and
reusable generic core. This checkpoint proves the guest before .3 starts the SDK.

**Size:** M
**Files:** Temporal envelopes/API generation, realization, guest and extension Go module.
**Touches:** [model/temporal/**, model/Cargo.toml, model/Cargo.lock, model/scripts/**, Makefile]

### Approach
- Add exactly the three envelopes in the parent contract under the Temporal proto tree. Generate the upstream Temporal API types from the existing descriptor-generation target, with complete imports and well-known types wired to the core's shared owner. Do not copy Testpilot schemas or download a source tree.
- Make clean builds generate the descriptor before compiling Temporal crates. Compile the real generated closure natively and for wasm32-wasip1; cached or stubbed generation is insufficient evidence. Do not silently introduce runtime libraries.
- Generate the extension Go package under a separate `model/temporal/hosts/go/` module named `umpire.local/temporal`, with generated `umpire/temporal/v1/temporal.pb.go`. Keep its generator/freshness checks outside the generic host output. This preserves the existing root ownership gate's nested-module rule.
- Implement temporal.rs with strict outer/inner Any URL and RPC validation, the actual Info.Status/Info.RunState projection and the declared binary/plain Complete/Fail policy. Use the independently authored Model's transitions and callbacks; do not infer missing delivery or make new edges to explain observations.
- Validate policy and the selected enabled response before committing callback evidence for Poll. Return only Model-decided fields in partial Respond* requests; document which upstream callback/request fields the SDK owns or cannot expose.
- Native and WASM scenarios must pin Scheduled/Started/terminal repeats, terminal-before-delivery, stale Scheduled after delivery, opposite terminal after closure, unknown/malformed projection, wrong/missing URLs, malformed policy/cardinality, callback at a closed state, freeze and Reset. Refused callbacks leave Scheduled unchanged; returning Answer leaves Started.
- Link the Temporal guest against the generic library through its public hook and compile the existing stock guest too. Exactly one instance belongs to one run; no hidden global sharing.

### Investigation targets
**Required:**
- `Makefile:346` - existing descriptor generation prerequisite.
- `cmd/tools/getproto/main.go:315` - descriptor closure validation.
- `model/umpire/wasm/build.rs` - admitted prost generation pattern after .2.
- `model/scripts/proto-go.sh` - pinned generator/freshness pattern.
- `model/archived/temporal/features/activity_mini/realization/MiniActivityRun.scala.disabled:40` - delivery evidence precedent, read-only.
- `tools/umpire/ir/ownership_test.go:190` - nested Go module ownership rule.

### Quick commands
Run the actual new Temporal crate/native tests and wasm32-wasip1 guest build through mise. Run CGO_ENABLED=0 mise exec -- go test ./... in the extension module and the pinned generator freshness check. Run make model-check and make model-time from the root.

## Acceptance
- [ ] Exactly three generated envelopes and their real upstream API closure compile natively and as WASM from clean descriptor generation; generated Go freshness passes in the Temporal module.
- [ ] The realization pins parent-contract projections, input policy, callback delivery and partial replies using original Model behavior, including positive zero-step matches and every listed negative case.
- [ ] Refusal/Unexplained/reset/allocation behavior is consistent with the reusable core, with no invented Poll or partial callback mutation.
- [ ] Both Temporal and stock guests build and their focused native/WASM/Go checks and full model gate pass with recorded timing; no Temporal code enters Umpire.


## Done summary
The mini activity guest now judges real upstream Describe messages and answers partial Poll callbacks through the reusable core. Exactly three generated envelopes belong to `umpire.temporal.v1`; the separate Go module is `umpire.local/temporal`. `features/activity_mini/temporal.rs` owns the strict registry, status/run-state projection, binary/plain Complete/Fail policy and partial replies. Answer validates policy and response enablement before committing Poll, leaves Started, and relies on later terminal evidence to close the run.

The API crate compiles the complete existing descriptor closure with shared `umpire_wasm::google::protobuf` types. Direct Cargo builds invoke the existing Make descriptor owner and track its inputs. Ordered generated maps avoid the forbidden WASI randomness import. The clean proof moved the prior ignored descriptor into preserved backups, removed API build artifacts, regenerated and compiled natively, and then compiled the closure and linked guest for wasm32-wasip1. `fn216-6-clean-generation.log` and `fn216-6-clean-generation-vv.log` retain cold compilation and raw Make-generator output. The decoded closure currently has 57 files; no cardinality invariant was added. `fn216-6-stale-input.log` proves a go.mod mtime change triggers Cargo and Make regeneration. Original bytes and timestamp were restored.

The independent Model and canonical goldens are unchanged. Its SHA256 remains `e9df73c0a44e0b4cd1ab5fcfdca7228b3653b9f9a27e531b76886c4beac87ad6`. The named guest adapter delegates transitions, internal classification and the registered postcondition to the original world. Six native integration tests cover explicit zero-step repeats, delivery and terminal replay, impossible observations, first-Unexplained freeze, strict malformed/unsupported inputs, rejected callback rollback, rejected publication and reset/isolation. The adapter report and ordered Rows compare directly with the existing canonical goldens. Eight Go tests exercise the actual WASM guest through the production host and test-only raw ABI allocation controls, including five response-exhaustion rollback cases. Missing guests fail. Pinned extension Go freshness and CGO-disabled module tests passed in the full gate. Stock/control guests, old protocol controls and generic host parity remain in that gate.

The test-first run `fn216-6-native-red.log` failed all five initial behavior controls against deliberate Unsupported stubs. `fn216-6-adapter-mutation-red.log` failed exact ordered Rows after a temporary catalog reorder; that mutation was reverted and all six native controls passed. `fn216-6-go-red.log` records the required missing-guest failure. The first built guest failed the production host's existing random_get restriction; the historically named `fn216-6-go-green.log` retains that failed attempt. The map-generation fix preserved the host restriction. Final full gate logs provide the green execution evidence. Additional raw proof detail is in `fn216-6-proof-notes.md`.

baseline: green. Pre-edit `make model-check` and `make model-time` passed. Baseline timing was 4.025s full and 1.741s inner. An initial inner shell observed the wrong directory and failed to find the target; that observation was inconclusive and corrected before edits. Baseline logs are `fn216-6-baseline-model-check.log` and `fn216-6-baseline-model-time.log`.

Post-review `make model-check` passed in 18.277s and wrote `.flow/tmp/green-receipts/fe00d08b-unittest.json`. The isolated warm default `make model-time` passed at 9.787s full and 2.243s checker edit/test. After its checker touch, the isolated mini-activity timing run passed at 34.313s full and 0.600s original Model edit/test. The earlier post-clean/source-relocation run passed at 39.589s full and 1.536s realization edit/test. Both 30s soft-target overruns remain recorded. The checker-invalidated run spent 4.290s in Clippy, 4.383s in docs, 7.518s in tests and 9.957s in WASM builds. `fn216-6-checker-dependencies.log` confirms the API depends on the checker through the shared-WKT owner, which is consistent with that rebuild cost. No gate or ownership change was made to improve the measurement. Timing logs are `fn216-6-{pre-review,default}-model-time.log` and `fn216-6-model-edit-time.log`; the final full verification log is `fn216-6-post-review-model-check.log`.

Three fresh reviewer contexts used explicit `codex:gpt-6.1-sol:high`. Correctness, contracts and integration each returned SHIP with zero findings; the finalizer recorded SHIP for source checkpoint `fe00d08becc0d1c73413244acf289ff42a86a5fa`. Writer and reviewers share the Codex family. Fresh reviewer builds were blocked by the read-only sandbox; two reviewers executed the cached six-test native binary. Worker logs provide fresh generation, compilation and test execution. Receipt `/tmp/impl-review-receipt-8f37faba39e2-fn-216-mini-activity-vertical-slice-the-model.6.json`; fan-out evidence `.flow/review-fanout/9153c56670db4b87922c4879533fe719/`.

Task .3 still owns the real server/SDK proof. Task .4 owns generic-boundary gate wiring and broader documentation. Record two findings there. The DSL's opaque generated World/Action requires a named guest adapter for export_guest!, pinned here by existing goldens. Shared-WKT ownership couples generated API recompilation to checker edits. The six-phase atomic-cancel/fatal-only restriction and terminal_remains vacuity disclosure from .1 remain unchanged. The known legacy generated-Case failure remains separately recorded in `.flow/tmp/fn216-legacy-baseline.md`; this task makes no passing legacy or live-SDK claim.

stage: impl-review - ran [2026-10-19T17:56:25Z..2026-10-19T17:59:01Z]

Tier: session (jev-unavailable(no_key)); project implementer routing preserved.

stage: plan-sync - skipped(config: planSync.enabled != true)

Tracker sync: n/a (bridge inactive).
## Evidence
- Commits: fe00d08becc0d1c73413244acf289ff42a86a5fa
- Tests: make model-check, make model-time, UMPIRE_INNER_CRATE=umpire-temporal UMPIRE_INNER_SOURCE=temporal/features/activity_mini/temporal.rs make model-time, UMPIRE_INNER_CRATE=umpire-temporal-tests UMPIRE_INNER_SOURCE=temporal/features/activity_mini/model.rs make model-time, cd model && mise exec -- cargo test -p umpire-temporal --test realization, cd model && mise exec -- cargo check -p umpire-temporal-api -vv, bash model/scripts/proto-temporal-go.sh --check, cd model/temporal/hosts/go && CGO_ENABLED=0 mise exec -- go test -tags test_dep -count=1 ./...
- PRs: