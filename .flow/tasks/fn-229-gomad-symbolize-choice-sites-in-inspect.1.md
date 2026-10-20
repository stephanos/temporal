---
satisfies: [R3, R4, R7, R8]
---
# fn-229-gomad-symbolize-choice-sites-in-inspect.1 Symbolize package: locate pclntab in the retained module and resolve text offsets

Touches: tools/gomad/internal/evidence/artifact/symbolize/{module.go,pclntab.go,symbol.go}, symbolize/testdata/

## Description
New package `internal/evidence/artifact/symbolize`. Parse the wasm binary's sections with a minimal reader (data segments, custom `name` section, Go linker symbol metadata); locate `runtime.pclntab` and `runtime.firstmoduledata` text start; decode with `debug/gosym`; expose `Resolve(offset uint64) Symbol` returning package, function, file, line and inline frames, or `Unavailable{Reason}` for site_missing, no_symbols, offset_out_of_range, symbol_mismatch, symbol_table_too_large. Cross-check function names against the `name` section. Enforce a fixed byte limit on symbol tables before decoding. Confirm the GOARCH=wasm synthetic PC layout (function index high bits, resume point low 16) round-trips; if not, implement the name-section fallback documented in the spec and record which path is active. Fixture: a small module built by the pinned toolchain with a select in a known function and one in an inlined function, with the expected symbols pinned.

## Acceptance
Fixture resolves both sites to the expected package/function and the inlined one to the innermost frame with its outer frame; out-of-range, stripped and oversized modules yield the named unavailable reasons; output is deterministic across two runs; the fixture test fails, never skips, when the module is missing.

## Done summary
TBD

## Evidence
- Commits:
- Tests:
- PRs:
