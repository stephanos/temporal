---
title: Retain backend capture handles across untrusted process execution
date: "2026-10-23"
track: bug
category: security
module: model/umpire/backends/tlc/src/process.rs
tags: [subprocess, containment, output, lifecycle]
problem_type: security
symptoms: Log substitution or inherited output descriptors bypass bounded capture cleanup
root_cause: Reopening tool-controlled paths and waiting for producer EOF after cleanup
resolution_type: fix
---

## Problem
Review found that reopening backend-writable log paths could follow a substituted symlink or block on a FIFO. Pipe-reader joins could also wait indefinitely when a descendant retained an output descriptor.

## What Didn't Work
Checking path metadata before reopening left a substitution window. Waiting for producer EOF extended the process deadline after cleanup.

## Solution
The shared process owner retains securely created log file handles, polls file identity and combined capture size, cleans up owned processes, and reads bounded diagnostics through those handles. ProcessFailure carries those diagnostics to Apalache without reopening paths. Darwin supervision prevents descendants with child-only NPROC limits and refuses privileged execution.

## Prevention
Retain focused controls that substitute log symlinks and FIFOs, exceed combined output budgets, and verify cancellation, owner death and signal cleanup with explicit owned PID disappearance. A preflight that prints an expected diagnostic and then hangs must fail rather than pass from text alone.
