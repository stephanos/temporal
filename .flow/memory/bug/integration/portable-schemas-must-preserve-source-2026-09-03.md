---
title: Portable schemas must preserve source semantic kinds and cardinalities
date: "2026-09-03"
track: bug
category: integration
module: proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto
tags: [umpire, protobuf, admission]
problem_type: integration
symptoms: Lean-shaped DrivePlans were rejected or admitted with crossed semantic and scalar types
root_cause: The successor schema was designed from a synthetic fixture instead of retaining each source artifact meaning
resolution_type: fix
---

## Problem
The first PortableTestPlan schema collapsed DrivePlan semantic role kinds into scalar encodings and imposed synthetic nonempty collection cardinalities. It also conflated authority, participant, and semantic capability sets, so structurally valid Lean-shaped plans could not be represented or admitted faithfully.

## What Didn't Work
The initial synthetic fixture happened to satisfy invented one-role/one-choice/one-variant assumptions and therefore did not exercise the actual source artifact shape or semantic/scalar type boundaries.

## Solution
Keep semantic definition kinds separate from scalar value kinds in `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto`, preserve independent source collections and capability sets, and validate caller-context semantic and scalar types in `tools/umpire/testplan/validate.go`. Mutation tests in `tools/umpire/testplan/plan_test.go` cover Lean-shaped cardinalities, crossed types, exact checkpoints, canonical collections, and model result reservations.

## Prevention
For protocol successors, derive every field type and collection cardinality from the existing typed source artifacts before designing fixtures. Include at least one representative source-shaped fixture plus crossed semantic/scalar mutations in the first admission test pass.
