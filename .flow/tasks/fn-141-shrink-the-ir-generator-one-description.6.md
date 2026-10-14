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
# fn-141.6 handover

Task: fn-141-shrink-the-ir-generator-one-description.6
Implementation commit: b0ebe61d66658f4efda5827a281a4370cb544f54
Task base: a20e99698df4c803b030245252fd2e19ebda062f
Scope: .plans/REFUSAL_LEDGER.md only; existing owner changes preserved.

Created the durable refusal ledger with 617 direct production sites, assigned exactly once to 61
kinds. Independent source recount gives 619 lexical occurrences minus Context's declaration and
shared sink; 606 fail calls and 11 direct LiftError constructors. Final intended outcomes are
149 macro sites (19 kinds), 379 construction sites (29 kinds), 89 source-lint sites (13 kinds),
0 Go-only and 0 proven deletions. Counts were reported to the owner.

Every kind records exact source sites, one intended enforcement home, an fn-174 owner, author
position transport and existing coverage or explicit gaps. G01 retains located construction
checks on constructed channel/state/action/binding metadata; Go continues independent admission.
Parser-only reconstruction arms have accepted/rejected spelling proof obligations and must not
become new factory shape restrictions. No unproved deletion or future enforcement is claimed.

L13 explicitly retains definition-site/source-lint validation of every declared effect, including
unused definitions and both If/all Match result branches (.4 implementation, .8 contract). Unused
and folded-away negatives remain explicit fixture gaps. Other retained contracts include disabled
targets, strict sugar argument evaluation, pre-discovery initialization lint, intermediate
derivations, excepted capability fields and all-sibling output suppression after a root failure.
Real-feature-only ScriptRejects remain sweep-held. Fn-174 preserves current total values/checks;
fn-173.3 owns their later retirement.

Verification: executed the ledger's exact fenced census through a read-only subprocess and
compared the artifact's inventory against actual current source: 617 matches, zero missing,
extra or duplicated sites; row/outcome/per-file counts reconcile. All 61 coverage rows and
68 cited individual source/fixture/doc paths resolve; all six sample files and activity_mini
directory exist. git diff --cached --check passed for the committed documentation after removing
one trailing blank line. No proof scripts or manifests were added.

stage: implementation-review - skipped(explicit owner REVIEW_MODE=none; independent review deferred to complete Scala source batch boundary)
stage: build-tests - skipped(documentation-only task selects census/path/diff verification; Scala/Go build, fixture execution and broad source-batch gates deferred to owner)
stage: verification - ran(production census, exact site reconciliation, outcome/per-file arithmetic, coverage/path existence, scoped git diff --cached --check)

This task closes inventory/design only. Fn-174 implements checks and proves diagnostic transport,
parser retirement and coverage. No test-pass, independent-review or batch-closure claim.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: b0ebe61d66658f4efda5827a281a4370cb544f54
- Tests: Exact ledger fenced production rg census executed read-only: 619 lexical occurrences; minus Context.scala:295,303 = 617 direct sites (606 fail, 11 LiftError), Read-only Node artifact/source reconciliation: 61 kinds; 617 uniquely inventoried sites; missing=0, extra=0, duplicate=0; macro=149, construction=379, source-lint=89; per-file totals matched, Read-only Node cited-path and coverage-row checks: 68 individual paths resolve; 61 coverage rows; shell existence checks for six framework samples and activity_mini, git diff --cached --check -- .plans/REFUSAL_LEDGER.md
- PRs: