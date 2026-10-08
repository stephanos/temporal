---
title: Recompute mapping summaries after final row classification
date: "2026-10-08"
track: bug
category: integration
module: tools/gomad3 test-preservation evidence
tags: [gomad, mapping, preservation, derived-summary]
problem_type: data
symptoms: Detailed mapping rows and aggregate status counts disagree
root_cause: Object spread retained a derived aggregate after replacing its input rows
resolution_type: fix
---

## Problem

The fn-112.9 final mapping report replaced its detailed rows after classifying failed adapter ancestors, while object spread retained the earlier counts_by_actual_status summary. All 282 rows remained present, but the summary counted 104 children under not observed in bounded focused selection even though their actual status was NOT REACHED; ancestor FAIL. The contracts reviewer found this R10 reporting inconsistency in final-links.mjs at line 20.

## What Didn't Work

Preserving the original report fields with {...map, rows:finalRows, ...} also preserved a derived aggregate whose inputs had changed. Row counts, source hashes, and executable preservation controls alone did not check agreement between the detailed rows and the summary.

## Solution

The conductor retained current-mapping-corrected.json under a fresh name and recomputed status totals from every final row. mapping-summary-correction-proof.json proves that all 282 rows and every other field remain exact, and that the original frozen report is unchanged. mapping-summary-red rejects the stale original summary; mapping-summary-green accepts the corrected report. Three negative controls reject the stale aggregate, a changed row status, and a missing row. Commit da10d3bdad0c10b844832fcc9ebb318ebdd936a8 carries the correction. The actual Codex re-review response marks the original finding fixed.

## Prevention

After finalizing mapping rows, derive every status aggregate from those rows and assert exact equality with a fresh aggregation. Check the expected row count and blocked-child count together with the status total. Keep the original frozen artifact intact and bind each corrected report to its original hash, so a reporting repair cannot silently rewrite historical evidence.
