---
title: Worker shutdown must retain completed RPC validation
date: "2026-10-25"
track: bug
category: runtime-errors
module: model/temporal/hosts/go/reference/worker
tags: [umpire, shutdown, temporal, evidence]
problem_type: runtime-error
symptoms: Normal Close could fault or hide a captured invalid terminal acknowledgment
root_cause: Peer cancellation ordering and broad shutdown suppression conflated unfinished waits with completed RPC validation
resolution_type: fix
related_to: [bug/runtime-errors/evidence-admission-refusals-must-remain-2026-10-25, bug/runtime-errors/field-alterations-must-range-over-2026-09-28, bug/runtime-errors/finite-ir-admission-must-count-work-2026-09-30, bug/runtime-errors/finite-search-requires-injective-2026-09-30, bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/replay-rejection-lost-to-2026-09-27, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05, bug/runtime-errors/workflow-replay-must-complete-2026-09-05]
---

## Problem
Worker shutdown could report an expected held-poll cancellation as DriverFault, or discard validation of an already captured terminal acknowledgment. The second case lost substantive errors and their actual RPC evidence when Run cancellation won a scheduling race.

## What Didn't Work
Canceling the poll hold before its worker owner allowed the poll to return while the owner was live. Checking only a derived SDK Run also missed owner-first cancellation propagation. Suppressing every error that matched context cancellation could hide joined validation failures. Merely recording transport snapshots did not ensure terminal validation ran during Close.

## Solution
Cancel the worker owner before releasing polls. Drain terminal expectations after handlers join, retain captured terminal completions across cancellation, and validate them even during shutdown. Suppress only unfinished wait cancellation using actual owner/Run provenance. Preserve substantive late Follow errors through Driver.Close; retry success may supersede prior failed terminal invocations, while local cancellation cannot erase unrecovered failures. Canceled heartbeat waits still earn no history bit.

## Prevention
Use synchronized controls that pause Close after poll release and pause terminal coordination until capture plus cancellation. Cover valid/invalid completed acknowledgments, unfinished local cancellation, owner-first propagation, failed-then-successful retries and mixed cancellation/validation errors. Exercise the actual pinned SDK transport and repeat race checks. See fn-246 task14 shutdown-reviewfix-4795ee4015 evidence and review primary session01a173d4-8077-7cf2-bfb2-ca35592b45aa; these controls are not live Temporal service proof.
