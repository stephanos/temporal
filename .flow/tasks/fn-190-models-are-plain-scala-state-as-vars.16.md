---
satisfies: [R2, R11, R12, R13, R14, R16]
---
# fn-190-models-are-plain-scala-state-as-vars.16 DSL completion: delete the record-state vocabulary from the DSL

## Description
**DSL completion phase (owner, 2026-10-17).** After fn-190.15 and fn-193.2 (cross-spec: plain compositions), delete what fn-190.4 made `private[framework]`: `RecordMachine` and its `Owner`/`Declaring` plumbing, `Rules`/`~>`/`where`/`always`/`rejects`/`enter`, `property`/`expect`/`across`, `query`/`Limits` construction, `assume`/`Assumption`, the named `leadsTo` overloads, `restrict`/`assuming`, and the internal tests that only exercised them. Rewrite DSL tests that still matter in the plain form. Export/testdata fixtures are out of scope (re-anchored in the IR/plugin phase).

**Native preservation seams (conductor, 2026-10-18).** Rewriting the remaining tests must preserve native fault-budget and durability behavior, including direct var budget selectors, budgeted native transition/crash binding, exact field ownership and nested immutable-record updates rebuilt atomically with the generated fault's own budget. Preserve selected finite/opaque validity evidence and reject malformed dormant metadata before callbacks. Preserve per-bound-effect choice-token validation without evaluating effects or guards; an unavailable declaration inventory must be explicitly unknown/refused, never invented empty or broadened to every choice of a Machine. Record any remaining separate-library metadata boundary honestly before completion.

**Construction/container split.** Delete `query`/`QueryDecl`/`QueryOn`/`QueryIn` and authorable `Limits` construction here; the broader `Query`/`ClaimQuery`/`Witness` container retirement belongs to fn-195.3. Any necessary interim bounds metadata is private and non-authorable. Rewrite substantive provenance, acquisition-laziness, crash/budget and environment tests in the plain form or as direct pure metadata tests; delete only the obsolete parser/authoring contracts and inventory the count/coverage delta.

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
