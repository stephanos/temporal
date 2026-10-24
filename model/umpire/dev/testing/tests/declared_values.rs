//! A declared World's local fieldless enums export their values without `derive(Finite)`.

#[umpire_dsl::machine]
mod phased {
    use umpire_dsl::{Action, State, World};

    #[derive(Clone, Copy, PartialEq, Eq, Hash, Debug)]
    pub(super) enum Mode {
        Low,
        High,
    }

    #[derive(World)]
    pub(super) struct World {
        mode: Mode,
    }

    #[derive(State)]
    pub(super) struct Model {
        mode: Mode,
    }

    #[derive(Action)]
    pub(super) enum User {
        Lower,
    }

    impl Model {
        #[starts]
        fn initial(world: &World) -> impl IntoIterator<Item = Self> {
            [Self { mode: world.mode }]
        }

        #[on(User)]
        fn user(&mut self, _: User) {
            self.mode = Mode::Low;
        }
    }
}

#[test]
fn world_and_state_enums_bind_without_finite() -> umpire_testing::TestResult {
    use umpire_dsl::{Machine, Value, Values};
    let world = phased::World::new(phased::Mode::High);
    assert_eq!(world.values(), [("mode", Value::Enum("Mode::High"))]);
    assert_eq!(world.starts()[0].values(), [("mode", Value::Enum("Mode::High"))]);
    let files = umpire_testing::bound(world, 8, None).run()?;
    assert_eq!(files["states.txt"].lines().count(), 2);
    Ok(())
}
