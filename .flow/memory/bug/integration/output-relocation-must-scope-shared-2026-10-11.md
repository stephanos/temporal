---
title: Output relocation must scope shared readers and publication
date: "2026-10-11"
track: bug
category: integration
module: tools/umpire/cmd/umpire-gen-cases
tags: [paths, feature-selection, publication]
problem_type: integration
symptoms: Relocated outputs broke shared readers and left standalone targets unscoped
root_cause: Central-tree and full-suite assumptions survived feature colocation
resolution_type: fix
---

## Problem
Colocating generated model outputs left shared readers, standalone Make entrypoints and scoped fixture compilation using central-tree or full-suite assumptions. A path-only happy-path check did not prove the actual consumer routes. The first implementation review found seven gaps.

## What Didn't Work
Changing caller directories without migrating shared manifest/name readers broke functional consumers. Passing a feature selection to the gate did not scope standalone Make targets, and publishing a subset through a whole-tree writer would discard unselected pins. Removing source positions from fixture comparisons hid location drift.

## Solution
Shared readers resolve the feature-owned primary and historical trees through neutral path leaves and retain logical Model/Query names. Actual Make entrypoints invoke the generator in selected source or generated folders. Pinned publication merges only selected identities inside the existing lock, retaining unselected rows, bytes and timestamps. Scoped compilation excludes full-feature fixture packages; equivalence keeps all applicable immutable baseline rows. Original fixture positions are verified by explicit complete coordinate oracles with corruption controls. Import exceptions admit only the exact neutral namespace and interpreter-test path leaf.

## Prevention
Exercise real Make entrypoints and shared consumers, not only the gate's internal command builder. Prove checks are read-only and partial updates preserve sibling bytes and mtimes, include a source-only no-pin feature and invalid scope, and retain negative controls for complete fixture metadata and namespace boundaries.
