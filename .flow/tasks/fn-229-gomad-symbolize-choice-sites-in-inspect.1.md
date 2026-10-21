---
satisfies: [R3, R4, R7, R8]
---
# fn-229-gomad-symbolize-choice-sites-in-inspect.1 Symbolize package: locate pclntab in the retained module and resolve text offsets

**Touches:** tools/gomad/rust/gomad/src/evidence/symbolize/**, tools/gomad/rust/gomad/src/evidence/mod.rs, tools/gomad/internal/backends/wasm/wasmhost/src/lib.rs (existing helper public re-export surface), tools/gomad/testdata/parity/**
**Semantic references only:** tools/gomad/internal/model/runtimesource/gomad_semantics.go.txt, tools/gomad/internal/backends/wasm/toolchain/{overlay.go,runtime_wasm.go.txt}; the Rust symbolizer paths above are proposed implementation surfaces.

## Description
Implement R3/R4/R7/R8 once in the Rust evidence layer after fn-232.5. Register the symbolizer in evidence/mod.rs and expose Wasmtime's existing wasmparser through the helper library's public re-export surface in lib.rs (reuse an equivalent existing public re-export if present). The host gomad crate consumes that helper re-export with the checked pinned Go metadata reader; it gains no direct Wasmtime or new parser dependency. Provide a Rust-internal resolver over a site value plus its missing flag and the authenticated retained-module handle; preserve the spec's Symbol shape and unavailable reasons. No Go-host implementation, Go decoder subprocess or new dependency.

## Approach
- Decode active constant-offset data into the bounded sparse view; handle split tables and zero gaps without whole-memory allocation. Locate a unique structurally validated pcHeader/moduledata pair, validate sorted functab/sentinel and names, and follow checked gofunc-relative funcdata/inline and PC-value tables. Never assume named runtime DATA symbols or use the zero header textStart word.
- Interpret compressed textOff at physical-function scale; validate text base and imported-function/defined-ordinal mapping. Derive exact Go names from pclntab; cross-check a present name section only after pinned ASCII-word/dot normalization. A missing required name cross-check means no_symbols even with retained pclntab; a present disagreement/malformed table means symbol_mismatch, under the spec's precedence.
- Recognize the complete pinned selectgo call-emission grammar from the retained function body, including every possible direct/indirect/tail transfer and dead branch. Associate concrete stack return-address stores with the actual target; check parent and textOff binding. Decode real return-PC minus one. Require complete nonempty unanimous chains for inline output and separate location consensus; unknown/conflicting patterns fall back physically. Never infer candidates from nearest constants or discard disagreement.
- Implement all fixed byte/entry/step/depth/name limits and cumulative checked counters from the spec before allocation/traversal. Exercise equal-to-cap, over-cap, sparse logical ranges, counter/address overflow and cyclic inline metadata.
- Retain the pinned-toolchain source, exact flags, module digest, compiler evidence of actual inlining and expected metadata/decoded call sites with the real R8 fixtures. Prove the positive wrapper reaches instrumented selectgo with a complete singleton (or unanimous complete set); fixture generation/regeneration is an implementation task, not an existing pass.

## Required semantic references
Use the Go 1.27.1 source pin at tools/gomad/.toolchain/stock/2198cf19890232fc/go/src:
- runtime/symtab.go:376..402,725,1195..1284 and runtime/symtabinl.go:67..101; cmd/link/internal/ld/pcln.go:195..254,771..810 and ld/symtab.go:596 for metadata, textOff, gofunc and inline decoding.
- cmd/link/internal/wasm/asm.go:101,208..236,259,520..610 for synthetic PCs/indexes, normalized names and split active data.
- cmd/compile/internal/walk/select.go:221..232; cmd/compile/internal/wasm/ssa.go:231..238,331; cmd/internal/obj/wasm/wasmobj.go:320..337,455..487 for select lowering, source positions, caller stack slot and complete return-PC association.
- .flow/artifacts/fn229-symbolization-format-gap-2026-10-20.md and the corrected follow-up in .flow/tmp/fn229-wasm-symbolization-research.md. Go debug/gosym is a name-splitting semantic reference only.

## Acceptance
- The retained real known-site and real inline fixtures satisfy R8's exact package/function/innermost/outer-frame expectations; absent fixture modules fail rather than skip. Keep the raw compiler/retained-module proof, not a synthetic guessed-PC substitute.
- Real conflicting-two-call and unknown-call-pattern cases force physical fallback without false inline/location claims. A same-chain/different-line case omits only non-consensus locations; complete multi-candidate consensus remains allowed.
- A real -s fixture proves its actual retained pclntab/name behavior and yields no_symbols because the required name cross-check is absent; do not replace it with an artificially metadata-free 'stripped' fixture. Actual pclntab absence separately yields no_symbols. Wrong normalized names, malformed/ambiguous metadata, out-of-range and all cap/overflow cases produce the spec's existing reasons.
- Cross-segment/zero-gap tables and valid module/index/name mappings resolve deterministically across repeat runs; retained-module digest failure precedes symbol reads. No record/identity/wire mutation or new dependency.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
