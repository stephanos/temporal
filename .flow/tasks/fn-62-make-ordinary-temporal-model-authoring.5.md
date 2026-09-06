---
satisfies: [R1, R6, R8]
---
# fn-62-make-ordinary-temporal-model-authoring.5 Deepen typed Observation construction and migrate Nexus

## Description
Complete the uncovered Observation authoring stage using the current inert declaration owner and migrate its established Nexus consumer after `.2`/`.3`.

**Size:** M
**Files:** `model/Umpire/Observation/Declaration.lean`, Observation tests/imports, established Nexus Observation declaration/tests
**Touches:** [model/Umpire/Observation/Declaration.lean, model/Umpire/Observation/ImportTests.lean, model/Umpire/Observation/Tests/**, model/Temporal/Feature/Nexus/Observation.lean, model/Temporal/Feature/Nexus/ObservationTests.lean]

## Approach
- Compose typed profile/rule/mapping constructors from existing `ObservationFieldSpec` leaves. Keep fields, output kinds, dispositions, providers, order, closures, and limits explicit.
- Delegate validation to `checkObservation`; checked construction continues to require proof. Add no parallel validation or interpretation language.
- Migrate the established declaration and compare exact raw/helper checked values, fingerprint, accepted facts, Evidence links, order/closure support and sources.
- Extend existing compile/evaluation matrices to cover helper-produced declarations; preserve full diagnostic precedence, not just error presence.

## Investigation targets
**Required:**
- `model/Umpire/Observation/Declaration.lean:60` — typed leaf projections.
- `model/Umpire/Observation/Language.lean:12` — checked proof seam.
- `model/Umpire/Observation/Tests/Compilation.lean:186` — declaration rejection matrix.
- `model/Temporal/Feature/Nexus/Observation.lean` — migration.
- `model/Temporal/Feature/Nexus/ObservationTests.lean:113` — exact identity and Evidence tests.

## Acceptance
- [ ] Inert constructors and migrated Nexus preserve exact checked mapping meaning, field identities, fingerprint, sources and public imports.
- [ ] Compile failures cover missing/unknown/wrong-type fields, duplicate mappings/dispositions, absent dispositions, conflicting/unresolved providers, wrong output kind, contradictory/cyclic order, missing/duplicate closures, and invalid bounds.
- [ ] Evaluation failures cover over-limit, missing/ambiguous/conflicting Evidence, profile mismatch, rejected fields, and causal/order/closure failures, with unchanged facts/links and diagnostic precedence.
- [ ] `cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests` passes; public import, trust and lint checks pass with no new issues.
- [ ] Structural cost inventory names helper callees and traversals: at most one construction pass per explicit input collection, no nested rescan, normalization, or duplicate checkObservation pass; 10× independent declarations adds at most 10× wrapper work.

## Done summary
Added inert typed Observation profile, kind, rule, and mapping constructors that project existing `ObservationFieldSpec` leaves, delegate checking to the existing checker with explicit proof evidence, and migrate the Nexus Observation declarations without changing their public identity or semantics. Expanded helper-backed compile/evaluation failure and precedence coverage, exact raw/helper compatibility and trust checks, and a named single-pass/10x structural cost audit; reviewed staged tree `02ab2b4c61bc761620eadc5bc3425b3b36f5951c` is unchanged, while an external `wip` commit advanced HEAD from `7774fdc7ac751ac959816c9829516ce54af57194` to `8f40772ebb8e56fac4d068c9990b3c1827f8a05f` during review.

Verification: focused build 72/72, public-import build 37 jobs, and lint-model 258/258 passed; inherited `lint-code` remained exactly 1316 normalized diagnostic headers with SHA-256 `aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077`, identical to the approved baseline.

stage: impl-review - ran [2026-09-06T04:23:51Z..2026-09-06T04:27:17Z] | codex:gpt-5.6-sol:medium | SHIP | receipt `/tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.5.json`
stage: plan-sync - skipped(config: planSync.enabled != true)
stage: tracker-sync - skipped(config: tracker inactive)
## Evidence
- Commits: 8f40772ebb8e56fac4d068c9990b3c1827f8a05f
- Tests: baseline: cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests (rc0, 72/72; /tmp/fn62-task5-baseline-build.log), TDD red: cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation Umpire.Observation.ImportTests (expected rc1: authoring APIs absent; /tmp/fn62-task5-tdd-red.log), TDD green: cd model && mise exec -- lake build Umpire.Observation.Tests.Compilation Umpire.Observation.ImportTests (rc0, 51/51; /tmp/fn62-task5-tdd-green.log), cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests (rc0, 72/72; /tmp/fn62-task5-final-build.log), cd model && mise exec -- lake build Umpire.Observation.ImportTests (rc0, 37 jobs; /tmp/fn62-task5-public-import.log), make lint-model (rc0, 258/258; /tmp/fn62-task5-lint-model.log), make lint-code GOLANGCI_LINT_FIX=false (inherited rc2, exactly 1316 normalized headers, SHA-256 aee7770bec1fe01dab8826427cc89e9ffa7e764fbac25ce6b68bf5f2e3c0b077, identical to approved baseline; /tmp/fn62-task5-lint-code.log), GREEN_RECEIPT:unittest:8f40772e - cd model && mise exec -- lake build Umpire.Observation.Tests Temporal.Feature.Nexus.ObservationTests, impl-review codex:gpt-5.6-sol:medium SHIP (reviewed staged tree 02ab2b4c61bc761620eadc5bc3425b3b36f5951c; /tmp/impl-review-receipt-fn-62-make-ordinary-temporal-model-authoring.5.json)
- PRs: