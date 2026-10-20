use umpire_dsl::{Path, path};
#[umpire_dsl::machine]
mod world {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model { pub done: bool }
    #[derive(Action)]
    pub enum User { Poll, Send { #[inputs(0..=1)] amount: u8 } }
    impl Model {
        #[on(User)]
        fn send(&mut self, _: User) { self.done = true; }
    }
}
fn main() {
    let _: Path<world::Model, world::Action> = path![world::User::Send { typo: _, .. }];
    let _: Path<world::Model, world::Action> = path![world::User::Poll { nonexistent: _ }];
}
