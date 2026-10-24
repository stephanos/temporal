use std::fmt::Write;
use umpire_dsl::{Assumption, AssumptionExt, Scope};

#[derive(Clone, PartialEq)]
struct Action(u8);

impl From<u8> for Action {
    fn from(value: u8) -> Self { Self(value) }
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut text = String::new();
    for (name, assumption) in [
        ("all", Assumption::<Action>::default()),
        ("legacy-empty", Scope::without(&[])),
        ("legacy", Scope::without(&[Action(1)])),
        ("only", Assumption::only(&[0u8, 2])),
        ("only-without", Assumption::only(&[0u8, 1]).without([1u8])),
        ("without-only", Assumption::without(&[Action(1)]).only([0u8, 1])),
        ("replace-only", Assumption::only(&[0u8]).only([2u8])),
        ("repeat", Assumption::fair(&[2u8, 0, 2]).fair([1u8, 0])
            .without([1u8, 1]).without([2u8, 1])),
    ] {
        write!(text, "{name} allows:")?;
        for action in 0..3 {
            if assumption.allows(&Action(action)) { write!(text, " {action}")?; }
        }
        write!(text, "; fair:")?;
        for action in &assumption.fair { write!(text, " {}", action.0)?; }
        write!(text, "; excluded:")?;
        for action in &assumption.excluded { write!(text, " {}", action.0)?; }
        write!(text, "; is_fair:")?;
        for action in 0..3 {
            if assumption.is_fair(&Action(action)) { write!(text, " {action}")?; }
        }
        text.push('\n');
    }
    Ok(std::collections::BTreeMap::from([("values.txt".into(), text)]))
}
