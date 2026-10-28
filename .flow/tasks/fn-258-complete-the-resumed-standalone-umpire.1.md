---
satisfies: [R1, R2]
---
# fn-258-complete-the-resumed-standalone-umpire.1 Freeze renewed source and reconcile candidate ownership

## Description
**Touches:** [.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/**, .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/history/**]

Freeze ac60daaf47 framework/archive manifests independently of fn-257 inventories. Inventory current integrated delta and every candidate/ref, preserving original recovery records. Produce a new isolated history projection and independent verification against the renewed manifest; extend the commit map without altering the original export. Preserve authors, dates, messages and retained parent edges; every pruned source commit must be projection-neutral.

## Acceptance
- [ ] New inventory covers every selected blob, mode, size and SHA256 at the fixed revision.
- [ ] Existing inventories and recovery evidence remain unchanged.
- [ ] All original candidates and current refs have explicit recovery/ownership, known task links and unresolved actions; no provisional entry is labelled transferred.
- [ ] The independently verified refreshed history matches every renewed-manifest path and covers all available source commits, including c88fdc5506..ac60daaf47.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
