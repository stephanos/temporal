---
satisfies: [R11, R12]
---
# fn-230-gomad-explore-site-scope-for-choice.3 Goroutine creation sites in the choice wire schema and runnable-decision scoping

**Touches:** tools/gomad/internal/model/choice/schema/**, tools/gomad/internal/model/runtimesource/**, tools/gomad/internal/backends/wasm/toolchain/**, tools/gomad/internal/backends/wasm/wasmhost/src/model/**, tools/gomad/rust/gomad/src/campaign/exploration/**, shared Go/Rust choice corpora
**Semantic references only:** tools/gomad/internal/model/choice/schema/choicewire.json, model/runtimesource/gomad_semantics.go.txt, backends/wasm/toolchain/overlay.go, wasmhost/src/model/imports.rs, model/choice corpora, runner/internal/exploration/choice.go

## Description
Implement once on the Rust host after fn-232.8 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Extend the goroutine identity material in the runtime overlay with the creation site offset (from the PC already passed to `gomadChoiceAssignGoroutineIdentity`), add a creation-site record or field to the choice wire schema as a new version, regenerate codecs with `make generate`, update the Rust decoder, and regenerate corpora with the expectation diff reviewed. In exploration, map runnable-decision alternatives to creation sites and treat a runnable decision as in scope when any alternative's creation package matches. Traces of the previous schema version replay and inspect unchanged and have no creation sites for scoping.

## Acceptance
Schema version bump with regenerated Go and Rust codecs; corpora byte-identical on second regeneration; previous-version traces still replay exactly; a scoped campaign forces runnable decisions whose goroutines were created in scope and skips the rest, with counts; model and helper identity rotation recorded for requalification.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
