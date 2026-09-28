---
title: "Differentials must enumerate every search caller, not only declarations"
date: "2026-09-28"
track: bug
category: test-failures
module: model/Umpire/Search/Tests/Differential.lean
tags: [umpire, lean, search, fn-88]
problem_type: test-failure
symptoms: Backend differential missed programmatic AdmittedQuery.search callers and unselected searches
root_cause: Query inventory came from the query-command registry only
resolution_type: fix
---

## Problem
The fn-88.9 backend differential first swept only `query` declarations and treated Queries whose admission ended in `.notSelected` as not admitted. Review found every other caller of `AdmittedQuery.search`/`searchWithIntent` uncompared (Promotion, Variations points, Replay re-admissions, exploration campaigns, Property-only and relocated admissions), and a limit-reached comparison that could not see the reference's unresolved endpoints.

## What Didn't Work
Registry-only enumeration, and comparing a `CheckedModel`'s stored run: a multi-instance Query whose search selected nothing has no CheckedModel. Sampling a campaign (`targets.take 40`) was flagged as uncovered.

## Solution
`model/Umpire/Search/Tests/Differential.lean`: re-admit through `admitOver`/`admitQuery` (the admission `checkAdmitted` makes, stopping before search); `sweep` reads each multi-instance declaration's `checkInstances` application from the environment and re-applies its arguments to `instancesLine`; `campaignLine` compares every campaign candidate, pairing checked Queries with one admitted view via `withQuery`; `examinedEndpoints` re-decides every endpoint the reference traversal visits for limit-reached comparisons. The Caller campaign is split over four modules for parallel builds.

## Prevention
For "every Query reachable through X", grep the callers of X first and inventory both declared and programmatic ones; a differential over interleaved-instance models needs explicit cost bounds recorded in the spec.
