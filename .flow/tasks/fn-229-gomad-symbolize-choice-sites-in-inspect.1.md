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
Implemented the authenticated retained-module symbol resolver for R3/R4/R7/R8 in fe0c2fff2a, with review repairs in 1b08590b24. Real Go 1.27.1 retained fixtures prove physical and complete-consensus inline attribution, conflicting/indirect fallback, actual stripped-name behavior, and fixture absence refusal. Fixed cumulative byte/code/entry/step/frame limits apply before parsing or allocation; requested results are immutable and whole-parse exhaustion discards partial attribution.

The three first-round findings (pointed metadata decoys, pre-budget data-expression traversal, and balanced unsupported exception controls) have true RED/GREEN regressions. Independent re-review SHIP, session01a15c0c-4e15-7b11-99c4-1575da10a108, receipt .flow/tmp/impl-review-receipt-fn229-symbolize.json. Fresh root candidate3 checks: all36 tests passed, mandatory fresh Go/Rust inspect parity0, foundation raw integrity/repeated goldens0, ordinary-release fixture refusals13/13. Worker fixture/default tests36/36 each, both Clippy/build/format checks0; two authenticated golden regenerations match all16 files. Original failures and exact commands remain in /home/agent/wt/evidence/fn229-symbolize/HANDOVER.md and its indexed raw receipts.

Candidate3 fixture SHA256 e75fedf086dab6ae0d17676976050dcd683955f466ff9e7d6ed49d4e491fffd8; release71aa958ffd73c9c66337b439c2ca954b0d2ccfef6f0594b7f1c1ef144c4e923a; immutable test executable1e7652b28757626259000b79a99e4d16ef34e1e94e49962e2976d466398d083a. Root source equals the repaired worker source for owned build inputs. Overlay prerequisites remain separately attributed to fn234. No new dependency, external symbol decoder, record/wire mutation or qualification claim.

Tier: session (jev-unavailable(no_key)); explicit AGENTS implementer/reviewer pins retained. Public inspect rendering and authenticated campaign divergence R2 belong to task .2 and fn232.7/.8; they remain open. The fn232 compiler/teardown deadline gap is not waived.

stage: plan-sync - skipped(config: planSync.enabled != true)
## Evidence
- Commits: fe0c2fff2aaad3e8332ce136c4165ef0a07c605d, 1b08590b24
- Tests: /home/agent/wt/evidence/fn229-symbolize/candidate3/gomad-symbolize-tests --nocapture (36 passed, exit0), python3 tools/gomad/testdata/parity/foundation-check.py --binary /home/agent/wt/evidence/fn229-symbolize/candidate3/gomad-fixture --work /home/agent/wt/evidence/fn229-symbolize/root-foundation-1b08590b24 (exit0), python3 tools/gomad/scripts/behavioral-parity --corpus tools/gomad/testdata/parity --verb inspect --inspect-binary /home/agent/wt/evidence/fn229-symbolize/candidate3/gomad-fixture --inspect-baseline /home/agent/wt/evidence/fn232/go-baseline-1d7b37f00a --inspect-source /home/agent/wt/fn232-go-source/temporal-1d7b37f00a --inspect-go-compiler /Users/stephan/Workspace/skunkworks/gomad/temporal/tools/gomad/.toolchain/stock/2198cf19890232fc/go/bin/go --output-root /home/agent/wt/evidence/fn229-symbolize/root-1b08590b24 (exit0), python3 tools/gomad/testdata/parity/foundation-release-check.py /home/agent/wt/evidence/fn229-symbolize/candidate3/gomad-release /home/agent/wt/evidence/fn229-symbolize/root-release-1b08590b24 (13/13, exit0), flowctl codex impl-review fn-229-gomad-symbolize-choice-sites-in-inspect.1 --base 9714711c2a --receipt .flow/tmp/impl-review-receipt-fn229-symbolize.json --spec codex:gpt-6.1-sol:high --json (SHIP), git diff --check (exit0)
- PRs: