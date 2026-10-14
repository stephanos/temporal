---
title: Check unbounded Lean numbers before protobuf narrowing
date: "2026-09-07"
track: bug
category: integration
module: model/Umpire/Case/Compiler.lean
tags: [umpire, lean, protobuf, numeric-bounds]
problem_type: integration
symptoms: Generated Testpilot Cases could silently wrap oversized Umpire numbers
root_cause: Unchecked toInt32 and toInt64 conversions crossed an unbounded-to-fixed-width protocol boundary
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Moving Umpire's unbounded Lean `Nat` and `Int` fields into fixed-width generated protobuf fields used unchecked `toInt64` and `toInt32` conversions. Oversized versions, enum numbers, reservations, horizons, or limits could wrap into valid but semantically different Cases.

## What Didn't Work
Direct conversions looked faithful for all existing fixtures because their values were small, but they silently changed values outside the protobuf numeric ranges.

## Solution
`model/Umpire/Case/Compiler.lean` now narrows numeric values through checked helpers that return typed `LoweringError` values. The compiler propagates these checks through nested values, paths, instructions, limits, horizons, and version construction. Boundary tests cover accepted maxima and rejected underflow or overflow for every affected family.

## Prevention
When an unbounded Lean numeric value crosses into a fixed-width generated protocol field, add explicit range checks and exercise both accepted boundaries and overflow in the producer's focused tests.
