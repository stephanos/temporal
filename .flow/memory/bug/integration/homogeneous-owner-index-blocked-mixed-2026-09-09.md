---
title: Homogeneous owner index blocked mixed-schema field Properties
date: "2026-09-09"
track: bug
category: integration
module: model/Umpire/Property/Evaluation.lean
tags: [lean, property, fields, type-index]
problem_type: integration
symptoms: request/state field relations rejected whenever operands came from different generated schemas
root_cause: a single owner/witness type index plus a global fieldBindings homogeneity guard on checked field admission
resolution_type: fix
---

## Problem
The first checked field-Property implementation indexed `CheckedFieldPredicate` and
`CheckedFieldProperty` by a single `owner`/`witness`, and a private `requireFieldOwner` rejected any
`PropertyCheckContext.fieldBindings` entry whose schema differed from that one owner. Because
`checkInput` also accepted only `List (PropertyFieldProjection owner witness)`, a same-step context
could not mix independently authored request, prior/resulting-state, outcome and event operands —
exactly the request/state and request/result relations ACT-3 and the task acceptance require. The
whole test module hid the defect because every binding was built from one mock schema.

## What Didn't Work
Relaxing `requireFieldOwner` alone would not have helped: the homogeneous
`List (PropertyFieldProjection owner witness)` input type still forbids heterogeneous evidence.

## Solution
Added `PropertyFieldEvidence` (private constructor, reachable only via
`PropertyFieldProjection.evidence`), which erases the generated index while the checked structural
path keeps the authority. `CheckedFieldPredicate`/`CheckedFieldProperty` lost their owner/witness
parameters, `requireFieldOwner` was deleted, and admission now relies on the per-operand binding
check already present in `Umpire/Property/Check.lean:validateFieldComparison` (each referenced path's
`(reference, schema)` must be an admitted `fieldBindings` entry).

## Prevention
Fixtures that construct every binding from one mock schema cannot exercise a heterogeneity rule.
When a checked context holds a *list* of authorities, the regression fixture must instantiate at
least two of them; `model/Umpire/Property/Tests/Fields.lean` now binds model state and semantic
events to a second `RpcOwner`/`Schema`.
