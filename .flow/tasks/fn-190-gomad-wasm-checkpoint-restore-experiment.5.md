---
satisfies: [R4, R6]
---
# fn-190-gomad-wasm-checkpoint-restore-experiment.5 Research finding and go/no-go

## Description
Write docs/research/gomad/<date>-wasm-checkpoint-restore.md: what the image needed beyond linear memory, every mismatch and cause, the measurements, and go/no-go. If go: the product design for a checkpoint-backed forced-prefix frontier (one image per prefix, provenance bound to prepared module and helper identities, retained failures fresh-replay verified). If no-go (including the .1 kill switch): the state that cannot be captured and the cheapest alternative. Link from docs/research/gomad/README.md and .plans/GOMAD_CMP.md. Confirm the dormant path left cooperative evidence unchanged on the integrated candidate. Depends on .1 only, so it also runs on the no-go path.

## Acceptance
Report exists and is linked; disposition recorded in the spec; cooperative qualification unchanged on the integrated candidate; unrun R2–R5 acceptance is stated as not met, never implied.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
