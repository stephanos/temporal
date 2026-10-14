---
title: "slices.Collect(maps.Keys) turns empty into nil; cmp.Or[error] trips SA4023"
date: "2026-09-27"
track: bug
category: build-errors
module: common/testing/testpilot/temporal/worker
tags: [go, stdlib, staticcheck, testpilot]
problem_type: build-error
symptoms: stdlib swap changed empty result to nil; cmp.Or error comparison flagged SA4023 always true
root_cause: slices.Collect returns nil for empty seqs; staticcheck misreads cmp.Or instantiated with error
resolution_type: fix
related_to: [bug/build-errors/glossary-renames-can-reintroduce-names-2026-09-13]
---

## Problem
Replacing hand-written map-key collectors (`setKeys`-style loops over `make([]T, 0, len(m))`) with `slices.Collect(maps.Keys(m))` changes an empty result from a non-nil empty slice to nil. The review flagged it against the task's "keep nil-vs-empty results" rule.

## What Didn't Work
`slices.Collect(maps.Keys(m))` (and `slices.Sorted(...)`) return nil for an empty map. Separately, `cmp.Or(errA, errB, errC)` followed by `result != nil` trips staticcheck SA4023 ("comparison is always true"), a false positive for `cmp.Or[error]`; inlining the first-non-nil chain instead was flagged as a hand-written cmp.Or equivalent.

## Solution
`slices.AppendSeq(make([]T, 0, len(m)), maps.Keys(m))` keeps the non-nil empty shape (worker/driver.go, worker/routing.go nexusCandidates). For cmp.Or over errors: keep cmp.Or and put `//nolint:staticcheck // SA4023 false positive ...` on both the cmp.Or line (golangci reports the related-information location as its own issue) and the comparison line (worker/registry.go finishAcquisition).

## Prevention
When swapping a collector for `slices.Collect`, check whether the old helper pre-sized with `make(..., 0, n)`; if so use `AppendSeq` into the same make. Pin with a `require.NotNil` + `require.Empty` test on the empty input.
