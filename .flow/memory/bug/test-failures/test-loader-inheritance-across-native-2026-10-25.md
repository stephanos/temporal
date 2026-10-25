---
title: Test loader inheritance across native fixtures
date: "2026-10-25"
track: bug
category: test-failures
module: model/dev/xtask/tests/guest_pipeline.rs
tags: [macos, environment, fixtures]
problem_type: test-failure
symptoms: Protected shell fixtures lose the loader marker before the native boundary
root_cause: The test modeled native inheritance through system-shell execution
resolution_type: fix
---

## Problem
A loader-variable control used protected system Bash scripts as compiler/previous-wrapper fixtures, so macOS could strip DYLD variables before the native launcher boundary under test.

## What Didn't Work
Exporting the marker in the test parent coupled success-path tests to the system shell's SIP behavior; that is not evidence about native wrapper inheritance.

## Solution
Reuse the existing native compiler fixture for both previous wrapper and compiler, inject the marker immediately before the native launcher, and assert it in both native endpoints. A Linux control simulating orchestration-layer purging failed before this correction and passes afterward, including raw marker-byte observation. Production guest/compiler/build inputs remain unchanged.

## Prevention
Test loader inheritance over the actual native process boundary. Keep shell orchestration separate, and label simulated purging independently from native macOS execution. Apple Runtime Protections documents protected-process dyld environment purging: https://developer.apple.com/library/archive/documentation/Security/Conceptual/System_Integrity_Protection_Guide/RuntimeProtections/RuntimeProtections.html
