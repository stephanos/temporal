---
title: Contract work bounds must follow typed values
date: "2026-09-04"
track: bug
category: integration
module: tools/umpire/verification
tags: [umpire, admission, bounds]
problem_type: integration
symptoms: Valid scalar correlations rejected with large response limits
root_cause: Whole-response byte ceiling substituted for projected value cost
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04]
---

## Problem
Contract work accounting charged a complete allowed RPC response for fixed-width capture assignments and scalar comparisons. A valid int64 correlation was rejected whenever the Program response ceiling exceeded the Contract's hard per-event work ceiling.

## What Didn't Work
Using one response-byte ceiling for all typed values hid the difference between inspecting a projected scalar and processing an entire response.

## Solution
Bound fixed-width scalars and enums by constant encoded size. Account for variable-size operands, literal sizes, capture support references, and path fanout separately. Bind paths to the same admitted Program fanout used in work estimates. Keep static binding work independent from runtime work limits, and bind ordered conditions without synthetic expression-depth inflation.

## Prevention
Test a fixed-width correlation with a 16 MiB response ceiling, reject undersized runtime work, and retain variable-size comparison bounds. Test authored expressions exactly at the depth ceiling and path expansion below, at, and above the Program fanout ceiling.
