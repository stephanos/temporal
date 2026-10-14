---
title: Transition-table reads must omit realization-only refinement metadata
date: "2026-10-06"
track: bug
category: integration
module: tools/umpire/model/compose.go
tags: [umpire, composition, lint, refinement]
problem_type: integration
symptoms: A refinement-map hole hides a constructible machine or composition table
root_cause: The table reader used a realizing binding that evaluates refinement fields
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/channel-catalogs-and-visible-results-2026-09-30, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
The first composition stuck-state implementation read tables through a realizing binding. For record-state machines, that binding evaluates refinement-map metadata. A hole in the map therefore hid an otherwise constructible machine table and any composition using it. The correctness and integration review axes independently identified the same omission.

## What Didn't Work
Skipping replacement refinement checks in composition construction was insufficient: member subject construction still evaluated realization-only field metadata.

## Solution
`tools/umpire/model/compose.go:162` reads through the existing checking binding, which omits realization metadata while retaining transition interpretation, structural errors and composition ceilings. `TestStuckStateWithRefinementMapHole` covers both machine and composition populations and confirms that the incomplete refinement receipt remains visible. Its pre-fix run failed for both owners; the scoped post-fix run passed.

## Prevention
When a diagnostic must inspect constructible transition tables independently of claim verdicts, test failed and incomplete refinement metadata as well as rejected replacement refinements. Preserve the separate claim diagnostics instead of treating a failed verdict as a blanket table exemption.
