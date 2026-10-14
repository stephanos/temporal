---
title: Behavior-neutral refactors must not strengthen validation
date: "2026-09-04"
track: bug
category: integration
module: model/Umpire/Observation/Evaluation/Admission.lean
tags: [umpire, lean, refactor, compatibility]
problem_type: integration
symptoms: A decomposition changed accepted-envelope results and added a traversal
root_cause: A new hardening invariant was mixed into a behavior-neutral relocation
resolution_type: fix
related_to: [bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03]
---

## Problem
A behavior-neutral module extraction added a new accepted-envelope consistency check. The check rejected a carrier that the pre-refactor validator accepted and introduced another nested traversal across links, dispositions, record support, and fields.

## What Didn't Work
Treating a stricter fail-closed invariant as an implied part of a decomposition mixed a semantic change into a relocation task, even though the new regression made that stricter behavior look intentional.

## Solution
Remove the new cross-check and its changed-behavior fixture so `model/Umpire/Observation/Evaluation/Admission.lean` contains the original validator unchanged. Keep mutation coverage limited to the established diagnostics and precedence named by the task.

## Prevention
For behavior-neutral refactors, compare the moved validator against the base implementation before adding guards. Record newly discovered hardening opportunities as separate semantic-change work, including their traversal and compatibility costs.
