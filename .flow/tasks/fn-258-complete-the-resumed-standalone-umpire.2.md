---
satisfies: [R2, R3]
---
# fn-258-complete-the-resumed-standalone-umpire.2 Refresh portable standalone from integrated framework source

## Description
**Touches:** [../umpire/**, .flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/**]

Transfer the integrated framework delta c88fdc5506..ac60daaf47 onto portable f7b42ce. Join task 1's verified filtered history as ancestry with the reviewed active tree unchanged, preserving the original export and all adaptation history; record source/export/receiving commit maps. Preserve canonical Go identities and independent-repository quick selection. Task 2 owns only standalone quick code; task 3 exclusively owns the Temporal quick caller. Regenerate Go descriptors from owning schemas, never hand-merge them. Run focused portability, boundary and generation checks.

## Acceptance
- [ ] An explicit per-path adaptation inventory against the ac60 manifest reports zero missing source files, zero unexpected modifications/additions outside justified allowlists, zero protected payload/snapshot-metadata/lock/LICENSE differences and no mode drift.
- [ ] All source deletions and additions since c88 are listed and verified; retired paths do not reappear.
- [ ] Every expected difference names identity rewrite, regenerated descriptor, quick-selection, portability or reviewed boundary-policy rationale.
- [ ] The commit map covers every available framework-touching commit through ac60 and both histories remain reachable from the portable tip without tree changes at the join.
- [ ] Focused checks and protobuf freshness pass; semantic fixtures remain exact.

## Done summary
Portable standalone05473210 is source-reviewed SHIP on all three independent Claude Fable5.1 high lenses. Exact adaptation inventory passes1189mapped/1130unchanged/59justified/499protected with zero missing/unexpected/mode differences;133 additions present122 deletions absent,54 historical adaptation claims and14 refusal controls verified. Both original/export and portable histories remain reachable through tree-preserving joinc71c309; receiver054 changes exactly the two approved gate paths afterac79114. Current core/Kettle protobuf freshness and real source-boundary receipts pass with clean source. Required tests supervise existing TLC setup and preserve failure;30 wiring controls, warnings-denied clippy and actual TLC regression pass. Fresh full standalone makevalidate passed18parent17child; recovery-only verification158objects7979members passes. Gate1679.762s/inner24.162s soft targets missed; validation-cost assessment retained. Metadata-only portability category clarified after review and exact verifiers rerun; narrow CLI guide sentence remains with task5 final docs. Task4 exact named release/paired/local-edit proofs and task5 publication/receiving/cutover remain open. No source removal or remote publication.

stage: impl-review - ran [2026-10-28 final054 round; all3 SHIP] (model: claude-fable-5-1 high)
stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 7c6d9fdcbd896b4903720350ae6706e32d203a5d, a4e97c8e16e71f5cea515910fe692b738d31a366, e4cf57dd013b6c86cc9d6e0e27984005608ad4ba, 1fb713f6fcdcab8f1078d52f363d948a195d65b2
- Tests: make validate at standalone054;18parent17child success; validation/standalone-0547321, verify-adaptation.py:1189mapped499protected14controls; verify-history-join.py:exact trees/parents/current054 ancestry, core/Kettle protobuf --check and source-boundary current054;30gate wiring+clippy+realTLC regression, verify-recovery.py:158objects7979members PASS without original caches
- PRs: