#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    struct World { limit: u8 }
    #[derive(State)]
    struct Model<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> { value: T }
    #[derive(Action)]
    enum User { Advance }
    impl<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User) {}
    }
}
fn main() {}
