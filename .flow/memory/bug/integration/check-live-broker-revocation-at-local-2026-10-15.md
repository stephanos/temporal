---
title: Check live broker revocation at local continuation sync
date: "2026-10-15"
track: bug
category: integration
module: tools/gomad/internal/backends/wasm/wasi/process.go
tags: [wasm, revocation, storage]
problem_type: integration
symptoms: Split storage sync continues after a crash or stop following successful delivery
root_cause: Sync reads only the error cached by Delivered instead of current session state
resolution_type: fix
related_to: [bug/integration/bound-encoded-storage-evidence-2026-10-15, bug/integration/share-offered-output-accounting-across-2026-10-15]
---

## Problem
Split storage sync checked only revocation cached while delivering a remote reply. A crash or stop after successful Delivered but before the next sync therefore sent continue and authorized local storage effects after revocation.

## What Didn't Work
Tests that revoked inside the reply writer covered the cached-error branch while missing the later interval. A helper-only interrupt test could prove the helper stopped but could not prove Go chose the interrupt at the right time.

## Solution
BrokerSession.localContinuation checks currentErrorLocked and the shared clock under the broker lock. revocationSync uses that state before sending continue. TestSplitRevocationAfterReplyInterruptsTheNextLocalOrRemoteCall now covers both during-reply and after-successful-delivery timing for crash and stop. Failing reproduction 7e75cccd3e and fix cfa09430a7 retain the red-to-green evidence.

## Prevention
Place deterministic revocations on both sides of the delivery completion boundary. Pair the transport control with a real-helper write/sync and storage-image equality control. Repeat the timing table under the required host load.
