# fn-248.7 joined validation

One canonical `BASH_ENV=/dev/null UMPIRE_APALACHE=require make model-validate` (TLC and Apalache required)
ran in the main checkout on `umpire` at `9b3e26654a`, which contains all six fn-248 lanes, the
reviewed fn-250.4/.5 lanes and the join commits below. Exit 0; all 18 gate stages passed.
Log: [model-validate.log](model-validate.log). Linux 7.0.14/aarch64; no macOS claim.

| Stage | Seconds |
| --- | ---: |
| temporal-guest-freshness | 43.059 |
| source-boundary | 16.693 |
| clippy | 13.594 |
| docs | 38.444 |
| tests (workspace, TLC+Apalache required) | 1086.256 |
| wasm-guest | 56.299 |
| kettle-go-host | 37.378 |
| temporal-go-host | 11.137 |
| other ten stages | < 4 each |
| **gate total** | **1316.090** (soft target 60) |
| inner loop (edit+test) | 9.953 (soft target 10) |

Join commits: 11862a858f (guest regenerated on joined inputs), eaa0d29ce7 (generated method-doc
spacing, Kettle UI snapshot re-accepted: indentation only), 763c6a5504 (guest and R18 docs),
713edcb334/43b10a5dc5/eb44d3113c/e4f0d148c5 (completion-review fixes: qualified `AnyMessage`
calls with the stale dispatch snapshot re-accepted, WKT docs, heck for `pascal`, shared
unchanged-probe refusal text), 9b3e26654a (guest republished).

Guest: 3507265 B `bc27af7d…` → 3516063 B `11c953c35455d99d003aa774c75b11ffe90239e52acaaad3ba04a42afad8f68a`;
a clean clone at a different path with an empty target rebuilt identical bytes, and the gate's
freshness stage passed.

Snapshots: [snapshot-review.md](snapshot-review.md) found 0 content changes across the insta
migration (345 renamed byte-identical, 349 metadata companions removed, 1 new darling UI control).
The completion review then found one stale expansion snapshot the diff-based review could not
see; it and the Kettle UI snapshot are the only two re-accepted, each explained above.

Attempts before the pass, none counted as receipts: run 1 stale guest (expected, fixed); run 2
sandbox `BASH_ENV` cwd reset broke a relative script call (rerun with `BASH_ENV=/dev/null`, as earlier
gates did); run 3 Kettle trybuild snapshot (fixed in eaa0d29ce7); runs 4–5 private build targets
filled the overlay disk (moved to the shared target in the main checkout).
