---
satisfies: [R2, R11, R12, R13, R14, R16]
---
# fn-190-models-are-plain-scala-state-as-vars.16 DSL completion: delete the record-state vocabulary from the DSL

## Description
**DSL completion phase (owner, 2026-10-17).** After fn-190.15 and fn-193.2 (cross-spec: plain compositions), delete what fn-190.4 made `private[framework]`: `RecordMachine` and its `Owner`/`Declaring` plumbing, `Rules`/`~>`/`where`/`always`/`rejects`/`enter`, `property`/`expect`/`across`, `query`/`Limits` construction, `assume`/`Assumption`, the named `leadsTo` overloads, `restrict`/`assuming`, and the internal tests that only exercised them. Rewrite DSL tests that still matter in the plain form. Export/testdata fixtures are out of scope (re-anchored in the IR/plugin phase).

**Size:** L

### Verification
- grep shows none of the retired names defined in model/framework/dsl.
- Native dsl/realize/samples suites and lint RuleSuite pass.

## Acceptance
- [ ] Record-state vocabulary deleted from framework.dsl
- [ ] Native suites pass

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
