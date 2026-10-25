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

#[cfg(test)]
mod capabilities {
    use author_kit::dsl::{Machine, Steps};
    use author_kit::runtime::{
        NoTarget, Projection, Realize, Runtime,
        capability::{self, Fill, Identity},
        evidence::Rpc,
        prost_types::Any,
        protocol::{self, Refusal},
    };
    use std::marker::PhantomData;

    #[derive(Clone, Copy, PartialEq, Eq, Hash)]
    enum Empty {}

    impl std::fmt::Display for Empty {
        fn fmt(&self, _: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
            match *self {}
        }
    }

    trait Declared {}
    impl Declared for u16 {}
    impl Declared for u32 {}

    struct World<T: Declared>(PhantomData<fn() -> T>);
    impl<T: Declared> Machine for World<T> {
        type State = u8;
        type Action = Empty;
        type Outcome = ();
        type Fact = ();

        fn starts(&self) -> Vec<u8> {
            vec![0]
        }
        fn actions(&self) -> Vec<Empty> {
            vec![]
        }
        fn internal(&self, action: &Empty) -> bool {
            match *action {}
        }
        fn step(&self, _: &u8, action: Empty) -> Steps<u8, (), ()> {
            match action {}
        }
    }

    struct Binding<M>(PhantomData<fn() -> M>);

    impl<M> Identity for Binding<M> {
        type Id = ();
        fn identify(&self, _: &Any) -> Result<(), Refusal> {
            Ok(())
        }
    }

    impl<M> Fill<Empty> for Binding<M>
    where
        M: Machine<State = u8, Action = Empty, Outcome = ()>,
    {
        type World = M;
        type Inputs = ();
        type Request = capability::NoRequest;

        fn fill(&self, action: &Empty, _: &M, (): &(), (): &()) -> Result<Self::Request, Refusal> {
            match *action {}
        }
    }

    impl<M> capability::Snapshot<M> for Binding<M>
    where
        M: Machine<State = u8, Action = Empty, Outcome = ()>,
    {
        type Target = NoTarget;
        fn project(
            &self,
            _: &M,
            _: &Any,
            _: &u8,
        ) -> Result<Projection<Empty, (), NoTarget>, Refusal> {
            Ok(Projection::Current)
        }
    }

    impl<M> capability::Evidence<Rpc> for Binding<M>
    where
        M: Machine<State = u8, Action = Empty, Outcome = ()>,
    {
        type World = M;
        type View = capability::NoEvidence;
    }

    impl<M> capability::ReplyPolicy<M> for Binding<M> where
        M: Machine<State = u8, Action = Empty, Outcome = ()>
    {
    }

    impl<M> Realize<M> for Binding<M> where M: Machine<State = u8, Action = Empty, Outcome = ()> {}

    #[test]
    fn renamed_facade_preserves_generic_bounds_empty_actions_and_refusing_defaults() {
        fn check<T: Declared>() {
            let world = World::<T>(PhantomData);
            let binding = Binding::<World<T>>(PhantomData);
            let value = Any { type_url: "control/Empty".into(), value: vec![] };
            assert_eq!(
                Realize::program(&binding).err().map(|e| e.code),
                Some(protocol::RefusalCode::Unsupported as i32)
            );
            assert_eq!(
                Realize::answer(&binding, &world, &value, &0).err().map(|e| e.message),
                Some("answer unsupported".into())
            );
            assert_eq!(
                Realize::reach_evidence(&binding, &value, &[]).err(),
                Some(Refusal::unsupported("evidence kind unsupported"))
            );
            let mut runtime = Runtime::new(world, binding);
            let result = runtime.dispatch(protocol::Method::Propose as u32, &[]);
            let Some(protocol::response::Outcome::Refusal(refusal)) = result.outcome else {
                panic!("empty action world must refuse a proposal")
            };
            assert_eq!(refusal.code, protocol::RefusalCode::Quiescent as i32);
        }
        check::<u16>();
        check::<u32>();
    }
}
