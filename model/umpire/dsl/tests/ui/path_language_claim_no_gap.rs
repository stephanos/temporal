#[umpire_dsl::machine]
mod world {
    use umpire_dsl::{Action, State, path};
    #[derive(State)]
    pub struct Model { pub done: bool }
    #[derive(Action)]
    pub enum User { Poll }
    impl Model {
        #[on(User)]
        fn poll(&mut self, _: User) { self.done = true; }
        #[progress]
        fn completes(&self) -> LeadsTo { path![!self.done, self.done] }
    }
}
fn main() {}
