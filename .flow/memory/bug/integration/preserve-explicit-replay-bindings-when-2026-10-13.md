---
title: Preserve explicit replay bindings when sharing a Model
date: "2026-10-13"
track: bug
category: integration
module: tools/umpire/check/checking.go
tags: [umpire, replay, sharing]
problem_type: integration
symptoms: Same-Model sharing bypassed the independent replay binding supplied by the property-row helper
root_cause: The default sharing shortcut ran before checking the existing replay override
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/retired-protobuf-names-must-guard-every-2026-10-09, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/schema-closure-tests-must-compile-2026-10-09, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/transition-table-reads-must-omit-2026-10-06, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
The first R6 implementation returned the checker's original binding before checking for an explicitly supplied replay binding. `newPropertyRowWalk` supplied a separate scoped interpretation, but replay used the original binding while existing independence assertions inspected the unused one.

## What Didn't Work
A same-Model pointer check at the start of `checker.again` bypassed the caller's `fresh` binding. Report parity alone did not expose this because both interpretations produced the same answers.

## Solution
Apply same-Model sharing only when `fresh` is nil. The property-row helper now asserts that `again()` returns the supplied replay binding. `TestPropertyRowWalkKeepsItsIndependentReplayBinding` runs that helper on a small counter Model and proves that replay uses an independent interpreter.

## Prevention
Trace explicit field assignments when optimizing a default path. Assert which supplied instance the consumer actually uses, alongside its observable answers. The focused regression failed before the ordering fix and passed afterward.
