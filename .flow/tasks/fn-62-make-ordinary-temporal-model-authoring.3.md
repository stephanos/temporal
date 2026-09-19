---
satisfies: [R4]
---
# fn-62-make-ordinary-temporal-model-authoring.3 Specialize Temporal identities and source contracts

## Description
Add only the Temporal-owned specialization left uncovered by fn-65. Reuse generic identities, sources, and named Limit constructors rather than rebuilding them.

**Size:** M
**Files:** `model/Temporal/Shared.lean`, `model/Temporal/Feature/Nexus/Operations/Internal.lean`, Temporal shared public tests and aggregate root as needed
**Touches:** [model/Temporal/Shared.lean, model/Temporal/SharedTests.lean, model/Temporal/Feature/Nexus/Operations/Internal.lean, model/TemporalModelTests.lean]

## Approach
- Add a narrow constructor fixing the `temporal` root and accepting explicit semantic family, kind, and suffix, returning existing `DefinitionFamily`/`DefinitionId` values.
- Reuse `Temporal.Shared.sourceLocation` and `QueryLimitSpec`; keep source locations explicit and stable for migrated declarations. Do not introduce a macro or implicit identity selection.
- Test equivalent raw/helper identities, declaration/source-order independence, and the inability to replace the owned root. Raw ID syntax remains unchanged; checked reference errors belong to languages.
- Prepare the ordinary Nexus shared identity inputs without migrating the three operation bodies (owned by `.4`).

## Investigation targets
**Required:**
- `model/Temporal/Shared.lean:7` — existing ID and source helpers.
- `model/Umpire/Target/Authoring.lean:7` — generic family implementation.
- `model/Umpire/Query/Authoring.lean:9` — existing `QueryLimitSpec`.
- `model/Temporal/Feature/Nexus/Operations/Internal.lean` — existing shared operation identity/source inputs.

## Acceptance
- [ ] Helper-generated IDs equal existing raw IDs and cannot change the fixed root; suffixes/kinds remain explicit.
- [ ] Source-location and ordering tests preserve identity/fingerprint meaning; malformed raw IDs remain rejected by existing syntax checking.
- [ ] Compiled specimens exercise duplicate/crossed-reference and invalid/zero/wrong-unit Limits through existing language checkers, preserving IDs and source diagnostics.
- [ ] Public Temporal tests are wired into `TemporalModelTests`; focused new root and `Temporal.Feature.Nexus.OperationsTests` build, with applicable lint checks.
- [ ] Structural cost inventory shows only explicit ID string construction and source/Limit assembly, with no registry or repeated declaration scan; quantify added work for 1×/10× independent declarations separately from unchanged checker work.

## Done summary
Added the Temporal-owned `definitionFamily` constructor with a fixed `temporal` root, prepared ordinary Nexus family and named Query Limit inputs, and preserved the existing operations source and query semantics. Public compiled tests cover raw/helper equivalence, root ownership, source/order-independent fingerprints, typed malformed/duplicate/crossed-reference and Limit diagnostics, trust, and the structural 1×/10× cost inventory.

Verification: the new Shared root, existing Operations root, and Temporal aggregate build passed; `make lint-model` passed 258 jobs. Go lint retained the inherited exit 2 with exactly 1,316 sorted diagnostic headers, byte-identical to the established baseline at SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`. The official staged-overlay review returned SHIP with zero findings, and reviewed source blobs equal the final staged source blobs.

No commit was created under the user's standing commit policy; HEAD remains `7774fdc7ac751ac959816c9829516ce54af57194` and all cumulative changes remain staged.

stage: impl-review - ran [2026-09-06T02:59:34Z..2026-09-06T03:02:25.394573Z] (SHIP; actual model codex:gpt-5.6-sol:medium; receipt /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.3.json)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests: baseline: cd model && mise exec -- lake build Temporal.SharedTests Temporal.Feature.Nexus.OperationsTests (expected red: task-required Temporal.SharedTests.lean absent; existing OperationsTests completed; /tmp/fn62-task3-baseline.log), baseline: cd model && mise exec -- lake build Temporal.Feature.Nexus.OperationsTests (green; 46 jobs; /tmp/fn62-task3-baseline-operations.log), TDD red: cd model && mise exec -- lake build Temporal.SharedTests (expected failure: Temporal.Shared.definitionFamily, Operations.Internal.family, and Operations.Internal.queryLimitSpec absent; /tmp/fn62-task3-red-sharedtests.log), cd model && mise exec -- lake build Temporal.SharedTests Temporal.Feature.Nexus.OperationsTests TemporalModelTests (green; 138 jobs; /tmp/fn62-task3-final-focused.log), cd model && mise exec -- lake build Temporal.SharedTests (green; trust audit: definitionFamily/Internal.family use only propext; queryLimitSpec has no axioms; /tmp/fn62-task3-final-trust.log), make lint-model (green; 258 jobs; /tmp/fn62-task3-final-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited exit 2; exactly 1316 sorted diagnostic headers; normalized SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077; byte-identical baseline; /tmp/fn62-task3-final-lint-code.log), git diff --cached --check -- task-owned paths (green), impl-review codex:gpt-5.6-sol:medium (SHIP; zero findings; /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.3.json), reviewed source blob equality (review tree 6b5728d31e8c7b1995000994a887d4231ba7d0ba equals final staged owned blobs)
- PRs: