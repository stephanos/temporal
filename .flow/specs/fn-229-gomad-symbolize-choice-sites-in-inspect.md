# Gomad: symbolize choice sites in inspect

## Goal & Context

Choice records carry a site value and a missing-site flag. Select sites use the runtime's compressed text offset of the caller PC; runnable sites currently marked missing stay unavailable. Inspect shows the value and a fingerprint hash without Go package/function attribution, so divergence reports and minimized failures read as lists of numbers.

This spec adds symbolization to inspect. For each site the output names package, function and, when known, file and line, derived only from the retained module in the artifact. It is a read-only projection computed at inspect time and bound to the module digest the artifact already pins. No record, identity or wire format changes. It is the first of three steps; the explore site scope and minimizer site ordering specs build on the attribution defined here.

## Architecture & Data Models

**Interleaved ordering (owner decision 2026-10-20):** implement on the Rust host after fn-232.5, without waiting for all of fn-232 or Go-host retirement. Go package locations and algorithms are semantic references only; no parallel Go-host feature is built. Preserve the existing R-IDs and same-lineage evidence/behavior contracts.

Use the existing `wasmtime::wasmparser` re-export for retained WASM sections, imports, optional function names and code, plus a checked reader for the pinned Go pclntab/moduledata/gofunc/inline grammar. No new dependency or Go subprocess is admitted. Runtime tables may span active constant-offset data segments and zero-filled gaps. Build a bounded sparse data view, validate a unique pcHeader/moduledata interpretation and all table pointers/counts, and obtain the text base from moduledata. The zero unused header textStart word and nonexistent named DATA symbols are not locators.

On WASM, textOff discards the low 16 resume bits. Map its function-scale offset through the already-shifted function table to the physical function; never treat it as a raw PC or shift it again. Validate the text base, defined-function ordinal and imported-function count against the actual WASM index. Derive exact package/function spelling from pclntab. When a name section exists, compare its physical-function entry to the pclntab name after Go's exact linker normalization (characters outside ASCII word characters and dot become `_`). Emitting a validated symbol requires that physical-function cross-check. Missing name sections or required entries yield `no_symbols` even when pclntab remains, because the required validation is unavailable; they do not imply pclntab was stripped. Sanitized names alone cannot reconstruct exact Go package/receiver spelling.

For select sites, a bounded recognizer of the pinned direct-selectgo call-emission grammar may enumerate compatible real return PCs from the physical function. Validate the complete body and all potential transfers, the selectgo index, paired return-address store, physical parent and textOff equality; include dead-branch calls unless exclusion is proved. Decode each real return PC minus one through pc-value tables, gofunc-relative inline rows and parent-PC links. Render inline frames only for a nonempty complete candidate set whose successful decodes agree on the full frame chain. File/line pairs require separate consensus. Unknown/indirect/tail call patterns that prevent completeness, empty sets or conflicting chains retain physical-function attribution without invented lines/frames. Never choose the nearest constant, first match or a subset that omits a conflicting candidate. This computes common attribution, not the lost PC.

Inspect calls the symbolizer when the record has a choice profile and attaches a `symbol` to each site and each listed decision. The symbolizer depends on the host filesystem primitives for reading the retained module and on nothing above the evidence layer. Before rendering, pre-resolve canonical unique requested sites in deterministic order into one immutable map of complete results, including unavailable results, and reuse it unchanged for sites, replay decisions and divergence decisions without resetting budgets per consumer. A whole-parse budget failure replaces partial results with `symbol_table_too_large` for every nonmissing requested site before rendering; missing-site results retain `site_missing`.

## API Contracts

Inspect `--json` site and decision entries gain exactly one field:

```
"symbol": {"package": "go.temporal.io/server/service/history", "function": "(*workflowTaskHandlerCallbacks).handleCommand", "file": "service/history/workflow_task_handler.go", "line": 412, "inlined_from": [{"package": "...", "function": "...", "file": "...", "line": 0}]}
```

or

```
"symbol": {"unavailable": "site_missing" | "no_symbols" | "offset_out_of_range" | "symbol_mismatch" | "symbol_table_too_large"}
```

`inlined_from` is empty when there is no inlining. `file` and `line` are omitted when the pclntab has no line for the offset. On physical fallback the same shape has an empty `inlined_from` and omitted unproved locations; this means no safely resolved inline frames, not proof that the compiler performed no inlining. An agreed frame chain may be rendered while a non-consensus file/line pair is omitted.

## Edge Cases & Constraints

Authenticate the retained module through the artifact handle before symbol reads; digest/read failures keep the existing open error. Then apply deterministic existing reasons: a missing-site flag yields `site_missing`; exhausting any analysis limit or overflowing checked size/count/address arithmetic yields `symbol_table_too_large`; absent/unsupported pinned Go metadata or a missing required name cross-check yields `no_symbols`; recognizable but malformed/truncated tables, multiple validated interpretations, inconsistent moduledata/indexes or a present name cross-check failure yield `symbol_mismatch`; a validated table with no containing function yields `offset_out_of_range`. Malformed name sections are not silently ignored. Metadata inconsistency takes precedence over range classification. These choices distinguish absent required validation from contradictory information without adding an enum value. Unsupported inline code patterns alone use the already-proved physical fallback, not a fabricated symbol or an artifact error.

Fix per-symbolizer limits at 64 MiB cumulative symbol/metadata bytes and allocations (including interpreted zero-filled table ranges), 64 MiB inspected code bytes, 1,048,576 aggregate section/segment/function/name/inline/candidate entries, 16,777,216 decode/traversal steps, 256 inline frames per chain and 1 MiB per name/path. Count work across header search, sparse reads, full-body call recognition and all requested sites; repeated lookups cannot reset budgets. Check counters and arithmetic before allocating or following offsets, detect inline cycles, and never materialize the entire linear address space. These fixed conservative limits bound malformed-input work; current-guest coverage has not been measured or promised. Boundary and over-limit fixtures must pin the reason, without host-dependent cutoffs or partial inline output after exhaustion.

A real `-s` module may retain pclntab while lacking the name section. Its existing `no_symbols` acceptance uses the deliberately required name cross-check, not a false claim that `-s` removes pclntab. Inspect and retain the real stripped fixture's metadata; do not substitute a fabricated metadata-free file for it. Test actual pclntab absence separately under the same `no_symbols` reason. The R8 real inline fixture must prove actual compiler inlining, the instrumented selectgo path, complete candidate recognition and expected innermost/outer frames. A guessed PC, handcrafted inline row or physical-only fallback does not satisfy it. Include conflicting-two-call and unrecognized-call-shape fixtures proving conservative fallback, plus sparse/ambiguous/corrupt tables, name mismatch, budget and artifact-digest negatives. Missing required fixture modules fail, never skip. No such fixture or cap coverage is claimed verified by this plan.

## Acceptance Criteria

- **R1:** Inspect on an artifact with a choice profile lists each site with its existing fingerprint, kind, count and maximum alternatives plus the `symbol` object. Errors: no symbol information in the module → `unavailable: no_symbols` and inspect still succeeds; module unreadable → the existing inspect open error.
- **R2:** Each replay decision and each divergence decision in inspect output carries the same `symbol` object for its site. No error surface beyond R1.
- **R3:** The symbolizer reads function and line information only from the retained module inside the artifact, never from a module on disk outside it. Errors: retained module digest mismatch → the existing artifact open error, before any symbol read.
- **R4:** An offset that falls outside every known function yields `unavailable: offset_out_of_range`; a site-missing record yields `unavailable: site_missing`; a pclntab name that disagrees with the `name` section yields `unavailable: symbol_mismatch`. No other error surface.
- **R5:** Runnable-kind decisions attribute to the scheduling point as recorded today. No error surface beyond R4.
- **R6:** Symbolization is deterministic: repeated `inspect --json` runs on one artifact produce byte-equal output on every host. Errors: none beyond R1 to R4.
- **R7:** The symbolizer refuses a module whose symbol tables exceed a fixed byte limit with `unavailable: symbol_table_too_large` instead of reading without bound. No other error surface.
- **R8:** A test fixture pins one known site in a retained module to its expected package and function, and one site inside an inlined function to the innermost frame with its outer frame in `inlined_from`, so an attribution change fails a test. Errors: fixture module absent → test fails, never skips.
- **R9:** Record hash, failure signature, execution identity and artifact contents are unchanged; symbolization appears only in the inspect result. No error surface.

