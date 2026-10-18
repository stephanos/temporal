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
