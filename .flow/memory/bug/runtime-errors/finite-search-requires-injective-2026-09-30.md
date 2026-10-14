---
title: Finite search requires injective identity and bounded admission
date: "2026-09-30"
track: bug
category: runtime-errors
module: model/go/umpire
tags: [umpire, identity, search, replay]
problem_type: runtime-error
symptoms: Separator-joined composition and monitor keys aliased distinct histories; search could erase an already found witness after exhausting work.
root_cause: Composite key encodings and late budget checks were treated as proof bookkeeping.
resolution_type: fix
related_to: [bug/runtime-errors/field-alterations-must-range-over-2026-09-28, bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/replay-rejection-lost-to-2026-09-27, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05, bug/runtime-errors/workflow-replay-must-complete-2026-09-05]
---

## Problem
The fn107 foundation review found distinct composition starts and monitor histories sharing separator-joined keys. Replay also accepted foreign Definition IDs, and late work-limit handling could erase a counterexample found within the bound.

## Solution
The generic checker detects composition key collisions, length-prefixes observer identity, validates witness Definition IDs, and checks the state ceiling before enqueue. It retains an in-bound counterexample even when subsequent work reaches the limit. Same-named assumptions merge their fair classes.

## Prevention
Exercise separator-bearing names, embedded NULs, foreign IDs, and adjacent state ceilings with meaningful mutation controls. Preserve original table answers and distinguish a bounded counterexample from an unsearched suffix.
