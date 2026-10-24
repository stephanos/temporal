#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    struct World { limit: u8 }
    #[derive(State)]
    struct Model { count: u8 }
    #[derive(Action)]
    enum User { Advance }
    #[starts]
    impl Model {
        fn starts(_: &World) -> [Self; 0] { [] }
    }
    impl Model {
        #[on(User)]
        fn user(&mut self, _: User) { self.count += 1; }
    }
}
fn main() {}
