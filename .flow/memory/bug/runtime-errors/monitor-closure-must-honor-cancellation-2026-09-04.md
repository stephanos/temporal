---
title: Monitor closure must honor cancellation without rebuilding proofs
date: "2026-09-04"
track: bug
category: runtime-errors
module: tools/umpire/verification
tags: [umpire, cancellation, monitor]
problem_type: runtime-error
symptoms: Close could return satisfied after context expiry during validation
root_cause: Only entry cancellation was checked before uncancelable closure work
resolution_type: fix
related_to: [bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04]
---

## Problem
A Monitor Close callback checked cancellation only on entry. Cancellation during a long Run event scan or Verdict construction could return authoritative satisfaction without an error.

## What Didn't Work
Adding an entry check did not bound later loops. Building a complete fallback Verdict after cancellation would still require scanning all historical support to preserve an earlier violation.

## Solution
Maintain rule results and supporting references incrementally at the atomic Observe commit. Close polls context during event validation, performs a final cancellation check, and transfers the already prepared frozen Verdict once. A cancellation changes an unproved result to inconclusive but preserves committed violation evidence. Repeated callbacks after Close are rejected.

## Prevention
Use deterministic cancellation-on-check contexts to cover callback entry, mid-scan and final-return boundaries, with both satisfied and previously violated states. Assert live/offline Verdict and trace equivalence, terminal callback behavior, and no support-history copying per event.
