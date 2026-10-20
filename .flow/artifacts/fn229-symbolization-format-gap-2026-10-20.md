# fn-229 WASM choice-site symbolization format gap

The recorded site loses the resume-point bits, so it cannot generally identify
one call PC, source line or selected inline frame. This does not prove literal
fn-229 R8 impossible: R8 requires one real inline fixture, and Edge Cases allows
physical-function fallback elsewhere. Bounded consensus over a complete set of
compatible retained-code call sites could satisfy that fixture without changing
recorded bytes. No such fixture has been demonstrated; R8 remains pending.
No spec, acceptance, producer, identity or wire change is authorized here.
Implementation remains after fn-232.5 under MILESTONES.md.

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
function and collide in the recorded site. Guessing a PC or merely selecting
one candidate does not establish correctness. A proven complete singleton can
establish the attribution for that site, not a general inverse of textOff.

## Literal R8 and conservative consensus

R8 requires a retained-module fixture resolving an inlined site to its
innermost frame with its outer frame in `inlined_from`. The earlier blanket
claim that an owner must change the contract before acceptance was too strong
and is retracted. General information loss remains real, but source-backed
candidate consensus is compatible with this literal fixture and the existing
fallback. It is a possible proof route, not an implemented or verified result.

Let C include every return PC compatible with the site under the supported
producer, including syntactically present dead branches unless exclusion is
separately proven. C must be nonempty and complete. Decode each real return PC
minus one through pclntab and the inline metadata; emit an inline chain only
if every decode succeeds and all candidates agree on the full chain. Emit
file/line only where those fields separately agree. Incomplete coverage,
unknown patterns, conflicting chains or locations require the physical fallback
for the unsupported attribution. Never ignore a disagreeing candidate, choose
the nearest constant or first matching call, or infer that no inlining exists.

Pinned Go emission provides a bounded structural route for select sites:

- `cmd/compile/internal/walk/select.go:221..232` lowers a general select to a
  direct `runtime.selectgo` call. Simplified one-channel/default cases may use
  other runtime operations; a fixture must reach the instrumented path.
- `cmd/compile/internal/wasm/ssa.go:231..238` preserves the source position and
  emits static ACALL for a known target; line 331 implements GetCallerPC by
  loading the corresponding Go stack slot.
- `cmd/internal/obj/wasm/wasmobj.go:455..487` stores the concrete synthetic
  return address, function symbol plus pcAfterCall, before the direct WASM
  call. `cmd/link/internal/wasm/asm.go:208..211` resolves both into retained code.

A recognizer must validate the complete body, all candidate transfers, actual
selectgo function index, return-address-store association, parent physical
function and textOff equality. An unresolved indirect/tail transfer that could
target selectgo, unknown call shape or malformed control flow prevents narrowed
consensus. Bound code bytes/instructions/candidates as well as metadata; this
does not authorize an interpreter or unbounded analysis. Enumerating all low-bit
values is only a broad superset: `wasmobj.go:320..337` advances PCs across inline
source-position boundaries, and `cmd/internal/obj/pcln.go:202` maps non-inline
positions to -1. Finding one inline row therefore proves no whole-function
consensus.

A real positive singleton fixture, a conflicting two-call fixture forcing
fallback, and an unknown-pattern fixture preventing silent candidate omission
remain required proof. A physical wrapper with one inlined selectgo call is
plausible from the compiler sources, but no compiler fixture was built or
proven. The narrower path should be verified before requesting a scope change;
no owner contract change is a prerequisite established by this research.

Function-only pclntab attribution remains the accurate fallback when consensus
cannot prove more; it alone does not meet the inline fixture. The name section
also cannot reconstruct exact Go package/receiver spelling because the linker
replaces characters outside ASCII word characters and dot with `_`. R5 keeps
runnable site-zero/site-missing results unavailable; no scheduling or creation
site may be invented.

## Bounded retained-metadata reader

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

The follow-up corrects the prior impossibility conclusion, not R8 or the
no-format/identity-change boundary. This artifact selects no new PC
representation or unavailable reason. Actual fixture and completeness evidence
is still missing; no implementation, guest execution, benchmark or qualification
is claimed.

## Pinned evidence

The authoritative detailed report, including its superseding appendix
"Follow-up correction: literal R8 and conservative consensus", is
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
- [Pinned call emission](https://github.com/golang/go/blob/go1.27.1/src/cmd/internal/obj/wasm/wasmobj.go#L455)
  and [pinned select lowering](https://github.com/golang/go/blob/go1.27.1/src/cmd/compile/internal/walk/select.go#L221)
  support candidate recognition, not a claim that a real R8 fixture exists.

This durable summary records the existing read-only research. It performs no
new research or Flow lifecycle operation and changes no acceptance authority.
