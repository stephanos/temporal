---
title: Finite IR admission must count work before listing catalogs
date: "2026-09-30"
track: bug
category: runtime-errors
module: model/scalav2/goir
tags: [umpire, admission, bounds, identity]
problem_type: runtime-error
symptoms: "Admission could allocate large products before Build enforced ceilings, and malformed steps or key aliases could suppress behavior."
root_cause: Validation and interpretation enumerated finite products through different helper paths.
resolution_type: fix
related_to: [bug/runtime-errors/field-alterations-must-range-over-2026-09-28, bug/runtime-errors/finite-search-requires-injective-2026-09-30, bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/replay-rejection-lost-to-2026-09-27, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05, bug/runtime-errors/workflow-replay-must-complete-2026-09-05]
---

## Problem
The fn107 IR review found validation helpers listing large catalogs and class products before applying interpreter ceilings. Saturated counts could admit overflow at MaxInt64, recursive catalogs exhausted the stack, and a singleton MaxInt64 range wrapped. Separator keys could alias distinct values or rows.

## Solution
Admission and interpretation share bounded product and class counting. LimitError preserves overflow, counts and owner context. Catalog-cycle checks precede enumeration; ranges terminate at the upper bound. Located errors reject ambiguous value, class, row and composition keys, and defensive step validation retains disabled actions separately from holes.

## Prevention
Use tight ceilings to test every admission path before allocation. Include recursive catalogs, aggregate products, separator-bearing identities, typed selectors and upper-bound ranges. A consumer must inspect hole metadata beside the concrete table and bind authored claims explicitly.
