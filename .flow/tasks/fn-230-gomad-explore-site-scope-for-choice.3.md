---
satisfies: [R11, R12]
---
# fn-230-gomad-explore-site-scope-for-choice.3 Goroutine creation sites in the choice wire schema and runnable-decision scoping

**Touches:** tools/gomad/internal/model/choice/internal/wire/**, tools/gomad/internal/model/volume/internal/hashwire/wire_generated.go, tools/gomad/internal/model/choice/schema/**, tools/gomad/internal/model/runtimesource/**, tools/gomad/internal/backends/wasm/toolchain/**, tools/gomad/internal/backends/wasm/wasmhost/src/model/**, tools/gomad/rust/gomad/src/campaign/exploration/**, shared Go/Rust choice corpora
**Semantic references only:** tools/gomad/internal/model/choice/schema/choicewire.json, model/runtimesource/gomad_semantics.go.txt, backends/wasm/toolchain/overlay.go, wasmhost/src/model/imports.rs, model/choice corpora, runner/internal/exploration/choice.go

## Description
Implement once on the Rust host after fn-232.8 and the existing site-spec predecessors; the Go locations below document existing semantics only. Port the behavior into the Rust ownership surface above and extend the shared golden/behavioral corpus. Do not create or extend a Go-host implementation.
Extend the goroutine identity material in the runtime overlay with the creation site offset (from the PC already passed to `gomadChoiceAssignGoroutineIdentity`), add a creation-site record or field to the choice wire schema as a new version, regenerate codecs with `make generate`, update the Rust decoder, and regenerate corpora with the expectation diff reviewed. In exploration, map runnable-decision alternatives to creation sites and treat a runnable decision as in scope when any alternative's creation package matches. Traces of the previous schema version replay and inspect unchanged and have no creation sites for scoping.

## Acceptance
Schema version bump with regenerated Go and Rust codecs; corpora byte-identical on second regeneration; previous-version traces still replay exactly; a scoped campaign forces runnable decisions whose goroutines were created in scope and skips the rest, with counts; model and helper identity rotation recorded for requalification.

## Done summary
R11/R12 accepted. Runtime overlay records creation sites in declared v4 metadata while preserving the original decision identities and tape projection. Runnable decisions match ANY alternative creation package before rank enumeration. Genuine mixed-package fixture evidence covers4 mixed runnable decisions; public forced ordinals3/4 exercise both select and runnable, with2 admitted alternatives and8 skipped.

Two fresh make generate/validate-protocol plus actual replay-plan corpus regenerations pass and produce identical aggregate c99ad4bfe525e4247c41143008a9553326b9d7558941b2848a80f2771a885fff across3 generated files and all replay-plan JSON filenames/bytes. Exactly3 new corpus cases were added; historical expectations are unchanged. Previous-version compatibility passes Go TestPreviousVersionTraceRemainsReadable, helper runtime_creation_sites_preserve_alternative_identities_and_exact_replay and replay_plan_go_corpus. Old ABI1/raw-v3 trace has no creation sites and exact replay preserves its bytes.

Root's post-change f30269e6 reader additionally inspects the genuine retained Rust-v3 artifact under fn232-replay/stable-reference2 with profile gomad-choice-trace/v3 and3 sites: workspace-parent .build-cache/fn230-site-scope/v3-public-inspect-postchange.json. Historical identity-matching Go v3 exact-replay/inspect receipts remain under /home/agent/wt/evidence/fn230-legacy-symbolization/fixture-proof after transfer from closed fn229; no cross-helper artifact identity migration or relabeling is claimed.

Public03/04 each pass34 checks with identical observations. Independent site_scope_review inspected source and authenticated real journals/artifacts with no blocking findings. Root Go choice/generator/toolchain/determinism packages pass. Integrated734 full helper tests/alltarget Clippy and host feature62tests/alltarget Clippy all exit0, authenticated by /home/agent/wt/evidence/fn232-replay/integrated734-source-freeze.*.

stage: impl-review - ran (fresh site_scope_review, same-family session-model fallback)

Declared runtime/model/helper identity rotation remains subject to final frozen Rust whole-system qualification under fn232.13. Task1's public plan seam remains open. This is not a qualification or whole-spec completion claim.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: 38572d7e49, 97ac127b01, 734bc7c566
- Tests: two fresh make generate/validate-protocol and replay-plan corpus generations: exit0; identical c99ad4bfe525e4247c41143008a9553326b9d7558941b2848a80f2771a885fff, previous-v3 Go trace decoding, helper exact old-trace replay and replay-plan corpus: exit0; no old corpus expectations changed, actual post-change public inspect of retained Rust-v3 artifact: profilev3 and3 sites, publicscope03/04:34checks each and byte-identical observations, integrated734 helperfulltests/alltargetClippy and host feature62tests/alltargetClippy: all exit0, source-freeze0, independent site_scope_review: R11/R12 supported, no blocking findings
- PRs:
