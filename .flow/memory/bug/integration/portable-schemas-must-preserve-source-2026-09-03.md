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
last_updated: "2026-09-04"
---

## Problem
The first PortableTestPlan schema collapsed DrivePlan semantic role kinds into scalar encodings and imposed synthetic nonempty collection cardinalities. It also conflated authority, participant, and semantic capability sets, so structurally valid Lean-shaped plans could not be represented or admitted faithfully.

## What Didn't Work
The initial synthetic fixture happened to satisfy invented one-role/one-choice/one-variant assumptions and therefore did not exercise the actual source artifact shape or semantic/scalar type boundaries.

## Solution
Keep semantic definition kinds separate from scalar value kinds in `proto/internal/temporal/server/api/umpire/v1/portable_test_plan.proto`, preserve independent source collections and capability sets, and validate caller-context semantic and scalar types in `tools/umpire/testplan/validate.go`. Mutation tests in `tools/umpire/testplan/plan_test.go` cover Lean-shaped cardinalities, crossed types, exact checkpoints, canonical collections, and model result reservations.

## Prevention
For protocol successors, derive every field type and collection cardinality from the existing typed source artifacts before designing fixtures. Include at least one representative source-shaped fixture plus crossed semantic/scalar mutations in the first admission test pass.

## Update 2026-09-04

## Problem
Portable Umpire schemas repeatedly lost source meaning when a synthetic fixture drove the wire shape: semantic definition kinds were omitted, presence-bearing values became scalars, graph-local identities disappeared, and invalid enum sentinels were reused as valid runtime states. These losses make distinct checked inputs serialize identically or force executors to invent conventions.

## What Didn't Work
A happy-path fixture covered the immediate example but not the complete closed source vocabulary, absent-versus-empty values, multi-node references, or every semantic state shared by protobuf and Lean.

## Solution
Keep semantic kinds separate from scalar kinds, preserve exact source collections and presence, give every referenceable graph a stable identity, and model every valid state explicitly in both protobuf and authored Lean. The Case IR tests in `tools/umpire/cmd/umpire-gen-lean-api/case_schema_test.go` now use source-shaped definitions, crossed oneofs, optional metadata, cleanup references, diagnostic presence, and bounded typed captures; `model/Umpire/CaseTests.lean` mirrors the same closed vocabulary.

## Prevention
Before freezing a protocol successor, inventory the complete source enums, optional fields, identity-bearing references, and bounded runtime state. Make the first regression fixture representative rather than minimal, then add crossed-type and absent/present round trips for each closed union.
