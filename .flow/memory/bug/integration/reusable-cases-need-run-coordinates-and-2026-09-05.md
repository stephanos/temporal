---
title: Reusable Cases need run coordinates and event-aligned captures
date: "2026-09-05"
track: bug
category: integration
module: model/Temporal/CaseRuntime.lean
tags: [umpire, case-runtime, correlation]
problem_type: integration
symptoms: Repeated Runs reused semantic IDs and cross-event history fields could satisfy one correlation
root_cause: The Case lacked a closed Run coordinate source and scalar projections discarded same-event pairing
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/contract-work-bounds-must-follow-typed-2026-09-04, bug/integration/full-integration-gates-must-select-the-2026-09-04, bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-model-plans-need-exact-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03, bug/integration/program-admission-must-validate-2026-09-04, bug/integration/validate-protobuf-descriptor-structure-2026-09-05]
---

## Problem
Implementation review found that a reusable Case authored fixed workflow and request identifiers, and its Contract correlated independently projected history fields. Repeated Runs could target the same Temporal execution, while fields from different History Events could accidentally satisfy one logical correlation.

## What Didn't Work
Static semantic identifiers did not carry the Executor's immutable Run coordinate. Separate scalar observations also lost the fact that `event_id`, `request_id`, and `scheduled_event_id` originated on the same History Event.

## Solution
Add one closed text Run-ID intrinsic that is admissible only in Program RPC request expressions, so the Lean Case explicitly chooses each assignment while the Executor supplies its immutable Run coordinate. Retain the complete declared `HistoryEvent` Observation under exact descriptor and byte bounds, then project correlated fields from that immutable capture and from each aligned current event.

## Prevention
Exercise one prepared artifact sequentially and concurrently while validating constructed request identities. Mutate the scheduled event's outer ID, started/completed references, and crossed completion fields; each mutation must remain inconclusive and live/offline evaluation must agree.
