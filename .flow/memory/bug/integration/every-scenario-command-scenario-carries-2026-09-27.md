---
title: Every scenario-command Scenario carries ordering; pinned schedules decide it
date: "2026-09-27"
track: bug
category: integration
module: model/Umpire/Search/Product/Scenario.lean
tags: [umpire, lean, scenario, fn-88]
problem_type: integration
symptoms: Spec-literal Unsupported for ordering would exclude all command Scenarios from veil
root_cause: Scenario.exactly emits an adjacent-chain ordering for every declared Scenario
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/dual-purpose-coverage-maps-need-per-2026-09-09, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/homogeneous-owner-index-blocked-mixed-2026-09-09, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
The fn-88 spec says Scenario `ordering` and `adjacencies` lower to `Unsupported` in the progress automaton. Taken literally, that routes every `scenario`-command Scenario off the state-space search, because `Scenario.exactly` (used by the command) always emits an adjacent-chain `ordering` for its occurrences. R9 (Caller and Pair on veil) would then be unreachable.

## What Didn't Work
The review first flagged the pinned-schedule exception as spec drift, assuming checked-in Scenarios carry no `ordering`.

## Solution
Under a pinned schedule (`actionsExactly`/`traceExactly`) every action-sequence constraint is one Boolean, computed once at lowering by `admits` on the pinned schedule (`model/Umpire/Search/Product/Scenario.lean`, `pinnedSchedule`). Only free-schedule `ordering`/`adjacencies` are `Unsupported`. The rationale is in the module header and was carried into fn-88.9.

## Prevention
Before treating a Scenario field as "rare", grep `Scenario.withSteps`/`Scenario.exactly`: the command fills `requiredOccurrences`, `occurrenceBounds`, `ordering` and `actionsExactly` for every declared Scenario.
