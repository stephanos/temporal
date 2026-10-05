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
TBD

## Evidence
- Commits:
- Tests:
- PRs:
