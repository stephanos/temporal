//! Nominal transport and scalar accessors preserve authored values without finite validation.
use umpire_dsl::{IntoValue, Machine, Value, Values};

#[umpire_dsl::machine]
mod counter {
    use umpire_dsl::{Action, Finite, State, World, check};

    #[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
    pub struct Ticks(#[inputs(1..=3)] pub u8);

    #[derive(::umpire_dsl::Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
    struct Remaining {
        #[inputs(0..=3)]
        ticks: u8,
    }

    #[derive(umpire_dsl::Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
    pub struct Signed(#[inputs(0..=3)] pub i32);

    #[derive(World)]
    pub struct World {
        pub configured: Option<Ticks>,
    }

    #[derive(State)]
    pub struct Model {
        pub configured: Option<Ticks>,
        pub raw: Ticks,
        remaining: Option<Remaining>,
        pub signed: Option<Signed>,
    }

    #[derive(Action)]
    pub enum User {
        Tick,
    }

    impl Model {
        #[starts]
        fn starts(world: &World) -> impl IntoIterator<Item = Self> {
            [Self {
                configured: world.configured,
                raw: Ticks(1),
                remaining: Some(Remaining { ticks: 0 }),
                signed: None,
            }]
        }
        #[on(User)]
        fn user(&mut self, _: User) {
            self.configured = None;
        }
        #[invariant]
        fn bounded(&self) {
            check!(Ticks::contains(self.raw.0));
            check!(Signed::contains(self.signed.unwrap_or(Signed(0)).0));
        }
    }
}

#[test]
fn local_newtypes_export_once_for_world_state_and_repeated_fields() {
    let world = counter::World::new(Some(counter::Ticks(3)));
    assert_eq!(
        world.values(),
        [("configured", Value::Optional(Some(Box::new(Value::Bounded { name: "Ticks", raw: 3 }))))]
    );
    let state = &world.starts()[0];
    assert_eq!(
        state.values()[2],
        (
            "remaining",
            Value::Optional(Some(Box::new(Value::Bounded { name: "Remaining", raw: 0 })))
        )
    );
    assert_eq!(counter::Ticks(9).value(), Value::Bounded { name: "Ticks", raw: 9 });
    assert_eq!(None::<counter::Ticks>.value(), Value::Optional(None));
}

#[test]
fn scalar_accessors_reconstruct_both_shapes_and_leave_bounds_to_authored_claims() {
    use umpire_dsl::__lifted::{Interprets, value::Value as Scalar};
    type World = counter::World;
    let world = World::new(Some(counter::Ticks(1)));
    let state = &mut world.starts()[0];
    assert_eq!(World::__umpire_get(state, 0), Scalar::Int(2));
    assert_eq!(World::__umpire_get(state, 2), Scalar::Int(1));
    World::__umpire_set(state, 0, &Scalar::Int(4));
    World::__umpire_set(state, 2, &Scalar::Int(0));
    assert_eq!(state.configured, Some(counter::Ticks(3)));
    assert_eq!(state.values()[2], ("remaining", Value::Optional(None)));
    for invalid in [-1, i128::MIN, i128::from(i32::MAX) + 1] {
        assert!(
            std::panic::catch_unwind(|| {
                World::__umpire_state(&[
                    Scalar::Int(0),
                    Scalar::Int(1),
                    Scalar::Int(1),
                    Scalar::Int(invalid),
                ])
            })
            .is_err()
        );
    }
    for invalid in [-1, i32::MAX] {
        state.signed = Some(counter::Signed(invalid));
        assert_eq!((world.invariants()[0].predicate)(&world, state), umpire_dsl::Verdict::Fails);
        assert!(std::panic::catch_unwind(|| World::__umpire_get(state, 3)).is_err());
    }
    state.signed = None;
    World::__umpire_set(state, 1, &Scalar::Int(9));
    assert_eq!(state.raw, counter::Ticks(9));
    assert_eq!((world.invariants()[0].predicate)(&world, state), umpire_dsl::Verdict::Fails);
    let state =
        World::__umpire_state(&[Scalar::Int(0), Scalar::Int(2), Scalar::Int(1), Scalar::Int(0)]);
    assert_eq!(state.configured, None);
    assert_eq!(
        state.values()[2],
        (
            "remaining",
            Value::Optional(Some(Box::new(Value::Bounded { name: "Remaining", raw: 0 })))
        )
    );
}
