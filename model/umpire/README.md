# Umpire framework

The framework is grouped by role: `author/` for Model declarations and authored tests,
`front/` and `ir/` for shared admission, `backends/` for checking, `protocol/` for wire
contracts and guest dispatch, `catalog/` for typed annotation catalogs, and `dev/` for
fixture tooling and shared Models. `hosts/go/` contains the host, Runner and generated
protocol packages; `reference/kettle/` contains the independent Kettle integration.

The framework is an independent Cargo workspace with local tool pins, lockfile,
lint and arithmetic policies. From this directory, use the installed pinned tools:

```sh
mise exec -- cargo build --workspace --locked
mise exec -- cargo test --workspace --locked
mise exec -- cargo fmt --all -- --check
mise exec -- cargo check-all
mise exec -- cargo doc-all
```

Run the complete framework gate from this directory:

```sh
mise exec -- cargo xtask gate
```

It checks this workspace's formatting, clippy, documentation, tests, source boundaries,
protobufs, guests and both Go host modules. `cargo xtask fmt`, `update`, `time` and
`validate` provide formatting, snapshot updates, timed checks and required TLC checks.
The full gate sets `UMPIRE_QUICK_CACHE=0` and `UMPIRE_CACHE_DISABLE=1` for all children.
Apalache remains opt-in through `UMPIRE_APALACHE=require`.

The native [xtask](dev/xtask/README.md) provides pinned tool installation and development
commands. Guest builds execute the independent [guest driver](dev/guest-driver/README.md).
`mise exec -- cargo xtask build-guests` builds the five generic guests and native parity reports;
the [Go host](hosts/go/README.md) and [Kettle reference](reference/kettle/README.md) describe
how to consume them. `mise exec -- cargo xtask proto-go core --check` and the `kettle` selector
verify local generated Go protocol sources. Tools, generators and backend caches stay
under this workspace; explicit Cargo target overrides remain supported.

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

## Model authoring

### Fixed parameters and computed starts

A module may declare one named-field `#[derive(World)] struct World`. Its fields stay fixed
through an exploration. Like `State`, the derive supplies conditional Clone/Eq/Hash and
Debug/Display implementations; it adds `World::new` with arguments in field order and the
struct's visibility. It requires neither Copy nor Default. The declared World is the Machine;
there is no implicit `world()` constructor for its parameters.

```rust
#[umpire_dsl::machine]
mod counter {
    use umpire_dsl::{Action, State, World, check};

    #[derive(World)]
    pub(super) struct World {
        initial: u8,
        limit: u8,
    }

    #[derive(State)]
    pub(super) struct Model {
        value: u8,
    }

    fn targets(world: &World) -> impl Iterator<Item = u8> {
        world.initial..=world.limit
    }

    #[derive(Action, Copy)]
    pub(super) enum User {
        Set {
            #[domain(targets)]
            value: u8,
            #[inputs([true])]
            enabled: bool,
        },
    }

    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: world.initial }]
        }

        #[on(User)]
        fn user(&mut self, action: User, world: &World) {
            match action {
                User::Set { value, enabled: true } if value <= world.limit => self.value = value,
                User::Set { .. } => {}
            }
        }

        #[invariant]
        fn bounded(&self, world: &World) {
            check!(self.value <= world.limit);
        }
    }
}

use counter::World;

umpire_testing::fixture!(World = World::new(1, 3), 64);

#[test]
fn selected_world() -> umpire_testing::TestResult {
    let files = bound().run()?;
    assert_eq!(files["states.txt"].lines().count(), 3);
    let smaller = bound_for(World::new(2, 3)).run()?;
    assert_eq!(smaller["states.txt"].lines().count(), 2);
    Ok(())
}
```

This exact block compiles as [the DSL README fixture](author/dsl/tests/readme/parameterized.rs),
checked against the README by `mise exec -- cargo test -p umpire-dsl --test main readme::` from
this directory. The fixture binds one selected World explicitly; `bound_for` selects another without
fabricating defaults. `World::new` assigns fields and does not validate their ranges or starts.

Checker tools are provisioned explicitly with `mise exec -- cargo xtask fetch-tlc` or
`mise exec -- cargo xtask fetch-apalache`. Their framework-owned HTTP declarations live in
`mise-tools/`; its default environment contains TLC 1.7.4 and its `apalache`
environment adds Apalache 0.62.4. Provisioning delegates checksummed downloads,
extraction and installation locking to mise. Execution never auto-installs tools.
The jar provider scopes mise data/cache/state to `build/mise-jars`, leaving the
ordinary Java and Rust installations in place. Backends require exact install
paths, regular files without symlinks, runtime JAR SHA-256 and tool version
admission before checking a Model. See [provider controls](mise-tools/README.md).
