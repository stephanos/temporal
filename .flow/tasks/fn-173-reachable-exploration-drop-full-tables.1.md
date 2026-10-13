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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
