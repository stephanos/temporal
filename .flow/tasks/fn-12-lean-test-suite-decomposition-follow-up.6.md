---
satisfies: [R2, R3, R4, R5, R6, R7]
---
# fn-12-lean-test-suite-decomposition-follow-up.6 Split shared Temporal configuration tests by concern

## Description
Split the final shared Temporal configuration suite into validation, resolution, and catalog concerns over generic fixtures (R2-R7). Leave the already-cohesive Callback-owned suite unchanged.

**Size:** M
**Files:** `model/Temporal/System/Configuration/Tests.lean`; new `model/Temporal/System/Configuration/Tests/{Fixtures,Validation,Resolution,Catalog}.lean`
**Touches:** [model/Temporal/System/Configuration/Tests.lean, model/Temporal/System/Configuration/Tests/**]

## Approach
- Confirm the shared owned tree matches the fn-10 closure baseline, then map its 19 assertions and comments; record the 11 Callback assertions and their entire source file as a byte-for-byte read-only baseline.
- Follow the approved final-layout design at `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:153-180`.
- Put only generic checked-use and view-construction helpers shared across concerns in `Fixtures`.
- Keep validation, deterministic resolution/context/typed-read behavior, and catalog/default-fixture checks in their named concern modules.
- Add a short module comment to every new file, make the shared root directly import every concern, and do not modify the Callback owner suite.

## Investigation targets
**Required** (read before coding):
- `model/Temporal/System/Configuration/Tests.lean:1-182` — shared fixtures and validation failures.
- `model/Temporal/System/Configuration/Tests.lean:183-260` — deterministic resolution, context isolation, typed reads, and immutable views.
- `model/Temporal/System/Configuration/Tests.lean:261-375` — catalog fixtures, defaults, replacement, and drift checks.
- `model/Temporal/System/Callback/ConfigurationTests.lean:1-249` — cohesive owner suite that must remain byte-for-byte unchanged.
- `docs/superpowers/specs/2026-08-25-lean-test-suite-structure-design.md:153-180` — approved Configuration boundary.

## Key context
The 30-assertion legacy inventory is now 19 shared plus 11 Callback-owned assertions; this task moves only the shared 19. This is a fresh-agent, serial current-branch task: stop for human direction on baseline drift, do not commit, and do not use a worktree.

## Acceptance
- [ ] The final shared suite is recorded as 375 lines; `Configuration/Tests.lean` is import-only and directly imports `Validation`, `Resolution`, and `Catalog`, with no fixtures or concern module importing the facade.
- [ ] The evidence map accounts for all 19 shared Configuration assertions, attached comments, and semantic fixture strings exactly once; every new file has a short module comment.
- [ ] `Callback/ConfigurationTests.lean` remains byte-for-byte unchanged with its 11 assertions, preserving the full 30-assertion legacy inventory.
- [ ] `Fixtures` and every concern module pass direct Lean elaboration, then `cd model && mise exec -- lake build TemporalModelTests` passes.
- [ ] Generated API/dynamic-config modules, production configuration behavior, public APIs, dependencies, build targets, documentation, commits, and worktrees remain unchanged.

## Done summary
Split the 375-line shared Temporal configuration suite behind an import-only facade into Validation, Resolution, and Catalog concerns, with only shared checked-use/result helpers in Fixtures. The 19 shared assertions and every original declaration/string are preserved exactly once; the 249-line, 11-assertion Callback suite remains byte-for-byte unchanged at SHA-256 `f960a90afad07c34c3d549dfd76ad631f3e35d8b474db5be33d9797295801f68`.

Declaration-level evidence map:
- `Fixtures`: `errorKindOf`, `maxRequest`, `checkedMaxUse`.
- `Validation`: `unknownUseResult`, `unclassifiedUseResult`, `emptyClassificationResult`, `missingInterpretationResult`, `incompatibleInterpretationResult`, `schemaDriftResult`, `defaultDriftResult`, `missingContextResult`, `illegalContextResult`, `malformedUseResult`, `duplicateOverrideResult`, `illegalOverrideResult`, `schemaMismatchOverrideResult`, `duplicateUseResult`, `duplicateConstrainedDefaultSetting`, `valuesOfList`, `addressRuleValue`, `addressRulesValue`, and `malformedUnselectedAddressOverrideResult`; four assertions cover checked-use failures, override failures, duplicate constrained defaults, and malformed unselected address interpretation.
- `Resolution`: `sameKeyView`, `sameKeyTypedReads`, `sameKeyViewsEqual`, `sameKeyTypedReadsMatch`, `originatingUseReadResult`, `immutableViewReads`, `immutableViewReadsMatch`, `representativeView`, and `representativeMetadataComplete`; five assertions cover deterministic ordering, typed reads, originating-context isolation, immutable views, and mixed Callback/Matching metadata.
- `Catalog`: `constrainedDefaultInterleaving`, `constrainedDefaultInterleavingMatches`, `mismatchedFixtureResult`, `opaqueMetadata`, `opaqueClassification`, `opaqueInterpretation`, `checkedOpaqueUse`, `selectedOpaqueDefaultResult`, `replacedOpaqueDefaultResult`, `staleOpaqueReplacementResult`, `malformedOpaqueReplacementResult`, and `replacedOpaqueDefaultMatches`; ten assertions cover classification count, constrained defaults, fixture count/conformance/drift, and opaque-default selection/replacement/drift/schema behavior.
- The original shared source contained zero explanatory comments. Sorted quoted-string and named-declaration inventories match the pre-task source exactly; each new file adds only its required module comment.

stage: impl-review - ran [2026-08-26T03:45:47Z..2026-08-26T03:49:08Z] (model: gpt-5.6-sol)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: bc7a52ac9c54279757fac8c53d508a4cf6e0255a
- Tests: GATE_SKIPPED:build:green-receipt 2a7de6d1 - baseline reused from prior post-gate pass, GATE_SKIPPED:unittest:green-receipt 2a7de6d1 - baseline reused from prior post-gate pass, git diff --check, cd model && mise exec -- lake env lean Temporal/System/Configuration/Tests/Fixtures.lean, cd model && mise exec -- lake build Temporal.System.Configuration.Tests.Fixtures, cd model && mise exec -- lake env lean Temporal/System/Configuration/Tests/Validation.lean, cd model && mise exec -- lake env lean Temporal/System/Configuration/Tests/Resolution.lean, cd model && mise exec -- lake env lean Temporal/System/Configuration/Tests/Catalog.lean, cd model && mise exec -- lake build TemporalModelTests, Configuration assertion/comment/declaration/semantic-string and import-boundary inventory checks, Callback/ConfigurationTests.lean SHA-256 and 11-assertion byte-identity checks, (cd model && mise exec -- lake build UmpireTests TemporalModelTests), make umpire-check-regression, git diff --check 905fe8d18a69292393c090f9759c3b688aeba4c0..HEAD
- PRs:
