---
satisfies: [R2]
---
# fn-153-gomad-retire-canonical-json-and-private.20 Replace remaining WASM strict JSON scanners with the shared decoder

## Description
The integrated requirements audit found that the explicit 2026-10-12 amendment still has handwritten duplicate-key and UTF-8/trailing scanners in backends/wasm/wasi/protocol.go, backends/wasm/qualification/spec.go, backends/wasm/installation/installation.go and backends/wasm/simulation/oracles.go. Migrate these named surfaces to the pinned shared strict JSON owner; retain field ownership, protocol framing, transport buffers and mutable caller-buffer ownership, numeric semantics, capacity/depth limits, refusal classes, error precedence, and model identity inputs. Update permitted architecture edges and tests before using the shared owner. No encoder, generic canonical comparison, compatibility fallback or new handwritten strictness scanner. This is a missing explicit spec requirement, not a new feature.

**Size:** M
**Touches:** [tools/gomad3/backends/wasm/wasi/protocol*, tools/gomad3/backends/wasm/wasi/helper_model*, tools/gomad3/backends/wasm/wasi/identity*, tools/gomad3/backends/wasm/qualification/spec*, tools/gomad3/backends/wasm/qualification/*_test.go, tools/gomad3/backends/wasm/installation/**, tools/gomad3/backends/wasm/simulation/oracles*, tools/gomad3/internal/strictjson/**, tools/gomad3/internal/archcheck/**]

### Revival trigger (2026-10-13)

The final fn153 requirements audit demonstrated that the named WASM readers still contain the owner-amended handwritten JSON scanners. Finish this missing migration within the authorized shared-owner rotation batch before fn153.18's final audit and completion review. Preserve existing transport/model semantics and resource/error contracts; update generated identities from the integrated candidate and keep historical qualification bound to its original sources. Native deferrals and the no-CI/PR/push boundary remain in force.


### Identity owner binding

The caller migration also binds the actual strictjson implementation sources and pinned library version/content sum into the WASI composite identity. `internal/strictjson/identity.go` embeds source without runtime host reads; tests verify its pin against go.mod/go.sum and reject missing-source/drift. `wasi/identity.go` consumes that identity and fails closed when absent. This exact source/identity edge was authorized before implementation. Caller depth enforcement uses only the pinned tokenizer API, preserving legacy root-zero empty-container boundaries; duplicate/UTF-8/schema syntax remains library-owned.

### Verification

Run focused strictjson and all four WASM reader packages with `-tags test_dep -count=1`, existing strict transport/mutable-buffer/error-order controls and actual helper parity; run both-platform architecture/purity and generated validation. The conductor runs the standard WASM suite and final integrated rotation-batch qualification.

### Integrated source review (2026-10-13)

Combined JSON/config scope reviewed at frozen `6ff6a117b2`, based on `603d168023`. Receipt `/tmp/wasm-json-network-config-source-review-20261013.json` now reports SHIP after all four merged findings were fixed and same-session re-review found no blockers. Model pin on that frozen source is `sha256:0b9d0091a80b036be2eec7a2d271bd4ca1ce9bc134965621c5e582a9d25ff444`. Main integrates the same fixes but newer connection/holding/expiry work is outside this review. Full reader, architecture/purity, generated validation and scoped lint checks pass; logs and task/R-ID scope are retained in the shared rotation handover. This is source acceptance only: final current-source WASM qualification, exact native requirements, task dependencies and other acceptance remain open. Do not mark the task done from this source verdict.

stage: impl-review - ran (model: gpt-6.1-sol/high, same-family; first round three axes, one single-session re-review)
## Revival trigger (2026-10-13)

The final fn153 requirements audit demonstrated that the named WASM readers still contain the owner-amended handwritten JSON scanners. Finish this missing migration within the authorized shared-owner rotation batch before fn153.18's final audit and completion review. Preserve existing transport/model semantics and resource/error contracts; update generated identities from the integrated candidate and keep historical qualification bound to its original sources. Native deferrals and the no-CI/PR/push boundary remain in force.


### Identity owner binding

The caller migration also binds the actual strictjson implementation sources and pinned library version/content sum into the WASI composite identity. `internal/strictjson/identity.go` embeds source without runtime host reads; tests verify its pin against go.mod/go.sum and reject missing-source/drift. `wasi/identity.go` consumes that identity and fails closed when absent. This exact source/identity edge was authorized before implementation. Caller depth enforcement uses only the pinned tokenizer API, preserving legacy root-zero empty-container boundaries; duplicate/UTF-8/schema syntax remains library-owned.

## Verification

Run focused strictjson and all four WASM reader packages with `-tags test_dep -count=1`, existing strict transport/mutable-buffer/error-order controls and actual helper parity; run both-platform architecture/purity and generated validation. The conductor runs the standard WASM suite and final integrated rotation-batch qualification.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

All four named readers use library-owned strict syntax and their handwritten duplicate/UTF-8/trailing scanners are removed. Recursive duplicate keys, invalid UTF-8, trailing data and unknown owned typed fields are refused; intentionally open maps/raw producer fields retain their declared ownership. Existing transport depth/byte and model/installation/qualification/oracle bounds, mutation and precedence negatives pass. Current-source architecture and generated validation pass; changed identities are regenerated and tested. Focused portable readers and actual helper/model parity tests pass; no historical qualification is claimed for the rotated model.

## Done summary
All four named WASM readers use the pinned shared strict decoder; depth, byte limits, mutable ownership, numeric and error precedence contracts remain. Original scoped SHIP and final shared formal implementation SHIP are retained; fn153 completion re-review is SHIP. Full frozen960 qualification passed with expected success/canary dispositions, all selected SQLite states and source/helper preservation. Exact gate-only correction plus fresh affected checks is joined under the milestone small-review-fix rule; original source8c evidence is not relabelled. Native execution remains deferred to fn128/fn149; no external action. stage: impl-review - ran (model: gpt-6.1-sol/high, same family). stage: completion-review - ran (model: gpt-6.1-sol/high, same receipt/session). stage: QA - ran (full frozen960 WASM pipeline exit0).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 960c966cdc9fba84736a268d8949f30214c16ac4, 94799507d4, ea9ca7c35c
- Tests: Complete frozen960 kKv0jE/run-qualification.sh exit0; stages.log finalPASS05:36:50Z, Current gate-fix all-platform architecture, validation, root/module lint, clock and actionlint exit0; /tmp/gomad-gate-integrated-* logs, Formal impl-review-rotation-integrated-20261014.json SHIP; completion-review-receipt-fn-153.json SHIP
- PRs: