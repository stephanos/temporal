---
title: Correlate pending proposals with operation completion evidence
date: "2026-10-23"
track: bug
category: integration
module: model/temporal/feature/activity/temporal.rs
tags: [umpire, proposal, correlation, retry]
problem_type: integration
symptoms: Delivery stayed pending across retry; unrelated Describe cleared pending Cancel
root_cause: Completion required an intermediate state and correlation matched only resource identity
resolution_type: fix
---

## Problem
A pending activity delivery proposal waited forever after a fast retry skipped the first Started read. A separate correlation fallback accepted a Describe for the same activity as completion of a pending cancellation, although no cancellation result had been observed. Both defects were reproduced against the committed Wasm guest during implementation review.

## What Didn't Work
Testing only Started before retry missed valid progress between reads. Matching namespace, activity and run identity did not establish the pending RPC's invocation or completion.

## Solution
The realization completes delivery when Model-validated state reaches Started for the selected attempt or advances beyond that attempt. Cancellation correlation requires its exact RPC method and complete actual request; Describe belongs only to observation instructions. Pending changes remain inside the Runtime publication transaction. Native and Go/Wasm controls first failed, then passed for both races.

## Prevention
Exercise skipped intermediate reads and crossed RPCs, including a refused Describe followed by the actual cancellation result. Preserve the pending request across failed publication and callbacks. Derive completion from the typed pending action and validated prospective state; resource identity alone cannot finish an operation.
