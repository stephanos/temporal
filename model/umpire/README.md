# Umpire framework

The framework is grouped by role: `author/` for Model declarations and authored tests,
`front/` and `ir/` for shared admission, `backends/` for checking, `protocol/` for wire
contracts and guest dispatch, `catalog/` for typed annotation catalogs, and `dev/` for
fixture tooling and shared Models. `hosts/go/` contains the host, Runner and generated
protocol packages; `reference/kettle/` contains the independent Kettle integration.

See [the crate map](../README.md#crate-map) for each crate's job and author-facing surface,
and [the workspace guide](../README.md) for authoring and verification commands.
