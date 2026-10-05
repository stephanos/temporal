---
satisfies: [R9]
---
# fn-108-gomad-reduce-code-size-without-removing.8 Record owner waiver of the linux/amd64 gates

## Description
The owner's explicit request on 2026-10-04, "close fn-108 without linux check",
waives the outstanding linux/amd64 execution requirement of R9 for fn-108 only.
Record the waiver rather than execute the former Linux gate checklist.
Tasks 1–7 retain their accepted darwin/arm64 evidence in
`.flow/artifacts/fn-108-gomad-reduce-code-size-without-removing/final.md`.
The original Linux check was not run and is not claimed to have passed.
D12, other specs' platform gates, baseline dispositions, and support claims
remain unchanged.

## Acceptance
- Retain the explicit owner waiver and amend fn-108 R9 to exempt only its outstanding linux/amd64 execution requirement.
- Preserve existing darwin/arm64 evidence and the original final report; state that Linux validation was waived, not performed or passed.
- Close this task and fn-108 under the amended scope, remove the completed spec from MILESTONES.md, and leave D12 and other specs' gates unchanged.

## Done summary
Closed under the owner's explicit waiver: "close fn-108 without linux check".
The outstanding linux/amd64 R9 execution requirement is waived for fn-108 only.
Existing accepted darwin/arm64 evidence and the historical final report are retained.
No Linux checks were executed or claimed to pass; this is scope acceptance, not
native Linux qualification. D12 and all other specs' Linux gates remain unchanged.

stage: plan-sync - skipped(config: planSync.enabled != true)

## Evidence
- Commits: 1610deb3852f7b8070bdae000070449ce422a092
- Tests: flowctl validate --spec fn-108 --json (valid, 0 errors, 0 warnings), git diff --check (pass; documentation and tracking only; no Linux execution)
- PRs:
