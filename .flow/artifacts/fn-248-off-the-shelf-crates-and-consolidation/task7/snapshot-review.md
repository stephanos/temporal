# fn-248.7 join review: snapshots, guests, R18 (read-only)

Range: `0a1a733c72..umpire` under `model/` (970 files). Note: the range also holds fn-252.3/.4 commits (badd05d621, fe1fe5384d, 6a8d3ee660), so they are classified too.

## Verdict: NEEDS_WORK

The snapshots are clean: no content changes at all. The verdict is NEEDS_WORK only because the committed Temporal guest has not been regenerated even though its inputs changed (task7 owns that step and has not done it yet), plus one small R18 documentation gap.

## Snapshot change classes

Method: compare every golden at base with its successor at `umpire`, with insta headers stripped (script: payload bytes compared exactly).

| Class | Count |
|---|---|
| Migrated `X.snap.EXT` / `X@EXT.snap.EXT` / plain `gaps.txt` -> `X.EXT.snap`, payload byte-identical | 345 |
| Macro expansion `expansion@rs.snap.rs` -> `expansion.rs.snap`, payload identical | 9 |
| Kettle `coverage.snap` (headerless) -> `coverage.txt.snap`, payload identical | 1 |
| Old insta binary-metadata companions (`X.snap` beside `X.snap.EXT`) deleted | 349 |
| Unchanged, same path (trybuild .stderr 134, catalog/annotate/realization insta .snap 15, historical originals 14 kept on purpose, archived export json/txt 23, tla/apalache txt 4) | 177 + 14 |
| New snapshot with no predecessor: `umpire/catalog/tests/ui/malformed_attributes.stderr` (new darling trybuild control, fn-248.4) | 1 |
| Content change in any golden (.snap, expected/, .stderr, TLA+/cfg, expansion) | **0** |

The 355 `.snap` files added by 4b0602c57f match one for one. No verdict, state, row, counterexample, TLA+/cfg or expansion payload changed. fn-252.4 rewrote 4 native fixture `model.rs` files onto shared support; their expected payloads are unchanged too, which confirms it preserved behaviour.

## Content or wording changes found (all in-code assertions, none in goldens)

- `temporal/annotate/syntax/src/realization.rs:609`, `catalog/build/tests/wire_generated.rs:203`: the derived Rust name for message `_42Foo` is asserted as `_42foo`. This comes from heck plus shared prost-compatible sanitization (f40149592b / e0eec761ce) and is pinned by the compile control `catalog/tests/pass/prost_names.rs`. Justified: it matches prost's own generated identifiers.
- A propose-attribute diagnostic changed from "unknown proposal hook" to darling's "Unknown field: `unknown`" (7996cb97a9). Justified: the wording comes from darling.
- `node0.r#new_unannotated_field` became `node0.new_unannotated_field`, and string `contains` checks became token-normalized comparisons. This comes from quote/prettyplease codegen; a raw identifier was never needed for a non-keyword. Justified.
- Unexplained semantic changes: none found.

Out-of-scope side note: fn-252.3 deleted the committed `model/design/native-symbolic/out*/` solver outputs (45 files) and now gitignores them. The regenerated evidence and the deletion list are kept under `.flow/artifacts/.../task3/evidence/`. This is not fn-248's change.

## Orphan and pending snapshots

- No tracked or untracked `*.snap.new` / `*.pending-snap`.
- No old-format files remain, except the intentional historical set `temporal/feature/activity/compatibility/original/*`. Those are listed in `HISTORICAL` and read by `compatibility/checkpoint.rs`.
- Ownership: the new `temporal/testing/snapshot_ownership.rs` refuses orphan, pending and missing-registered snapshots outside fixtures. The fixture runner (`dev/testing/src/runner.rs`) refuses expected files with zero or several consumers, and pending goldens. I found no orphan by inspection.
- Limit: this review did not execute these gates.

## Guest regeneration status: OWED

- The only committed guest is `temporal/hosts/go/umpire_temporal_guest.wasm`. It is unchanged in the range: sha256 `bc27af7d…9217872`, last changed in 8cbe8ee453 (fn-247), matching GUEST_BUILDS.md.
- Its normal closure (umpire-temporal-guest -> runtime, temporal realization/api/annotate, catalog, ir, checker, dsl, front…) contains files the lanes changed:
  - temporal/api (4 files): prost-types WKT swap, .5
  - catalog src/build/syntax (14 files): .4
  - ir/src: .3
  - protocol/runtime src+build.rs (7 files)
  - temporal/realization (6 files)
  - front, dsl, checker
  - Cargo.lock (removes enum-iterator, adds darling/ident_case). The lock-hash identity over-hashes, so this alone invalidates the guest.
- No regenerated asset, GUEST_BUILDS.md update or task7 artifact directory exists. The working tree is clean. Expect `guest-freshness --check` inside `make model-validate` to report the guest as stale. If so, run `make umpire-guest` once on the joined inputs, record the old and new hash in GUEST_BUILDS.md plus a receipt, then rerun the gate.
- Kettle, stock and control guests are built fresh and have no pins.

## R18 status: provenance OK, docs gap

- The R18 dependency list lives in `.flow/specs/fn-210-…md`, not in `model/DESIGN.md`. Line 59 lists thiserror, syn/quote/proc-macro2, darling 0.24, prettyplease, insta and trybuild. The dated 2026-10-24 note admits heck 0.5 and prost-types 0.14 and removes enum-iterator. The 2026-10-25 fn-248.4 clarification adds darling/darling_core/darling_macro 0.24.1 and the transitive ident_case 1.0.1, with evidence in `task4/crates-io-admission.json`.
- Actual lock delta: +darling, darling_core, darling_macro 0.24.1, +ident_case 1.0.1; -enum-iterator 2.3.0, -enum-iterator-derive 1.5.0. heck, prost-types, thiserror, quote, insta and serde were already resolved. All are accounted for.
- enum-iterator is gone from every Cargo.toml and from Cargo.lock (git grep is empty). The deny.toml comment was updated. syn 2.0.119 is still resolved through other crates, and multiple-versions stays at "warn".
- Gap: `model/DESIGN.md:271` (R18 paragraph) and `model/README.md:985-992` still name only the fn-247 admissions and fn247 inventories. Suggest one line naming the fn-248 admissions (darling+ident_case, heck, prost-types; enum-iterator removed) and linking the task4 inventory.
