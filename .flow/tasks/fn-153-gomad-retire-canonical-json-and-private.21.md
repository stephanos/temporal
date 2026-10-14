---
satisfies: [R1, R2]
---
# fn-153-gomad-retire-canonical-json-and-private.21 Close remaining WASM cache publication and World syntax audit gaps

## Description
Close two demonstrated gaps in the existing fn153 requirements, found by the committed-source inventory at fe84b745d7. This is not a new feature or a blanket reader migration.

**Size:** M
**Touches:** [tools/gomad3/backends/wasm/backend/provider.go, tools/gomad3/backends/wasm/backend/provider_test.go, tools/gomad3/backends/wasm/simulation/retained.go, tools/gomad3/backends/wasm/simulation/retained_admission_test.go, tools/gomad3/backends/wasm/simulation/retained_canonical_test.go, tools/gomad3/world/replay.go, tools/gomad3/world/replay_test.go, tools/gomad3/runner/internal/campaign/choice_exploration_start_test.go, tools/gomad3/internal/archcheck/edges.go, tools/gomad3/internal/archcheck/inventory.go, tools/gomad3/internal/archcheck/architecture_test.go]

### Approach

- WASM preparation cache module/receipt publication currently uses os.Rename from a .build directory. Both are mutable whole-file replacements, not the directory/immutable carve-outs. Stage through the existing shared internal/hostfs owner in the cache directory, seal/validate both exact staged files before publishing either, and publish module before receipt. Receipt remains the commit marker: a module without a receipt is an interrupted publication and may be rebuilt/replaced. Preserve compiler/source verification, current modes, bounds, cleanup, cancellation, primary error identities, contextual classification and rename/durability disposition. Use the existing shared operations rather than a new publisher or public facade. Existing non-atomic prepared-output writes are not automatically in scope.
- Add focused positive and negative cache publication controls, including stage/validation failure leaving old destinations, cancellation, receipt-last ordering, interruption recovery, successful rename followed by directory-sync failure, and cleanup failures. Shared hostfs's existing operation tests may supply its own fault proof; caller tests must prove its transaction composition. Do not fabricate native prepared inputs or weaken preparation/identity admission.
- Remove world/replay.go's redundant raw utf8.Valid JSON preflight. The pinned shared strict decoder owns raw syntax, including invalid UTF-8 and duplicates; preserve typed original-string and generated binary-wire UTF-8 validators. Keep bounded admission, invalid-input refusals, semantic replay validation, hashes and exact wire reencode ownership. Preserve documented error classifications and tested precedence; explicitly report any unavoidable library error-message difference rather than writing another scanner.
- Update only the exact internal/hostfs architecture edge needed by the host preparation owner, if absent. Do not widen guest/WASI effects or general imports. No new package, generic encoder, compatibility fallback, hand-written strictness scan or cache optimization is admitted.
- Classify WASM provenance and the retained cluster bundle as ordinary evidence JSON, not a new wire exemption. Their raw-byte SHA/length authentication is separate from producer reencoding equality. Retire only their spelling-only comparisons, using the existing strict decoder to preserve malformed/duplicate/raw-invalid-UTF-8/unknown/trailing refusals. Preserve schema, required shapes, complete identities, semantic validation, nil/empty and pointer distinctions, original record hashes, raw authentication and large-bundle allocation bounds. Keep the streaming producer writer; do not introduce a second whole-bundle encoding. Add noncanonical-spelling positive controls alongside tamper/identity/shape negatives. Older unrelated stdlib readers are not admitted for a blanket rewrite.
- Investigate the frozen campaign-plan hash failure reproduced at unchanged baseline15119dbbeb (actual fd169c192d69477d5dd2f6463e29ee4d2edcf0c67e49eb83f5d228af63b31771 versus retained afd609dfe85609a457d678d14ee350b0f3b22f63c95f68a2e80e6af814708d65). Trace which serialized inputs changed under accepted migrations. Refresh this exact first-baseline pin only if evidence proves an intentional identity/input rotation with preserved semantics; otherwise request scope binding for the actual defect. Never change an expected value just to obtain green.

