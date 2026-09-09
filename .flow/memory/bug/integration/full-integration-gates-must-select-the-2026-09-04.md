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
`Makefile` now provides `umpire-check-live-tests`, runs a complete prefix selector rather than a representative test name, and compares any red result against the exact recorded failure identity set. CI and the aggregate regression target invoke that target, while `tools/umpire/regression/ci_workflow_test.go` pins the selector, wiring, and baseline.

fn-81 later deleted the Umpire2 and Umpire3 trees and with them the nine inherited failure identities. The gate kept this entry's whole-set rule and moved to the `^TestTestpilot` prefix with an empty baseline. An empty baseline needs one more guard than a populated one, so the gate also requires at least one `--- PASS` identity: an empty expected set on its own cannot distinguish "everything passed" from "the selector matched nothing", and a selector that matches nothing exits zero.

## Prevention
For migrated integration suites, guard the complete test-name prefix in both CI and aggregate Make dry runs. When a full suite inherits failures, compare the entire recorded failure identity set and fail on every addition or deletion rather than narrowing the selector to a green subset. When the inherited set is empty, pair the comparison with a floor on passing identities, or the gate passes on a selector that matched nothing.
