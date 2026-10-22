---
title: Rejected delivery steps must not authorize callback replies
date: "2026-10-22"
track: bug
category: integration
module: model/temporal/feature/activity/temporal.rs
tags: [umpire, realization, rejection, callback]
problem_type: integration
symptoms: Mismatched activity callbacks receive replies after rejected Poll delivery
root_cause: Replayability includes rejected self-loops and does not establish successful delivery
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/every-scenario-command-scenario-carries-2026-09-27, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/nested-admission-diagnostics-must-2026-09-05, bug/integration/paired-level-validators-must-check-each-2026-10-06, bug/integration/portable-carriers-must-keep-the-checked-2026-09-09, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/preserve-explicit-replay-bindings-when-2026-10-13, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/reconciliation-must-preserve-authority-2026-09-05, bug/integration/required-numeric-generation-must-sample-2026-10-21, bug/integration/retired-protobuf-names-must-guard-every-2026-10-09, bug/integration/reusable-cases-need-run-coordinates-and-2026-09-05, bug/integration/schema-closure-tests-must-compile-2026-10-09, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27, bug/integration/transition-table-reads-must-omit-2026-10-06, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Adding stale Poll rejection to the activity Model changed delivery from a disabled action to a rejected self-loop. After valid attempt-one delivery, the realization could answer callbacks for attempts two and three because generic replay accepts emitted rejected steps and completion remained enabled in Started(1).

## What Didn't Work
Checking that Poll was replayable did not establish accepted delivery. The initial controls exercised current attempts and terminal observations, missing mismatched callbacks after delivery.

## Solution
The existing state-aware answer handler checks the observed callback attempt against the current activity attempt before constructing its delivery plan. Mismatch returns State refusal. Native and actual committed-guest regressions first failed, then passed while preserving Started and the later terminal observation.

## Prevention
When a Model replaces disabled actions with rejected self-loops, revisit callback consumers that treat replayability as authorization. Exercise rejection after an earlier valid delivery and verify refusal, unchanged state and the next valid observation. Keep generic replay able to explain legitimate rejecting observations.
