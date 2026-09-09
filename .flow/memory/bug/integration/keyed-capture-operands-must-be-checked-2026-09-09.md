---
title: Keyed capture operands must be checked against their declaration
date: "2026-09-09"
track: bug
category: integration
module: model/Umpire/Property/Check.lean
tags: [lean, property, captures, scoped, admission]
problem_type: integration
symptoms: malformed capture operands were admitted and failed later as missing evidence
root_cause: "predicate validation carried only capture names, not the declared retained path and lifetime"
resolution_type: fix
---

## Problem
Keyed capture operands were admitted on the strength of the capture *name* alone. Because the
declaration already fixes the exact coordinates an occurrence retains, an operand could name a
declared capture and then reference different root/reference/schema/steps/type coordinates, or an
ordinal past the declared lifetime. `checkProperty` accepted both. Neither could ever bind at
runtime, so the malformed Property failed only later, as a `missingPredicateInput` on every step of
the operation — a static contract error reported as missing evidence.

## What Didn't Work
Threading only `List DefinitionId` (the declared capture names) through `validatePropertyPredicate`
was enough to reject an unbound name but carried no information to check the operand against.

## Solution
`model/Umpire/Property/Check.lean:validateFieldComparison` now receives the declared
`PropertyScopedCapture` records. A capture operand must resolve to a declaration, its path with the
capture key erased must equal that declaration's retained path, and its ordinal must be below the
declaration's lifetime.

## Prevention
When a declaration already pins a value's coordinates, the reference to it must be checked against
the declaration, not against its name. A name-only scope check turns a static contract violation
into a runtime "missing evidence" failure that looks like a data problem.
