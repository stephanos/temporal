#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    struct World { limit: u8 }
    #[derive(State)]
    struct Model { count: u8 }
    #[derive(Action)]
    enum User { Advance }
    impl Model {
        #[on(User)]
        fn user(&mut self, selected: &World, _: User) { self.count += selected.limit; }
    }
}
fn main() {}
