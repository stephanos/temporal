#[umpire_dsl::machine]
mod sized {
    use umpire_dsl::{Action, State, World};

    #[derive(World)]
    pub struct World {
        pub sizes: u8,
    }

    pub fn sizes(world: &World) -> impl Iterator<Item = u8> {
        0..world.sizes
    }

    #[derive(State)]
    pub struct Model {
        pub last: u8,
    }

    #[derive(Action)]
    pub enum User {
        Set {
            #[domain(sizes)]
            size: u8,
        },
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) {}
    }
}

fn main() {
    let _ = umpire_dsl::path![sized::Model { last: 0 }, sized::User::*];
    let _ = umpire_dsl::path![sized::Model { last: 0 }, sized::User::Set { size: _ }];
    let _ = umpire_dsl::path![sized::User::*, .., |state: &sized::Model| state.last == 0];
}
