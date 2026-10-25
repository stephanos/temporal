# Source-size closeout

After integration and final source freeze, inventory the authenticated Go source
revision and the final Rust revision from the repository root:

```sh
cargo +1.99.1 run --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad-source-size -- --repo . --rev GO_SOURCE_SHA --output /absolute/fresh/go-source-size.json
cargo +1.99.1 run --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad-source-size -- --repo . --rev RUST_SOURCE_SHA --output /absolute/fresh/rust-source-size.json
```

Each invocation inventories one revision. Compare the two reports for the final
baseline/candidate comparison. Omit `--output` to print JSON to stdout; output
files must be new. Each report contains the tracked blob inventory, authenticated
tree/blob IDs, SHA-256 hashes, and source groups.

Run the counter checks with:

```sh
cargo +1.99.1 test --manifest-path tools/gomad/rust/Cargo.toml --locked -p gomad-source-size
```

Report the preexisting Rust helper separately from the new Rust host, retained
Go model/oracle/toolchain sources and supporting languages. The prospective
retirement tree has `internal/model/wasi` as a distinct retained reference group.
Do not report the entire preexisting helper as newly written replacement code,
or subtract its removal while adding the relocated copy. Use whole-repository
bytes/files and both path inventories for the net comparison after deletion.
No preliminary worktree count is the requested final integrated comparison.
