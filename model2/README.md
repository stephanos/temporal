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

End to end, by golden files (fn-210 R8, R22, R23). A test is a fixture, an input plus the output it
produces, in its owning crate's `tests/fixtures/<name>/` (`model.rs` and `expected/`). "Fixture", not
"case": Case is Testpilot's term.

- behaviour: each fixture's `expected/report.txt`, `states.txt`, `rows.txt` and `paths/<claim>.txt`
  (`insta`; review with `cargo insta review`);
- refusals: `trybuild` `.rs` + `.stderr` under `framework/dsl/tests/ui/`;
- expansion: insta snapshots of each macro's implementation function, formatted by
  `prettyplease`;
- one property test against a brute-force reference explorer, the check that goldens are right
  rather than merely stable.

Goldens are deterministic, human-readable and timing-free. `make model2-update` rewrites them; a
fixture folder without `model.rs` or without `expected/` fails the build. Unit tests only when a defect cannot be
written as a case.

Behaviour goldens use insta binary snapshots with text payloads. Each `report.txt`, `states.txt`,
`rows.txt`, or `paths/<claim>.txt` is a symlink to its `.snap.txt` payload; the adjacent `.snap`
stores insta's review metadata. This keeps the required text paths readable by other hosts and
lets `cargo insta review` (or `cargo insta accept`) update their single source of expected content.
Run `make model2-update` to create or refresh snapshots and their aliases together.

## Dependencies

Only admitted crates (fn-210 R18): a release in the last 12 months, or finished and very widely used.
They are declared once in `[workspace.dependencies]`; a crate opts in with `name.workspace = true`.
`deny.toml` bans the unmaintained alternatives (`im`, `derivative`, `proc-macro-error`,
`proc-macro-error2`, `deluxe`, `index_vec`, `stateright`); `cargo deny check bans` enforces it. A new
dependency needs the same check and a line in the spec.
