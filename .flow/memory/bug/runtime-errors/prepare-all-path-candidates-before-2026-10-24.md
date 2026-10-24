---
title: Prepare all PATH candidates before allocation-free exec fallback
date: "2026-10-24"
track: bug
category: runtime-errors
module: model/umpire/backends/process/src/syscall.rs
tags: [subprocess, path, execve]
problem_type: runtime-error
symptoms: An unusable first PATH candidate prevented a valid later executable from launching
root_cause: Permission and file-type probes were incorrectly used to predict execve success
resolution_type: fix
---

## Problem
A native supervisor selected the first executable regular file in PATH before actually calling execve. A missing shebang interpreter or invalid executable format then prevented a valid later candidate from running. Primary implementation review found this functional launch regression.

## What Didn't Work
Parent-side access and regular-file probes established permissions and type, but could not prove that execve would succeed.

## Solution
Prepare every candidate's CString and a sentinel pointer table in the parent. The target tries each execve using only prepared storage and stack scalars, retaining the first substantive errno when all candidates fail. The path_search_tries_candidates_after_exec_errors control failed for both interpreter and format rows before the correction and passed afterward.

## Prevention
Test real exec failures followed by a working PATH entry. Permission checks are not an execution oracle, especially when the child path must remain allocation-free.
