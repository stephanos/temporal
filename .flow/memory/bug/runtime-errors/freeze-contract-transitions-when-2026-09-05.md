---
title: Freeze Contract transitions when execution becomes incomplete
date: "2026-09-05"
track: bug
category: runtime-errors
module: tools/umpire/verification
tags: [umpire, monitor, precedence]
problem_type: runtime-error
symptoms: Late positive evidence promoted an already incomplete Run to violated
root_cause: Incompleteness suppressed horizon expiry but not positive transitions
resolution_type: fix
---

## Problem
Execution incompleteness suppressed horizon expiry but still allowed later positive Contract transitions. A drained in-flight fact could therefore produce a new violation after an earlier execution failure and promote an incomplete Run to stopped/violated, contradicting the fixed precedence table.

## What Didn't Work
Masking only the recorder's returned Verdict would diverge from offline replay. Suppressing absence-based horizon expiry alone did not freeze positive transitions or captures.

## Solution
The shared Evaluator Observe path freezes transitions, captures and support when the current or earlier event marks execution incomplete, while continuing validated sequence/elapsed progression. Violations committed before that boundary remain authoritative. The recorder marks incomplete before callbacks and uses the same default Monitor semantics without inventing a failed-callback cutoff.

## Prevention
Table-test failure before a potentially violating fact, failure on that same fact, and violation before failure. Compare live/offline decisions, Verdicts, supporting sequences and transition traces; separately retain successful-callback cancellation and failed-callback cutoff regressions.
