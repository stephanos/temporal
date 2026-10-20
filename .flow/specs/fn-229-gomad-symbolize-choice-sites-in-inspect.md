# Gomad: symbolize choice sites in inspect

## Goal & Context

Every choice record carries a site offset: the caller program counter of the select or scheduling point, converted by the runtime overlay to an offset into the module's text. Inspect shows that offset and a fingerprint hash. Nothing maps either back to a Go package, function, file or line, so divergence reports and minimized failures read as lists of numbers.

This spec adds symbolization to inspect. For each site the output names package, function and, when known, file and line, derived only from the retained module in the artifact. It is a read-only projection computed at inspect time and bound to the module digest the artifact already pins. No record, identity or wire format changes. It is the first of three steps; the explore site scope and minimizer site ordering specs build on the attribution defined here.

## Architecture & Data Models

Go's wasm linker emits the runtime's pclntab into the module's data segment and a `name` custom section with function names. The site offset is the runtime's text offset of the caller PC, so the exact inverse is the same pclntab. A symbolize package under the evidence layer locates the pclntab in the retained module, decodes it with the standard Go symbol decoder, and resolves text start plus offset to a function and file:line. The `name` section is a cross-check: the function the pclntab names must match the name section entry for the same function index.

Inspect calls the symbolizer when the record has a choice profile and attaches a `symbol` to each site and each listed decision. The symbolizer depends on the host filesystem primitives for reading the retained module and on nothing above the evidence layer.

## API Contracts

Inspect `--json` site and decision entries gain exactly one field:

```
"symbol": {"package": "go.temporal.io/server/service/history", "function": "(*workflowTaskHandlerCallbacks).handleCommand", "file": "service/history/workflow_task_handler.go", "line": 412, "inlined_from": [{"package": "...", "function": "...", "file": "...", "line": 0}]}
```

or

```
"symbol": {"unavailable": "site_missing" | "no_symbols" | "offset_out_of_range" | "symbol_mismatch" | "symbol_table_too_large"}
```

`inlined_from` is empty when there is no inlining. `file` and `line` are omitted when the pclntab has no line for the offset.

## Edge Cases & Constraints

On GOARCH=wasm the PC is synthetic: function index in the high bits, resume point in the low 16 bits. The implementation must confirm the retained module's text offset round-trips through the symbol decoder for that layout; the pinned fixture in R8 is the guard. If it does not, the fallback is to decode the function index from the offset and use the `name` section alone, giving package and function but no line. Output must be byte-equal across runs and hosts. The symbol table read is bounded by a fixed byte limit.

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

The pclntab was chosen over DWARF because the runtime computes the offset from the same table, so the inverse is exact, and because DWARF may be stripped from the module. The `name` section cross-check turns a decoding mistake into an explicit `symbol_mismatch` rather than a wrong name. Rejected writing symbols into the record as unnecessary state: they are derivable from the retained module at any time and would otherwise change identities.

## Quick commands

```bash
cd tools/gomad && go test ./internal/evidence/artifact/symbolize/... ./internal/runner/inspect/... && .bin/gomad inspect --json ARTIFACT | jq '.choices.sites[0].symbol'
```
