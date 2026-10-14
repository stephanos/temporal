---
title: Moved conformance tests must not import functional adapters
date: "2026-09-06"
track: bug
category: integration
module: common/testing/testpilot/conformance_test.go
tags: [testpilot, dependencies, conformance]
problem_type: integration
symptoms: Common Testpilot tests imported the functional testcore Driver
root_cause: A moved test reused an adapter-owned descriptor helper
resolution_type: fix
related_to: [bug/integration/reconciliation-must-preserve-authority-2026-09-05]
---

## Problem
Moving a conformance test into the reusable Testpilot owner retained a helper import from the functional testcore Driver, reversing the intended dependency direction.

## What Didn't Work
Reusing the moved adapter descriptor helper was convenient but made common Testpilot tests compile the functional SDK dependency tree.

## Solution
Build the exact WorkflowService descriptor closure locally from the public protobuf descriptor in common/testing/testpilot/conformance_test.go.

## Prevention
After moving an ownership test, run go list -deps -test for the destination package and reject dependencies on its adapters.
