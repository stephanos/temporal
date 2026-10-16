---
title: Reconstruct crash resets and budget updates atomically
date: "2026-10-16"
track: bug
category: runtime-errors
module: model/framework/dsl/StateField.scala
tags: [native, crash, atomic-update]
problem_type: runtime-error
symptoms: A valid generated crash throws while constructing intermediate reset state
root_cause: Each field reset and budget write reconstructed records separately
resolution_type: fix
---

## Problem
Generated crashes reset each classified field through a separate case-class reconstruction. Valid simultaneous resets could therefore throw on an invalid intermediate record. Nested records and an own-budget decrement could create the same failure at either constructor boundary.

## What Didn't Work
Sequential StateField.write calls preserve individual field types but do not preserve constructor invariants spanning multiple fields. Grouping only resets would still leave a reset-before-budget intermediate state.

## Solution
Compute durability replacements and the own-budget replacement from the original state, then rebuild each affected nested record and the root once with final fields. Keep single-field writes for their existing callers. NativeCrashSuite covers sibling invariants and a nested invariant tied to the numeric budget, with exhaustion/negative-budget controls retained. Source correction: 461120e29d; formal full-batch review resolved the finding in round two.

## Prevention
For any transition that writes multiple fields, include a fixture whose constructor requires those fields to agree. Cover joint budget consumption and nested records, rather than testing only unconstrained case classes.
