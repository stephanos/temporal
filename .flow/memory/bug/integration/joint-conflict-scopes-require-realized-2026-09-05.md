---
title: Joint conflict scopes require realized trigger identity
date: "2026-09-05"
track: bug
category: integration
module: model/Umpire/Planning/CaseAnalysis.lean
tags: [umpire, lean, joint-analysis, temporal]
problem_type: integration
symptoms: Distinct temporal triggers merged and absent-trigger continuations entered conflict evidence
root_cause: Continuation grouping matched only prefix and Action instead of evaluator-owned realized occurrences
resolution_type: fix
---

## Problem
Bounded joint analysis grouped temporal obligations only by modeled prefix, transition, prior state, and selected Action. Distinct observation occurrences could therefore collapse into one trigger, and a continuation with the same Action but no matching temporal trigger could be counted as admitted conflict evidence.

## What Didn't Work
Filtering continuations by prefix and Action established that the transition was reachable, but did not establish that each selected obligation's trigger occurred on that continuation.

## Solution
`model/Umpire/Property/Evaluation.lean` now records the realized trigger field, value, coordinate unit, and coordinate. `model/Umpire/Planning/CaseAnalysis.lean` groups by that identity and admits a continuation only when evaluator-owned observations for every selected expectation match the exact trigger scope. Nexus2 regressions cover two observation triggers in one transition and an alternative same-Action branch where a trigger is absent.

## Prevention
For model-relative evidence, test trigger identity as part of the evidence payload and include traces that share prefix and Action while differing in trigger occurrence. Require the admitted continuation count to exclude absent-trigger alternatives.
