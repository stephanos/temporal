use umpire_dsl::{Path, path};
#[umpire_dsl::machine]
mod world {
    use umpire_dsl::{Action, State};
    #[derive(State)]
    pub struct Model { pub done: bool }
    #[derive(Action)]
    pub enum User { Send { #[inputs(0..=1)] amount: u8 } }
    impl Model {
        #[on(User)]
        fn send(&mut self, _: User) { self.done = true; }
    }
}
fn main() {
    let _: Path<world::Model, world::Action> =
        path![world::User::Send { amount: 0, amount: 1, .. }];
    let _: Path<world::Model, world::Action> =
        path![world::User::Send { amount: 0, r#amount: 1, .. }];
}
