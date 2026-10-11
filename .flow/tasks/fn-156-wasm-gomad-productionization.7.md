---
satisfies: [R7]
---
# fn-156-wasm-gomad-productionization.7 Optimize supported profiles and deliver backend documentation

## Description
Original fn-151 Phase 10: re-measure the corrected main-based artifact, admit tested provider exclusions and immutable engine-code caching, document install/discover/replay/support and measured costs. Native replacement remains a separate owner decision.

The provisional delivery surface adds an opt-in WASM command adapter through the existing Gomad command and Runner owners, preserving native defaults and grammar. Installation/discovery verifies pinned helper/compiler/runtime/model inputs; retained replay must run without original sources or compiler access. Advertised profiles remain bounded by actual integrated qualification; network/storage and cost claims stay pending until their owning gates pass. Do not infer a checkpoint or provider exclusion without its explicit safety/identity/native-default gate.

**Touches:** `tools/gomad3/cmd/gomad/**`, `tools/gomad_wasm/cmd/**`, `tools/gomad_wasm/installation/**`, `tools/gomad_wasm/CLI.md`

README/Makefile integration is serialized with the helper owner. Final admission follows `.6`. Gates include installation/path/identity negative tests, native command characterization/architecture checks, an actual clean-install guest failure discovery/inspect/fresh replay, and final measured support/cost documentation.
## Acceptance
Original fn-151 Phase 10 exit gate.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
