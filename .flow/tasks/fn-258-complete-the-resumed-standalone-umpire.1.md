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
Renewed source inventory and candidate ownership at ac60daaf47, preserving the original fn-257 inventories. All1,189 selected records agree with independent Git archive/blob/mode comparison; all81 original candidates and observed refs have explicit recovery/ownership and unresolved dispositions, with zero transfers claimed. Refreshed isolated history export a92db061 retains627 raw metadata/tree/parent identities and6,634 projection-neutral pruned snapshots; all original7,149 map rows remain unchanged. Two fresh corrected exports and independent bundle recovery reproduce exactly. Exact graph/map and unique source-frontier guards reject old-pass same-tree/foreign-parent injections. Current map file inventory,15 ancestry commands and6 representative histories reproduce at ac60. Source/worktree isolation controls hit their intended guards; all39 source worktree preservation captures and56 dirty paths remain intact. Ledger handles dirty standalone namespaces, missing original objects and orphan candidates through explicit control/report paths. R6 transfer/publication and current runtime gates remain open in later tasks.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 05051b84a18f29e8cafbfa6b5ac57981a2aee72f, ba8ede3a6a078d3ede43245a7764c081a5daa1d6, e4c4642cdc760bd42e1ec29ef422bb814c924694
- Tests: python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py, python3 .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/source/candidate-ledger.py --controls-only, Independent verify-export.py full/bundle verification,627 retained+6634 pruned; exact commands in history/tool-provenance.json and current proof artifacts, checkpoint-controls.py old-pass/new-fail unmapped-tip and foreign-parent controls, verify-history-provenance.py all15 definitions,6 representative histories,current1189-file inventory, git diff --check
- PRs: