---
satisfies: [R2, R6]
---
# fn-24-lean-native-verification-receipts-and.2 Map finite native planning into honest receipts

## Description
Implement `Umpire.Formal.Native` over the existing checked Query and dependent `IncrementalPlannerKernel query.target`. Admit only single-Property verify/counterexample forms with exhaustive policy and exact finite completeness, call the existing planner internally with that exact Query/kernel pair, derive request/evidence/toolchain fields, validate returned completeness/bounds/evidence digests, and map every outcome through the exact table into the private `NativeCheckResult`. A found violation becomes an opaque `NativeCounterexample` retaining its candidate, lineage, and diagnostics for task `.3`, not a terminal receipt. Provide fixture builders only under tests and preserve existing comments.

**Size:** M
**Files:** `model/Umpire/Formal/Native.lean`, `model/Umpire/Formal/Tests/NativeOutcomes.lean`, `model/Umpire/Formal/Tests/NativeAdmission.lean`
**Touches:** [model/Umpire/Formal/Native.lean, model/Umpire/Formal/Tests/NativeOutcomes.lean, model/Umpire/Formal/Tests/NativeAdmission.lean]

## Acceptance
Exact matrices cover verified, complete no-counterexample, opaque violating context, budget exhausted, unsatisfiable, invalid, and unexpected found reasons, plus missing completeness, query form, policy, Property count, and dependent-kernel construction. Compile-time/API tests prove no free `PlannerRun`, target, or kernel lineage can be crossed. Only the two complete outcomes establish within bounds; budget stays unknown, invalid/unsatisfiable stay invalid, violating selection remains an opaque context, and no path emits kernel trust or reads `ExperimentSpec`.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
