//! A Model author with exactly one direct dependency, under a consumer-selected name.
#![forbid(unsafe_code)]
#![allow(dead_code)]

#[author_kit::machine]
mod counter {
    use author_kit::{
        Finite,
        dsl::{Action, State, World, check},
    };

    #[derive(World)]
    pub(crate) struct World {
        limit: u8,
    }

    #[derive(State)]
    pub(crate) struct Model {
        pub(crate) value: u8,
    }

    #[derive(Finite, Clone, Copy, PartialEq, Eq, Hash, Debug)]
    pub(crate) enum Direction {
        Up,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Increment { direction: Direction },
    }

    impl Model {
        #[starts]
        fn starts(_: &World) -> impl IntoIterator<Item = Self> {
            [Self { value: 0 }]
        }

        #[on(User)]
        fn user(&mut self, action: User, world: &World) {
            match action {
                User::Increment { direction: Direction::Up } if self.value < world.limit => {
                    self.value += 1;
                }
                User::Increment { .. } => {}
            }
        }

        #[invariant]
        fn bounded(&self, world: &World) {
            check!(self.value <= world.limit);
        }
    }
}

#[author_kit::machine]
mod unsupported {
    use author_kit::{Action, State};

    #[derive(State)]
    pub(crate) struct Model {
        pub(crate) value: u8,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        Tick,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Tick => self.value = self.value.wrapping_add(1),
            }
        }
    }
}

struct Limits;

#[derive(author_kit::Catalog)]
#[mirror(response = "google.protobuf.Duration", context = Limits)]
struct DurationCatalog {
    seconds: author_kit::catalog::kinds::Counter,
    nanos: author_kit::catalog::kinds::Counter,
}

#[test]
fn authors_a_machine_path_suite_and_catalog_through_one_dependency()
-> Result<(), Box<dyn std::error::Error>> {
    use author_kit::dsl::__lifted::Lifts;
    use author_kit::{Run, Space, path, verify_all};
    use counter::{Direction, User, World};

    let embedded = World::__umpire_lifted();
    assert!(embedded.model.is_ok(), "{:?}", embedded.model);
    assert!(unsupported::World::__umpire_lifted().model.is_err());
    let space = Space::<counter::Model>::new(World::new(2));
    let report = verify_all(&space)?;
    assert_eq!(report.counts.reached, 3);
    assert_eq!(report.outcome(), author_kit::CheckOutcome::Verified);
    let run = Run::<counter::Model>::along_in(
        World::new(2),
        path![
            counter::Model { value: 0 },
            User::Increment { direction: Direction::Up },
            ..,
            |state: &counter::Model| state.value == 2
        ],
    )?;
    assert_eq!(run.state().value, 2);
    let duration = author_kit::runtime::prost_types::Duration { seconds: 2, nanos: 0 };
    let catalog = DurationCatalog::catalog(&Limits)?;
    assert_eq!(author_kit::catalog::validate(&duration, &catalog), []);
    Ok(())
}
