**Scope reviewed:** task fn-258.2 source readiness (R2/R3) via the standalone adaptation patch relative to export a92, the allowlist/equivalence/history-join evidence, the focused logs, and the Temporal in-tree parent source the seam must integrate with. The sibling `../umpire` checkout itself is outside this sandbox's readable scope, so byte-identity claims are taken from the committed verifier outputs (`adaptation-equivalence.json` pinned head = live head = `ac79114`, tree `823a5dc`).

## What I verified

- **Boundary seam vs. unchanged `forbidden()`**: Traced the organization-exception token logic in `model/umpire/dev/xtask/src/gate/boundary.rs:123-146`. The exception already admits `github.com/temporalio/umpire…` when preceded by a non-path byte, which is why only three descriptor lines need exact-hash admission: `model.pb.go` (`BCZAgithub…`), `kettle.pb.go` and `probe.pb.go` (`BFZDgithub…`) have an alphanumeric length byte immediately before the prefix; `annotations`/`realization`/`run` have `?`/`=` and pass without exception. Policy entries match exactly those three paths.
- **Descriptor regeneration is real, not hand-edited**: Recomputed the protobuf FileOptions/go_package length bytes for all six `.pb.go` raw-descriptor tails (63/65, 65/67, 61/63, 63/65, 68/70, 68/70) — all consistent with protoc output for the new paths. Only the `go_package` option changes; package names and field tags are untouched.
- **Masking cannot widen admission**: `descriptor_line` only masks when path *and* full-line SHA-256 match; tests cover forged prefixes, injected controls (with and without a re-hashed policy), missing `const …_rawDesc` context, non-generated header, and same-bytes copy at another path.
- **Policy deserialization**: `deny_unknown_fields` plus required `license_sha256`/`descriptor_lines` means the standalone policy is self-contained; the Temporal in-tree copy keeps its own older struct/policy pair, so no cross-tree breakage.
- **Quick multi-repository `changes()`**: `Selection.reasons` is a `BTreeSet<String>` (`selection.rs:128`), `Options::parse` always yields `Some(base)` defaulting to `HEAD`, and `changed()` semantics are unchanged. Single-repo behaviour reduces to the prior code path exactly.
- **Portability (R3 negative conditions)**: `scripts/proto-go.sh` uses `.bin/` under the framework root; `scripts/xtask.sh` resolves from its own dir; all `path =` Cargo deps and the kettle `replace` resolve inside the root (checked by the unchanged `contains()`); `mise.toml`, `mise.lock`, `Cargo.lock`, `LICENSE` are in the frozen manifest (`LICENSE` mapped from Temporal root, which contains "Temporal Technologies" — hence the exact-digest exception). CI uses `${{ runner.temp }}` target.
- **Inventory arithmetic**: 57 allowlisted modifications counted path-by-path = 54 prior (51 unchanged, 3 updated: `dev/xtask/README.md`, `quick/mod.rs`, `quick/tests.rs`) + 3 new (policy, `boundary.rs`, `protocol/proto/README.md`); 1189−57 = 1132; 6 standalone-only files; 122 deletions / 133 additions enumerated; manifest sha `a29ea70…` consistent across both verifiers.
- **History join**: join `c71c309` has ordered parents `[5ee8911, a92db06]`, tree equals first parent; all 12 portable/export ancestors reachable; `ac79114` changes only `boundary.rs` test text with production text unchanged.
- **Failed-gate recovery bundles**: `cleanup-audit.json` records the two preserved failures as `source-boundary DESIGN.md:83` at 45eb554 and `clippy xtask lib tests` at 5ee8911 — both are fixed by later commits (`069624d`, `ac79114`), and the current DESIGN.md text has no forbidden token.

## Findings

No P0/P1.

- **Severity**: P3 · **Confidence**: 100 · **Classification**: introduced · **File:Line**: `.flow/artifacts/fn-258-complete-the-resumed-standalone-umpire/standalone/verify-adaptation.py:39-44` · **R-IDs**: [R3]
  **Problem**: `protected()` treats any path with a component named `fixtures` as protected, so `dev/fixtures/src/*.rs` and `dev/fixtures/build.rs` (Rust source of the fixtures crate) are counted among the "499 protected fixture/input/expected/snapshot/metadata/lock/sum/LICENSE records". Today nothing there differs, so the result is correct, but a future reviewed adaptation to that crate's source would be refused as `protected-difference` even after allowlisting, with a misleading error kind.
  **Suggestion**: Restrict the directory rule to payload subtrees (e.g. `fixtures/**` under `tests/`, or exclude `src/`/`build.rs`), or document that fixtures-crate source is intentionally frozen.

FYI (non-blocking):
- The task-2 artifact set has no readable protobuf-freshness/source-boundary receipt at `ac79114`; those stage receipts exist only as hashed entries inside the two failed-gate tarballs. The task acceptance line "protobuf freshness pass" therefore rests on the live task-4 gate, which the focus note already states. Process observation only.
- Quick multi-repository mode requires the caller directory to lie inside an admitted repository even when the effective base is the default `HEAD` everywhere. This is documented in `dev/xtask/README.md` and tested, so it is a recorded decision; it could be relaxed when `--base` is not explicitly given.

## Requirements coverage

| R-ID | Status | Evidence |
| --- | --- | --- |
| R1 | deferred | Task 1 (closed) owns the inventory; task 2 consumes its manifest unchanged (sha match in both verifiers) |
| R2 | met | Join preserves portable tree and both ancestries; export selected tree equals frozen source (`history-join-verification.json`) |
| R3 | met | Canonical Go identities, regenerated descriptors, go_package-only proto changes, root support/CI/docs, no out-of-root paths, boundary controls retained with exact LICENSE/descriptor admission; fresh full gate explicitly belongs to task 4 |
| R4 | deferred | Task 3 |
| R5 | deferred | Task 4 |
| R6 | deferred | Task 5 |

Unaddressed R-IDs: []

Suppressed findings: 0
Classification counts: 1 introduced, 0 pre_existing.

```json
{"classification_counts":{"introduced":1,"pre_existing":0},"unaddressed":[]}
```

<verdict>SHIP</verdict>