---
title: Separate archived scan roots from historical source paths
date: "2026-10-18"
track: bug
category: integration
module: tools/umpire/ir
tags: [migration, layout, scanner]
problem_type: integration
symptoms: Archive relocation broke layout assertions and dropped a vocabulary match
root_cause: Filesystem scans and exclusions assumed historical logical paths were physical roots
resolution_type: fix
---

## Problem

Moving the Rust workspace to model/ and freezing Scala under model/archived/ broke Go layout assertions and changed scanner coverage. A scanner tied to the old physical root also inspected Rust files and archived build caches.

## Solution

Keep a separate physical Scala root while preserving historical logical model/... paths for namespace checks. Assert the new Rust and archived Scala directories independently. Adapt the archived-path vocabulary branch and retain red/green controls for cache exclusion, Rust decoys and the previously lost vocabulary match. The final diagnostic set exactly matches the pinned baseline's 922 locations.

## Prevention

For source moves, compare both failing-test names and normalized diagnostics against the exact pre-move tree. Equal failure counts do not prove equal coverage. Keep negative controls that distinguish physical filesystem roots from logical source names; baseline failures remain failures rather than being skipped to claim a green package.
