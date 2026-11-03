use umpire_runtime::{capability::Identity, prost_types::Any, protocol::Refusal};
#[umpire_dsl::machine]
mod empty {
    use umpire_dsl::{Action, State};
    #[derive(Action)]
    pub enum Nobody {}
    #[derive(State)]
    pub struct Model {
        enabled: bool,
    }
    impl Model {
        #[on(Nobody)]
        fn nobody(&mut self, action: Nobody) {
            match action {}
        }
    }
}
struct Empty;
impl Identity for Empty {
    type Id = ();
    fn identify(&self, _: &Any) -> Result<(), Refusal> {
        Ok(())
    }
}
empty::action_metadata!(umpire_temporal::registry; empty; {
    #[registry(inputs = ())]
    impl Empty {}
});
fn main() {}
