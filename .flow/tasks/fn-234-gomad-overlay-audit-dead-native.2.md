---
satisfies: [R2, R3, R4, R5]
---
# fn-234-gomad-overlay-audit-dead-native.2 Split inspection code out of the wasip1 runtime overlay with the declaration and behavioral gates

## Description
Touches: tools/gomad/internal/backends/wasm/toolchain/{runtime_wasm.go.txt,runtime_inspect_wasm.go.txt (new),overlay.go,overlay_test.go,debug_test.go}, tools/gomad/internal/backends/wasm/README.md.
Before editing embedded sources or refreshing caches, retain the pre-split revision, patched-runtime declaration list, actual compiled modules with hashes, installation/configuration bytes and identities, compiler/helper/host identities, arguments, seeds, profiles and bounds for every required workload. Keep this separate from the immutable fn-232 baseline and authenticated extracted source.
Move every gomadDebug* declaration (state, buffers, gomadDebugMain, gomadDebugFindRunnable, gomadDebugAllowed, dump/probe plumbing) and nothing else into a new embedded source with the same //go:build wasip1 && wasm constraint, copied beside the runtime file into the patched tree; keep names, types and the rewrite table unchanged. Add the new source to ImplementationSHA256 ordered input; publish and verify it like the others. Add a toolchain test that lists sorted top-level declarations (name, kind) of the patched runtime package before and after the split and asserts equality; keep the before list as a golden.
Refresh preparation and compile caches for the new identity only after pre-split custody. Retain post-split modules/configurations separately, and freshly execute both versions on the same fixed host/helper/compiler with matching workload arguments, seeds, profiles and bounds. The workloads are the toolchain choices fixture, helper feature cases and TestFrontendSystemInfo through the Temporal installation. Retain raw results and each version's own identity-matching replay evidence; compare outcome class/termination, first-divergence ordinal/reason and failure-group partitions over paired workload/seed labels. Missing inputs/executions or semantic differences fail. Keep actual identity/hash differences; do not normalize them into equality. Conformance corpora stay unchanged and make validate passes.
Do not substitute Go/Rust agreement on the new overlay for this before/after gate. The fn-232 host-parity lane keeps its frozen baseline-source argument and source/archive hash checks, never the edited checkout as baseline; no baseline replacement or guard weakening. Coordinate integration before final qualification, since earlier reports cannot qualify the new overlay identity. Document the two files, the .go.txt rationale and the identity change.
## Acceptance
Declaration-list test passes with the retained pre-split golden; runtime file contains no gomadDebug declarations and the inspection file contains only them; ImplementationSHA256 changes once and includes the new source; stale installation refused by the existing identity check. Authenticated pre-split modules/configurations and workload inputs were captured before edits/cache refresh; corresponding post-split inputs are retained separately. Fresh executions of both versions, on the same fixed host/helper/compiler with matching arguments/seeds/profiles/bounds, match outcome class/termination, first-divergence ordinal/reason and failure-group partitions; each version retains actual identity bindings, raw results and its own replay evidence. Missing inputs/executions or differences fail. Conformance corpora unchanged and make validate green. The frozen fn-232 baseline, extracted source and source-authentication checks remain unchanged; one-overlay host parity is not substituted for the before/after comparison. README names both files, the .go.txt rationale and the identity change; the done summary records the commands, seeds and comparison.
## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
