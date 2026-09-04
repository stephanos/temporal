---
title: Full integration gates must select the complete migrated suite
date: "2026-09-04"
track: bug
category: integration
module: Makefile
tags: [umpire, integration, ci, baseline]
problem_type: integration
symptoms: Make and CI ran one green relocated test while other migrated live tests were omitted
root_cause: The live gate used a representative exact-name selector instead of the suite prefix
resolution_type: fix
related_to: [bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03]
---

## Problem
The Umpire Make and CI live-test gates selected only one relocated generated test, so other relocated Run Evaluation, HTTP/gRPC portable executor, and hand-written caller-closure integration tests could regress without either aggregate gate noticing.

## What Didn't Work
Treating one representative relocated test as the live integration command satisfied the destination/tag check but did not prove that every test in the migrated `tests/` surface remained covered.

## Solution
`Makefile` now provides `umpire-check-live-tests`, runs the complete `^TestUmpire` prefix selector, and compares any red result against the exact inherited Umpire2/Umpire3 failure identity set. CI and the aggregate regression target invoke that target, while `tools/umpire/regression/ci_workflow_test.go` pins the selector, wiring, and baseline identities.

## Prevention
For migrated integration suites, guard the complete test-name prefix in both CI and aggregate Make dry runs. When a full suite inherits failures, compare the entire recorded failure identity set and fail on every addition or deletion rather than narrowing the selector to a green subset.
