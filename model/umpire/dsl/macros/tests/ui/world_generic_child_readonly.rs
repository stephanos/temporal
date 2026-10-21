#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};

    trait Token: Clone + Eq + std::hash::Hash + std::fmt::Debug {}

    #[derive(World)]
    struct World<T: Token> { initial: T }
    #[derive(State)]
    struct Model<T: Token> { value: T }
    #[derive(Action)]
    enum User { Tick }

    #[starts]
    impl<T: Token> Model<T> {
        fn starts(world: &World<T>) -> [Self; 1] {
            [Self { value: world.initial.clone() }]
        }
    }

    impl<T: Token> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User) {}
    }

    mod child {
        impl<T: super::Token> super::Model<T> {
            fn hidden(&mut self) {}
        }
    }
}

fn main() {}
