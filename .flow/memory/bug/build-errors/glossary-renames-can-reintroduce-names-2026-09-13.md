---
title: Glossary renames can reintroduce names the vocabulary gate retired
date: "2026-09-13"
track: bug
category: build-errors
module: tools/umpire/internal/retiredvocabulary/check.go
tags: [umpire, testpilot, rename, vocabulary]
problem_type: build-error
symptoms: retired-vocabulary gate red after a follow-on consistency rename
root_cause: new identifier matched a compound retired by an earlier spec
resolution_type: fix
---

## Problem
During the fn-87 glossary rename (RunStatus to RunDisposition), a hand-written Go method
`recorder.terminalStatus` was renamed to follow the new word after the retired-vocabulary gate had
already passed. fn-82 had retired that exact compound in `tools/umpire/internal/retiredvocabulary/check.go`,
so the gate went red and the review returned NEEDS_WORK.

## What Didn't Work
Running `make umpire-check-retired-vocabulary` mid-task and then making "consistency" renames afterwards.

## Solution
Reverted the extra rename (`common/testing/testpilot/internal/execution/recorder.go`); the method keeps the name
the gate holds and only its return type moved to `RunDisposition`.

## Prevention
Before renaming a hand-written identifier toward a glossary word, grep `check.go` exactTokens for the new
compound; run the vocabulary gate after the last edit, not mid-task.
