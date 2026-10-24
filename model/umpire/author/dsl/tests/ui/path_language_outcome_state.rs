use umpire_dsl::{Path, path};
#[umpire_dsl::machine]
mod world {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model { pub done: bool }
    #[derive(Action)]
    pub enum User { Poll }
    impl Model {
        #[on(User)]
        fn poll(&mut self, _: User) { self.done = true; }
    }
}
fn main() {
    let _: Path<world::Model, world::Action> = path![world::Model { done: false } => (), true];
}
