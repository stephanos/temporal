---
satisfies: [R1, R2, R3]
---
# fn-96-gomad-f2-close-the-go127-port-on.1 Run the darwin upgrade dossier and review the go1.26.4 to go1.27.1 boundary diff

## Description
Run `make -C tools/gomad3 upgrade-dossier GOMAD3_BASELINE_REF=<last go1.26.4 commit>`, review each boundary diff entry, rerun with GOMAD3_APPROVED_BOUNDARY_DIFF_SHA256 set to the reviewed digest, and record the digest and reviewed entries.

## Acceptance
- dossier reports every non-root gate passed
- boundary diff empty or approved by recorded digest

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
