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
last_updated: "2026-09-05"
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

## Update 2026-09-05

## Problem
Program admission verified that Await referenced a local ancestor but did not check that the target produced a compatible Nexus operation handle. RPC authorization repeatedly scanned endpoint method lists, bypassing the aggregate preparation-work budget. Reservation-carrier topology initially repeated a linear handler scan for every potential StartNexusOperation and did not charge its indexing, source, route, or reconciliation work.

## What Didn't Work
Graph reachability alone did not establish opcode compatibility. Bounding collection lengths independently did not bound repeated cross-product work, and compiling an immutable result did not itself account for the work needed to build it.

## Solution
Require Await targets to be StartNexusOperation while inspecting source opcodes independently of authored node order. Compile charged per-endpoint method sets once, reject duplicates, and use constant-time authorization lookups. For carrier topology, build one service/operation handler index in reservation order, reject duplicate keys there, perform constant-time source lookups, and charge every topology record, handler insertion, potential source, emitted route, and final count reconciliation.

## Prevention
Test each incompatible producer opcode, legal forward source order, duplicate policy entries, and topology compilation with only one fewer work unit than it needs. Audit repeated lookup loops and derived collection construction against the aggregate work budget rather than only individual collection limits.
