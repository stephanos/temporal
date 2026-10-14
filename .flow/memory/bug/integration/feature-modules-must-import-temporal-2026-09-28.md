---
title: Feature modules must import Temporal conventions or IDs change family
date: "2026-09-28"
track: bug
category: integration
module: model/Temporal/Case/Conventions.lean
tags: [definition-id, conventions, feature-module]
problem_type: integration
symptoms: New Temporal.Feature module mints temporal.feature.* Definition IDs
root_cause: Module imported Umpire.Command without Temporal.Case.Conventions
resolution_type: fix
---

## Problem
A new command-authored module under `Temporal.Feature` that imports only `Umpire.Command` elaborates cleanly, but its Definition IDs land in the family `temporal.feature.<x>` instead of `temporal.<x>`. The IDs are fixed at elaboration, so importing the module beside other Temporal Models later does not correct them. fn-92.1's worker module shipped this way until review.

## What Didn't Work
Pinning only the machine table and axioms: nothing in those pins reads an ID, so the wrong family passed every test.

## Solution
Import `Temporal.Case.Conventions` (the `model_conventions root "temporal" under Temporal.Feature` declaration) as the first import of every feature Model module, directly or through `Temporal.Case.Syntax`. Pin at least the entity, one action, and the machine `targetId` in the module's Tests (`model/Temporal/Feature/Worker/Tests.lean`).

## Prevention
A new feature module's Tests pin its Definition IDs. The same fix loop showed a second gap: a declaration lint that imports aggregate roots misses feature modules that no aggregate imports. Feed it the discovered source inventory instead.
