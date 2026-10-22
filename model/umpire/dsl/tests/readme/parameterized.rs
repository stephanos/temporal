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
