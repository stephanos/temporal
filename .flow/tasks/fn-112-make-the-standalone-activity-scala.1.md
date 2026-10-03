---
satisfies: [R1, R18]
---
# fn-112-make-the-standalone-activity-scala.1 Freeze the original Scala Model outputs and current source metrics

Touches: [tools/umpire/internal/golden/**, tools/umpire/model/*migration*, tools/umpire/lower/*migration*, model/lifter/test/**, .flow/tmp/fn112-1/**]

## Description
Build the original-baseline equivalence harness before changing the author surface.

**Size:** M
**Files:** tools/umpire/internal/golden and migration golden tests; model gate/lifter fixture support; .flow/tmp/fn112-1 receipts.

### Approach
- Archive all six current IRs, positive fixture JSON, manifests, reject sets and Case JSON before any fn112 migration.
- Extend the closed comparison so every later task compares with this original archive: decoded IR, finite catalogs, Definition IDs, tables, refinements, fingerprints, Query answers and ordinary-admission Case bytes. Permit only source positions, a named allow-list of lifter-internal function symbols, Query total assertions, inert names on existing fn-120 Part A result alternatives, and the exact R20 queue entity declaration/attachments. Prove choice names preserve branch count/order and all behavior and identity outputs. Derive entity-sensitive expected fingerprints from the baseline plus that exact delta; never omit fingerprint comparison.
- Add object/package-move probes for actions, monitors, assumptions, channels and realizations. Record the exact former compiler-owner plus captured-name map and prove one generic `DefinitionScope` pin per former owner reproduces every existing ID; do not widen the projection when a probe fails.
- Record current post-fn107/post-fn117 line and syntax-aware string-literal counts with category output reproducible by one command.

## Acceptance
- [ ] The equivalence command fails for a changed ID, table, result catalog/order, state key, Query answer, manifest field, exploration identity or Case byte and accepts only R1's exact position/function/total/choice-name/entity delta. Mutation controls prove unrelated choice/entity edits or fingerprint changes fail.
- [ ] Baseline receipts cover all six IRs, every positive lifter fixture, manifests/rejects and all checked-in Cases through ordinary Go admission.
- [ ] Object/package probes prove the exact DefinitionScope mechanism for every symbol-based declaration kind before feature files move, with no per-declaration legacy-ID map.
- [ ] The current 2,830-line source and exact literal count/classification are recorded with the command and source hash.
- [ ] Focused Go golden tests and the model gate pass without rewriting frozen artifacts.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
