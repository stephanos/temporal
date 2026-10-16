---
satisfies: [R7]
---
# fn-173-reachable-exploration-drop-full-tables.13 Native Bank and TwoPhaseCommit maps/sets with retained sample proof inputs

Touches: [model/framework/samples/**, model/framework/README.md, model/framework/migrate/README.md, model/README.md]

## Description
Implement the first source-tail checkpoint under the parent's Native source-tail contract. Re-anchor after committed fn-153.7; no held engine/lifter dependency is an admission prerequisite.

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
Blocked:
Native source-tail admission hold. Release only after fn-153.7 native typed illustrations is committed on local umpire, following fn-123.9/.10. Cross-spec task edges are unsupported; do not wait for whole IR-bearing spec closure. Then execute .13 -> .14 -> .15 -> .16; original .2-.9 engine/IR/Go/graph/backend holds remain unchanged and Temporal stays excluded.
## Evidence
- Commits:
- Tests:
- PRs:
