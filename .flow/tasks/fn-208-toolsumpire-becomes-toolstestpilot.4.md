---
satisfies: [R4]
---
# fn-208-toolsumpire-becomes-toolstestpilot.4 Amend SPEC, Vision, READMEs, module map, measurement note and MILESTONES; close fn-196.4 and R5 by reference

## Description
**Owner decision 2026-10-18:** amend the SPEC and Vision Go-replay clause to native replay through Checker.replay plus TLC reachable-set agreement (fn-207 R4) as the independent reading.

Replace the Go-replay clauses in the SPEC and Vision with `Checker.replay` under the owner's authorization; rewrite the tools README, the module map, the model README pipeline section and the measurement note's obligation table; set MILESTONES' Direction to one language reads Models; close fn-196.4 and fn-196 R5 by reference to this spec.

**Files:** .plans/UMPIRE4_SPEC.md, .plans/UMPIRE4_VISION.md, .plans/UMPIRE_MODULES.md, tools/testpilot/README.md, model/README.md, model/design/scala-checker-measurement.md, MILESTONES.md, .flow/specs/fn-196-*.md.

**Quick commands:** `grep -rn 'tools/umpire' --include=*.md .` shows only historical notes.
## Acceptance
- Docs amended; fn-196.4 closed with a reference; Direction updated.

## Done summary
Closed without implementation on 2026-10-24 at the owner's request. The Scala framework this task targets was frozen under `model/archived/` on 2026-10-18 and Umpire continues in Rust (`model/umpire`); the Scala-era backlog is retired rather than carried. Any intent here that still matters is re-filed against the Rust workspace when it is needed.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits:
- Tests:
- PRs: