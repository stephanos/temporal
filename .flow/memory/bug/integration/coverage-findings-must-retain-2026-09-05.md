---
title: Coverage findings must retain independent obligations and evidence
date: "2026-09-05"
track: bug
category: integration
module: model/Umpire/Planning/CaseAnalysis.lean
tags: [umpire, lean, case-coverage, evidence]
problem_type: integration
symptoms: Missing replacement was suppressed and uncovered witnesses lost case evidence
root_cause: Finding construction conflated completeness with replacement and discarded observed cases
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Finite case-analysis findings correctly classified excluded and uncovered behavior, but two evidence-construction choices weakened the result. Missing-replacement evidence was tied to a complete-group obligation, and uncovered findings discarded the declared cases whose guards failed.

## What Didn't Work
The first implementation treated missing replacement as a completeness failure and passed an empty case list to the uncovered finding constructor. That conflated separate obligations and removed the exact case IDs, sources, clauses, and guards needed to explain the witness.

## Solution
In `model/Umpire/Planning/CaseAnalysis.lean`, emit missing replacement whenever the parent applies, exclusions exist, and no case applies, independent of `complete`. Construct uncovered complete-group findings from all observed declared cases so their checked evidence remains attached. Focused regressions cover a non-complete excluded group and assert exact uncovered case evidence.

## Prevention
For every diagnostic classification, test both its trigger condition and the full evidence payload. Keep completeness, exclusivity, exclusion, and replacement as independent predicates rather than reusing one obligation as another's gate.
