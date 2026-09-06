---
title: Keep raw semantics behind checked input boundaries
date: "2026-09-05"
track: bug
category: integration
module: model/Umpire/Property/Evaluation.lean
tags: [umpire, lean, admission, public-api]
problem_type: integration
symptoms: Unchecked raw evaluation converted missing predicate input into a Boolean result
root_cause: Raw semantic helpers remained public beside the checked evaluator facade
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
A checked Boolean predicate API also exposed its raw evaluator over an arbitrary field projection. A caller could represent missing input as an empty list and negate the resulting false value into true, bypassing the required diagnostic boundary.

## What Didn't Work
Hiding the checked predicate constructor and indexing checked inputs by the predicate did not protect evaluation while the raw recursive semantics remained public.

## Solution
Keep literal, atom, and recursive predicate semantics private in `model/Umpire/Property/Evaluation.lean`. Expose evaluation and denotation only through the dependent `CheckedPropertyPredicateInput` interface, and assert at the facade boundary that raw evaluation names cannot be accessed.

## Prevention
For fail-closed checked APIs, audit every public semantic helper as a possible bypass. Add narrow-import compile failures for unchecked evaluators in addition to construction-forgery tests.
