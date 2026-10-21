//! Readonly guards retain declaration-bound names in their original lexical scope.

use umpire_dsl::Machine;

#[umpire_dsl::machine]
mod bounded {
    use umpire_dsl::{Action, State, World};

    pub(super) trait Token: Clone + Eq + std::hash::Hash + std::fmt::Debug {}
    impl Token for u8 {}

    #[derive(State)]
    pub(super) struct Model<T: Token> {
        value: T,
    }

    #[derive(World)]
    pub(super) struct World<T: Token> {
        initial: T,
    }

    #[derive(Action)]
    pub(super) enum User {
        Tick,
    }

    #[starts]
    impl<T: Token> Model<T> {
        fn starts(world: &World<T>) -> impl IntoIterator<Item = Self> {
            [Self { value: world.initial.clone() }]
        }
    }

    impl<T: Token> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User, world: &World<T>) {
            self.value.clone_from(&world.initial);
        }
    }

    pub(super) mod child {
        struct Counter(u8);

        impl Counter {
            fn inc(&mut self) {
                self.0 += 1;
            }
        }

        pub(crate) fn increment() -> u8 {
            let mut counter = Counter(0);
            counter.inc();
            counter.0
        }
    }
}

#[test]
fn child_helpers_do_not_need_the_models_local_generic_bounds() {
    assert_eq!(bounded::World::new(7_u8).starts()[0].to_string(), "Model { value: 7 }");
    assert_eq!(bounded::child::increment(), 1);
}
