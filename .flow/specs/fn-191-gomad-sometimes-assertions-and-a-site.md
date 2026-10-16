# Sometimes assertions and a site catalog on WASM

## Goal & Context

`softassert.That` and `softassert.Fail` are already Gomad's `Always` oracle: the runner's `internal/evidence/record/softassert` observer scans guest output for the "failed assertion: " prefix and the failed-assertion tag, deduplicates messages and treats a hit as a failure with identity. Sixty-six production call sites use it. What is missing is the coverage half: a way for code to say "this path should be reached in some execution" and for a campaign to report when it never was. Without it, a 200-seed campaign that stays green says nothing about whether the workload exercised the code the engineer cares about.

This spec adds `softassert.Sometimes`, a generated catalog of assertion sites, a first-hit semantic-probe channel on the cooperative WASM guest, and campaign-level reporting. It follows the Antithesis SDK pattern, where a build-time catalog of assertions is what makes "never fired" reportable, and reuses the registry discipline of `common/gomaddebug`: declare before `main`, stable IDs, late registration fails.

Prerequisites delivered: fn-180 (registry descriptors retained in recordings, runtime-state witness, debug import ABI separate from WASI) and the cooperative runtime overlay under `tools/gomad/internal/backends/wasm/toolchain`. Plan context: [.plans/GOMAD_FIND_BUGS.md](../../.plans/GOMAD_FIND_BUGS.md) step 4.

## Architecture & Data Models

- **API.** `softassert.Sometimes(staticMessage string)` declares that the calling site should be reached in at least one execution of a campaign. `softassert.Unreachable(logger, staticMessage, tags...)` is an alias of `Fail`. In ordinary builds `Sometimes` compiles to nothing observable. The message must be a string literal; the catalog generator refuses anything else.
- **Catalog.** A generator in the `gomadtool` command set scans the server module for `softassert.That`, `Fail`, `Sometimes`, `Unreachable`, `UnexpectedInternalErr`, `UnexpectedDataLoss` and for static span names in `tracer.Start` calls, and writes one checked-in Go file under the `gomad && wasip1 && wasm` tag that declares every site at package initialization with a stable ID, kind (always, sometimes, span), package path, file, line and message. IDs follow declaration order and are stable within one catalog version; the catalog hash is the identity. `make -C tools/gomad validate` fails when the checked-in file is stale.
- **Channel.** A semantic-probe facility in the cooperative guest runtime: a bitset sized from the catalog, reserved at initialization identically in every incarnation, set by `Sometimes` with one load, one branch and one store, and flushed through one dedicated host import at process exit and at every inspection or terminal boundary. `Fail` under the gomad tag additionally emits its ID and up to 32 raw PCs from `runtime.Callers` through the same import on the failing path. The import never yields, reads the clock, consumes draws or enters the WASI transcript or choice trace; it writes into the helper's evidence stream with the current local choice ordinal and virtual time attached by the host.
- **Evidence.** The artifact record gains a `semantic` section: catalog hash, the sorted set of hit IDs, and for `Fail` hits the ID, choice ordinal, virtual time and symbolized frames. Hit sets enter the coverage summary and corpus features. The catalog hash is bound into prepared-target identity; replay and resume refuse a mismatch.
- **Campaign.** `--coverage=semantic` and `--require-probe=NAME` accept catalogued `sometimes` names on WASM. A campaign summary lists catalogued `sometimes` entries never hit, with counts of executions. `--must-hit=NAME` (repeatable, globs allowed) fails the campaign with a distinct outcome when a named entry is never hit. `--keep-successes=novel` retains the first execution that hits each new entry.
- **Limits.** Raise the gomaddebug registry from 256 entries and 128-byte names to a shared string table and a catalog bound of 16,384 entries; both registries share the descriptor retention fn-180 introduced.

## API Contracts

- `softassert.Sometimes(staticMessage string)`: no logger, no tags, no return value. Ordinary builds: no-op. Gomad WASM builds: sets the bit for the site's catalog ID. Calling it from any goroutine is legal.
- `gomadtool catalog generate --module DIR --output FILE` writes the catalog; `gomadtool catalog check` exits 1 with the first stale or refused site. Refusals: non-literal message, duplicate (package, message) pair, message over 4,096 bytes, a site in a file excluded by build tags the generator cannot evaluate.
- `gomad explore` and `qualify` on cooperative WASM: `--coverage=semantic`, `--require-probe=NAME`, `--must-hit=NAME` and the existing `--keep-successes=novel` apply to catalogued names. Unknown names are refused before execution (status 2). A never-hit `--must-hit` entry ends the campaign with a dedicated outcome and status 1.
- `gomad inspect ARTIFACT` prints the semantic section: hits by name, and for `Fail` hits the choice ordinal, virtual time and frames. `inspect --choices` marks decisions adjacent to a `Fail` hit.
- Record schema and catalog format are versioned; older artifacts without a semantic section remain inspectable and replayable and report "no catalog".

## Edge Cases & Constraints

- A site hit before the runtime has initialized the bitset (package initialization order) must be counted; reserve the bitset in the runtime overlay before user package initialization, as fn-180 reserves the debug goroutine and emit buffer.
- The channel does identical work in recording and replay. The compulsory runtime-state witness must not differ because a `Sometimes` fired; the bitset lives outside the witnessed allocation and GC counters.
- `Fail`'s PC capture allocates on the failing path only. That path is rare and identical in recording and replay; it is permitted and documented as a declared difference from ordinary builds.
- Paths in the catalog are module-relative; the generator strips the module root. The guest never captures paths at runtime.
- A catalog hash mismatch between the prepared module and the runner's catalog is a preparation refusal, not a runtime divergence.
- Multi-instance clusters: each incarnation has its own bitset; the artifact records per-incarnation hit sets and the campaign summary unions them.
- Truncated or lost output does not affect the channel; the log scanner remains the fallback for artifacts recorded before this spec and keeps its existing inconclusive semantics.
- The catalog may include span names for the later OTEL tracer work, but this spec emits nothing for spans; declaring them only reserves IDs.

## Implementation Design

### Server package and generator (.1)

Add `Sometimes` and `Unreachable` to `common/softassert` with a no-op body in the default build and a gomad-tagged body that calls into the semantic channel through the same guest initialization link seam `common/gomaddebug` uses. Write the generator under the WASM backend's `gomadtool` with the scanner built on `go/packages`; emit `common/softassert/zz_catalog_gomad.go`. Add the `validate` check. Convert at least one existing site that an engineer identified as "should happen under load" to `Sometimes` as the demonstration, for example the sync-match path in matching or the speculative workflow task timeout path.

### Guest channel (.2)

Extend the cooperative runtime overlay with the bitset, the `Sometimes` hook, the `Fail` PC capture and the flush import, under the existing runtime owner. Add the import to the helper's ABI beside the fn-180 debug import, writing to the evidence stream with host-attached coordinates. Rotate the runtime, ABI and helper identities together.

### Evidence and campaign (.3)

Add the semantic section to the WASM artifact record and cluster evidence, the catalog hash to prepared-target identity, and the coverage, `--require-probe`, `--must-hit` and `--keep-successes=novel` handling to the runner for cooperative WASM. Feed hit sets into corpus features. Extend `inspect`.

### Qualification and demonstration (.4)

Run the cooperative runtime qualification on the integrated candidate to show unchanged recording and replay evidence for workloads without `Sometimes` sites. Demonstrate on the Temporal one-box cell: one `Sometimes` site reached under a seed campaign and reported in the summary; the same site reported never-hit and failing `--must-hit` when the workload that reaches it is removed. Retain commands, exits and identities in the batch evidence handover.

### Verification

Host-source and package tests with `-tags test_dep`, the Rust helper suites, `make -C tools/gomad validate` including the catalog check, and `test-wasm` once on the integrated candidate. Plan review authorizes no implementation, push, PR or CI action.

## Acceptance Criteria

- **R1:** `softassert.Sometimes` and `softassert.Unreachable` exist. In ordinary builds `Sometimes` has no observable effect and `Unreachable` behaves as `Fail`. Errors: none beyond compilation.
- **R2:** The catalog generator produces a checked-in file declaring every assertion site and static span name with stable IDs, kinds, module-relative paths, lines and messages; `make -C tools/gomad validate` fails on a stale catalog. Errors: a non-literal message, a duplicate (package, message) pair, an oversized message or an unevaluable build-tagged file is refused with its location.
- **R3:** On cooperative WASM, a `Sometimes` hit is recorded once per incarnation in the artifact's semantic section, and a `Fail` hit carries its ID, choice ordinal, virtual time and symbolized frames. Recording and ordinary replay evidence for a workload without hits is unchanged, and the runtime-state witness is unaffected by hits. Errors: a catalog hash mismatch is refused at preparation; a flush that exceeds its bound reports truncation with a count.
- **R4:** `--coverage=semantic`, `--require-probe`, `--must-hit` and `--keep-successes=novel` accept catalogued names on cooperative WASM; the campaign summary lists never-hit `sometimes` entries; a never-hit `--must-hit` entry ends the campaign with a dedicated outcome. Errors: unknown names are refused before execution; the existing restrictions on guidance with forced-prefix strategies are unchanged.
- **R5:** `gomad inspect` prints the semantic section and marks decisions adjacent to `Fail` hits; artifacts without a semantic section report "no catalog" and remain replayable. No error surface beyond R3.
- **R6:** On the Temporal one-box cell, one `Sometimes` site is demonstrated as reached under a seed campaign and as never-hit, failing `--must-hit`, when its workload is removed. The cooperative runtime qualification passes on the integrated candidate. No error surface beyond R1 to R5.

## Boundaries

- No edge or block coverage instrumentation; the choice trace remains the coverage signal.
- No OTEL tracer implementation; span names are catalogued only to reserve IDs for that later work.
- No change to the log-scanner semantics for `That` and `Fail` in ordinary builds.
- No native backend work.
- No search-policy change; hits feed existing guidance features only.
- No numeric guidance assertions (`AlwaysGreaterThan` and similar) in this spec.

## Decision Context

Owner request (2026-10-16) following the discussion recorded in `.plans/GOMAD_FIND_BUGS.md`: extend `softassert` with a `Sometimes` rather than introduce a second assertion API, catalog sites the way the Antithesis instrumentor does, and keep the instrumentation in Gomad's tooling rather than in `cmd/gomad`, which is a launcher. The research notes of 2026-10-01 list the application assertion and probe API as the prerequisite for workload generation, swarm profiles and `sometimes` coverage. The semantic channel is the kill switch: without it `Sometimes` is a log line.
