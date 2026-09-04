---
title: Program admission must validate producer kinds and bound repeated lookups
date: "2026-09-04"
track: bug
category: integration
module: tools/umpire/internal/execution
tags: [umpire, admission, bounds]
problem_type: integration
symptoms: Invalid Await targets accepted and repeated authorization scans escaped work accounting
root_cause: Reachability substituted for producer compatibility; per-collection limits omitted multiplicative work
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03]
---

## Problem
Program admission verified that Await referenced a local ancestor but did not check that the target produced a compatible Nexus operation handle. RPC authorization repeatedly scanned endpoint method lists, bypassing the aggregate preparation-work budget.

## What Didn't Work
Graph reachability alone did not establish opcode compatibility; bounding collection lengths independently did not bound repeated cross-product work.

## Solution
Require Await targets to be StartNexusOperation while inspecting source opcodes independently of authored node order. Compile charged per-endpoint method sets once, reject duplicates, and use constant-time authorization lookups.

## Prevention
Test each incompatible producer opcode, legal forward source order, and duplicate policy entries. Audit repeated lookup loops against the aggregate work budget rather than only individual collection limits.
