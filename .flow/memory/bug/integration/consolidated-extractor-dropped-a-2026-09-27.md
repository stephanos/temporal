---
title: Consolidated extractor dropped a caller's nonempty-field rejection
date: "2026-09-27"
track: bug
category: integration
module: common/testing/testpilot/temporal/driver.go
tags: [testpilot, refactor, validation]
problem_type: integration
symptoms: empty binding field returned worker.ErrInvalid instead of composite ErrInvalid
root_cause: descriptor extractor rejects absent fields only; old typed path also rejected empty values
resolution_type: fix
related_to: [bug/integration/behavior-neutral-refactors-must-not-2026-09-04, bug/integration/moved-conformance-tests-must-not-import-2026-09-06, bug/integration/test-output-parsers-must-accept-tlog-2026-09-27]
---

## Problem
Replacing the composite Driver's Marshal/Unmarshal binding read with delivery's descriptor extractor dropped the old path's nonempty-field check: the extractor rejects absent message fields but returns empty strings as values, so an empty workflow ID fell through to CreateCarrier and surfaced worker.ErrInvalid instead of the composite ErrInvalid.

## What Didn't Work
Assuming the two extraction paths validated the same thing because they read the same fields.

## Solution
`carrierBinding` in common/testing/testpilot/temporal/driver.go wraps `delivery.StartBinding` and keeps the composite's complete-binding check; a table test pins empty and absent fields.

## Prevention
When collapsing two extraction paths into one, diff their rejection sets (absent vs empty vs oversized) per caller and pin each caller's error identity with a table test before deleting the old path.
