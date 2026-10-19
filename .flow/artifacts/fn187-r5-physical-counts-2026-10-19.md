# fn-187 R5 physical line counts — 2026-10-19

Static counts are ready for the integrated R4 source. Same-host canonical test
timing, frozen-candidate qualification and final independent batch review remain
pending; this is not R5 acceptance or a performance result.

| Physical classification | Original | Integrated candidate | Delta |
| --- | ---: | ---: | ---: |
| Rust `src/` outside recognized test modules (`rust_production`) | 24,429 | 27,455 | +3,026 |
| `src/conformance.rs` outside recognized tests, included above | 1,513 | 1,535 | +22 |
| Rust recognized inline/separate test modules | 11,640 | 7,404 | −4,236 |
| All Rust `src/`, sum of the two disjoint buckets | 36,069 | 34,859 | −1,210 |
| Rust `tests/`, excluding `testdata/` | 6,734 | 7,157 | +423 |
| Rust `src/` plus `tests/` | 42,803 | 42,016 | −787 |
| Go `wasi/*_test.go` | 21,898 | 21,061 | −837 |
| Go `wasi/integration_test.go`, included above | 11,831 | 3,786 | −8,045 |

These are physical lines, including comments/blanks and a final unterminated
line if present. They are not compiled production LOC. The unchanged historical
counter assigns complete files named `tests.rs` and recognized inline
`#[cfg(test)]`/`#[cfg(all(test,...))] mod NAME { ... }` blocks to its test bucket.
Other Rust source lines enter `rust_production`, including feature-only
`conformance.rs`, individual test-only helper items and separate corpus modules
such as `cluster/network/corpus.rs`, `network/stream_abi/corpus.rs` and
`storage_transport/vector_corpus.rs`. Moving a test into such a file can increase
that bucket without adding production executable code. The counter does not
evaluate Cargo features or Rust conditional compilation. No net compiled-code
reduction is inferred from this table.

Baseline: exact Git object
`bad3c401b5e7dbdf8bc8afaca3d08cad4da154fe`, read without using dirty baseline
worktree bytes. The existing original-baseline TSV is
`/home/agent/wt/evidence/fn187/r5-current/original-baseline-physical-counts-2026-10-19.tsv`.
The retained counter at
`/home/agent/wt/evidence/fn187/review3/count_lines.py` is unchanged, SHA-256
`92d858a03836e2e7ba7b4163253c0d035b85589bb9748f60bcb18d5fc7aad95b`.

Candidate: uncommitted integrated primary bytes under
`/Users/stephan/Workspace/skunkworks/gomad/temporal`, HEAD
`5951d8bb96f41fdd1b146ee4a8cbb17613495aa0`. HEAD alone does not identify these
edits. All 131 counted candidate inputs match root's retained integrated Rust
test `source-inputs.json`; 74 original inputs were read from Git. The exact
path/SHA-256 inventories and both result dictionaries are retained at
`/home/agent/wt/evidence/fn187/r5-current/r5-physical-count-inputs-2026-10-19.json`
(SHA-256 `da1132cf955b09c42aa2d64dad4f5dd79fc9fac45bdfde2daffed61ad3c59bdd`).
Raw before/after TSV:
`/home/agent/wt/evidence/fn187/r5-current/r5-physical-counts-2026-10-19.tsv`
(SHA-256 `af829340346cc7069c818ca9a0b15590bc164b1c4572b55af500ef879c842ae7`).

Count host: `Linux dev-build 7.0.14 #1 SMP PREEMPT Mon Sep 28 07:01:01 UTC 2026
aarch64 GNU/Linux`. Both sides used the same host and counter. The small
working-tree adapter reuses the historical `count`, `CFG`, brace scanner and
line-count functions; it changes only file enumeration/read callbacks, freezes
input bytes and checks the baseline values. Reproduce from the primary root:

```sh
python3 -I .flow/tmp/fn187-r5-count-current.py
```

Adapter SHA-256:
`78c98ba6ce6cd8894cfe043ceec575bb23d878bb9fdfad7ec8a0f746138f41ed`.
Rerun against the final source freeze if any counted input changes. README and
ARCHITECTURE already describe the shared corpus contract, repeated regeneration,
byte stability, assertion/coverage-before-deletion policy, retained invariants
and separate `wasmmeasure` invocation/execution; no further doc edit was needed.

No timing median/range or gain is claimed. Root retains the failed baseline
attempts and the accepted fixture-only spin-count repair (`2000`→`4000`) with
the original fuel limit, assertions and runtime budgets unchanged. Its current
canonical baseline run uses an independent `GOMAD_WASM_TEST_HELPER` clone and
the unchanged original Makefile. Actual successful baseline/candidate repetitions,
medians/ranges, final identities and qualification remain due.
