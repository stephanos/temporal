# model2: the Rust prototype

A Cargo workspace mirroring `model/framework`'s layout. Scope is only what
`model/framework/samples/Kettle.scala` uses today; see `framework/samples/options/README.md` for the
authoring-surface options under discussion.

The toolchain is pinned in the repository's `mise.toml` (`rust = "1.99.1"`); run through mise:

```
mise exec -- cargo build --workspace
mise exec -- cargo test --workspace
```
