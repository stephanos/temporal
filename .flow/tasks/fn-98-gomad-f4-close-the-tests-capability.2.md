---
satisfies: [R1, R2]
---
# fn-98-gomad-f4-close-the-tests-capability.2 Close the ./tests closure on darwin/arm64

## Description
Run closure analysis of `go-test ./tests` with tags disable_grpc_modules,gomad,test_dep; add a darwin counterpart of temporal-functional-tests-linux-amd64 with exact facts if needed.

## Acceptance
- zero unsupported_target findings on darwin

## Done summary
Observed on darwin/arm64 at a861e0dbb3 that closure analysis of `go-test ./tests` with tags disable_grpc_modules,gomad,test_dep reports `supported` with zero blockers and no `unsupported_target` findings over 1043 packages. The selected packs are modernc-libc-xsys-v047, modernc-libc-xsys-v047-isatty-v021, reflect2-go126, temporal-functional-compute-darwin-arm64, and temporal-functional-tests-darwin-arm64. `temporal.json` names no packs: a pack binds to the workload its request names, and the pack qualification set is the per-platform `COMPATIBILITY_PACK_QUALIFICATIONS` list in tools/gomad3/Makefile. The darwin tests pack is already in that list, placed the same way as the linux one. The missing piece was the CI assertion, so the darwin `core` job in .github/workflows/gomad3.yml now runs the same closure step as the linux job and asserts on the darwin pack. The observation is recorded in the F4 status in .plans/GOMAD_MILESTONES.md.

baseline: green (make -C tools/gomad3 validate compatibility-pack-qualification)

stage: impl-review - ran [codex fan-out, 3 draws SHIP .. finalize SHIP]
## Evidence
- Commits: 98f47614c239feeaeee0a51b69cda81f93b6ce2f
- Tests: make -C tools/gomad3 validate compatibility-pack-qualification, tools/gomad3/.bin/gomad analyze --capability-mode=closure --format=json --timeout=15m --build-tag disable_grpc_modules --build-tag gomad --build-tag test_dep go-test ./tests (supported, 0 blockers, 0 unsupported_target, 1043 packages), jq darwin closure assertion from .github/workflows/gomad3.yml core job
- PRs: