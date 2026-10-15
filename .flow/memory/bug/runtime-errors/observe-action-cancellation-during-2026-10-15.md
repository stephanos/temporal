---
title: Observe action cancellation during split helper revocation grace
date: "2026-10-15"
track: bug
category: runtime-errors
module: tools/gomad/internal/backends/wasm/cluster/provider.go
tags: [wasm, revocation, cancellation]
problem_type: runtime-error
symptoms: Canceled Crash or Stop waits for the full helper grace
root_cause: The synchronous grace select omits the operation context
resolution_type: fix
---

## Problem
Cluster Crash and Stop waited synchronously for a split-owned helper to reach another broker call. The 30-second grace ignored the action context, so an already-canceled action could block until the grace expired.

## What Didn't Work
Normal unbound durable restart tests reached another callback quickly and never exercised a blocked helper with a shorter action context.

## Solution
awaitRevocation observes the action context, cancels the helper and returns its context error. The node remains owned by execution for mandatory Shutdown. TestProviderCanceledSplitRevocationCancelsAndReaps requires prompt return, helper cancellation, a Reaped transition and successful Shutdown. Reproduction 7e75cccd3e fails the one-second control; fix cfa09430a7 passes.

## Prevention
Any host grace wait inside a context-taking lifecycle operation must observe that operation context. Test a blocked node separately from an actual-helper cleanup/restart control, and retain cleanup ownership when the operation returns early.
