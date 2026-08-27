---
satisfies: [R1, R3, R4, R6, R7]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.7 Synchronize documentation, downstream plans, and the legacy-name gate

## Description
Close R1, R3, R4, R6, and R7 after both code branches converge. Update active documentation and downstream flow plans, capitalize defined Ubiquitous Language nouns, and add a scoped gate that prevents the retired public vocabulary from returning.

**Size:** M
**Files:** model architecture/readmes, active `.plans/UMPIRE4_*` documents, affected open `.flow/specs`, existing Make validation surface
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_*.md, .flow/specs/fn-*.md, .flow/specs/fn-*.json, Makefile]

### Approach
- Update authoring, architecture, artifact, Observation, example, and tool documentation to the final code names and human descriptions.
- Capitalize defined Ubiquitous Language nouns such as Capability, Limit, Model Trace, Evidence Link, Generated View, and Claim Assessment when used as those concepts.
- Update the Umpire4 order so this hard cut sits after fn-31/fn-4 and before fn-32/fn-18 consumers.
- Reconcile downstream plans identified by spec-scout, especially fn-18, fn-32, and fn-20: remove v1 compatibility, Refinement, Conformance, Projection, Qualification, semantic identity/digest, bound, and omission contracts when they denote replaced concepts. Use flowctl for all `.flow` plan/dependency state changes.
- Make fn-32 depend on this spec so fn-18, fn-5, and their transitive consumers cannot start against the old vocabulary.
- Add a repository-local validation target scoped to live Umpire/Temporal source, current generated views, active Umpire4 docs, and open downstream specs. Exclude Umpire3, completed historical specs, and memory/history records.
- Preserve existing comments and avoid renaming unrelated engineering uses of the same English words.

### Investigation targets
**Required** (read before coding):
- `model/README.md` — model-author entry point.
- `model/ARCHITECTURE.md` — cross-layer ownership guide.
- `model/Umpire/ARCHITECTURE.md` — reusable module architecture.
- `.plans/UMPIRE4_SPEC.md` — approved Ubiquitous Language source.
- `.plans/UMPIRE4_ORDER.md` — vertical-slice priority.
- `.flow/specs/fn-18-versioned-umpire-artifact-boundary.md` — v1 premise to supersede.
- `.flow/specs/fn-32-add-umpire-refinement-and-the-first.md` — Implementation Link vocabulary consumer.

### Key context
The legacy-name gate targets public model concepts and exact retired wire/API tokens, not every occurrence of ordinary words such as projection or refinement in unrelated technical prose. Active `.flow` changes must go through the bundled flowctl rather than direct file editing.

## Acceptance
- [ ] Current model/architecture docs and Umpire4 plans describe the final code and wire vocabulary consistently.
- [ ] Defined Ubiquitous Language nouns are capitalized when used as domain concepts.
- [ ] Fn-18 starts from v2 with no v1 reader/migration promise; fn-32 uses Implementation Link; fn-20 uses Run Evaluation; affected downstream plans use Generated View and Claim Assessment consistently.
- [ ] Flow dependencies prevent downstream vocabulary consumers from becoming ready before this spec.
- [ ] The scoped legacy-name gate passes and has negative fixtures or command-level proof that representative retired API/wire names fail it.
- [ ] Full Lean, Go, regression, fixture, and generated-view verification commands pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
