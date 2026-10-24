//! Bounded nominal optional values retain their source identity.
use umpire_tla::Lift;

const SOURCE: &str = r#"
use umpire_dsl::machine;
#[machine] mod countdown {
    use umpire_dsl::{Action, Finite, State, World};
    #[derive(Finite, Clone, Copy, Debug, PartialEq, Eq, Hash)]
    struct Ticks(#[inputs(1..=3)] u8);
    #[derive(World)] struct World { timeout: Option<Ticks> }
    #[derive(State)] struct Model { timeout: Option<Ticks> }
    #[derive(Action)] enum User { Clear }
    impl Model {
        #[starts] fn starts(w: &World) -> impl IntoIterator<Item = Self> {
            [Self { timeout: w.timeout }]
        }
        #[on(User)] fn user(&mut self, action: User) {
            match action { User::Clear => self.timeout = None }
        }
    }
}
"#;

#[test]
fn source_local_optional_ticks_are_admitted() -> Result<(), Box<dyn std::error::Error>> {
    Lift::parse("optional_ticks.rs", SOURCE)?;
    Ok(())
}
