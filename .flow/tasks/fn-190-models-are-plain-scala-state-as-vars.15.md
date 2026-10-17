---
satisfies: [R6]
---
# fn-190-models-are-plain-scala-state-as-vars.15 DSL completion: native runtime moves to the product checker under fn-196.1

## Description
**Carried by fn-196.1 (owner, 2026-10-17):** the destination is the product checker module `model/framework/checker`, package `framework.checker`. Execute this move once through fn-196.1, not through a separate test-support implementation. The measurement decision in fn-196.3 still gates Go-checker retirement under fn-196.4; this move alone grants no Go/IR/Temporal integration credit.

**DSL completion phase (owner, 2026-10-17).** Move native execution (`effectOf`, `guardHolds`, input-effect adapters, finite/domain/state enumeration and budget runtime, `NativeCheck` and var snapshots) out of `model/framework/dsl`. Keep declarations and the references the plugin reads in the DSL; expose the native questions and unified result through the product checker as fn-196.1 specifies. Preserve the author's declaration surface, and make tests import the checker explicitly. The earlier test-support destination and its later deletion by an IR evaluator are superseded by fn-196; they are not acceptance requirements for this move.

**Size:** L

### Verification
- No production source in model/framework/dsl invokes an effect or guard callable (grep recorded).
- The native dsl/realize/samples suites pass with the same test count; lint RuleSuite and boundary suites pass.

## Acceptance
- [ ] Native execution lives outside framework.dsl in the product checker module, as fn-196.1 specifies
- [ ] Native suites pass with unchanged counts
- [ ] fn-196.1's source and verified committed closure carry the move; this tracking task is closed afterward, without a second implementation

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
