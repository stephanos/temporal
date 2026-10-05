---
satisfies: [R10]
---
# fn-105-gomad-follow-ups-deferred-scope.10 D10: downstream-seam guide

## Description

Origin: fn-104 R4. Deferred 2026-09-29 pending another downstream adoption. Revived 2026-09-30 by fn-107 R12, which requires a reusable guide for the completed integration. Follow fn-105.9. Write the generic Gomad-side seam guide without downstream repository or component names; cover paired tags, native defaults, injected resources, closure/linked analysis, exact dependency policy, GOWORK=off replacements, CLI driver, qualification/replay and external-service limits. Add concrete downstream instructions alongside its profile, manifest and packs. Files/Touches: tools/gomad3 documentation and downstream localcell/gomad README. Quick: check commands against retained successful execution/pack evidence, documentation link checks.

Deferral (2026-10-04): fn-107 now closes its implementation scope only.
Its original qualification requirements remain the acceptance reference, not
passing evidence. This task stays open under its unchanged acceptance; references
to fn-107.5 consuming/reconciling evidence describe the pre-closure workflow.
D8 retains adapters and supported analyses; D9 retains final consumer/source/native
reconciliation and reviews, packs/driver and both-platform workflow/exact replay;
D10 retains qualification-bound guidance and documentation. Resume when the
downstream checkout and qualified hosts are available. No qualified downstream
support claim or D12 waiver follows from the fn-107 closure.

## Acceptance

Revival is recorded. Generic guide satisfies original fn-104 R4 without naming downstream repositories/components. Consumer guide gives concrete repeatable commands and bounded support claims matching measured dual-platform success. No commands rely on undocumented local edits or treat classified failures as passing. fn-107.5 reconciles the roadmap after consuming this guide.

## Done summary

TBD

## Evidence

- Commits:
- Tests:
- PRs:
