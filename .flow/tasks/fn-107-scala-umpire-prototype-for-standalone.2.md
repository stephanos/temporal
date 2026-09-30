---
satisfies: [R2, R3, R9]
---
# fn-107-scala-umpire-prototype-for-standalone.2 Extend finite model IR and TASTy lifting

Touches: [proto/internal/temporal/server/api/modelir/v1/**, api/modelir/v1/**, model/scalav2/lifter/**, model/scalav2/SEMANTICS.md, model/scalav2/gen.sh, model/scalav2/run.sh, model/scala/umpire/**]

## Description
Extend model admission and lifting for the reviewed bounded contracts, passive monitors, composition, refinement interfaces, and explicit support gaps. Scenario execution lowering stays in its own task.

**Size:** M
**Files:** modelir ir.proto and generated outputs; Lift.scala; native framework declarations only where the reviewed declaration inventory requires extension; SEMANTICS.md; lifter diagnostic fixtures; existing generation/lift gates where required.

### Approach
- Inventory source constructs needed by both specimens before extending schema. Preserve stable definition identity independently of source locations. The task-1 handoff identifies Action/Domain/Refine and small new framework declaration types as possible seams; the framework Touches includes them so no parallel declaration layer is needed.
- Express first-class passive monitor state, typed channels, visible-result projection, assumptions, and explicit holes in the portable finite subset. Retain native functions and guarded transitions.
- Preserve disabled actions and existing Nexus artifact semantics. Add reference/type/version/bounds rejection fixtures through ordinary lifting/admission.
- Regenerate Go and JVM protocol classes with existing gates. Use existing checked Nexus IR as the behavior pin for unaffected declarations.

### Investigation targets
**Required:** proto/internal/temporal/server/api/modelir/v1/ir.proto:19; model/scalav2/lifter/Lift.scala:392; model/scalav2/gen.sh; model/scalav2/goir/diagnostics_test.go; model/scala/umpire/Claims.scala.
**Optional:** model/scalav2/ir/nexus-caller.json; model/scalav2/run.sh.

### Quick commands
`make protoc`; `make umpire-gen-scala`; `make umpire-check-scala`; `make lint-scala`.

## Acceptance
- [ ] Both reviewed model/monitor surfaces lift without handwritten IR construction.
- [ ] Valid union/presence and bounded channel cases round-trip; rejection fixtures identify source declarations.
- [ ] Existing Nexus tables, identities, and supported fingerprints retain their declared comparison behavior.
- [ ] Required protocol generation and focused Scala gates pass.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
