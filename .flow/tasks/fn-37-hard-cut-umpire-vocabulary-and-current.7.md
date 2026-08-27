---
satisfies: [R1, R3, R4, R6, R7]
---
# fn-37-hard-cut-umpire-vocabulary-and-current.7 Synchronize documentation, downstream plans, and the legacy-name gate

## Description
Close R1, R3, R4, R6, and R7 after both code branches converge. Update active documentation and every open downstream spec/task pair, capitalize defined Ubiquitous Language nouns, and add a scoped gate that prevents retired public vocabulary from returning.

**Size:** L
**Files:** model architecture/readmes, active `.plans/UMPIRE4_*` documents, affected open `.flow/specs` and `.flow/tasks`, existing Make validation surface
**Touches:** [model/README.md, model/ARCHITECTURE.md, model/Umpire/ARCHITECTURE.md, .plans/UMPIRE4_*.md, .flow/specs/fn-*.md, .flow/specs/fn-*.json, .flow/tasks/fn-*.md, .flow/tasks/fn-*.json, Makefile]

### Approach
- Update authoring, architecture, Artifact, Observation, example, and tool documentation to the final code names and human descriptions.
- Capitalize defined Ubiquitous Language nouns such as Capability, Limit, Model Trace, Evidence Link, Generated View, and Claim Assessment when used as those concepts.
- Keep the hard-cut spec first in the remaining order while recording that it waits for fn-31/fn-4.
- Reconcile the plans and every todo task for the complete open downstream closure: fn-5, fn-18, fn-19, fn-20, fn-21, fn-22, fn-24, fn-25, fn-26, fn-27, fn-28, fn-29, fn-30, fn-32, and fn-33. Remove v1 compatibility and retired Refinement, Conformance, Projection, Qualification, Qualification Receipt/Profile, semantic identity/digest, bound, and omission contracts where they denote replaced concepts. Use Implementation Link, Run Evaluation, Generated View, Observation Evaluation, Evaluation Receipt/Profile, Behavior Fingerprint, Artifact Checksum, Limit, Known Gap, and Claim Assessment by meaning.
- Use bundled `flowctl` for every `.flow` plan, task Markdown, task JSON, and dependency mutation; do not directly edit flow state. Validate each changed spec and derive its task waves after synchronization.
- Audit the dependency graph for the complete closure above. Preserve a direct or transitive dependency path from every downstream spec to this spec; add a direct dependency through flowctl only where no such path exists, so no consumer can start against the old vocabulary.
- Add a repository-local validation target scoped to live Umpire/Temporal source, current generated views, active Umpire4 docs, and all open downstream spec/task Markdown and JSON. Exclude Umpire3, completed historical specs/tasks, and memory/history records.
- Preserve existing comments and avoid renaming unrelated engineering uses of the same English words.

### Investigation targets
**Required** (read before coding):
- `model/README.md`, `model/ARCHITECTURE.md`, and `model/Umpire/ARCHITECTURE.md` — active model documentation.
- `.plans/UMPIRE4_SPEC.md` and `.plans/UMPIRE4_ORDER.md` — approved language and prototype priority.
- `.flow/specs/{fn-5,fn-18,fn-19,fn-20,fn-21,fn-22,fn-24,fn-25,fn-26,fn-27,fn-28,fn-29,fn-30,fn-32,fn-33}-*.md` and matching `.flow/tasks/*.md` — complete open downstream reconciliation set.
- Bundled `flowctl task set-description`, `task set-acceptance`, `task set-spec`, `spec set-plan`, dependency, validation, and wave commands — required mutation path for flow state.

### Key context
Updating only spec prose is insufficient: stale task contracts could still implement v1 or retired terms. The legacy-name gate targets public model concepts and exact retired wire/API tokens, not every ordinary use of words such as projection or refinement.
## Acceptance
- [ ] Current model/architecture docs and Umpire4 plans describe the final code and wire vocabulary consistently; defined Ubiquitous Language nouns are capitalized as domain concepts.
- [ ] Every todo task as well as the plan for fn-5, fn-18, fn-19, fn-20, fn-21, fn-22, fn-24, fn-25, fn-26, fn-27, fn-28, fn-29, fn-30, fn-32, and fn-33 is reconciled through flowctl and no longer promises v1 or retired vocabulary.
- [ ] Fn-18 starts from v2; fn-32 uses Implementation Link; fn-20 uses Run Evaluation; fn-27 through fn-30 use Evaluation Receipt/Profile and Claim Assessment rather than Qualification/Conformance APIs; all affected tasks use Generated View, Observation Evaluation, Behavior Fingerprint, Artifact Checksum, Limit, and Known Gap consistently.
- [ ] A direct or transitive flow dependency prevents every spec in the complete downstream closure from starting before this spec, and all changed specs validate with coherent task waves.
- [ ] The scoped legacy-name gate covers active open spec/task Markdown and JSON as well as live source/docs, and negative fixtures or command-level tests prove representative retired API/wire names fail it.
- [ ] Full Lean, Go, regression, fixture, and Generated View verification commands pass.
## Done summary
Hard-cut active Umpire vocabulary across the open downstream closure, active model/architecture documents, and Umpire4 plans; v2 is now the sole artifact baseline and reduced downstream plans preserve only their intended current contracts. Added a scoped repository legacy-vocabulary gate with command-level negative fixtures, Make integration, and semantic cleanup of Definition/Behavior Fingerprint/Artifact Checksum, Limit/Known Gap, Observation Evaluation, Evaluation Receipt/Profile, Implementation Link, Generated View, and Claim Assessment usage.

Verification is green for the scoped scanner fixtures and legacy gate, focused Target Lean compatibility, full Lean build, pinned Go suite, Generated View drift, regression aggregate, Lean API fixture, all 15 downstream spec validations/readiness checks, and the direct/transitive dependency closure audit. Codex review fixed four P1 scope/coverage defects and returned SHIP in session `01a044aa-eeac-7b62-9416-d0e5ce9869d3`; memory capture was skipped because this checkout reports memory as not initialized.

stage: impl-review - ran [Codex NEEDS_WORK -> SHIP; 2026-08-27T19:29:08Z..2026-08-27T19:43:53Z; session 01a044aa-eeac-7b62-9416-d0e5ce9869d3]
## Evidence
- Commits: e10de777f57ff185093565f7727abf76f91550f7, 3487371bd3be99e64328dce65ed80f230c0dd618
- Tests: mise exec -- go test ./tools/umpire/vocabulary -run 'TestLegacyVocabularyGate|TestCheckPathsRejectsLegacyTokens|TestCheckPathsAllowsOrdinaryEnglishAndExcludedHistory', mise exec -- make umpire-check-legacy-vocabulary, cd model && mise exec -- lake build Umpire.Target.Tests.Compatibility, cd model && mise exec -- lake build UmpireTests TemporalModelTests TemporalExperimentalTests temporal-model-inspect, mise exec -- go test ./tools/umpire/..., mise exec -- make umpire-check-regression-views, mise exec -- make umpire-check-regression, mise exec -- make umpire-gen-lean-api-fixture, /home/agent/.codex/scripts/flowctl validate --spec <each of fn-5,fn-18,fn-19,fn-20,fn-21,fn-22,fn-24,fn-25,fn-26,fn-27,fn-28,fn-29,fn-30,fn-32,fn-33>, /home/agent/.codex/scripts/flowctl ready --spec <each of fn-5,fn-18,fn-19,fn-20,fn-21,fn-22,fn-24,fn-25,fn-26,fn-27,fn-28,fn-29,fn-30,fn-32,fn-33>
- PRs: