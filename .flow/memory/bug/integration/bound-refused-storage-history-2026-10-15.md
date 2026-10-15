---
title: Bound refused storage history independently of recovery image
date: "2026-10-15"
track: bug
category: integration
module: tools/gomad/internal/backends/wasm/wasi/storage_evidence.go
tags: [storage, evidence, bounds]
problem_type: integration
symptoms: Correct helper prefix rejected by image-derived receive budget
root_cause: Refused history need not fit a recovery image
resolution_type: fix
related_to: [bug/integration/bound-encoded-storage-evidence-2026-10-15, bug/integration/check-live-broker-revocation-at-local-2026-10-15]
---

## Problem
Permitted refused-run storage history exceeded the recovery-image-derived
receive allowance. Rust retained the correct failure prefix, but Go rejected
its evidence before exposing the original image-capacity terminal.

## What Didn't Work
An image-size coefficient bounds complete recovery images, not independent
call-created history when final image encoding fails.

## Solution
Recognize the complete 16-byte partial-v2 header before selecting a separately
derived finite allowance. Bound operations by calls and cumulative allocated
nodes, paths by node depth and admitted component length, and snapshot entries
by initial plus call-introduced paths. Include JSON escaping, scalar widths,
volume metadata and saturating arithmetic. Keep complete-v1 and image caps.
See storage_evidence.go and fn176-task7-partial-evidence-budget-2026-10-15.md.

## Prevention
Use actual-helper/Go-owner large-history parity regressions and exact/+1 stream
bounds. Check incomplete headers 0–15 bytes as well as full-header splits;
zero-padded short buffers must never count as recognized headers.
