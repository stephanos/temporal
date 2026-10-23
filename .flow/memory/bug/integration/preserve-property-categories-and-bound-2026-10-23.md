---
title: Preserve property categories and bound native start-failure replay
date: "2026-10-23"
track: bug
category: integration
module: model/umpire/backends/apalache/src/replay.rs
tags: [replay, typed-identity, arithmetic]
problem_type: integration
symptoms: A selected monitor could alias an authored claim; start failure capture enumerated whole domains
root_cause: Bare-name lookup lost property categories and generated failure loops lost range type constraints
resolution_type: fix
related_to: [bug/integration/preserve-arithmetic-evaluation-roots-2026-10-23, bug/integration/preserve-explicit-replay-bindings-when-2026-10-13]
---

## Problem
Apalache replay accepted a selected synthetic safety monitor or negative claim by evaluating an authored invariant with the same bare name. Generated start-failure capture also enumerated whole domain products and lost the field's integer inference.

## What Didn't Work
Combining native registration lists erased the emitted property's category. Collecting all failed prefixes required unbounded work before matching one witness, and an empty terminal visitor stopped constraining range literals through state construction.

## Solution
Retain the selected metadata category, isolate native registrations, and require arithmetic confirmation for the arithmetic monitor. Generate direct selected-prefix checks through native typed ranges in declaration order. Preserve the original panic payload and compare the failed suffix against emitted defaults without accepting a fallback edge.

## Prevention
Keep equal-name category controls, wide unsuffixed range compilation, and a billion-wide selected-prefix regression. Verify empty-domain suppression and actual later-field arithmetic witnesses separately from legal native edges.
