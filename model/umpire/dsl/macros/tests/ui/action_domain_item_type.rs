#[umpire_dsl::machine]
mod sized {
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub struct World {
        pub sizes: u8,
    }

    fn names(_: &World) -> impl Iterator<Item = &'static str> {
        ["a"].into_iter()
    }

    #[derive(State)]
    pub struct Model {}

    #[derive(Action)]
    pub enum User {
        Set(#[domain(names)] u8),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {}
    }
}

fn main() {}
