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
Froze the original (post-fn-107/113/117, base b5f405e68a) Scala Model outputs and built the equivalence harness every later fn-112 task, fn-120.1 and fn-114 compares against.

**Comparison command** (after `make umpire-gen-model`; ~35 s; also runs inside the gate's Go checks):
`CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower`

- Archive `tools/umpire/internal/golden/testdata/original/` (292 KB): 6 IRs, 6 positive lifter fixtures, rejects.txt, 17 Cases + manifest (gz); `model.json` (tables/catalogs, refinements, Property rows, Query receipts, Definition IDs, canonicals, fingerprints, refined reads: 36 digests), `lower.json` (explorations), `owners.json` (symbol-ID owner -> kind -> captured names).
- Checks: IR inventory closed (same 6 IRs and Cases; archived fixtures/refusals must persist, new ones may be added); each Model equal to baseline except positions/Model.source and the closed delta in `golden/original.json` (`function_name_substitutions`, `inert_fields`, `entity_attachments`, all empty now); derived outputs as streamed per-part SHA-256 (no 677 MB materialization; on mismatch both sides are re-derived and the first differing part shown); Cases from `lower.GenerateCases` and checked-in Cases byte-equal except Scala source paths; exploration candidates named by projected-model digest.
- Delta rules: inert field must exist in the IR schema and never be set in the baseline (task 11 adds `temporal.server.api.umpire.v1.Query.total`, fn-120.1 its choice-name field); renames bijective and actually made; attachments (machine `entity`, action `on`/`creates`) applied to the baseline, and expected fingerprints/Cases re-derived from baseline+attachments after proving the archive still derives as frozen. Mutation controls cover ID, table row, state key, branch order/count with inert names, limits/answers, manifest field, Case byte, fingerprint, exploration priority, unlisted/unmade rename, unlisted/wrong entity (real activity-system: attaching matchingQueue changes declaration fingerprints, which are re-derived, never waived).
- DefinitionScope probe `model/lifter/test/DefinitionScope.test.scala`: declarations/channels/realizations fixtures (all six symbol kinds incl. channel-derived actions) moved under `object Moved` and into subpackage `moved`; one pin of the former owner reproduces every expected ID exactly.
- Source metrics: `mise exec -- scala-cli run --suppress-outdated-dependency-warning model/metrics -- model/temporal/standaloneactivity` (more dirs => per-dir + combined). Start point: **2,830 lines, 462 literals**, sources sha256 d4b228af58da67724c565b1813f333e06f948b35949ce8e41a6a444cd553afd7 (path+NUL+bytes, sorted; *.test.scala excluded). Files: Claims 261/45, Model 512/56, Realization 860/94, System 1197/267. Categories: computed name 31, prose 12, evidence 39, composition key 132, Temporal API name 0, own name 112, repeated name 43, declared name 28, id 65. Full listing: .flow/tmp/fn112-1/source-metrics.txt.

Decisions: archive under golden/testdata (inside Touches) with derived outputs as digests to keep the repo small; Model.source treated as a source position; metrics entry point is its own `model/metrics` project so the gate keeps one main class (adding a mainClass directive to the gate project would force API-jar regeneration); classifier and test live in model/gate. Per conductor: fn-115's TestMigrationProjectionPreservesSemantics now streams its meaning (was OOM-killed under shared load; peak 4.4 GB).

Review: claude-fable-5-1 high, round 1 SHIP. Fixed P3 (Case diff names the differing file). Deferred P3: share one `golden.OriginalInputs` loader between model/lower tests; model/metrics is outside fmt-model/lint-model (formatted manually).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 6b8f30c912, 85dac83c5a, 4ea4c49bae
- Tests: CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -run 'OriginalBaseline' ./tools/umpire/internal/golden ./tools/umpire/model ./tools/umpire/lower (exit 0, 33 s), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- make umpire-check-model MODEL_GATE_ARGS=--skip-go-checks (exit 0, 500 s; incl. lifter DefinitionScope probe; model/ir and model/cases unchanged), CC=/usr/bin/gcc GOMEMLIMIT=4500MiB mise exec -- go test -tags test_dep -count=1 -p 2 -json ./tools/umpire/... ./common/testing/testpilot/... ./tools/canary/... (exit 1: model OOM-killed under shared load, lower 10m timeout, testpilot/internal/execution + umpire-ir-bridge build failure from another agent's uncommitted fn-119 edits; model and lower rerun green below), go test -tags test_dep -count=1 -p 1 -timeout 40m ./tools/umpire/lower (exit 0, 932 s), go test -tags test_dep -count=1 -p 1 -timeout 40m ./tools/umpire/model ./tools/umpire/internal/golden (exit 0, 225 s), mise exec -- scala-cli test model/lifter --test-only umpire.lift.DefinitionScope (exit 0), mise exec -- scala-cli test model/gate (exit 0), mise exec -- make lint-model (exit 0), GOLANGCI_LINT_FIX=false GOLANGCI_LINT_BASE_REV=origin/main mise exec -- make lint-code-fast (exit 0)
- PRs: