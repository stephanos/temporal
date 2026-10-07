---
satisfies: [R2]
---
# fn-123-declare-faults-as-the-environments.2 Durability classification and the crashes binding in the DSL and lifter (fixtures only)

## Description
The durability classification and the `crashes(…)` binding in the DSL, the lifter and the IR (R2), proved on lifter fixtures. It is split from the Go derivation (task 3) so the Scala surface and its refusals land and are reviewed on their own. No Model changes here.

**Size:** M
**Files:** `model/umpire/Machine.scala` (durability section, `crashes(…)`, the Scala crash row), `model/umpire/Syntax.scala` (if the binding sits beside `Rules`), `model/irgen/Declarations.scala` (lift classification and binding), `proto/internal/temporal/server/api/umpire/v1/ir.proto` + regenerated Go, new fixtures under `model/irgen/testdata/lifts/` and refusal fixtures, `model/irgen/test/Fixtures.test.scala`, `model/umpire/*.test.scala`
**Touches:** [model/umpire/**, model/irgen/**, proto/internal/temporal/server/api/umpire/v1/**, api/umpire/**]
**Order:** After task 1. Re-read the targets first; they were verified before the DSL batch.

### Approach
- DSL: the three forms of spec Part B (`durable`, `ephemeral(resetTo)`, `inMemory(…)(a -> d, …)`) in a durability declaration, and a `crashes(fault, facts…)` binding whose outcome is the machine's internal outcome (the `Ok` given, `TaskQueue.scala:53`). Follow how `object rules extends Rules(_.field)` takes a field selector (`Syntax.scala:266`).
- Plain Scala: by default the binding also gives the Scala rule book a crash row computed from the same classification, so munit tests that step a crash keep working (spec Open Questions, "Plain-Scala crash row"). The lifter emits no function for that row.
- Derivations: `rebind`, `extend` and `restrict` (`Machine.scala:315-360`) must carry the classification through `Built`, which records nothing about durability today (`Machine.scala:378-394`). Decide and test each case the gap analysis raised: `restrict` that drops the crash (the classification is then refused per R2), `extend` that adds a crash (the classification comes with the derivation), and `rebind` over a derived crash (classification still required, binding recorded as authored over derived, consumed by task 4).
- IR: a durability record on `Machine` (next free number after `assumes = 15`, `ir.proto:354-377`) and a derived-crash marker on the crash binding. Default-empty, so machines without a crash keep their bytes.
- Lifter refusals, each at its line and naming the field: unclassified field, field classified twice, in-memory value whose fallback is in-memory, value listed twice, reset value outside the field's domain, classification on a machine with no crash, product field classified both whole and by inner path.

### Investigation targets
**Required** (read before coding):
- `model/umpire/Machine.scala:90-125, 315-394` - `Rebinding`, `StepBinding`, derivations, `Built`
- `model/umpire/Syntax.scala:17, 266-330` - `enter`, `Rules`, `on`
- `model/irgen/Declarations.scala:700-830, 900-910` - step-binding lifting and assumption checks
- `proto/internal/temporal/server/api/umpire/v1/ir.proto:354-387` - `Machine`, `StepBinding`

**Optional** (reference as needed):
- `model/temporal/shared/taskqueue/system/System.scala:110-120` - `crashDetail`, the target the fixture mirrors
- `model/umpire/Rules.test.scala:94` - munit pattern for rule books

## Acceptance
- [ ] A fixture machine with all three forms lifts, and its golden shows the durability record and the derived-crash binding.
- [ ] A munit test steps the Scala crash row of that fixture and gets the classification's result.
- [ ] One refusal fixture per R2 error and per derivation case above, each naming the field and position.
- [ ] Machines without a crash lift byte-identical: the lift goldens and `model/ir` are unchanged apart from the new fixture.
- [ ] The irgen and umpire munit tests and `make lint-model` pass.


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
