#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    struct World { limit: u8 }
    #[derive(State)]
    struct Model { #[starts(0..=1)] count: u8 }
    #[derive(Action)]
    enum User { Advance }
    impl Model {
        #[starts]
        fn starts(_: &World) -> [Self; 0] { [] }
        #[on(User)]
        fn user(&mut self, _: User) { self.count += 1; }
    }
}
fn main() {}
