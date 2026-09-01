---
satisfies: [R3, R4, R5]
---
# fn-42-centralize-configuration-authoring-with.2 Hard-cut Callback and Matching to ConfigUseSpec

## Description
Migrate all six owner declarations and their cross-owner metadata coverage to the proven authoring interface (R3, R4, R5). This task owns the atomic source-level cutover and final model gates.

**Size:** M
**Files:** `model/Temporal/System/Callback/Configuration.lean`, `model/Temporal/System/Matching/Configuration.lean`, `model/Temporal/System/ConfigurationIntegrationTests.lean`
**Touches:** [model/Temporal/System/Callback/Configuration.lean, model/Temporal/System/Matching/Configuration.lean, model/Temporal/System/ConfigurationIntegrationTests.lean]

### Approach
- Replace each classification/interpretation/definition-result/witness/get cluster with one owner-authored spec and one explicitly proven checked definition.
- Remove obsolete individual intermediate declarations and classification aggregate lists; keep the public context constructors, `callbackUseDefinitions`, `matchingUseDefinitions`, and typed `*Use` functions stable.
- Preserve all six Definition IDs, authored catalog expectations, fingerprints, decoders, policies, lifecycle metadata, registry ordering, and existing comments exactly.
- Strengthen integration coverage around the erased checked-definition metadata so the hard cut proves the same six classifications/uses without depending on removed intermediate names.
- Leave callback address parsing, callback domain projection/behavior, file layout, facades, generated modules, and resolver implementation untouched.

### Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Callback/Configuration.lean:258-444` — four repeated declarations, registries, contexts, and use functions
- `model/Temporal/System/Matching/Configuration.lean:19-109` — two repeated declarations and use functions
- `model/Temporal/System/ConfigurationIntegrationTests.lean:12-72` — cross-owner keys, count, resolution, and provenance assertions
- `model/Temporal/System/Callback/ConfigurationTests.lean:92-270` — decoder, override, projection, and callback behavior regressions

**Optional** (reference as needed):
- `model/Temporal/System/Matching/ConfigurationTests.lean:8-30` — registry validation and default resolution
- `model/Temporal/System.lean:1-3` — facade import direction that must remain unchanged

### Key context
- This is an authorized hard cut of unused authoring symbols, not permission to rename or remove the consumer-facing use functions and registries.
- Shared configuration must remain owner-independent, and owner-specific scalar/address decoders and exact-context constructors stay in their current modules.

### Lifecycle reconciliation

The hard-cut implementation and its acceptance hardening remain reachable at commits
`b5868effd0c023a6efef893ed9aec502bb40b1a0`, `cd9cfc2454fed801f95811972b4523a9c4d30d92`,
and `32ffa50c84d1892b42270c90b70b128a7b850ee8`. This reopened lifecycle run verifies
that existing implementation rather than introducing duplicate source changes.

### Acceptance
- [ ] Exactly four Callback and two Matching specs produce the same ordered checked-definition metadata and registry keys as before.
- [ ] No owner-local `DefinitionResult`, named `isSome` witness, duplicate classification/interpretation record, or obsolete classification aggregate remains.
- [ ] Context, defaults, overrides, illegal-context errors, address-decoder failures, callback traces, resolution sources, and provenance regressions pass unchanged.
- [ ] Existing comments remain intact; imports, facades, generated files, and external documentation remain accurate without unrelated edits.
- [ ] `cd model && mise exec -- lake build TemporalModelTests`, `make umpire-build-model`, and `make lint-model` pass.

## Acceptance
- [ ] Six settings complete the hard cut and preserve exact checked metadata/order for R3.
- [ ] Focused and integration regressions demonstrate R4 behavior and diagnostic equivalence.
- [ ] Full model and lint gates satisfy R5 with no generated or unrelated changes.

## Done summary
Reconciled the reopened task lifecycle to the reachable hard-cut implementation at `b5868effd0c023a6efef893ed9aec502bb40b1a0` and its acceptance hardening at `cd9cfc2454fed801f95811972b4523a9c4d30d92` and `32ffa50c84d1892b42270c90b70b128a7b850ee8`, without duplicating production changes. Fresh current-tree verification and a same-session Codex review confirm the four Callback and two Matching specs retain their exact checked metadata, stable consumer interfaces, behavior, and R3-R5 coverage.

baseline: green via handoff (verified at 51437e6d by fn-42-centralize-configuration-authoring-with.1); fresh aggregate model, full model build, and model lint baselines passed

GATE_SKIPPED:focused-config:docs-only - cumulative diff classified tier-B (no executable paths touched)

GATE_SKIPPED:aggregate-model:docs-only - cumulative diff classified tier-B (no executable paths touched)

GATE_SKIPPED:umpire-model:docs-only - cumulative diff classified tier-B (no executable paths touched)

stage: impl-review - ran [2026-09-01T06:54:24Z..2026-09-01T06:56:41Z] (codex; SHIP)

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 9d8a2c159f290ee9e464afacc9b30c7d4a4c63c6, 6b21cd8f17d38814d65cc3cfa99e20b56e992b80
- Tests: baseline: green via handoff (verified at 51437e6d by fn-42-centralize-configuration-authoring-with.1), cd model && mise exec -- lake build Temporal.System.Callback.ConfigurationTests Temporal.System.Matching.ConfigurationTests Temporal.System.ConfigurationIntegrationTests, cd model && mise exec -- lake build TemporalModelTests, make umpire-build-model, make lint-model, GATE_SKIPPED:focused-config:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:aggregate-model:docs-only - cumulative diff classified tier-B (no executable paths touched), GATE_SKIPPED:umpire-model:docs-only - cumulative diff classified tier-B (no executable paths touched)
- PRs: