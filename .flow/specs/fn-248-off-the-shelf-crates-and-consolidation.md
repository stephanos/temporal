# Off-the-shelf crates and consolidation in existing model code (model)

## Conversation Evidence

> user (2026-10-24): "for model/ ie umpire check if the rust code (existing and planned - see specs) could be simplified by using off the shelf crates"
> assistant: a scored list of 42 options (−5..5) across existing code and the open specs.
> user (2026-10-24): "do 1-27 (allow new cargo dependencies) but drop enum-iterator unless found useful; drop any byte-for-byte compatibility requirements; why have hand-rolled insta-file handling?; yes move custom goldens to insta (break is fine!)"
> user (2026-10-24): "we don't want to apply/impl yet. we want to file them as flow next specs." / "update existing tasks or add missing tasks for new crates and changes"

Items for planned work (fn-247 xtask and process crate, fn-240, fn-245, fn-246, fn-243, fn-211) are filed in those specs. This spec owns the items that change existing code.

## Goal & Context

The model workspace (~86k lines of Rust) hand-rolls a few things that admitted or widely used crates do, and repeats several small helpers across crates. This spec replaces those with crates or one shared copy, moves hand-rolled golden comparisons onto insta, and drops the binary-snapshot layout that forced the fixture runner to decode insta's private file naming. Checker semantics, Model meaning and the semantic protocol contract are preserved; internal and wire representations may change with coordinated regeneration. Byte-for-byte compatibility of goldens, generated code and artifacts is not required (owner, 2026-10-24): goldens are re-accepted after review of their diffs.

## Compatibility policy for remaining work

Owner, 2026-10-24: remaining work does not require backward compatibility or byte-for-byte equality with old outputs. Where needed for the planned change, internal and wire representations, generated code, diagnostics and goldens may change; update their in-tree producers/readers and regenerate affected guests and goldens together. Do not add legacy decoders, dual formats or old-guest support. Verify Model meaning, the current native contract, typed replay, reproducibility and safety instead. Deterministic ordering is a requirement of the new contract, not an obligation to retain old bytes. Pinned tool checksums, content-addressed integrity and actual proofs/negative controls remain required.

## Architecture & Data Models

- **Crates (R1).** `heck` replaces three hand-written snake/camel converters, so generated names follow prost-build by construction. `darling` (admitted under fn-210 R18, unused so far) parses the flag, `key = value` and rename parts of catalog `ApiAttributes` and `proposal_handler`; source-ordered tag/constraint families, positional `also(uuid, ...)`, ordered `oneof(...)`, `rejects(...)` and the rule/mirror split stay hand-written. `thiserror` replaces hand-written `Display`+`Error` impls in umpire-catalog and umpire-dsl where it reads cleaner. Prost-build's default WKT mappings replace in-house well-known-type generation consistently across runtime, Kettle and shared Temporal codegen: nonprimitive types use `prost-types`, Empty uses unit and scalar/string/bytes wrappers use the corresponding Rust primitives; descriptor-derived paths use the same mapping. A thin wrapper keeps the Malformed-versus-Unsupported refusal split of `any.rs`. Serde derive with `deny_unknown_fields` decodes the ITF envelope; the strict value visitor and budgets stay. catalog/build generates Rust with `quote` + `prettyplease` instead of `write!`/`push_str`. `enum-iterator` is removed (unused; `derive(Finite)` generates `domain()`).
- **One copy (R2).** `panic_message(&dyn Any) -> &str` in umpire-checker (five copies today); one SplitMix64 (two copies); one `ProbeApplicability` mapping in umpire-catalog used by Kettle and Temporal realization (magic `0..3` today); one `Errors` accumulator in umpire-front (three today); test source-edit helpers (`replace_once`, `replace_in`, `replace_code`) in umpire-testing; one hex decoder in runtime `tests/support`.
- **IR traversal (R3).** `ExprKind::children()` and `children_mut()` in umpire-ir, non-allocating, one exhaustive match each. Recursion-only matches use them: `Expr::infallible` (only its non-recursive `false` arms stay explicit), `umpire-tla` `names::rename` and `emit::prime`, the fixtures' interpreted `rewrite`. Matches that do per-variant work (TLA+ `render`, safety `value::emit`, native lowering, front `vector`/`effects`) stay. fn-245 then adds its ~13 variants touching one match per helper instead of every recursive walk.
- **Goldens on insta (R4).** Kettle `tests/probes.rs` (`UMPIRE_UPDATE_KETTLE_COVERAGE`), Kettle `tests/realization.rs` and Temporal `realization/tests/realization.rs` (`assert_eq!` against `include_str!("*.snap.txt")`) become named insta snapshots. The fixture runner (`umpire/dev/testing/src/runner.rs`) moves from `assert_binary_snapshot!` (metadata `.snap` plus payload `.snap.<ext>`) to text snapshots, one `<name>.snap` per golden: the logical name is the path minus `.snap`, so `logical_file`/`golden_files` shrink to a suffix strip; consumer ownership and the incomplete/pending checks for skipped backends stay. Readers of payload files (`include_str!` of `.snap.txt` in other crates and guests, Go tests reading sibling goldens) read through one small helper that strips the insta header, or assert their own snapshot.

## API Contracts

- umpire-checker: `pub fn panic_message(payload: &(dyn Any + Send)) -> &str`.
- umpire-ir: `ExprKind::children(&self) -> impl Iterator<Item = &Expr>`, `ExprKind::children_mut(&mut self) -> impl Iterator<Item = &mut Expr>`.
- umpire-catalog: one public applicability mapping returning `ProbeApplicability`.
- umpire-testing: `replace_once`, `replace_in`, `replace_code` (signatures from the current copies), and a golden-reading helper returning snapshot contents without the insta header.
- No Model-surface change. Protocol representations (`umpire.model.v1`, `umpire.run.v1`), generated types and their Rust/Go readers may be updated where required by the planned crate and snapshot consolidation. Regenerate affected guests and in-tree consumers together; preserving old wire bytes or supporting old guests is not required.

## Edge Cases & Constraints

- Seeded generation uses one shared deterministic implementation. SplitMix64 remains the intended consolidation target, but the old random sequence, `% len` choice and witness bytes are not compatibility requirements; verify reproducibility for a fixed seed under the new implementation and semantic validity of generated values/witnesses.
- Guest closure: `thiserror` (proc macro, no runtime) and `prost-types` reach wasm guests; every guest still builds for `wasm32-wasip1` and the guest closure inventory is updated. `sha2`, `heck`, `darling` stay host or build/macro side.
- `heck`: the two existing `camel` copies differ (one filters empty parts); snapshot diffs decide which behavior is kept, and the change is recorded.
- darling changes diagnostic text: trybuild `.stderr` goldens are re-accepted; refusals stay at least as strict and as well located.
- Orphaned, pending and unowned current goldens fail through ownership/pending checks inside the existing test pass. Extend the existing runner registry for text snapshots and explicit Rust/Go/live/external readers; consuming a snapshot as data or validating a skipped backend is ownership evidence, not execution/agreement evidence. Cover non-fixture named snapshots with explicit live-test ownership too; no blanket ignore list or unchecked snapshot subtree. Do not add `cargo insta test` as another workspace run or require an installed cargo-insta CLI. Keep plain text insta snapshots and remove binary filename decoding.
- The process crate (`umpire/backends/process`) is owned by fn-247 until fn-247.4 closes; `tempfile` and `rustix` adoption there is fn-247's task, not this spec's.

## Acceptance Criteria

- **R1:** `heck`, `darling`, `thiserror`, `prost-types`, serde derive and `quote`/`prettyplease` replace the hand-rolled code named above; `enum-iterator` is gone from every manifest and `deny.toml`; fn-210's R18 list records each admission. Errors: malformed attributes and malformed ITF envelopes retain their strict refusal controls (wording may change).
- **R2:** Each helper named under "One copy" exists once and every former copy uses it. Errors: none beyond existing semantic behavior; fixed seeds reproduce the new outputs and generated values/witnesses remain valid, without matching a legacy sequence.
- **R3:** `ExprKind::children`/`children_mut` exist and recursion-only walks use them; per-variant walks are unchanged. Errors: none; the differential interpreter-versus-compiled checks and TLC agreement still pass.
- **R4:** Active mutable golden producers use named text insta snapshots instead of hand-rolled comparisons/update switches; the fixture runner contains no binary-snapshot filename decoding. Rust/Go readers may compare current native outputs with registered snapshot payloads through the shared reader; historical originals and deliberate decoder/fixture inputs remain evidence, not mutable goldens to re-accept. Errors: orphaned, pending or unowned current goldens fail inside the existing test pass; a skipped backend with an incomplete golden still refuses and ownership never claims execution.
- **R5:** After all six source lanes join, one complete Model gate passes through the existing canonical `UMPIRE_APALACHE=require make model-validate` entrypoint: one TLC/Apalache-required model-check and its existing benchmark, with clippy, all tests, wasm guests, native/Go consumers and existing gate stages retained. Review current snapshot diffs and perform necessary current guest publication/reproducibility/freshness on joined inputs. No repeated per-lane full gate or extra cargo-insta workspace pass is required; failures remain failures.

## Boundaries

- No change to the process crate (fn-247), the xtask (fn-247), the symbolic back end (fn-240), sets and maps (fn-245) or realization refinement (fn-246).
- No checker, interpreter, TLA+ semantics or Model surface change.
- Historical artifacts under `model/temporal/feature/activity/compatibility/**` remain evidence of their recorded runs; they do not impose old-output, old-wire or old-guest compatibility on this work. Regenerate current fixtures and guests when the planned consolidation requires it.

## Isolated implementation lanes and central join

