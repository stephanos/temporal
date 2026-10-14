---
satisfies: [R11]
---
# fn-166-gomad-readable-package-structure-and.11 Rename the gomad3 module and tree to gomad

## Description
After the identity-rotating structure batch settles the final tree, rename tools/gomad3 to tools/gomad and change the nested Go module/import path from go.temporal.io/server/tools/gomad3 to go.temporal.io/server/tools/gomad. Update workspace references, commands, scripts, CI routing, generated/source identities, fixtures, and documentation in one atomic move. Preserve the gomad CLI name and behavior; do not retain a gomad3 compatibility tree or import shim.

## Acceptance

Current scope (owner decision, 2026-10-14): apply [native hibernation](../../MILESTONES.md#native-backend-hibernation); native execution obligations transfer to fn-128/fn-149, superseding earlier native completion blockers in this task. All retained source/portable and actual WASM criteria remain required; this amendment supplies no native pass.

The repository has no surviving tools/gomad3 path or go.temporal.io/server/tools/gomad3 import/module reference outside intentional historical records. All workspace, build, generator, architecture, lint, WASM, qualification, documentation, and fixture references use tools/gomad. The rename is atomic, identities and generated outputs are rotated from the final source closure, and the full post-rotation gate set passes without compatibility shims.

## Done summary
Renamed the module/tree atomically to tools/gomad and go.temporal.io/server/tools/gomad. Updated build, workspace, CI, generators, identities, fixtures and operative documentation without a compatibility tree or import shim. Preserved runtime protocol/env names and intentional historical fixtures.

All retained R11 source/portable and actual WASM gates passed in the joined post-rename qualification. Frozen execution source75835196d7/source manifest0bd4d29c remains explicit. Original session6808 ended exit1 solely after a successful network producer because the independent one-line IDEAS.md commit changed HEAD to07066d09. Reviewed source-equal continuation87187 ended exit0 at16:57:21Z, retaining both revisions, exact doc-only delta, original receipts and unchanged source/compiler/helper/ledger checks; it ran every remaining gate. Independent final evidence audit verified85 stages,62 mandatory named gates,800 runtime records/32 exact replays,20 artifact replays,14 authenticated Result/publication/report pairs, five qualified successes/five expected-failure canaries, four qualified SQLite selections and exact preservation. Original whole-driver success is not claimed.

Formal Codex implementation review completed SHIP on correctness/contracts/integration, zero findings/unaddressed R-IDs, explicit gpt-6.1-sol/high. Receipt /tmp/impl-review-receipt-657da2bc4466-fn-166-gomad-readable-package-structure-and.11.json; fanout37a2246c5410481eaaf697d44e4e4596, primary session01a13b61-4c79-7422-be25-80c4219d3c90. Same GPT family as writer; fresh reviewer tests were blocked by the required read-only sandbox, while actual producer/qualification evidence was independently checked.

Native execution stays hibernating under fn128/fn149; portable/source checks establish no native platform pass. Whole fn166 completion review and spec close remain separate gates. Current checkpoint and MILESTONES record the completed qualification and other unaccepted source preparation.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b9d19bf37b3a12319905a3d9d538133f9e0b92d0, 87e3442715a91a2c5d6f0b35ae1506ecda63766e, 4bd0932e700595f82a4c184d5d763933147d8436, fb77e50c1022a12651656f7ceee5db1e34376fc0
- Tests: Joined R11 segment: bash /Users/stephan/Workspace/skunkworks/gomad-postrename-retry-20261014.TMcAas/run-qualification.sh 75835196d7345cc0f2d9886d79851fdd3b05eeb3 a1f0be7e96; all executed stages pass; original overall exit1 after networkPASS solely for IDEAS-only HEAD drift, preserved, bash /Users/stephan/Workspace/skunkworks/gomad-postrename-doc-continuation-20261014.bL7lrR/continue-qualification.sh; exit0, every remaining gate and source/corpus preservation pass at16:57:21Z; driverSHA6ed6e56ae8b44c92c8b58406d938e936fdc699d31a42ad45440797e06a66fa86, Independent final joined evidence audit YES:85 stage exit0 pairs,62 mandatory named gates,800records/32replays,20artifactreplays,14authenticatedResult/receipt/report pairs and exact four-state SQLite selection mapping/preservation, sha256sum -c source.sha256 helper.sha256 compiler.sha256; current accepted execution closure unchanged; cmp primary-before.sha256 primary-after.sha256 PASS, flowctl codex impl-review-fanout fn-166.11 --base a3127c9403 --spec codex:gpt-6.1-sol:high; all three draws SHIP; fanout-finalize37a2246c5410481eaaf697d44e4e4596 SHIP with zero introduced findings and unaddressed R-IDs
- PRs: