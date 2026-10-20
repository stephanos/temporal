# fn-229 WASM choice-site symbolization format gap

Pinned source inspection establishes a conflict with fn-229 R8: the recorded
site can identify a physical function, but cannot generally recover its call
PC, source line or selected inline frame. No spec, acceptance, producer,
identity or wire change is authorized by this finding. Implementation remains
after fn-232.5 under MILESTONES.md.

## Exact information loss

The retained producer path is:

- `tools/gomad/internal/backends/wasm/toolchain/overlay.go:271`: patched
  `selectgo` captures `sys.GetCallerPC()` and calls `gomadChoiceSite`.
- `tools/gomad/internal/model/runtimesource/gomad_semantics.go.txt:282`:
  `gomadChoiceSite` calls `firstmoduledata.textOff(pc)`. The resulting uint64
  occupies trace bytes 24..32 (line 252), and helper decision-input bytes
  8..16 in `toolchain/runtime_wasm.go.txt:503`. Neither retains the resume bits
  separately.
- Go 1.27.1 `runtime/symtab.go:725`: `textOff` computes `pc - md.text`, shifts
  it right 16 on WASM, then returns uint32. PCs in the same 65,536-wide interval
  therefore have the same stored site. The overlay does not patch this routine.

The pinned WASM linker represents a synthetic PC as `F<<16+B`, where F is
`0x1000 + defined-function ordinal` and B is the resume point. Discarding B is
lossy, not a reversible text-address encoding. Subtracting one for a return PC
afterward cannot recover it. The pclntab function offsets are already shifted
right 16; treating the stored site as a raw PC, or shifting it again, is wrong.
Validate the module text base and count imported functions, not all imports:
the actual WASM function index is `imported-function count + F - 0x1000`.

Inline metadata exists, but `runtime/symtabinl.go:67..101` chooses a frame using
the actual call PC. Several inline/select call sites can share a physical
function and collide in the recorded site. A fixture that guesses a PC or has
only one candidate does not establish a general inline resolver.

## Contract consequence and bounded feasible subset

R8 requires a retained-module fixture resolving an inlined site to its
innermost frame with its outer frame in `inlined_from`. The spec also forbids
record/identity/wire changes and assumes an exact inverse from the current
offset. These contracts cannot generally hold together with the lossy producer.
Function-only pclntab attribution is defensible, omitting unknown line/inline
data, but does not discharge R8. The name-only fallback does not discharge it
either: the linker replaces non-ASCII-word/non-dot name characters with `_`, so
the name section cannot reconstruct exact Go package/receiver spelling. R5
also keeps runnable site-zero/site-missing results unavailable; no scheduling
or creation site may be invented.

A later Rust parser may reuse the existing Wasmtime 47.0.3 wasmparser re-export
for WASM structure and a bounded checked reader for pinned Go metadata. Active
data can span segments and omitted zero runs; a contiguous per-segment magic
scan is insufficient. The unused zero textStart header word is not the text
base. Validate a unique pclntab/moduledata interpretation, sparse ranges,
arithmetic, names, function indexes and traversal bounds without materializing
all linear memory. The research's 64 MiB cap is only a proposal, not an admitted
limit or measured guest guarantee. `-s` removes the name section while pclntab
can remain, so absent-name and malformed/ambiguous metadata dispositions also
need explicit resolution within the existing unavailable contract.

The owner must resolve R8 versus the current no-format/identity-change
boundary before anyone claims full acceptance. This artifact neither drops R8
nor selects a new PC representation or unavailable reason. No implementation,
compiler fixture, guest execution, benchmark or qualification is claimed.

## Pinned evidence

The authoritative detailed report is
`.flow/tmp/fn229-wasm-symbolization-research.md`. It checked three Go 1.27.1
source copies, including the repository pin at
`tools/gomad/.toolchain/stock/2198cf19890232fc/go/src`, with equal hashes for
the examined runtime/linker/compiler/inline sources. In particular:

- `runtime/symtab.go`: SHA-256
  `50ce6d14b3885a8847086281a221ace995603c2b942e06c46935a15c0d66e17d`;
  [pinned textOff](https://github.com/golang/go/blob/go1.27.1/src/runtime/symtab.go#L725).
- `cmd/link/internal/wasm/asm.go`: SHA-256
  `1538a8b55ae94db9c094c53f2785cb607180eb88b7febb7e024af1b8b9647dd9`;
  synthetic PCs at line 101, function-index mapping at 211, name normalization
  at 236/605, data segments at 520..580 and missing DATA symbol names at 610.
- `cmd/link/internal/ld/pcln.go:771..810`: shifted function offsets/sentinel;
  `runtime/symtab.go:376..402`: pcHeader/moduledata;
  `cmd/compile/internal/wasm/ssa.go:31`: resume-point meaning;
  [pinned inline unwinder](https://github.com/golang/go/blob/go1.27.1/src/runtime/symtabinl.go#L67).

This durable summary records the existing read-only research. It performs no
new research or Flow lifecycle operation and changes no acceptance authority.
