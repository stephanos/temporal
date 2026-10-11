---
satisfies: [R1, R2, R3]
---
# fn-149-safety-and-liveness-groups-for-object.2 Enforce safety and liveness declaration placement

## Description
Enforce safety and liveness declaration placement. Advances R1, R2, R3 of the parent spec.

**Size:** M
**Files:** `model/irgen/Structure.scala`, `model/irgen/Order.scala`, `model/irgen/Claims.scala`, `model/irgen/test/Fixtures.test.scala`, `model/irgen/test/PropertyGroups.test.scala`, `model/irgen/testdata/propertyGroups/**`
**Touches:** [model/irgen/Structure.scala, model/irgen/Order.scala, model/irgen/Claims.scala, model/irgen/test/Fixtures.test.scala, model/irgen/test/PropertyGroups.test.scala, model/irgen/testdata/propertyGroups/**]

### Approach
- Enforce placement from resolved declaration kind: Property and safety monitor references in safety, Progress in liveness. Resolve aliases and factory/bundle outputs before checking; attribute the misplaced authored binding to its source.
- Check every resolved claim-bearing field of a shared bundle, not only its first classified field. Include mixed-kind and aliased/factory bundle negatives at the authored binding, with an independent expected population in the dedicated PropertyGroupsFixtures suite.
- Include supported Property subtypes (for example `never` and `stays`/`unless`) in resolved-kind admission. Discovery task .1 exposed exact-type matching as a silent unqueried-claim omission; declaration placement must likewise use the existing resolved/base-kind machinery rather than missing or reclassifying these claims. Pin direct, aliased and bundle subtype bindings in both valid and crossed groups, without treating helper definitions as authored claims.
- Reuse the actual section/order lint machinery left by fn-156, not a hypothetical fn-141 successor or a second semantic classifier. Accept omitted empty groups and allow helper definitions without treating them as claims.
- Retain existing progress bound/assumption admission and compiler type checks. Add source-attributed negative specimens for crossed kinds, aliases that conceal a crossed kind, bad bounds and incompatible owners.
- Prepare the final flat-layout refusal, activated with task .6's migration so intermediate tasks retain a usable tree.

### Investigation targets
**Required:**
- `model/irgen/Structure.scala:655` - section ownership.
- `model/irgen/Order.scala:325` - current unified Claim kind and nested-section refusal at :694.
- `model/irgen/Claims.scala:444` - progress bound admission.
- `model/irgen/test/Fixtures.test.scala` - refusal assertions.

### Key context
Selected Batch 2 entry follows the committed fn-140.10 witness source seal. Re-anchor to the actual fn-155 mapping, fn-156 enforcement and fn-140 vocabulary; future APIs remain unknown here. Fn-141 comes later. Root owns activation/readiness and source gates; .7's committed Scala seal releases fn-164; its expectation/colon seal and a separate pre-fault step comparison precede fn-123.1. Legacy .4/.5 retain deferred consumer/full closure. Shared heavy work uses the real `/tmp/umpire-heavy-gates.lock`.

Owner-mode source scope: the dedicated PropertyGroupsFixtures suite must execute a nonzero independently expected population. Native initialization/registration and full scratch-lift metadata are separate proofs, including a non-form owner with unqueried Progress. Go Query/progress outcomes, finite-bound counterexamples/incomplete checks and Cases/assessments remain required in deferred .4/.5. Only Scala source work runs here; managed artifacts/full gates/review/live stay deferred.

### Quick commands
```bash
mise exec -- scala-cli test model/irgen --test-only '*PropertyGroupsFixtures*'
```

### Conductor activation — attempt 1 (2026-10-11)

The committed discovery source/closure receipts b751748caef54e2ec8b7ffa666098a4443015658 / 1c6ee11392ffbb315d03ee04eba084477eca6294 are integrated in the local umpire history; root integration tracking is 77cb07ebb7. Root freshly authenticated all 792 final-seal files (manifest 630b1ef1a7c73993886dc9d6f608d26bbe7c4365b6e4d05ca3d07360278417b6), all 477 sealed source/config files against the committed checkout, six complete fixture IRs, and every native step-table compressed/decoded payload. Supplemental root native audit index remains 0d45f2096dcd804d197e35da02c227d4887b12d7e420f9f9c83799786ce63a4b. These immutable .1 proofs remain read-only.

Task claimed 13:40:04.424395 UTC. A fresh-context single implementation worker owns w-1d; attempt 1 begins 13:41:00 UTC and expires 15:41:00 UTC (120 minutes), with at most two consecutive non-done returns and no third respawn. No inherited green baseline is handed off. Re-run the exact nonzero PropertyGroupsFixtures Quick baseline; use test-first crossed-kind/refusal controls and actual authored coordinates for direct, alias, every bundle field and supported Property subtype bindings.

Only declared Scala source Touches plus worker lifecycle writes are authorized. Heavy commands serialize on /tmp/umpire-heavy-gates.lock. Root owns MILESTONES and admission. Go/report checkpoint .3, consumer .4/.5, migration .6, final .7 source seal, managed regeneration, full gates, independent review, live runs and publication stay held. Review mode none records a held skip, not SHIP. Existing flat declarations remain temporarily accepted until .6. Preserve the main user-owned SEMANTICS draft and protected Go bytes.
## Acceptance
- [ ] Valid, empty and omitted groups pass; direct and aliased kind mismatches fail at their source with the expected group.
- [ ] Positive, missing and non-positive bound cases retain the correct compiler/lifter disposition, with checker-result obligations handed to deferred .4/.5.
- [ ] Helper/bundle and shared-capability fixtures neither bypass placement checks nor duplicate registrations.

## Done summary
Model authors now get source-attributed safety/liveness placement refusals for resolved direct claims, aliases, factories, every bundle field, supported Property subtypes, and monitor references. The existing Order admission classifies all fields, Claims checks actual retained homes, helper claims stay unregistered, and effect-val/Step-def admission remains intact.

Flat Property/Progress declarations remain accepted until task .6. The kind split deliberately changes the existing hidden-helper diagnostic from `a Property` to `a progress claim`; its position and refusal assertion remain fixed. No IR schema or DSL meaning changed.

Baseline was green at a3417baac6b75b3d15cf0dae7e052b85eca9d5ea. The fresh nonzero PropertyGroupsFixtures Quick command exited 0 with seven tests before edits. Final Quick used `--server=false --require-tests` and exited 0 with 12 tests. Its independent inventories pin eight Properties, three Progress claims, five owners and one original monitor attachment; its exact negative matrix pins 29 authored-coordinate refusals, seven separately lifted foreign-owner roots including later-field controls, E171@11:46/E007@14:27,16:29 compiler dispositions, and zero/negative bound refusals. The original seven tests, six complete fixture IRs, fixed inventories/source maps and native 30 predicate plus eight monitor rows per variant remain supported.

Frozen red5 exited 1 with nine passes and three intended failures. Effect-red2 exited 1 with eleven passes and one intended failure, exposing both silently omitted step controls. Earlier setup failures are recorded without regression credit. Immutable proofs remain at `.flow/tmp/scala-dsl/fn149-2-baseline-proof`, `fn149-2-red-proof` and `fn149-2-effect-red-proof`.

Other terminal checks exited 0. IrFileDiscoveryTest selected 7; RulesTest 25; LayoutWitnessFixtures 1; ConventionsFixtures 9; QualifiedNames 4; Overlap 2. Source syntax, comment and block-form checks passed. Scoped formatting/check used the absolute model/.scalafmt.conf; git diff --check passed. Heavy commands ran serially under /tmp/umpire-heavy-gates.lock with foreground ownership.

Final proof is `/Users/stephan/Workspace/skunkworks/umpire/temporal/.worktrees/w-1d/.flow/tmp/scala-dsl/fn149-2-final-proof`. PROVENANCE.md and provenance.json name exact commands, terminal dispositions and original paths. The final scratch, copied authored sources, actual runtime main/test compiler outputs, exact 1,894 model class/TASTy members, current model jar/classpath/dependencies and root-authored audits are frozen there. The fixed placement audit rerun on this frozen final scratch exited 0. Final Quick log SHA256 is 7e54ca35fe4c07a3ffdd5477acd00e947de8c31c489ae099981b7a89ecb1fdcf. The 3,654-file manifest.json SHA256 is 81c2def73044e66438d8973a8299fe2f222a09ff409f8feab589816f931cc1fc; SHA256SUMS also covers the manifest and its check exited 0.

Original-admission and current-admission full-source scratch diagnostics both exited 0. All seven artifacts match byte-for-byte. Their compiler/source provenance and outputs are frozen. Root's separate witness bridge pins the historical discovery addition of 19 section roots plus the previously unqueried asyncStarts Property/function against fn140; task .6 must retain that ledger and the immutable witness seal.

Source commit is 77825c5bd2ab402e9552f6c3a5def9570be23c0d. The approved metadata-only fccab0d7d6 was merged normally as 8e302728545c64ae8241184efbf989b193716c58. Gate classify returned FULL; owner-held full/consumer gates, managed regeneration, Go/report checkpoint .3, checker-result obligations in .4/.5, flat migration .6, final source seal .7, independent review, live runs and publication received no passing credit. No live command handles remain.

stage: impl-review - skipped(config: REVIEW_MODE=none)
Tier: session (jev-unavailable(no_key)); repository AGENTS implementer routing gpt-6.1-sol medium applies.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fccab0d7d683c9ed4a071127933e08891a2bc6a7, 77825c5bd2ab402e9552f6c3a5def9570be23c0d, 8e302728545c64ae8241184efbf989b193716c58
- Tests: mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*PropertyGroupsFixtures*', mise exec -- scala-cli test model/project.scala model/framework --server=false --require-tests --test-only '*IrFileDiscoveryTest*', mise exec -- scala-cli test model/project.scala model/framework --server=false --require-tests --test-only '*RulesTest*', mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*LayoutWitnessFixtures*', mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*ConventionsFixtures*', mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*QualifiedNames*', mise exec -- scala-cli test model/irgen --server=false --require-tests --test-only '*Overlap*', mise exec -- scala-cli run --server=false model/check -- --check-syntax, mise exec -- scala-cli run --server=false model/check -- --check-comments, mise exec -- scala-cli run --server=false model/check -- --check-block-form, git diff --check
- PRs: