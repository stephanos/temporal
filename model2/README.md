# model2: the Rust prototype

A Cargo workspace mirroring `model/framework`'s layout. Scope is only what
`model/framework/samples/Kettle.scala` uses today; see `framework/samples/options/README.md` for the
authoring-surface options under discussion.

The toolchain is pinned in the repository's `mise.toml` (`rust = "1.99.1"`); run through mise:

```
mise exec -- cargo build --workspace
mise exec -- cargo test --workspace
```

## Hygiene

One lint policy for all crates lives in the workspace `Cargo.toml` (`[workspace.lints]`: clippy
`all` and `pedantic` at warn, a few pedantic lints that fight the Model style allowed, `unwrap_used`,
`expect_used`, `dbg_macro`, `todo` and `print_*` warned, `unsafe_code` forbidden, `missing_docs` and
`unreachable_pub` warned). Each crate opts in with `[lints] workspace = true`. Thresholds are in
`clippy.toml`, formatting in `rustfmt.toml` (stable options only, 2024 style, 100 columns).
`rust-toolchain.toml` matches the mise pin and names the components (rustfmt, clippy, rust-src,
rust-analyzer). `deny.toml` is the cargo-deny policy for licenses, duplicate versions, sources and
advisories; it needs the tool installed and the network.

The gate, from the repository root (`temporal/`):

```
make model2-check   # fmt --check, clippy -D warnings on all targets, docs with warnings denied, tests
make model2-fmt     # rustfmt the workspace
```

From inside `model2/`, `cargo check-all` and `cargo doc-all` are aliases for the clippy and doc steps.

## Testing

End to end, by golden files (fn-210 R8). A test is a case plus the output it produces, checked in
beside it:

- behaviour: every sample Model's `cases/<model>/expected/report.txt` and `paths/<claim>.txt`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `framework/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model2-update` rewrites them; a
golden with no case, or a case with no golden, fails. Unit tests only when a defect cannot be
written as a case.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec.
