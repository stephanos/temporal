# fn-222 Rust one-box gates and support-list receiving design

Design only, recorded 2026-10-22 during the fn-232.10 measured-source freeze. No build, runtime gate, source change, task transition or qualification pass is recorded here. Implement after .10 acceptance and integrate fn-232.11 before freezing the qualification candidate. The conductor owns lifecycle and the final release freeze.

The mandatory cell remains `onebox/unchanged-workflow-type-smoke`, unchanged `./tests` selection `^TestWorkflowTypeEncodingSuite$/^TestPlainASCII$/^Succeeds$`. `onebox/frontend-system-info`, unchanged `./tests/gomadfunctional` selection `^TestFrontendSystemInfo$`, is the second required support row. Current .10 frontend evidence supplies neither mandatory-smoke acceptance nor both support rows.

## Source contracts

- `.flow/specs/fn-222-gomad-qualify-the-one-box-cell-and.md` and tasks .1/.2 own gate order, both required selections and the support claim.
- `.flow/artifacts/fn222-rust-onebox-gates-2026-10-22.md` records gate 1 unrun because no Rust prepare-only entry exists. It records no measured attempt.
- `tools/gomad/internal/backends/wasm/QUALIFICATION.md`, sections One-box cell gates and Per-test support list, remains the acceptance contract.
- Go semantic owners are `qualification/onebox_test.go`, `qualification/support_list.go`, `qualification/corpus.go`, `backend/temporal_inspection_integration_test.go` and `wasi/temporal.go`, relative to `tools/gomad/internal/backends/wasm`.
- Existing Rust owners are `rust/gomad/src/campaign/{prepare.rs,mod.rs}`, `qualify/{cells.rs,result.rs,report.rs,cost.rs}`, `replay/{preflight.rs,inspection.rs,cluster.rs}` and the shared authenticated artifact opener, relative to `tools/gomad`.
- `.flow/artifacts/fn232-rust-qualification-design-2026-10-22.md` supplies the broader .13 receiving map. This design supplies its fn222 subsection and does not replace the remaining runtime, workload, inspection or 39-root gates.

## Ownership and implementation order

1. fn-232.11 extracts the installation/configuration owner from `campaign/prepare.rs`, implements public `install`, and exposes the existing public `plan --output FILE` grammar. Reuse `prepare::prepare`, `prepared_publication` and `retain_prepared`. Extract seed plan construction before its current call to `run_observed`; planning must return before any guest executes.
2. The same batch exposes a typed preparation result and bounded receipt writer to the qualification owner. A small internal API takes already parsed `ExploreOptions` and returns the prepared target plus retained plan references. Public plan and qualification preparation share this operation. It does not add `qualify --prepare-only`, a support-list public verb or a fixture-only release control.
3. fn222.1 adds registered one-box gate definitions and support-list producer/readers under the existing Rust `src/qualify` owner. The task's older `src/qualification/**` Touches name refers to this owner. Add bounded repository tooling for gate orchestration and retained JSON/Markdown production. Keep execution in the release CLI or its exact linked release library.
4. Implement and verify tooling before the final source freeze. Run the ordered gates on the frozen release candidate. .13 may consume those receipts only while all candidate bindings still match. Workspace consolidation, host removal or another identity-changing change requires a new final-candidate run.

## fn-232.11 preparation interface

Recommended internal types are `PreparedPlan` and `PreparationReceipt`. The exact Rust names may follow surrounding code; the contract below is required.

`PreparedPlan` holds `prepare::Prepared`, the authenticated retained preparation artifact reference, portable plan path and digest. It preserves the original source kind/package, build tags, exact argv, selected seed envelope, limits and provenance. For seed planning, `guest_executions` is zero. The existing preparation artifact already has outcome/termination `prepared`; never count that synthetic retention record as a successful execution or replay.

Public `plan --json --output FILE` returns the existing plan response facts, with a documented Rust JSON field for the preparation receipt path/digest if exposed there. A qualification gate can instead call the internal typed writer from its release-library harness. No new public verb is needed. Publish the plan and receipt exclusively, use private artifact storage and sync before returning their references.

The immutable `gomad.rust-preparation-receipt/v1` is canonical bounded JSON, at most 32 MiB excluding the separately authenticated module. It contains:

- Plan path/digest, prepared artifact reference and its authenticated record hash; contained module payload name, exact module bytes and SHA-256.
- Complete `gomad.rust-wasm-provenance/v1` bytes or a contained payload reference plus size/digest, build key, source kind/package, working directory, build tags and exact final target arguments.
- Actual host/helper/runtime implementation identities, compiler executable/version, compiler/runtime inventory preimage and digest, selected source inventory preimage and digest, engine identity, complete configuration preimage/digest, model and namespace inputs, and captured-schema payload bindings.
- Preparation command vector and numeric exit, preparation elapsed measurement and raw available process/compiler observations. Unavailable observations remain explicit. This receipt makes no measured-cost or qualification claim.
- A zero guest-execution count for the seed preparation lane. Scoped choice planning may record its required base execution, so it is a separate lane and cannot stand in for one-box prepare-only.

Receipt admission uses pinned handles, bounded typed/canonical decoding, plan and artifact authentication, module byte/hash checks and provenance recomputation. It works after source/compiler removal when reading retained evidence. Validate source/compiler inputs while producing the receipt. Do not adopt a prior receipt or preparation measurement into a later measured operator process.

Gate 1 plans the success selection only. Retain its receipt permanently in its own preparation root before success/canary start. Measured cells independently prepare their own source in fresh processes; compare both modules against the gate-1 receipt. Their compiler/resource accounting includes their own preparation. Full build-key/configuration equality with the canary is not required because the registered canary argv differs. Authenticate all those distinct identities rather than changing their bytes.

## Exact one-box policy and limits

The qualification policy is `wasi.TemporalQualification`, not `TemporalInstallation`. Installation enables a network model; qualification has nil network. Inspection is a third complete policy with network and different bounds. Keep a single shared Rust policy derivation with explicit policy selection inside tooling. Public installation keeps its existing `--config` path for a complete qualification configuration. Do not add a public policy flag or mutate a TemporalInstallation configuration after derivation.

Qualification policy captures `schema/sqlite` under `/workspace/schema/sqlite`, preserves its complete descriptor/payload identities, uses writable `/workspace`, `/tmp` and the selected guest directory, and sets epoch `946684800000000000`, read step zero and entropy key SHA-256 of `gomad-wasm-prototype/v1`. Policy arguments are nil; Runner supplies argv. Guest directories are `/workspace/tests` and `/workspace/tests/gomadfunctional`. Preserve `PWD`, `TEMPORAL_ROOT=/workspace`, `TMPDIR=/tmp`, `TZ=UTC`, and `TEMPORAL_TEST_LOG_LEVEL=warn`.

| Quantity | Required value |
| --- | --- |
| Go compiler / profile | Stock Go 1.27.1 / cooperative |
| Build tags | `disable_grpc_modules,gomad_wasm_slim,sqlite3_dotlk,test_dep` |
| Seed / repetitions / resident guests | 7 / 2 / 1 |
| Guest memory / fuel | 4 GiB / `1<<50` |
| Policy output / transcript / calls | 256 MiB / 1 GiB / `1<<28` |
| Descriptors / files / filesystem / pending events | 4096 / 100000 / 512 MiB / 4096 |
| Campaign execution / overall / terminate grace | 15 min / 30 min / 1 s |
| Campaign output / World / choice / retained success budget | 256 MiB / 32 MiB / 64 MiB / 2 GiB |
| Failure policy / budget | all / 1 |
| Measured preparation / each execution or replay | at most 10 min / at most 15 min |
| Compiler / operator / cold store / warm store / resident-store RSS ceilings | 6 GiB / 2 GiB / 6 GiB / 4 GiB / 8 GiB |

Use .10's existing linked-process accounting and raw phase observations. CPU is compiler children plus linked self once. An unattributed required memory component leaves the budget pending. These are the existing ceilings; successful frontend execution or a process exit cannot waive them. The .13 qualification driver retains its unchanged 3600-second outer bound. The historical Go test command timeouts do not widen it.

Each support success runs exactly `gomad-target -test.v -test.count=1 -test.timeout=72h -test.run=<registered selection> -persistenceType=sql -persistenceDriver=sqlite`. The canary changes only that selection to `[`. Argument order matters to the existing support contract. The current .10 corpus starts frontend argv with `-test.run`, so those receipts cannot automatically earn a support row.

## Gate order and process exits

Run smoke preparation first without `GOMAD_WASM_QUALIFY_WORKLOADS`; retain command, stdout/stderr hashes, numeric exit and receipt. A failure stops before either smoke measured mode. Its missing later gates remain unrun, with no numeric exit invented.

After successful preparation, run smoke success then smoke canary in distinct fresh release processes with mandatory-workload intent. Both final Results and their publication receipts must validate, be deterministic, retain exactly two executions, replay both to matches and satisfy the unchanged measured budget. Success has target success and zero replayable bugs. Canary has target failure and exactly one replayable bug. Require the preserved preparation module digest in both Results.

Public Rust `qualify` returns 0 for success and 1 for the expected target-failure canary. Retain those raw CLI exits. The canary gate wrapper returns 0 only after authenticating that expected exit and all canary evidence. This preserves the former Go test gate's zero successful assertion exit without falsifying a CLI result. Other nonzero CLI exits stop the gate.

Run the storage/transport inspection cell and validate both Result/receipt pairs. Then run frontend preparation, success and canary in its own root without mandatory-workload intent, preserving the same stop rules and exact frontend registry selection. The shared inspection receipt may satisfy both rows only when its full candidate identities match. Support-cell roots contain only their row's final success/canary Results; keep the preparation and inspection roots outside them and keep the mandatory corpus root separate.

The four row gate names remain, in order, `unchanged-source-preparation`, `measured-pass-replay-success`, `measured-pass-replay-canary`, `storage-transport-inspection`. Each records its exact wrapper command, zero validated-stage exit and absolute retained log path. Nested commands retain their own real exits. Missing/nonzero gates leave the row pending. Empty lists are pending reports, not support-list publications. A smoke-only partial list does not complete R2.

## Retained one-box storage/transport inspection

Implement the `onebox` subtest of `backend/temporal_inspection_integration_test.go`, using the unchanged staged Go files from `testdata/prototype_temporal` and `TestReplayInspectionWorkflowFailure`. Reuse public release `explore` plus `replay` inspection or the exact linked release library. Generic debug fixtures and the frontend cell are insufficient substitutes.

- Use TemporalInspection's full policy, 4 GiB memory, `1<<50` fuel, `1<<28` calls, 16 MiB policy output, 256 MiB transcript, 4096 descriptors, 100000 files, 512 MiB filesystem, 4096 events and the original bounded network. It has zero entropy key, frozen clock, writable `/workspace` and `/tmp`, and `GOMAD_DEBUG_CANARY=1`.
- Build with Temporal guest tags plus `gomad`. Target args are `-test.run=^TestReplayInspectionWorkflowFailure$`, `-test.v`. Run one-node typed scenario, boot main, address `127.0.0.1`, start then wait for `history` incarnation 1, seed 7, parallel 1, 30-minute execution and 45-minute overall bound, grace 1 s, policy first, failure budget 1, campaign output and World bounds 1 MiB. Preserve the original zero caller choice override and the actual scenario owner's 1 GiB incarnation trace bound. Rust scenario preparation currently substitutes 8 MiB when zero is supplied; receiving tooling must restore the original effective bound rather than silently accept that smaller fixture default.
- Retain exactly one failure artifact. Its authenticated stdout includes `workflow completed [step 0 attempt 2 step 1 attempt 2]` and `Temporal inspection canary after workflow timers and activity retries completed`.
- Make only the staged source and private compiler/runtime aliases unavailable. Keep the dormant retained module. Fresh replay must reproduce the authenticated complete canonical cluster record.
- Inspect stop=failure, instance=history, incarnation=1, dumps stacks/heap/temporal.mutable_states/temporal.shards, probe `temporal.*.snapshot`, original default total/per-dump bounds 8 MiB/4 MiB. Require inspected_prefix, verified prefix and matching witness, nonempty stack/heap/probes, both exact snapshot probe names, successful useful snapshot dumps, `values may precede stop`, and shard `snapshot=` content.
- Reauthenticate the original artifact record, backend evidence and all opened retained payloads after inspection and reopen. They remain byte-identical. Retain diagnostic report paths and hashes separately.

This supplies fn222's shared onebox inspection gate only. The shard/workflow subtests and all other inspection obligations remain required under .13's wider map.

## Rust support-list producer and identity derivation

Port `BuildSupportList`, `ValidateSupportList`, read/write and Markdown projection into `src/qualify/support.rs`, with a closed two-cell registry and no sweep input. Add a small repository tool or linked release-library harness to invoke those APIs; no new public `support-list` command. Consume only retained Rust Results opened through `qualify::result::open` and `open_receipt`, which already authenticate canonical raw bytes and independently recompute identities, costs, budgets and receipt process bindings.

The old Go document is `gomad.wasm-support-list/v1`. Preserve its per-test statement, registry, refusal rules and gate semantics. Give the Rust representation an explicitly documented lineage/schema difference if adding the typed raw-reference fields below; do not silently reinterpret Go fields or admit Go Results. Recommended Rust envelope is `gomad.rust-wasm-support-list/v1`, described as the Rust port of that contract, with the existing fixed statement unchanged. This format decision must be recorded in fn222's documentation diff before acceptance.

Two current Result facts prevent a mechanical Go-field copy. Rust `model_sha256` hashes configured model bytes including the working-directory digest; smoke and frontend legitimately differ. Rust `host_config_sha256` hashes the full execution Configuration including argv and the build-key-derived choice runtime identity; success and canary legitimately differ. Keep every raw Result/configuration/model/toolchain hash unchanged.

Use a typed support view with the following explicit bindings:

- Candidate identity binds integrated source digest, actual release executable digest, host implementation digest, linked helper implementation digest, compiler digest, source-bound runtime-overlay implementation plus compiler/runtime inventory digest, engine digest and OS/arch. The linked helper identity binds its actual model implementation source closure. Name that binding as the model implementation owner, rather than substituting a configured-model hash and calling the two equivalent. If a separate model implementation digest is introduced, retain its exact source preimage and build binding.
- Row identity binds the registered package/function/selection, complete guest-policy preimage, captured schema/namespace/model inputs, source inventory, module digest and bytes, exact build tags, seed, repeat and capacities.
- Derive the row's common success/canary policy through an explicit typed constructor that removes only the separately retained target arguments and build-key-derived choice runtime identity from Configuration. Recompute and validate both removed fields against each Result's registered exact argv and authenticated provenance. Compare every other policy field exactly. This is a support-view derivation, never artifact or golden normalization. Document the preimage and preserve each Result's full raw host/model/configuration/toolchain hashes in its reference.
- Different registry rows may have their declared working directories and corresponding configured-model identities. Require the same compiled implementation candidate, while validating each row's directory/schema/policy against its own registry entry. Do not require the two legitimate configured-model hashes to equal.

For every input require a final non-provisional measured Result, valid single receipt, unique nonzero operator PID, complete fresh-process preparation observations, deterministic expected outcome, no failure/cost/budget failure, budget within bounds with no pending components, exactly Repeat executions and matched fresh replay of every own artifact. Pair exactly one success/canary per registry cell. Reject unknown cells, duplicate modes/processes, mixed candidates, mixed module/size/tags within a pair, changed argv/limits/policy, missing evidence and unknown/duplicate/missing gates.

Authenticate each execution artifact through the shared opener. Its complete `stdout` payload must match the record's stream bytes/digest and retained payload entry. Refuse missing/truncated/capacity output. Every success execution needs exactly one verbose `--- PASS: <registered function> (...s)` and no matching skip. Wrapper PASS, package PASS, zero selected tests, `-test.list`, extra flags or duplicate named PASS cannot establish a row. Canary must preserve the invalid-regexp target-failure witness and both own matched replays.

Sort unique rows by package/function. Publish JSON and Markdown exclusively; Markdown is a pure projection. The independent retained-list validator repeats registry, candidate digest, pair/reference, gate order, actual named-PASS and measured-Result validation. A shape-only edited document cannot become proof. The first accepted fn222 list contains both required rows; a new candidate starts with no rows.

## Narrow validation and final receipts

Before the measured freeze, add public plan/install corpus cases for zero-execution preparation, source/package/module/argv binding, exclusive receipt writes, changed compiler/schema/policy refusal and retained read after inputs disappear. Add support corpus fixtures for both complete rows and the old producer's refusal matrix, especially missing/nonzero gate, repeated PID, missing receipt, provisional Result, wrong selection/order, no-tests/skip/truncated stdout, changed candidate, mixed module, forged raw hash and changed budget observation. Add inspection receiving cases that assert every original onebox witness and detached-input condition. Golden comparison follows raw authentication and two fresh agreements; second regeneration must be byte-identical.

The executable gate driver records actual argv, environment intent, start/end, numeric exit and stdout/stderr digest per subprocess, then publishes the four ordered stage receipts per row. It must stop dependent work after the first preparation failure, authenticate fresh Results through Rust readers, and reject a missing required case or skipped assertion. No measured run is authorized by this design document alone during .10's freeze.

After all source changes and the .13 workspace/retirement identity rotations settle, record one final frozen candidate's source/compiler/runtime/module/model/engine/helper/host/executable identities, commands, raw receipts and exits under the unchanged 3600-second qualification driver. Copy the accepted support JSON and its Markdown projection unchanged beside the fn222 checkpoint, then update the QUALIFICATION row. Until those live receipts exist, fn222 R1/R2 and the receiving .13 gates remain pending.
