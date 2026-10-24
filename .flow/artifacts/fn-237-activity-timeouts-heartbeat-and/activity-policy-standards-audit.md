## Quality Audit — Standards axis: fn-237.3

### Summary

- Files changed: 38, including four untracked additions · Should Fix 1 · Consider 0 · Blocking: none possible (standards axis).

### Should Fix

- **model/README.md:456** (Conf 100): This diff crosses the 1,000-line threshold, from 999 to 1,045 lines (executed: current line count plus diff numstat) — move the detailed activity-policy walkthrough into a feature README and retain a concise overview/link.

### What's Good

- Walked: policy and proof tests exercise distinct cancellation, countdown, fairness and replay obligations; their volume serves explicit acceptance criteria.
- Walked: compatibility projection validates complete state and exact facts before erasure; live changes remain limited to constructor compatibility.

No builds or tests run. Final gates remain pending as documented. Repository files were left untouched; the read-only auditor contract prevents saving the requested `.flow/tmp` report.
