#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State, World};
    #[derive(World)]
    struct World<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> { value: T }
    #[derive(State)]
    struct Model<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> { value: T }
    #[derive(Action)]
    enum User { Advance }
    #[starts]
    impl<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> Model<T> {
        fn starts(w: &World<T>) -> [Self; 1] { [Self { value: w.value.clone() }] }
    }
    impl<T: Clone + Eq + std::hash::Hash + std::fmt::Debug> Model<T> {
        #[on(User)]
        fn user(&mut self, _: User) {}
        fn hidden(&mut self) {}
    }
}
fn main() {}
