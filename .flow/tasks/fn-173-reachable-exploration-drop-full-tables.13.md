---
satisfies: [R7]
---
# fn-173-reachable-exploration-drop-full-tables.13 Native Bank and TwoPhaseCommit maps/sets with retained sample proof inputs

Touches: [model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Implement the first source-tail checkpoint under the parent's Native source-tail contract. Native fn-153.7 is committed (source 44e07a813211ef212f39e096e8a012fc3aa1ace8, receipt 9fb590c01efe09fb7e9abf82f4c9e1cfa91c1533). Re-anchor from the actual committed admission tip before editing; no held engine/lifter dependency is an admission prerequisite.

**Size:** M (four implementation/test files; retained text fixtures and guidance are mechanical).
**Files:** model/framework/samples/Bank.scala, TwoPhaseCommit.scala, Expect.test.scala, a focused NativeMaps.test.scala; samples/testdata/native-migration/ source texts and mapping/expectation inventory; framework README/migrate README and model/README.md sample guidance.

### Approach
- Before any sample rewrite, retain all six current sample sources byte-for-byte under samples/testdata/native-migration/ using a non-Scala text suffix. Record the actual committed pre-edit baseline, source paths and native discovery exclusions in the durable fixture guide, not an ignored report or new hash manifest. Later tasks consume this same snapshot bundle.
- Reuse Bank helpers/effects/Properties (pinned bac173188b Bank.scala:39-57,86-115,125-159). Replace fixed balances with the approved sparse nested maps and native Int. Canonicalize zero denomination entries and empty accounts; use safe absent-key reads. Preserve explicit finite send amounts with final record-field domains, identities, accepted/refused transfers, self-transfers and leaky control. No global Finite[Int].
- Preserve Bank's existing amount domain 0..3 and effect branch order, not the proposed design snippet's -1..3 domain or `amount <= 0` refusal. A zero send from a funded sender remains accepted and unchanged; a zero send from an empty sender remains refused, including a self-send. Insufficient funds are checked before the self-send shortcut; retain the authored overflow branch and distinct Leaky effect. Pin these cases independently alongside sparse-map zero cleanup.
- Reuse TwoPhaseCommit.scala:28-101,108-149; replace fixed States/Members/Messages records with exact manager maps and member/message sets. Messages remain ever-sent observations, not Inbox queues. No composition redesign from its TODO.
- Migrate shared Expect.test.scala constructors without dropping positive/negative cases. Add independent handwritten ordinary-dispatch and Property controls, retaining selectors/owners and actual inputs. Construction must not enumerate state.
- Start the enduring six-sample state-map/delta and intended-Query-answer inventory; complete both migrated samples now. Preserve all names, owners, starts, schedules, bounds, Property/monitor/progress associations and negative controls. A map is a later proof specification, not graph evidence.

### Investigation targets
Required: final framework README; model/design/reachable-exploration.md sections 8.1/8.2 and Phase 5; samples/Bank.scala, TwoPhaseCommit.scala, Expect.test.scala; dsl/NativeData.test.scala. Pinned coordinates in .flow/tmp/fn173-tail-repo-plan.md are research only and must be refreshed.

### Verification
Run focused framework-only native sample/expect suites with required nonzero counts and full native warning flags, scoped formatting and relevant lint. Do not run check/Samples lift-and-Go harness, tables, exporter, schema, Go, Temporal or regeneration. Native cases do not prove every Query answer, graph isomorphism, refinement, checked-int64 or operation liftability.

## Acceptance
- [ ] Six pre-rewrite source text fixtures and a durable baseline/map/expectation inventory survive independently of retired tooling and are excluded from Scala source discovery.
- [ ] Bank sparse maps and TwoPC maps/sets use final record/domain/expect APIs; finite inputs, source identities, protocol bounds and negative controls remain explicit. Bank retains amount domain 0..3, funded/empty zero-send distinctions and refusal-before-self-send order. Absent keys, zero cleanup, self-transfer/refusal and prepared/commit/abort controls pass.
- [ ] Independent native construction/dispatch/Property cases retain prior expectations and do not enumerate state; scoped native checks have nonzero passing counts.
- [ ] Original R7 graph/Query-answer and R5/R6 numeric/lifting proofs remain open; no full requirement or integration closure is claimed.

## Done summary
Implemented native Bank sparse nested maps and TwoPhaseCommit manager/member/ever-sent-message maps/sets. Preserved Bank amount domain0..3, refusal-before-self order, funded/empty zero distinctions and Leaky; preserved TwoPC exact manager keys, partial required lookups, sent-versus-received evidence, guards, idempotence and EagerCommit. Existing actions, Property selectors/owners, Progress/fairness, starts, Query names/forms/bounds and negative controls remain. All six pre-rewrite source texts are retained byte-for-byte at baseline 53f05f9bce8434ee069ec7717212700217358bbc, with durable guide and completed Bank/TwoPC maps plus all25 intended Query answers.

Source commit: d45444ba6064f093ed7ad1a404a2d04350f5b820. Tier: session (jev-unavailable(no_key)); judge unavailable no_key. REVIEW_MODE:none; no review verdict.

Baseline terminal rc0: SampleExpectTest6, NativeDataTest9, RulesTest26 (41). New runtime red before rewrite: NativeMapsTest rc1, 1passed/1failed because amount7 hits old UpTo2. Initial unused-import setup failure and later13 positional-record-call compile errors are recorded separately in evidence; both corrected in tests without changing DSL. Final terminal rc0: NativeMapsTest12, SampleExpectTest6, NativeDataTest9, RulesTest26, scoped lintRuleSuite29 (82). Changed Scala formatting check, git diff --check and all6 gitshow-baseline byte comparisons rc0. Logs/stages are in .flow/tmp/fn17313-done.json. Both disjoint source children terminal completed; no child compiler/Git/Flow operations.

Guide: model/framework/samples/testdata/native-migration/README.md. Map/delta/intended-answer inventory: model/framework/samples/testdata/native-migration/inventory.md. Snapshots and maps specify future proofs; no graph result or Query answer was recomputed. R7 graph/Query, checked-int64, operation lifting, refinement, interpreted progress/fairness, IR/Go/backend and original .2-.9 integration remain held. No Temporal sources read/compiled/restored, no exporter/schema/regeneration/Go/full gate run, no new DSL feature or migration harness. Formal review and broad native gates remain at batch end after .16. Only .13 closes; parent and .14-.16 remain open.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: d45444ba6064f093ed7ad1a404a2d04350f5b820
- Tests: /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeMapsTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.SampleExpectTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.NativeDataTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/dsl model/framework/realize model/framework/samples --require-tests --test-only framework.dsl.RulesTest, /usr/bin/timeout 600 mise exec -- scala-cli test --server=false model/framework/check --require-tests --test-only 'framework.check.lint.*RuleSuite', mise exec -- scala-cli fmt --check --scalafmt-conf model/.scalafmt.conf model/framework/samples/Bank.scala model/framework/samples/TwoPhaseCommit.scala model/framework/samples/Expect.test.scala model/framework/samples/NativeMaps.test.scala, git diff --check
- PRs: