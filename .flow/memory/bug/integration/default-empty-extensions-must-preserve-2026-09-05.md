---
title: Default-empty extensions must preserve canonical fingerprints
date: "2026-09-05"
track: bug
category: integration
module: model/Umpire/Property/Check.lean
tags: [canonicalization, compatibility, fingerprint]
problem_type: integration
symptoms: Existing same-step Properties changed fingerprints after adding an unused temporal field
root_cause: Canonical JSON emitted the new default-empty field unconditionally
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/nested-admission-diagnostics-must-2026-09-05]
---

## Problem
Adding optional case-scoped temporal clauses unconditionally serialized an empty `temporalClauses` array, changing canonical metadata and behavior fingerprints for existing same-step-only Properties.

## What Didn't Work
The first implementation treated a default-empty compatibility field like ordinary new canonical data even when it had no semantic content.

## Solution
`model/Umpire/Property/Check.lean` omits the temporal field when the list is empty and emits it only for the new semantic form. `model/Umpire/Property/Tests/GuardedCases.lean` freezes the exact fingerprint independently rebuilt from the persisted task-start tree.

## Prevention
Whenever a public record gains a default-empty semantic extension, compare the old tree's exact canonical bytes or fingerprint and add a golden regression before review.
