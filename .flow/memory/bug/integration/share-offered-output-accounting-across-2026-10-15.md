---
title: Share offered output accounting across split namespace and remote writes
date: "2026-10-15"
track: bug
category: integration
module: tools/gomad/internal/backends/wasm/wasmhost/src/engine.rs
tags: [wasm, output, split]
problem_type: integration
symptoms: Closed standard-descriptor writes bypass split output capacity or fail Go-owner terminal validation
root_cause: Remote ABI refunds and independent namespace budgets disagree with Go offered-byte semantics
resolution_type: fix
---

## Problem
Split WASM storage runs accounted remote stdout writes separately from local writes to closed stdout. A guest could exceed its offered-output budget without a capacity outcome. The Go-owner reference also rejected below-limit failed writes at terminal validation because the helper refunded their offered bytes.

## What Didn't Work
The remote ABI settled its counter to bytes actually written, while Go's namespace and helper-owned namespace charge offered bytes before descriptor validation. Preserving descriptor-based routing after close exposed the inconsistent counters.

## Solution
The ABI retains offered-byte charges across partial and errno replies. Split namespace and remote operations synchronize a cumulative counter, and split callbacks carry the pre-call offered total to Go. Admission rejects regressing, over-limit or incorrectly owned totals; final split totals cannot regress or exceed the limit. Remote-owner final totals retain exact equality. The configured namespace image is already bound to the execute output limit before execution.

## Prevention
Compare both owners through real Run for remote write, local close, rejected write and subsequent remote write. Assert capacity, offered totals, call/transcript history, actual stdout/stderr and remote callback counts. Keep partial/failed-write ABI tests separate from their actual written-result assertions. See TestStorageSplitOutputBudgetAfterClose, TestHelperBridgeSplitOutputStateAdmission and TestTransportSplitOutputTerminalBounds.
