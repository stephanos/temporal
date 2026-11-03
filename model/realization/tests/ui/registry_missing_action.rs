use umpire_dsl::{Action, State, machine};
#[machine]
mod specimen {
    use super::*;
    #[derive(Action)]
    pub enum Client {
        #[metadata(temporal(event = "first", evidence = event))]
        First,
        Added,
    }
    #[derive(State)]
    pub struct Model {
        active: bool,
    }
    impl Model {
        #[on(Client)]
        fn client(&mut self, _: Client) {
            self.active = true;
        }
    }
}
struct Binding;
specimen::action_metadata!(umpire_temporal::registry; specimen; {
    #[registry(inputs = ())]
    impl Binding {}
});
fn main() {}
