//! Bounds keep their authored meaning when declarations share a machine.

use umpire_dsl::Machine;

#[umpire_dsl::machine]
mod conditional_debug {
    use std::marker::PhantomData;
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World<T> {
        marker: PhantomData<T>,
    }
    #[derive(State)]
    pub(super) struct Model<T> {
        marker: PhantomData<T>,
        pub(super) changed: bool,
    }
    #[derive(Action)]
    pub(super) enum User<T: Default> {
        Replace(#[inputs([T::default()])] T),
    }
    impl<T: Default> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User<T>, _: &World<T>) {
            self.changed = true;
        }
    }
}

#[test]
fn carrier_debug_retains_the_actors_conditional_bounds() {
    let world = conditional_debug::World::<u8>::new(std::marker::PhantomData);
    let action = world.actions().remove(0);
    assert_eq!(format!("{action:?}"), "User(Replace(0))");
    let state = world.starts().remove(0);
    assert!(world.step(&state, action)[0].2.changed);
}

#[umpire_dsl::machine]
mod actor_defaults {
    use std::marker::PhantomData;
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub(super) struct World<T, U> {
        marker: PhantomData<(T, U)>,
    }
    #[derive(State)]
    pub(super) struct Model<T, U> {
        marker: PhantomData<(T, U)>,
        pub(super) count: u8,
    }
    #[derive(Action)]
    pub(super) enum User<T = u8> {
        Go(#[inputs([PhantomData])] PhantomData<T>),
    }
    #[derive(Action)]
    pub(super) enum Clock<U> {
        Tick(#[inputs([PhantomData])] PhantomData<U>),
    }
    impl<T, U> Model<T, U> {
        #[on(User)]
        fn user(&mut self, _: User<T>, _: &World<T, U>) {
            self.count = 1;
        }
        #[on(Clock)]
        fn clock(&mut self, _: Clock<U>, _: &World<T, U>) {
            self.count = 2;
        }
    }
}

#[test]
fn distinct_actor_parameters_do_not_merge_authored_defaults() {
    let _: actor_defaults::User = actor_defaults::User::Go(std::marker::PhantomData);
    let world = actor_defaults::World::<u8, u16>::new(std::marker::PhantomData);
    let state = world.starts().remove(0);
    let mut actions = world.actions().into_iter();
    assert_eq!(
        world.step(&state, actions.next().unwrap_or_else(|| panic!("missing User")))[0].2.count,
        1
    );
    assert_eq!(
        world.step(&state, actions.next().unwrap_or_else(|| panic!("missing Clock")))[0].2.count,
        2
    );
    assert!(actions.next().is_none());
}

#[umpire_dsl::machine]
mod declaration_self {
    use std::marker::PhantomData;
    use umpire_dsl::{Action, State, World};

    mod helpers {
        struct Counter(u8);
        impl Counter {
            fn increment(&mut self) {
                self.0 += 1;
            }
        }
        pub(super) fn ticked() -> bool {
            let mut counter = Counter(0);
            counter.increment();
            counter.0 == 1
        }
    }

    pub(super) trait Config<S> {}
    pub(super) struct ModelSettings;
    pub(super) struct WorldSettings;
    impl Config<Model<ModelSettings, WorldSettings>> for ModelSettings {}
    impl Config<World<ModelSettings, WorldSettings>> for WorldSettings {}

    #[derive(World)]
    pub(super) struct World<T, U>
    where
        U: Config<Self>,
    {
        marker: PhantomData<(T, U)>,
    }
    #[derive(State)]
    pub(super) struct Model<T, U>
    where
        T: Config<Self>,
    {
        marker: PhantomData<(T, U)>,
        pub(super) changed: bool,
    }
    #[derive(Action)]
    pub(super) enum User {
        Go,
    }
    impl<T, U> Model<T, U>
    where
        T: Config<Self>,
        U: Config<World<T, U>>,
    {
        #[on(User)]
        fn user(&mut self, _: User, _: &World<T, U>) {
            self.changed = helpers::ticked();
        }
    }
}

#[test]
fn declaration_self_bounds_keep_the_model_and_world_identities() {
    let world = declaration_self::World::<
        declaration_self::ModelSettings,
        declaration_self::WorldSettings,
    >::new(std::marker::PhantomData);
    let state = world.starts().remove(0);
    assert!(world.step(&state, world.actions().remove(0))[0].2.changed);
}
