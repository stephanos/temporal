---
title: Raw clone needs architecture-specific arguments and child results
date: "2026-10-24"
track: bug
category: runtime-errors
module: model/umpire/backends/process/src/syscall.rs
tags: [subprocess, fork, linux, abi]
problem_type: runtime-error
symptoms: s390x received flags as a stack pointer and SPARC child identity was not normalized
root_cause: The raw syscall wrapper assumed libc fork argument and return normalization was universal
resolution_type: fix
---

## Problem
A raw clone wrapper assumed one argument order and a zero child return on every Linux architecture. Primary review identified s390x's stack-first flags slot; the bounded ABI sweep also found SPARC's separate child-indicator convention. Either error breaks native ownership and can return a forked child into general Rust code.

## What Didn't Work
Zero pointer arguments do not make the nonzero SIGCHLD flag position immaterial. Generic libc syscall assembly does not necessarily perform the normalization supplied by a libc fork wrapper.

## Solution
Select s390x's stack-first prefix, provide all six machine-word slots, and normalize SPARC child identity with raw PID queries before and after clone. The focused argument and result tables failed before correction, then passed. Pinned Linux kernel and libc assembly receipts are retained under task4/clone-abi; native evidence is Linux/aarch64, not s390x or SPARC.

## Prevention
Audit both syscall arguments and return registers for each admitted architecture. Use executable kernel definitions rather than only Kconfig prose, and distinguish source/scalar controls from actual non-host execution.
