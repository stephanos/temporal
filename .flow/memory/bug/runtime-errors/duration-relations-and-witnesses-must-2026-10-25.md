---
title: Duration relations and witnesses must use the shared projection
date: "2026-10-25"
track: bug
category: runtime-errors
module: model/temporal/foundation/views.rs
tags: [duration, normalization, probes]
problem_type: runtime-error
symptoms: Retry ordering produced false violations above cap and missed noncanonical inversions
root_cause: Relation and witness code bypassed the shared duration projection
resolution_type: fix
---

## Problem
Retry ordering compared raw protobuf duration fields while validation used the shared normalized/capped projection. It falsely rejected intervals above the cap and missed inversions with excess nanos.

## Solution
Commit 755fd7b9b9 compares through View<kinds::Duration>. Inversion generation verifies the same projected relation and refuses atomically when the maximum is already at the cap.

## Prevention
Use the existing semantic projection for validation relations and generated witnesses. Cover capped equality, noncanonical nanos, disabled/default policies, validity, and failed witness construction without mutation.
