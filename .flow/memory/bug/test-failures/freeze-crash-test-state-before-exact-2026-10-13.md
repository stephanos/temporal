---
title: Freeze crash-test state before exact prefix comparison
date: "2026-10-13"
track: bug
category: test-failures
module: tools/gomad3/backends/wasm/command/kill_resume_linux_test.go
tags: [wasm, testing, race]
problem_type: test-failure
symptoms: Built-CLI SIGKILL gate could observe additional commits between readiness and process discovery
root_cause: The test captured a durable prefix while the campaign process group was still running
resolution_type: fix
---

## Problem

The built-CLI crash gate read a committed outcome count while the campaign was still running, then discovered descendants before sending SIGKILL. A correct campaign could commit another outcome or finish during that interval, making an exact post-kill prefix assertion flaky and weakening the proof that the intended in-flight state was killed.

## What Didn't Work

Committed-log readiness alone did not freeze the process state. Discovering the process tree after the readiness read left a race between the observed durable prefix and the actual kill boundary.

## Solution

Send SIGSTOP to the CLI process group immediately after readiness, wait until the CLI is stopped, and run lock-free JSON inspection while the stopped writer still owns its lock. Use that second inspection as the stable durable prefix, discover the now-frozen descendants, SIGKILL the group and descendants, wait for confirmed SIGKILL exit and descendant disappearance, then require post-kill inspection to report exactly the frozen prefix before resume.

## Prevention

Crash tests that compare exact durable state must establish a quiescent process boundary before recording the comparison value. Run the gate repeatedly so process-discovery and scheduling races are exercised before review.
