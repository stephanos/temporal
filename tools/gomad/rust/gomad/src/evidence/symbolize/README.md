# Retained choice-site attribution

`resolve(&Opened, sites)` authenticates `target/module.wasm` through the pinned
artifact handle before reading symbols. It returns an immutable `Symbols`;
`get(Site { value, missing })` performs no parsing or budget work. Inspect should
collect every requested raw site, including nonbranching sites, and resolve once
before rendering any sites or decisions. Missing flags always return
`site_missing`. An unrequested key fails closed with `no_symbols`.

The checked decoder supports the pinned Go 1.27.1 pclntab/moduledata grammar.
Active constant-offset data segments form a sparse view; omitted bytes inside
the declared memory are zero. It never expands the linear address space. The
moduledata text base and imported **function** count bind shifted function-table
entries to actual defined WASM functions. Compressed offsets identify one
physical function; gaps do not inherit the preceding function's name.

Exact Go spelling comes from pclntab. Every physical name requires its actual
WASM name-section entry after Go's ASCII-word/dot normalization. Missing entries
yield `no_symbols`, while contradictory names or malformed maps yield
`symbol_mismatch`. Optional producers are not required; an explicit conflicting
Go compiler version yields `no_symbols`. This does not establish compatibility
with other compiler versions.

Select attribution scans the complete retained body for direct `runtime.selectgo`
calls. The recognized sequence decrements cached SP, writes the concrete return
address into that stack slot, and calls the validated target with PC_B zero.
Every syntactic candidate participates, including calls in dead branches. An
indirect/tail transfer or unsupported store/call shape prevents narrowed
attribution. Balanced unsupported exception controls also retain only physical
attribution. Each candidate's return PC minus one is decoded through PC streams
and gofunc-relative inline rows. All candidates must agree on the full chain;
each frame's file/line pair needs its own consensus. Otherwise the physical
function remains available with no unproved locations or inline frames.

Global validation checks referenced PC streams, files, inline rows and parent
cycles before classifying requested offsets. Fixed cumulative limits are
64 MiB of symbol/metadata bytes and allocations, 64 MiB of code, 1,048,576
entries, 16,777,216 steps, 256 frames per chain and 1 MiB per name/path. Checked
arithmetic and interpreted zero ranges consume the same budgets. A whole-parse
limit failure discards partial entries and stores one compact unavailable
result for every nonmissing lookup. No lookup resets a counter.
Allocation charges use fixed conservative layout allowances, checked against
Rust type sizes at compilation, so analysis cutoffs do not vary by host layout.
Data-section byte work is charged before decoding any segment. Its pinned
constant-offset grammar is read directly with bounded binary primitives, never
by a lazy expression parser that could first traverse or allocate control stacks.

The [real fixture proof](../../../../../testdata/parity/symbolize/README.md)
retains compiler inlining diagnostics, actual trace sites, complete call
candidates and literal expected symbols. Tests also derive malformed/sparse
inputs from it and use separate synthetic boundary cases. They do not execute
a Go decoder or consult source files during symbolization. Fixture absence
fails tests. Fixed limits bound analysis; they make no coverage promise for
unmeasured Temporal guests.

```sh
cargo +1.99.1 test --release --locked --manifest-path tools/gomad/rust/Cargo.toml -p gomad symbolize
```
