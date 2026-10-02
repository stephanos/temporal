---
satisfies: [R5, R6]
---
# fn-104-gomad-run-a-downstream-cell-under-the.6 Measure the downstream smoke test and record the result

## Description
Run gomad analyze (linked) and gomad qualify --repeat 2 on seeds 11 and 17 from the downstream checkout against this branch's toolchain, with the downstream pack in an external directory and any downstream seams applied as local, uncommitted scratch edits that are reverted afterwards. Classify every non-qualified outcome (capability blocker, unmodeled operation, watchdog, evidence divergence) with its finding. Record the result in MILESTONES.md F9 and GOMAD_CLOUD.md without naming downstream components; update README/ARCHITECTURE for downstream targets.

## Acceptance
- the smoke test qualifies with exact replay or every outcome is classified with a finding
- the downstream checkout is clean afterwards; no downstream names in committed docs


## Done summary
Measured on 2026-09-30 from the downstream checkout against this branch's toolchain on darwin/arm64, in linked mode, with a scratch server replace reverted afterwards and no downstream source change. Live blockers: 78 at the baseline, 67 with the address-library adapter and the standalone x/sys pack, 44 with a downstream-owned external pack (23 facts in 17 packages) authored through discover/review/generate/check against --compatibility-root in /private/tmp and loaded with GOMAD3_COMPATIBILITY_PACKS; nothing downstream-specific was committed. The remaining 44 are downstream-owned: 10 remain_unsupported (five of the downstream's own subprocess and signal sites, five cloud credential chains through its CLI package and blob-store provider) and 34 model_operation, each with a C4 disposition and injection point. gomad qualify --repeat 2 --capability-mode=linked classifies seeds 11 and 17 as unsupported_target (capability blocker) at the first live boundary, a credential chain's os/exec import, which meets R5's classified branch. Recorded in GOMAD_CLOUD.md "Measurement" and the F9 status without downstream names; ARCHITECTURE.md gains "Targets outside this repository" and the README already documents --working-dir, the forced environment, and external packs (R6).

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 417675e69
- Tests: gomad analyze --capability-mode=linked (downstream cluster test): 78 -> 67 -> 44 live, gomad qualify --seed=11/17 --repeat=2: unsupported_target
- PRs: