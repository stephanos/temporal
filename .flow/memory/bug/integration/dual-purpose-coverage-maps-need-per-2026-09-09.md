---
title: Dual-purpose coverage maps need per-entry admissibility
date: "2026-09-09"
track: bug
category: integration
module: model/Umpire/Observation/Projection/Coverage.lean
tags: [lean, coverage, observation, lowering]
problem_type: integration
symptoms: Evidence replay failed on coverage entries only the portable lowering could use
root_cause: Rebuildability was validated per clause but applied per whole map at replay
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/check-unbounded-lean-numbers-before-2026-09-07, bug/integration/coverage-findings-must-retain-2026-09-05, bug/integration/homogeneous-owner-index-blocked-mixed-2026-09-09, bug/integration/joint-conflict-scopes-require-realized-2026-09-05, bug/integration/keep-raw-semantics-behind-checked-input-2026-09-05, bug/integration/keyed-capture-operands-must-be-checked-2026-09-09]
---

## Problem
A coverage map that binds modeled field coordinates to declared Observations was reused for two
different jobs: naming a retained value in the portable lowering, and rebuilding an admitted
projection for evidence-driven model replay. Request-rooted coordinates are legitimate for the
first and impossible for the second, but replay iterated every stored entry, so a lowering-only
request mapping made an unrelated Run fail with a coverage error at its first event.

The same map also supported only a bare field chain. Steps the checked field language already
admits -- a keyed map lookup in particular -- were rejected as "unsupported coverage step", so an
authored operand that evaluated fine could never pass the coverage gate, and the single generic
diagnostic hid which of several very different reasons applied.

## What Didn't Work
Treating "not rebuildable" as a whole-map property and validating it only where a clause reads a
field. Admission checked the fields a clause actually names, but the rebuild loop had no such
filter, so the two boundaries disagreed.

## Solution
`CoverageEntry.rebuildable` became a per-entry predicate, and `Coverage.evidence` skips entries that
are not rebuildable instead of failing on them
(`model/Umpire/Observation/Projection/Coverage.lean`). Keyed map lookup was added to both the
rebuild and the Program-construction comparison, and the three steps that genuinely cannot be
rebuilt -- presence, repeated element, cardinality -- now reject with a reason naming why: the
payload witnessing them would carry data no declared Observation reported.

## Prevention
When one checked map serves two consumers with different admissibility rules, make the rule a field
on the entry rather than a check at one consumer's door, and add a regression that runs the second
consumer under a map containing an entry only the first can use.
