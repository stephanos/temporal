---
title: Evidence admission refusals must remain distinct from predicate rejection
date: "2026-10-25"
track: bug
category: runtime-errors
module: model/umpire/protocol/runtime/src/runtime.rs
tags: [evidence, reach, refusal]
problem_type: runtime-error
symptoms: Malformed or unsupported evidence became Unexplained and could freeze pending runs
root_cause: Reach converted Err into EvidenceMatch::Reject before search
resolution_type: fix
---

## Problem
Typed evidence admission returned MalformedRequest or Unsupported, but Reach converted every error into a rejected candidate. Exhaustive search then published Unexplained and could freeze a valid pending run.

## Solution
Commit 79beb81e4c propagates Result<EvidenceMatch, Refusal> through new observations and retained-prefix validation. Admitted predicate rejection remains separate from failed admission.

## Prevention
Keep controls for both new and retained evidence. Assert the original refusal code/message, pending request, immutable prior reports, unchanged history, repeated refusal without freeze, and successful retry. Test non-RPC Snapshot adapters separately; do not impose a generic RPC decoder on every evidence kind.
