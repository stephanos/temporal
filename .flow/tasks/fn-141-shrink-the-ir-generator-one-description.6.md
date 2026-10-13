---
satisfies: [R12]
---
# fn-141-shrink-the-ir-generator-one-description.6 Refusal ledger: one recorded outcome per refusal kind

## Description
Classify the current refusal kinds before fn-174 retires tree-based export, so no semantic rule is lost when its matcher goes. Recount production sites at task start; do not retain the historical 450-site estimate as current evidence.

**Size:** M
**Files:** `.plans/REFUSAL_LEDGER.md`, read from `model/framework/export`, `model/framework/{lint,inspect,dsl,realize}` and the Temporal plug-in/kit.
**Touches:** .plans/REFUSAL_LEDGER.md
**Scope:** current framework and shared code, six framework samples, `activity_mini` and existing fixture contracts. Real-feature-only refusals are recorded as sweep-held, never proved by modifying real features. Fn-174 replaces fn-141 Part B; the ledger is consumed by fn-174.8.

### Approach
- Group production sites by kind, retaining a site-to-kind inventory so every site is accounted for. For each kind record its fixture (or explicitly missing coverage) and one outcome: deleted because the spelling now exports; kept at compile time by the capture macro; kept at construction by a factory; kept in source lint; or left to an existing Go admission check with the enforcing source. Planned outcomes are not already verified enforcement.
- A kept refusal names the author's file and line. Note for each whether a capture point supplies it.
- Carry over every validation of a path that is folded away; memory `consolidated-extractor-dropped-a-2026-09-27` records a regression from skipping this.
- Name the fn-174 task that handles each kind (.1–.8), so each has its checklist. Retain every folded-path validation and initialization-order check. Report counts by outcome to the owner without requiring routine approval.
- Account for checks on declarations that no capture reaches: `Declarations.givesNoEmpty` checks every declared effect, both conditional branches and every match result, including unused effects. A caller-only macro path cannot silently replace that coverage; identify its retained owner and fixture explicitly.

### Investigation targets
**Required:**
- `model/framework/export/*.scala` and `model/temporal/export/*.scala`: every production `fail(` and `LiftError(`, including wrapper/helper refusals
- `model/framework/export/testdata/lifts/` refusal fixtures and `expected/rejects.txt`, plus split fixture suites beside exporter modules
- `model/framework/{dsl,realize,lint,inspect}` and `model/temporal/realize`: construction and lint enforcement
- `tools/umpire/ir` and `tools/umpire/interp`: what the Go reader validates again

### Verification
- Independently recount production refusal sites and reconcile every site to a ledger kind, with outcome totals matching the inventory.
- Check cited source/fixture paths exist and distinguish retained coverage from proposed coverage; no new bespoke proof scripts or immutable manifests.
- `git diff --check` for the documentation change. Model compilation/equivalence is not selected by this documentation-only task; broad source-batch gates remain deferred.

## Acceptance
- [ ] The ledger covers every current production refusal kind and its sites, with fixture/coverage gap, outcome and fn-174 task.
- [ ] No kind is marked deleted unless the declaration it refused is legal Scala that will export.
- [ ] The owner has seen the count per outcome.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