### Quick commands

From tools/gomad3 with stock Go 1.27.1 and tags test_dep: focused provider publication/preparation tests, full ./world tests, shared ./internal/hostfs tests, affected architecture selection and scoped canonical lint. Root owns final integrated identity refresh, review, generated validation, WASM qualification and Flow completion. Required supported-native gates remain under their existing owners.

### Integration-discovered shared-decoder correction (2026-10-13)

**Additional Touches:** [tools/gomad3/internal/strictjson/decode.go, tools/gomad3/internal/strictjson/decode_test.go]
Restore the standard-library numeric time.Duration representation using the pinned library's FormatDurationAsNano option, after inspecting its implementation; the retained bundle's existing Request.Timeout exposed this missing v1 compatibility. No local retained shim, scanner, encoding change or unrelated option change. Add duration bounds/type/null regressions and rerun shared strictness controls; root refreshes all affected identities on the integrated candidate.

The exact native qualification testdata fixture prefix in archcheck/inventory.go is admitted for fn166's tagged fixture relocation. The fn166 worker owns architecture_test.go for library retirement and exact backend internal-hostfs import controls; this worker must not edit that shared file.

### Integrated source checkpoint (2026-10-13)

Original81ae933d0a/60881a8706 normalize to eb13e65f85/e986f971c3, joined with fn166 library retirement6fd1a8f4b2. Source handover /tmp/fn153-audit-corrections-summary.md and /tmp/fn153-audit-corrections-evidence.json reports focused tests/lint exit0. Independent integrated review, generated/source identity checks and current-candidate WASM qualification remain open. The shared numeric-duration compatibility correction intentionally rotates dependent model identities; no historical qualification transfers to this source.
## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

- [ ] WASM mutable module/receipt replacement uses shared staged publication with exact staged validation and preserved module-before-receipt transaction, modes, bounds, cancellation, cleanup and error/disposition controls.
- [ ] World replay raw syntax is library-owned without a redundant UTF-8 scan; malformed/duplicate/raw-invalid UTF-8, semantic and wire negative controls pass and typed/binary validators remain.
- [ ] Ordinary WASM provenance/retained-bundle spelling checks retire without weakening raw authentication, schema/semantic/identity/presence validation or large-bundle bounds; positive alternate spellings and strict/tamper negatives pass.
- [ ] The reproduced campaign-plan pin discrepancy has an evidence-backed disposition; any changed expectation follows traced intentional input rotation, not a weakened gate.
- [ ] Focused source tests, exact architecture edge and canonical affected lint pass; no private publication helper, new package, broad reader migration or host-effect widening is added.
- [ ] Root integrates and reviews the correction and binds final identities/qualification separately; no historical or skipped native evidence supplies current acceptance.

## Done summary
WASM cache stages and validates exact module/receipt files before ordered module-first/receipt-last publication using the shared owner; cancellation, cleanup and error/disposition controls pass. World raw JSON strictness is library-owned while typed strings and exact binary/JSON wire validation remain. Ordinary provenance and retained evidence admit alternate spelling without weakening raw authentication, shapes, identities or bounds. Numeric duration compatibility and the campaign pin discrepancy have traced source-backed controls. Final shared formal implementation and fn153 completion re-reviews are SHIP. Full frozen960 qualification passed; exact gate-only corrected-source checks join under the documented rule, without relabelling source8c evidence. Native execution remains deferred. stage: impl-review - ran (model: gpt-6.1-sol/high, same family). stage: completion-review - ran (model: gpt-6.1-sol/high, same receipt/session). stage: QA - ran (full frozen960 WASM pipeline exit0).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, ea9ca7c35cc46db340b648126cb8edd8affbd571
- Tests: Frozen960 full kKv0jE qualification exit0; expected success/canary dispositions plus exact fourSQLite selections/catalogue, Current corrected-source all-platform architecture, validation, root/module lint, clock and actionlint exit0; retained /tmp/gomad-gate-integrated-* logs, Formal shared impl-review and fn153 completion receipts SHIP; independent completed-corpus audit YES
- PRs: