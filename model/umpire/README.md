# Umpire framework

The framework is grouped by role: `author/` for Model declarations and authored tests,
`front/` and `ir/` for shared admission, `backends/` for checking, `protocol/` for wire
contracts and guest dispatch, `catalog/` for typed annotation catalogs, and `dev/` for
fixture tooling and shared Models. `hosts/go/` contains the host, Runner and generated
protocol packages; `reference/kettle/` contains the independent Kettle integration.

See [the crate map](../README.md#crate-map) for each crate's job and author-facing surface,
and [the workspace guide](../README.md) for authoring and verification commands.

## World-selected assumptions

A declared World can select one exact assumption leaf with exhaustive tail `if`/`else`
and nested `else if`. Named helpers, registered assumptions and progress `.assuming(..)`
use the same source subset. For example, inside a machine with a World-sized Poll catalog:

```rust,ignore
#[assumption]
fn selected_polls(world: &World) -> Assumption {
    if world.maximum == 1 {
        fair([Worker::Poll(1), Worker::Close])
    } else if world.maximum == 2 {
        fair([Worker::Poll(1), Worker::Poll(2), Worker::Close])
    } else {
        fair([Worker::Poll(1), Worker::Poll(2), Worker::Poll(3), Worker::Close])
    }
}
```

Guards admit primitive World Bool fields, Bool equality/inequality, Int comparisons
with matching literal values or source constants already admitted as primitive literals,
parentheses, `!`, `&&` and `||`. The source reader checks every authored operand before
folding and admits every branch, including inactive leaves. Guards cannot read state,
action inputs, outcomes, nominal or optional payloads, call methods or perform arithmetic.
Action arrays and payloads remain literal; loops, `match`, missing `else`, recursive
helpers and builders attached to conditional results refuse. Existing builder chains
belong inside each leaf. Helpers must receive the selected World parameter itself.

The source interpreter and TLA binding validate the World and select every registered
and progress plan before exact catalog membership. An inactive larger Poll domain is
admitted; a selected absent value refuses, including an unused registered assumption.
Selection preserves registration identity and emits one WF obligation per selected
concrete fair value. A bound conditional tree retains its authored branches, so binding
it again selects the new World, just as binding the original source does. The independent compiled native callback selects its own scope.
The shared `world_fairness` fixture checks maxima 1/2/3 through compiled, interpreted,
embedded and independently read source, with explicit close and heartbeat starvation controls.

Fixture agreement also compares emitted progress names, allowed/fair variants, exact
catalog ordinals and numeric bounds with the independently compiled scope at each reached
state. Empty graphs retain structural and static metadata checks; they invoke no progress
method and establish no agreement for an unevaluated callback.

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
