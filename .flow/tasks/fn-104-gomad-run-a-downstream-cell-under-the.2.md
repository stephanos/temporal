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
Downstream-owned packs never enter this repository: the authoring commands (discover, review, generate, check, qualify) take --compatibility-root=/absolute/dir with the internal/compatibilitypack layout, and GOMAD3_COMPATIBILITY_PACKS=/absolute/dir/packs loads those packs next to the embedded ones for every command, including the Runner's supervisor and coordinator processes. They pass the same strict validation (an external pack admitting os/exec is rejected with 'capability import:os/exec is never admitted'); ID collisions with embedded packs, relative or missing directories, and misnamed or non-pack entries fail closed. Selected pack IDs and digests are already part of the target identity, so VerifyIdentities fails when a recorded external pack is no longer available (tested). Implemented as an environment variable, like GOMAD3_TOOLCHAIN_DIR, rather than a per-command flag, so every subprocess sees it. A follow-up fix restored root-relative review output when no --compatibility-root is given.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 501b775ff, 4bd232127
- Tests: go test ./internal/compatibilitypack -run External, go test ./cmd/gomadtool -run CompatibilityPath
- PRs: