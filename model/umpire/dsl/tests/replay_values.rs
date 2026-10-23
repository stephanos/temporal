//! Native projections preserve structural identities independently of their rendering.
use umpire_dsl::{ActionValues, IntoValue, Machine, Value, Values};
#[umpire_dsl::machine]
mod legacy {
    use umpire_dsl::{Action, Rejection, State, reject};
    #[derive(State)]
    pub(super) struct Model {
        pub(super) done: bool,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Set(bool),
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, a: User) {
            match a {
                User::Set(true) => self.done = true,
                User::Set(false) => {
                    reject!(Rejection::NotFound);
                }
            }
        }
    }
}
#[test]
fn implicit_world_exports_state_payload_and_outcome() {
    let world = legacy::World;
    assert_eq!(world.values(), []);
    let state = &world.starts()[0];
    assert_eq!(state.values(), [("done", Value::Bool(false))]);
    let actions = world.actions();
    assert_eq!(
        ActionValues::<legacy::World>::action_value(&actions[0]),
        ("User::Set", vec![("0", Value::Bool(false))])
    );
    let step = world.checked_step(state, actions[0].clone()).unwrap_or_default();
    assert_eq!(step[0].0.value(), Value::Enum("Outcome::Rejected(NotFound)"));
    assert_eq!(().value(), Value::Sequence(vec![]));
}

#[test]
fn vector_projection_preserves_order() {
    let values: imbl::Vector<u8> = [2, 1, 2].into_iter().collect();
    assert_eq!(values.value(), Value::Sequence(vec![Value::Int(2), Value::Int(1), Value::Int(2)]));
}

#[umpire_dsl::machine]
mod native_only {
    use std::collections::BTreeSet;
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub(super) struct Model {
        values: BTreeSet<u8>,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Idle,
    }
    impl Model {
        #[on(User)]
        fn step(&mut self, a: User) {
            match a {
                User::Idle => self.values.clear(),
            }
        }
    }
}
#[test]
fn unsupported_projection_keeps_native_model_usable() {
    assert_eq!(native_only::World.starts().len(), 1);
}
