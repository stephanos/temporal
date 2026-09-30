---
satisfies: [R2]
---
# fn-104-gomad-run-a-downstream-cell-under-the.2 Accept reviewed compatibility packs from an external, digest-bound directory

## Description
Downstream-specific packs name downstream modules and must not be committed in this repository. Add an explicit --compatibility-packs DIR (and the qualify-set manifest equivalent) that loads strict gomad3.compatibility-pack/v2 packs from a directory whose canonical content digest is recorded in Campaign, Artifact, qualification, and plan identity; replay and resume fail closed when the digest differs. Embedded packs stay the default and take precedence on conflict; an external pack may not admit remain_unsupported capabilities, same validation as embedded packs. The discover/review/generate flow writes to that directory when asked.

## Acceptance
- a campaign with an external pack directory records its digest; replay with a changed directory fails closed
- an external pack admitting os/exec, os/signal, os/user, plugin, or runtime/cgo is rejected
- no downstream module path is committed in this repository


## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
