---
satisfies: [R7]
---
# fn-156-let-the-scala-compiler-and-lint-enforce.5 Expand finite-domain checks and every machine table pin

## Description
Implement R7 using task 4's discovered universe and the completed fn-155 interpreter dump seam. Full domains and independent baseline tables replace omissions in example-only checks.

**Size:** M
**Files:** framework finite-domain tests, capability tests, activity/Nexus pin tests, generic table-pin support and immutable test data.
**Touches:** [model/framework/*test.scala, model/temporal/capabilities/*test.scala, model/temporal/**/*test.scala, model/check/**, model/testdata/**, .flow/tmp/fn156/**]

### Approach

- Inventory tests whose subject is Finite and replace example samples by their complete finite catalogs. Preserve every distinct assertion. Count domain/product sizes before expensive iteration; document actual size/time bound and chosen sample only for R7's explicit large-domain exception.
- Re-anchor fn-155's actual dump command, helper and input hashes after its committed closure. Use that existing public interpreter/script seam to build tables from lifted IR for every exported activity and Nexus machine, including compositions and derived/negative/failure designs. Make no repository Go edits and add no new runtime CLI. If the inherited seam cannot support this complete use, report the exact blocker rather than fall back to Product/System-only pins.
- Store baseline tables from task 1's pre-change IR independently of candidate tables. Compare complete catalogs, classes, starts/ends, enabled/disabled/hole pairs, ordered results/outcomes/facts and explanations through the existing canonical dump contract. Do not normalize identities, positions or omitted rows.
- Keep Scala tests for role/refinement evidence and native effects separate from interpreter pins. Cross-check the machine identity universe between exports, lifted IR and pins; seeded dropped machines, changed rows, reordered results/facts and binding-order mutants must fail. No baseline is generated from the candidate during an ordinary test.
- Promote only the necessary immutable pin data and existing dump/test support; keep large generated dumps in scratch. No invented machine count or baseline pass is recorded.

### Investigation targets

**Required:**
- `.flow/tasks/fn-155-name-the-standalone-activitys-repeated.1.md` - dump baseline and pending command ownership.
- `model/temporal/features/activity/standalone/StandaloneActivityPins.test.scala:104` - full table assertions.
- `model/temporal/features/nexus/product/NexusOutcomePins.test.scala:51` - Nexus cells.
- `model/temporal/capabilities/Closable.test.scala:43` - finite example tests.
- `model/temporal/capabilities/Pausable.test.scala:68` - finite capability domain.
- `model/framework/Inputs.test.scala:33` - finite input examples.
- `model/framework/Effects.test.scala:98` - exhaustive existing effect pattern.

### Quick commands

Run complete framework/Temporal tests with the canonical interpreter dump replay and named seeded pin mutants. Use the actual inherited dump invocation after re-anchor; production-sized generation/dumps take the shared heavy lock. Keep measured bounds and complete inventories beside evidence.

## Acceptance
- [ ] Every Finite-subject test is inventoried and iterates its whole catalog, or carries R7's explicit size/time/sample exception with measured evidence.
- [ ] All activity and Nexus machine identities have independent baseline interpreter-built table pins; missing/extra machines and row/result/fact-order mutants fail.
- [ ] Pins include all declared alternatives and composition/failure/negative-control machines; native role/effect checks remain covered independently.
- [ ] Existing dumper/test support suffices without any repository Go edits; candidate outputs never become their own baseline and any unavailable seam is reported as a blocker.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
