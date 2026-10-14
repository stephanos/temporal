---
title: Retain opaque completion authority across rejected acceptance
date: "2026-09-05"
track: bug
category: runtime-errors
module: tools/umpire/temporal/server
tags: [umpire, capability, cancellation]
problem_type: runtime-error
symptoms: Rejected completion acceptance stranded a consumed capability Slot
root_cause: Destructive bridge consumption preceded effect acceptance
resolution_type: fix
related_to: [bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04]
---

## Problem
Controller execution consumes a private capability Slot before calling the Host completion method.
If acceptance then fails due to cancellation or shared capacity, destructively consuming the Slot
loses the authority needed by cleanup. A plain mutex also lets cancellation expire while the caller
waits and then permits mutation after its deadline.

## What Didn't Work
Leaving the capability unused in a Host map did not make it recoverable: the Slot was consumed and
execution discarded the opaque value on error. Rechecking context only before mutex acquisition
was not sufficient, and rollback under the same canceled-context lock would still lose authority.

## Solution
The server bridge retains authority and returns a private context-bound claim. Rejected completion
releases the claim atomically without a rollback lock; canceled claims can also be replaced by fresh
cleanup consumption. Acceptance checks the current claim identity under the context-aware Host
lock and permanently marks the authority used. Stale claims cannot affect their replacements.
Profile nested method counts are bounded before any ownership clone.

## Prevention
Exercise Publish -> Consume -> canceled/capacity-rejected completion -> fresh cleanup Consume ->
stale claim rejection -> successful replacement, plus cancellation after acceptance. Hold Host
serialization while canceling each exposed method and assert it returns before the lock releases.
See tools/umpire/temporal/server/ownership_test.go.
