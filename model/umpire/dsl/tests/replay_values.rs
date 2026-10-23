//! Native projections preserve structural identities independently of their rendering.
use umpire_dsl::{ActionValues, IntoValue, Machine, Value, Values};
#[umpire_dsl::machine]
mod declared {
    use umpire_dsl::{Action, Rejection, State, World, reject};
    #[derive(World)]
    pub(super) struct World {}
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
fn declared_world_exports_typed_outcomes() {
    let world = declared::World::new();
    assert_eq!(world.values(), []);
    let state = &world.starts()[0];
    assert_eq!(state.values(), [("done", Value::Bool(false))]);
    let actions = world.actions();
    assert_eq!(
        ActionValues::<declared::World>::action_value(&actions[0]),
        ("User::Set", vec![("0", Value::Bool(false))])
    );
    let step = world.checked_step(state, actions[0].clone()).unwrap_or_default();
    assert_eq!(step[0].0.value(), Value::Enum("Outcome::Rejected(NotFound)"));
    assert_eq!(().value(), Value::Sequence(vec![]));
}