## Boundaries

- No DWARF line table reading; the pclntab is what the runtime used to compute the offset.
- No attribution of runnable decisions to goroutine creation sites; that belongs to the explore site scope spec.
- No new field in records, artifacts or wire schemas.
- No symbolization in explore, replay or minimize output in this spec.

## Decision Context

Pinned Go 1.27.1 research corrected the original exact-inverse assumption. textOff loses resume bits, while retained pclntab still provides exact physical names and retained call emission can sometimes prove unanimous inline attribution. Literal R8 therefore stays mandatory and unproved, with conservative physical fallback for other sites. The existing five unavailable reasons describe missing, inconsistent, out-of-range and over-budget metadata; name absence leaves the required cross-check unavailable, while disagreement is contradictory metadata. Rejected guessing a resume PC or changing the producer/wire/record identities. The retained-module reader and complete candidate proof, not an owner scope change, are the next implementation test.

## Quick commands

```bash
cargo test --manifest-path tools/gomad/rust/Cargo.toml -p gomad symbolize
# Run the Rust gomad inspect corpus lane and review its golden output.
```

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | Inspect on an artifact with a choice profile lists each site with its existing fingerprint, kind, count and maximum alternatives plus the `symbol` object. Errors: no symbol information in the module → `unavailable: no_symbols` and inspect still succeeds; module unreadable → the existing inspect open error. | fn-229-gomad-symbolize-choice-sites-in-inspect.2 | — |
| R2 | Each replay decision and each divergence decision in inspect output carries the same `symbol` object for its site. No error surface beyond R1. | fn-229-gomad-symbolize-choice-sites-in-inspect.2 | — |
| R3 | The symbolizer reads function and line information only from the retained module inside the artifact, never from a module on disk outside it. Errors: retained module digest mismatch → the existing artifact open error, before any symbol read. | fn-229-gomad-symbolize-choice-sites-in-inspect.1 | — |
| R4 | An offset that falls outside every known function yields `unavailable: offset_out_of_range`; a site-missing record yields `unavailable: site_missing`; a pclntab name that disagrees with the `name` section yields `unavailable: symbol_mismatch`. No other error surface. | fn-229-gomad-symbolize-choice-sites-in-inspect.1 | — |
| R5 | Runnable-kind decisions attribute to the scheduling point as recorded today. No error surface beyond R4. | fn-229-gomad-symbolize-choice-sites-in-inspect.2 | — |
| R6 | Symbolization is deterministic: repeated `inspect --json` runs on one artifact produce byte-equal output on every host. Errors: none beyond R1 to R4. | fn-229-gomad-symbolize-choice-sites-in-inspect.2 | — |
| R7 | The symbolizer refuses a module whose symbol tables exceed a fixed byte limit with `unavailable: symbol_table_too_large` instead of reading without bound. No other error surface. | fn-229-gomad-symbolize-choice-sites-in-inspect.1 | — |
| R8 | A test fixture pins one known site in a retained module to its expected package and function, and one site inside an inlined function to the innermost frame with its outer frame in `inlined_from`, so an attribution change fails a test. Errors: fixture module absent → test fails, never skips. | fn-229-gomad-symbolize-choice-sites-in-inspect.1 | — |
| R9 | Record hash, failure signature, execution identity and artifact contents are unchanged; symbolization appears only in the inspect result. No error surface. | fn-229-gomad-symbolize-choice-sites-in-inspect.2 | — |