[direct owner, 2026-10-24] "can we paralleize more aggressively?" and "can we take more risks, too?" authorize concurrent isolated source lanes, ordinary integration conflicts/rework, coordinated interface breaks and narrow inherited fixes. Tasks .1–.6 independently implement after whole-spec fn247 closure; .4/.5/.6 have no source prerequisite on .1. Shared Cargo/lock, generated-type paths, helper exports and snapshots are reconciled centrally, with reviewed-source immutability and private indexes/build targets. Keep honest overlapping Touches instead of pretending shared files are disjoint. Scientific oracle independence, semantics, safety, R18 admission and current integrity remain required.

The .1 lane establishes the text snapshot naming/payload-reader/ownership interface early. Other lanes may implement against current snapshots or that agreed interface in isolated copies; they do not wait for the entire migration. Preserve useful local negative tests; prepare candidate diffs without treating auto-acceptance as review. Consolidate snapshot reacceptance and necessary current guest publication after the join, then rerun focused controls affected by conflicts. No source lane's local pass certifies combined code. The newly allocated integration task depends on all six existing IDs and owns R5 once. The existing six task IDs and their scopes remain intact.

## Decision Context

2026-10-24, owner: "can we drop byte-for-byte compatability from open tasks to speed things up? we don't need backwards compatabiltiy". This supersedes old-output/backward-format requirements for remaining work. Current-contract integrity and semantic correctness remain required. Fresh plan review is required before implementation of this amended scope.


2026-10-24, owner: new Cargo dependencies are allowed under R18; byte-for-byte compatibility requirements are dropped; custom goldens move to insta ("break is fine"); enum-iterator is dropped as unused. Scored and rejected in the same review: strum/derive_more/databake (syn 2), BTOR2 crates and `itf` (stale under R18), walkdir/pretty_assertions/lasso, regex for the `grep -E` port (ERE syntax differs), regex-automata/petgraph for pattern search, petgraph BFS in the native checker, clap for xtask, a process-supervisor crate. `similar` diffs for custom goldens are subsumed by R4.

Why the runner decoded insta file names: binary snapshots kept a raw payload file beside the metadata, which other crates `include_str!`, TLC could read as `.tla`, and artifact manifests pinned by hash. With byte compatibility dropped, plain text snapshots remove the need.

## Requirement coverage

| Req | Description | Task(s) | Gap justification |
| --- | --- | --- | --- |
| R1 | `heck`, `darling`, `thiserror`, `prost-types`, serde derive and `quote`/`prettyplease` replace the hand-rolled code named above; `enum-iterator` is gone from every manifest and `deny.toml`; fn-210's R18 list records each admission. Errors: malformed attributes and malformed ITF envelopes retain their strict refusal controls (wording may change). | fn-248-off-the-shelf-crates-and-consolidation.4, fn-248-off-the-shelf-crates-and-consolidation.5, fn-248-off-the-shelf-crates-and-consolidation.6 | — |
| R2 | Each helper named under "One copy" exists once and every former copy uses it. Errors: none beyond existing semantic behavior; fixed seeds reproduce the new outputs and generated values/witnesses remain valid, without matching a legacy sequence. | fn-248-off-the-shelf-crates-and-consolidation.2 | — |
| R3 | `ExprKind::children`/`children_mut` exist and recursion-only walks use them; per-variant walks are unchanged. Errors: none; the differential interpreter-versus-compiled checks and TLC agreement still pass. | fn-248-off-the-shelf-crates-and-consolidation.3 | — |
| R4 | Active mutable golden producers use named text insta snapshots instead of hand-rolled comparisons/update switches; the fixture runner contains no binary-snapshot filename decoding. Rust/Go readers may compare current native outputs with registered snapshot payloads through the shared reader; historical originals and deliberate decoder/fixture inputs remain evidence, not mutable goldens to re-accept. Errors: orphaned, pending or unowned current goldens fail inside the existing test pass; a skipped backend with an incomplete golden still refuses and ownership never claims execution. | fn-248-off-the-shelf-crates-and-consolidation.1 | — |
| R5 | After all six source lanes join, one complete Model gate passes through the existing canonical `UMPIRE_APALACHE=require make model-validate` entrypoint: one TLC/Apalache-required model-check and its existing benchmark, with clippy, all tests, wasm guests, native/Go consumers and existing gate stages retained. Review current snapshot diffs and perform necessary current guest publication/reproducibility/freshness on joined inputs. No repeated per-lane full gate or extra cargo-insta workspace pass is required; failures remain failures. | fn-248-off-the-shelf-crates-and-consolidation.7 | — |

2026-10-24, owner (throughput ordering): `sha2` for the TLC and Apalache jar digests moved to fn-250.5, which consolidates jar admission into `PinnedJar::admit`, so the digest change lands once. This spec follows fn-247 and precedes fn-250, fn-251 and fn-252, which run as parallel lanes ahead of fn-246.
