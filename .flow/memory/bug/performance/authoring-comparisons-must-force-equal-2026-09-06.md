---
title: Authoring comparisons must force equal checker work
date: "2026-09-06"
track: bug
category: performance
module: model/Temporal/Feature/Nexus2/AuthoringTests.lean
tags: [umpire, lean, authoring, measurement, lsp]
problem_type: performance
symptoms: Frontend appeared slower and editor evidence appeared adequate under asymmetric work
root_cause: Timing and LSP matrices applied different operations to the compared surfaces
resolution_type: fix
---

## Problem
A constructor/frontend performance table compared constructor declarations that deferred checker evaluation with closed frontends that evaluated the checker during elaboration. The same report mixed constructor completion with frontend hover/navigation, so neither comparison performed equal work.

## What Didn't Work
Timing declaration elaboration alone treated deferred constructor admission as if it had already run. Reporting different LSP operations for each surface could not support a side-by-side editor conclusion.

## Solution
Keep declaration elaboration as a separately labeled syntax-cost observation, then force each named checker result through matched 1x/10x `#guard` admission commands. Run completion, hover, definition navigation, and error recovery against each corresponding constructor/frontend Property, Behavior, and Query declaration. Document caching and single-warm-run limits and base the default choice only on comparable evidence.

## Prevention
For interface comparisons, define an operation matrix first. Each measured row must use the same semantic work, input multiplicity, tool operation, environment, and limitations on both candidates.
