---
title: Replay rejection lost to finalizePlanning's unsatisfiable-Scenario priority
date: "2026-09-27"
track: bug
category: runtime-errors
module: model/Umpire/Search.lean
tags: [umpire, lean, search, fn-88]
problem_type: runtime-error
symptoms: "Faulty-backend witness on an unsatisfiable Scenario finalized to unsatisfiable, not invalid"
root_cause: "finalizePlanning checks isUnsatisfiable before the termination, overriding an invalid termination"
resolution_type: fix
related_to: [bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05, bug/runtime-errors/workflow-replay-must-complete-2026-09-05]
---

## Problem
fn-88.6 added kernel replay in `finalizeBackendResult`: a rejected witness became a `BackendResult.invalid` termination. For a Query whose Scenario is statically unsatisfiable, the private `finalizePlanning` checks `query.behavior.isUnsatisfiable` before the termination, so the result finalized to `unsatisfiable` and the `unreplayableWitness` error and diagnostic were lost.

## What Didn't Work
Expressing the rejection only as a termination (`.invalid error`) and relying on `finalizePlanning` to surface it.

## Solution
`finalizeBackendResult` keeps the replay rejection (`BackendResult.replayRejection`) and makes it the outcome (`| some error, _ => .invalid error`) after `finish`, without touching `finalizePlanning`'s gates. `model/Umpire/Search/Tests/Replay.lean` pins the unsatisfiable-Scenario case.

## Prevention
When a new gate maps to an existing termination, check every precedence rule the finalizer applies before the termination (unsatisfiable, completeness gates) and add a negative control for each.
