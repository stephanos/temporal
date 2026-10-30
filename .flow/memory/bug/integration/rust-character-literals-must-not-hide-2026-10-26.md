---
title: Rust character literals must not hide source-path macros
date: "2026-10-26"
track: bug
category: integration
module: model/umpire/dev/xtask/src/gate/boundary.rs
tags: [source-boundary, rust-literals]
problem_type: integration
symptoms: A valid character literal concealed an escaping include from the source guard
root_cause: Ad hoc tokenization opened strings inside character literals
resolution_type: fix
---

## Problem
The framework source guard treated a double quote inside a Rust character literal as a string opener. That hid a following include macro and let an existing file outside the framework escape source-containment checks. The organization exception also accepted an embedded path that the existing policy rejected.

## What Didn't Work
Normal source-tree and happy-path scanner checks passed. They did not exercise valid character/byte-character literals before an include, or the complete occurrence-level organization boundary matrix.

## Solution
Recognize character and byte-character literals before strings, preserve lifetime handling, and apply the existing exact organization occurrence rule through generic policy data. Register every graph prerequisite as an actual stage; declaration-only dependencies are not executable scheduling information.

## Prevention
Run the real compiled guard on existing outside files after quote/byte/lifetime prefixes, positive inside paths and the full negative/positive policy matrix. Validate that every listed prerequisite names a registered stage. The fn-254.10 revision1 controls and same-session review receipt bind these regressions to the corrected source.
