---
title: Gate retry polling before observing Scheduled evidence
date: "2026-10-22"
track: bug
category: test-failures
module: tests/testpilot_model_slice_test.go
tags: [umpire, retry, polling, evidence, synchronization]
problem_type: test-failure
symptoms: Delayed retry Describe observes Started before its Scheduled transition can be judged
root_cause: Callback blocking happens after server dispatch and cannot hold the next attempt Scheduled
resolution_type: fix
related_to: [bug/test-failures/differentials-must-enumerate-every-2026-09-28]
---

## Problem
The SDK retry proof held the second callback after delivery while expecting Describe to report Scheduled. An active long poll could already make the server mark attempt two Started. Delaying Describe past the two-second backoff reproduced a State refusal although the execution was valid.

## What Didn't Work
A callback channel and a short retry interval did not order server dispatch against Describe. SDK activity entry happens after the server records Started.

## Solution
The live harness serializes actual Poll RPCs with one permit. Empty and failed polls restore it; a nonempty delivery consumes it. After the actual failed-attempt RPC succeeds, the test delays a fresh Describe beyond the retry interval, judges Scheduled attempt two, then releases Poll permission. The independent Model remains unchanged.

## Prevention
Keep the delayed-observation control. Verify that concurrent poll invocations cannot bypass the permit and that failed or empty polls restore permission. Use actual RPC acknowledgment to order observations rather than assuming callback blocking holds server state.
