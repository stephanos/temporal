---
satisfies: [R1]
---
# fn-173-reachable-exploration-drop-full-tables.1 Governance (GOV-02): SPEC finite-types wording, Behavior Fingerprint as IR structural hash, VISION backends-may-decide, module map's orchestration module

## Description
Amend the shared architecture documents for the owner's accepted reachable-exploration decisions D1–D8, before the implementation phases that rely on them. The approval provenance is in this spec's Conversation Evidence and `model/design/reachable-exploration.md` §10.1; this task records those decisions under GOV-02 rather than asking the owner to decide them again. Do not approve unrelated pending amendments or claim the redesign is implemented.

**Touches:** .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_VISION.md, .plans/UMPIRE_MODULES.md

### Approach
- Reconcile all normative definitions affected by reachable state: Model IR, Model, Machine, Table and Behavior Fingerprint. State values may use native collections/checked integers while action inputs remain finite. Tables are reached graphs, closed or explicitly accepted as open under bounds. Preserve source authority, independent inconclusive statuses, versioning and stable rule IDs.
- Define Behavior Fingerprints as canonical IR structural hashes of behavior-affecting declarations: names/source positions/order erased, own machine starts included, Scenario starts excluded from machine identity and included in their own identity. Cite the design's dependency hashing and limitations; meaning-preserving expression rewrites may change the fingerprint. Do not present it as an extensional behavior-equivalence proof.
- Replace VISION's restriction that external checkers never decide with selected backend decision and confirmation, coverage admission, Go replay of backend traces and fail-closed disagreement/incomparability. Preserve Model-only semantic authority; backend selection is not license to reinterpret missing exports.
- Add the planned `tools/umpire/answer` orchestration owner above `check` and `export`, with a narrow public answer/confirmation contract and permitted imports. Keep the checker and exporter dependency directions explicit and avoid claiming the module exists before fn-173.8.
- Record the targeted approval provenance and implementation ownership (.2/.3 for reachable/static checking, .8 for backend orchestration) in place. Existing eager behavior remains documented as current where appropriate until those tasks land.

### Verification
- Read the complete shared SPEC, VISION high-level architecture and module map, plus design §3, §4.13, §5.5, §7.3–§7.5 and §10.1. Reconcile every affected normative definition and cross-reference.
- `git diff --check`; verify existing rule IDs are neither renumbered nor repurposed, retained requirements remain intact and planned behavior is distinguished from implementation claims.
- Documentation-only task: no Scala/Go gate or regeneration is selected. Independent source-batch review remains deferred and is explicitly recorded.

## Acceptance
- [ ] Every R1 architecture amendment reflects the previously approved decisions with explicit GOV-02 provenance, without unrelated approval changes.
- [ ] Finite inputs, reachable state, closed/open scope and canonical structural fingerprint semantics agree across the shared documents.
- [ ] Selected backends may decide only with coverage admission, Go trace replay and fail-closed agreement; the planned orchestration owner has clear module boundaries.
- [ ] Documentation verification passes; implementation phases and deferred gates receive no completion credit from this governance task.

## Done summary
Implemented fn-173.1 / R1 in c41633bf540af96ce181059310f63c095842b0b2.

The SPEC, VISION and module map record the owner's D1–D8 approval of 2026-10-13 under GOV-02, linked to fn-173 Conversation Evidence and reachable-exploration design §10.1. They reconcile finite input domains with native partial collections and checked int64 state, reached Tables only, per-machine/default reach bounds, closed/open root regions, per-Query acceptance and independent inconclusive statuses. Existing rule IDs, source authority, artifact versioning and unrelated pending approval text are retained.

Behavior Fingerprints are canonical IR structural dependency/Merkle hashes, independent of graph exploration: machine own starts are included and Scenario starts belong to Scenario/Query identity. Incidental function/local names and source positions/order are erased; semantic type/case names and meaningful sequence order remain. Equivalent expression rewrites may change the hash, so it is no extensional-equivalence proof.

Selected coverage-admitted backends may decide/confirm Queries, with Go replay of every returned trace and fail-closed disagreement/incomparability. Planned tools/umpire/answer sits above check/export with a narrow receipt contract and explicit one-way imports, no Testpilot dependency. Existing eager/catalog/totals/equivalence and backend-export restrictions are labelled current/historical until their fn-173 implementation tasks land; no implementation acceptance is credited by this documentation task.

Verification: committed-range git diff --check passed. All existing SPEC rule ID/title headings compare byte-for-byte against task base 4363a3f50979dcebd2490d0104d1eb6993fc4852. Approval links, affected definitions, canonicalization limits, scope/limit distinction, staged paths and module directions were inspected.

stage: implementation-review - skipped(none: owner deferred independent review to the complete Scala source-batch boundary)
stage: verification - passed(documentation diff, rule-heading equality and cross-reference/approval inspection)
stage: broad-gates - skipped(documentation-only task: Scala/Go gates and regeneration not selected; later implementation tasks retain them)
stage: plan-sync - skipped(disabled: planSync.enabled=false)
Tracker sync: n/a (bridge inactive)

Preserved all pre-existing owner changes; task implementation changed only its three declared documentation paths. Model requested by dispatch: gpt-6.1-sol at medium; actual host model was not separately evidenced. No worker.md exists in the installed skill; used its documented single-worker re-anchor, implement, verify, scoped commit and done/verified-status lifecycle. Scoped staging follows the owner's preservation instruction.
## Evidence
- Commits: c41633bf540af96ce181059310f63c095842b0b2
- Tests: git diff --check 4363a3f509..c41633bf54 -- .plans/UMPIRE4_SPEC.md .plans/UMPIRE4_VISION.md .plans/UMPIRE_MODULES.md, diff -u <(git show 4363a3f509:.plans/UMPIRE4_SPEC.md | rg '^\- \*\*[A-Z]+-[0-9]+') <(git show c41633bf54:.plans/UMPIRE4_SPEC.md | rg '^\- \*\*[A-Z]+-[0-9]+'), Manual cross-reference, targeted approval provenance, normative definition, meaningful order and one-way module-boundary inspection
- PRs: