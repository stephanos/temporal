---
title: Preserve caller context across nested workspace commands
date: "2026-10-25"
track: bug
category: integration
module: model/umpire/dev/guest-driver
tags: [workspace, cwd, cargo-home]
problem_type: integration
symptoms: Timed stages selected the wrong workspace and Cargo used inconsistent path roots
root_cause: Changing cwd without normalizing forwarded environment and command context
resolution_type: fix
---

## Problem
The workspace split left path-sensitive commands using inconsistent caller context. Timed framework recipes switched to the Temporal workspace, and Cargo inherited a relative or symlinked Cargo home that differed from the roots used for compiler identity remapping.

## What Didn't Work
Metadata/build success and helper-level tests did not establish which workspace the actual timed Make recipe executed. Making Cargo home absolute alone retained a lexical symlink path in emitted guest bytes.

## Solution
Resolve and canonicalize existing Cargo home before changing directories, forward that same path through bootstrap/probes/metadata/build, preserve caller cwd in timer step mode, and pass explicit native transcript paths at root Go boundaries. Keep standalone framework defaults local.

## Prevention
Exercise actual timed and untimed recipes, unset/relative/absolute target paths and distinct caller/framework roots with a relative Cargo-home alias. Reproduce guest bytes independently after driver changes. The fn-254.13 revision1 controls enforce these cases; see its current-index.json and review receipts.
