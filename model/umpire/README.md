# Umpire framework

The framework is grouped by role: `author/` for Model declarations and authored tests,
`front/` and `ir/` for shared admission, `backends/` for checking, `protocol/` for wire
contracts and guest dispatch, `catalog/` for typed annotation catalogs, and `dev/` for
fixture tooling and shared Models. `hosts/go/` contains the host, Runner and generated
protocol packages; `reference/kettle/` contains the independent Kettle integration.

See [the crate map](../README.md#crate-map) for each crate's job and author-facing surface,
and [the workspace guide](../README.md) for authoring and verification commands.

The workspace guide owns the [exact assumption and World-selected fairness grammar](../README.md#writing-a-model).

## Bounded optional values

Stored optional values admit `Option` of a source-local one-field integer wrapper derived
with `Finite`. Its single `#[inputs(low..=high)]` range supplies checked `try_new` and
`contains` operations and the nominal source type. The source subset supports canonical
`None`/`Some`, equality, presence, eager `unwrap_or`, and tuple/named payload reads.
Exact core/std Option and constructor imports preserve their built-in identities; aliases
and glob imports remain outside optional source admission.
Backend decoding preserves the wrapper's standard Debug spelling and distinguishes absent
values from exhausted zero payloads. Nested options, arbitrary inner types, pattern matching
and Option combinators remain outside this subset.

The shared `optional_ticks` fixture uses positive `Ticks(1..=3)` and zero-inclusive
`Remaining(0..=3)` independently of handler facts. Its explicit checked factory validates
configuration and starts by field. `World::new` remains an infallible ownership operation.
Native exploration does not automatically validate a wrapper's finite domain: directly
forged values bypass the factory, source binding refuses them, and an authored bounds
invariant refutes invalid native starts or updates.
