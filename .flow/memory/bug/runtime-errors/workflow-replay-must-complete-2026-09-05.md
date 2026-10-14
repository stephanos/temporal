---
title: Workflow replay must complete unfinished admissions once
date: "2026-09-05"
track: bug
category: runtime-errors
module: tools/umpire/temporal/worker
tags: [umpire, replay, reservation]
problem_type: runtime-error
symptoms: Reconstructed workflow finishes while its reservation remains open
root_cause: A replay boolean conflated live history reconstruction with duplicate terminal delivery
resolution_type: fix
related_to: [bug/runtime-errors/freeze-contract-transitions-when-2026-09-05, bug/runtime-errors/interface-nil-checks-must-cover-every-2026-09-04, bug/runtime-errors/monitor-closure-must-honor-cancellation-2026-09-04, bug/runtime-errors/retain-opaque-completion-authority-2026-09-05]
---

## Problem

Treating every repeated workflow admission as terminal replay skipped reservation completion when the Temporal SDK reconstructed an unfinished workflow after cache eviction. The reconstructed execution could finish successfully while its reservation and unused child reservations stayed open.

## What Didn't Work

A boolean replay flag conflated duplicate terminal delivery with reconstruction of live history. A second testsuite environment also did not expose the bug because `TestWorkflowEnvironment` never sets the SDK replay state.

## Solution

Keep the immutable activation in an exact admission record and guard terminal reporting with per-admission synchronized state in `tools/umpire/temporal/worker/sdk.go`. Reconstruction may complete a still-live admission once, while later terminal replay remains idempotent. Verify this with `worker.WorkflowReplayer` over partial history followed by completion events.

## Prevention

For durable workflow runtimes, test replay with the SDK history replayer and separate admission identity, execution reconstruction, and terminal idempotence in the state model.
