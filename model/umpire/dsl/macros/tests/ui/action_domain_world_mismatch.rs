#[umpire_dsl::machine]
mod sized {
    use umpire_dsl::{Action, State, World};

    pub struct Other;

    #[derive(World)]
    pub struct World {
        pub sizes: u8,
    }

    fn sizes(_: &Other) -> impl Iterator<Item = u8> {
        0..2
    }

    #[derive(State)]
    pub struct Model {}

    #[derive(Action)]
    pub enum User {
        Set(#[domain(sizes)] u8),
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {}
    }
}

fn main() {}
