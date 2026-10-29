//! Imported const-generic countdowns compile with native and interpreted state access.
#![forbid(unsafe_code)]

use umpire_dsl::__lifted::{Interprets, Lifts, value::Value as Scalar};
use umpire_dsl::countdown::{Remaining, Ticks};
use umpire_dsl::{IntoValue, Machine, Value, Verdict, machine};

#[machine]
mod clock {
    use umpire_dsl::countdown::{Remaining, Ticks};
    use umpire_dsl::{Action, State, World, check, countdown, stay};
    #[derive(World)]
    pub(super) struct World {
        configured: Option<Ticks<3>>,
    }
    #[derive(State)]
    pub(super) struct Model {
        pub(super) left: Option<Remaining<3>>,
        configured: Option<Ticks<3>>,
    }
    #[derive(Action, Copy)]
    pub(super) enum User {
        Tick,
        Arm,
    }
    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            [Self { left: None, configured: world.configured }]
        }
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Tick => {
                    self.decrement();
                    stay!();
                }
                User::Arm if self.left.is_none() => self.left = countdown::arm(self.configured),
                User::Arm => {}
            }
        }
        #[effect]
        fn decrement(&mut self) {
            if countdown::armed(self.left) {
                self.left = countdown::tick(self.left);
            }
        }
        #[invariant]
        fn bounds(&self) {
            if !countdown::expired(self.left) {
                countdown::within(self.left, self.configured);
            }
            check!(Remaining::<3>::contains(self.left.unwrap_or(Remaining { ticks: 0 }).ticks));
        }
        #[postcondition]
        fn descent(&self, before: &Self) {
            if before.left.is_some() {
                countdown::descends(self.left, before.left);
            }
        }
    }
}

#[test]
fn imported_values_have_real_admission_native_behavior_and_decoders() {
    let world = clock::World::new(Some(Ticks(2)));
    let embedded = clock::World::__umpire_lifted();
    let model = embedded.model.unwrap_or_else(|error| panic!("admission: {error:?}"));
    let starts = world.starts();
    let actions = world.actions();
    assert_eq!(model.fields.len(), 2);
    let agreement =
        umpire_testing::interpreted(|| clock::World::new(Some(Ticks(2))), None, 8, None)
            .unwrap_or_else(|error| panic!("{error:?}"));
    assert_eq!(agreement.states, 4);
    let mut before = starts[0].clone();
    assert_eq!(clock::World::__umpire_get(&before, 0), Scalar::Int(0));
    before = world.step(&before, actions[1].clone()).remove(0).2;
    assert_eq!(clock::World::__umpire_get(&before, 0), Scalar::Int(3));
    for expected in [1, 0, 0] {
        let edge = world.step(&before, actions[0].clone()).remove(0);
        assert_eq!(edge.2.left, Some(Remaining { ticks: expected }));
        assert_eq!((world.invariants()[0].predicate)(&world, &edge.2), Verdict::Holds);
        before = edge.2;
    }
    let state = clock::World::__umpire_state(&[Scalar::Int(1), Scalar::Int(4)]);
    assert_eq!(state.left, Some(Remaining { ticks: 0 }));
    let mut state = state;
    clock::World::__umpire_set(&mut state, 0, &Scalar::Int(0));
    assert_eq!(state.left, None);
    assert_eq!(
        Ticks::<3>(2).value(),
        Value::Bounded { name: "umpire_dsl::countdown::Ticks<3>", raw: 2 }
    );
    assert_eq!(
        Remaining::<255> { ticks: 0 }.value(),
        Value::Bounded { name: "umpire_dsl::countdown::Remaining<255>", raw: 0 }
    );
}

#[test]
fn admitted_countdown_model_binds_emits_and_validates_its_schema() {
    let source = include_str!("countdown.rs");
    let module = umpire_testing::bound(clock::World::new(Some(Ticks(2))), 8, None)
        .lift_source("countdown.rs", source)
        .unwrap_or_else(|error| panic!("{error}"));
    let schema = module.schema().unwrap_or_else(|error| panic!("{error}"));
    assert_eq!(schema.fields.len(), 2);
    let umpire_dsl::__lifted::Type::Optional(remaining) = &schema.fields["left"] else {
        panic!("remaining type")
    };
    assert_eq!(remaining.name, "umpire_dsl::countdown::Remaining<3>");
    assert!(module.tla.contains("Next =="));
    let mut bad_field = module.clone();
    bad_field.metadata.fields[0].name = "bad::field".into();
    assert!(bad_field.schema().is_err());
    let mut bad_module = module;
    bad_module.name = "bad::module".into();
    assert!(bad_module.schema().is_err());
}
