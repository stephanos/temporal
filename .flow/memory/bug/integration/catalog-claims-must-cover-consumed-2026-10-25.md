---
title: Catalog claims must cover consumed leaves and preserve map value kinds
date: "2026-10-25"
track: bug
category: integration
module: model/umpire/catalog/build/src/claims.rs
tags: [catalog, claims, probes]
problem_type: integration
symptoms: Unclaimed consumers compiled and valid probe controls were suppressed
root_cause: Generated structural coverage was mistaken for semantic admission and actual fill inventory
resolution_type: fix
---

## Problem
Generated Ignored leaves could still be consumed by mapped reads/fills, and synthetic map entries lost Secret value kinds. A later audit found production Describe selectors filled without claims, removing twelve original controls under an invalid unclaimed-field reason.

## Solution
Commit da3ece4961 rejects consumed unclaimed leaves and preserves typed map carriers; 755fd7b9b9 restores consumed Flags, Poll headers and Start link identities. Explicit ignores remain valid. Independent descriptor identities remain in removal provenance.

## Prevention
Check actual constructors and owned transport consumers against claims, not only declarations against themselves. Keep omitted-claim compile failures, map-value suppression/ancestor-constraint controls, actual filled-selector assertions and per-original-case reconciliation. Preserve original strict metadata and distinguish Default initialization or opaque forwarding from semantic use.
