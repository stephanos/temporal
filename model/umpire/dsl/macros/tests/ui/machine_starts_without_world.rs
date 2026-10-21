#[umpire_dsl::machine]
mod model {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    struct Model { count: u8 }
    #[derive(Action)]
    enum User { Advance }
    impl Model {
        #[starts]
        fn starts() -> [Self; 0] { [] }
        #[on(User)]
        fn user(&mut self, _: User) { self.count += 1; }
    }
}
fn main() {}
