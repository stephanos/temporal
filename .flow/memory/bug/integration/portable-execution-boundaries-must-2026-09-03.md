---
title: Portable execution boundaries must preserve runtime context
date: "2026-09-03"
track: bug
category: integration
module: tools/umpire/runner/runner.go
tags: [umpire, runner, bounds]
problem_type: integration
symptoms: "runtime slots, overflow identity, and invariants crossed execution seams"
root_cause: typed runtime context was omitted or generalized at runner and result boundaries
resolution_type: fix
related_to: [bug/integration/portable-schemas-must-preserve-source-2026-09-03]
---

## Problem
Portable-plan execution initially projected runtime binding slots as ordinary roles and omitted their declarations at the runner seam. The same boundary also returned reserved overflow results without the completed run identity and converted post-dispatch invariant failures into semantic results.

## What Didn't Work
Treating every runner error after dispatch as an evaluable operational failure fabricated successful tooling status for invariant errors. Reserving only a UUID-sized placeholder also underestimated the runner's documented 512-byte identity ceiling.

## Solution
Carry cloned typed slot declarations in `runner.InputBinding`, require a resolver for declared slots, and type-check and enforce their closed preconditions before participant construction. Reserve the maximum run identity in the mandatory result envelope, replace it with the completed run identity on overflow, and keep runner invariants on the internal-error path.

## Prevention
Pair each execution-boundary addition with focused tests for satisfied/unmet/type-crossed adapter values, exact overflow run correlation, post-dispatch invariant classification, and pre-dispatch cancellation.
