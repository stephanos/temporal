---
title: Portable model plans need exact artifacts and checked obligations
date: "2026-09-03"
track: bug
category: integration
module: model/Temporal/Tool/PortableEvaluationContract.lean
tags: [umpire, portable-plan, artifact-identity, obligations]
problem_type: integration
symptoms: Generated model plans failed preparation or silently skipped unsupported checked semantics
root_cause: The portable projection omitted identity-bearing fields and obligation fixtures bypassed checked lowering
resolution_type: fix
related_to: [bug/integration/portable-execution-boundaries-must-2026-09-03, bug/integration/portable-schemas-must-preserve-source-2026-09-03]
---

## Problem
Lean-generated portable plans retained model artifact checksums while reconstructing identity-bearing ExperimentSpec fields from lossy execution limits, and unsupported checked Property/Observation constructs aborted or were manually represented instead of becoming compiler-owned obligations.

## What Didn't Work
Treating execution limits as a sufficient artifact projection changed checksums, selection metadata, explored counts, provenance, and observation configuration. A hand-appended obligation fixture also bypassed the checked semantic input path and left unsupported Property clauses fail-fast.

## Solution
The portable plan now carries a closed artifact projection sufficient to reconstruct exact ExperimentSpec and RuntimeConfiguration identities. Property and Observation lowering accumulate unsupported checked constructs, retain supported checks, and emit deterministic required obligations; generated fixtures pass the same admission and executor preparation path as ordinary plans.

## Prevention
For model-compiled wire formats, test exact decoded artifact equality and run every generated fixture through executor preparation. Obligation tests must start from actual checked semantic inputs and cover every unsupported pattern/clause branch, rather than constructing obligations after compilation.
