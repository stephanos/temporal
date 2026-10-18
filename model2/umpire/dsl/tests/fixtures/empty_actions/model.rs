use std::collections::HashSet;
use std::fmt::Write;

use umpire_dsl::{Action, Spell};

#[derive(Action)]
enum Empty {}

pub(crate) fn run() -> umpire_testing::RunResult {
    let actions = Empty::actions();
    let cloned = actions.clone();
    let mut output = String::from("actions: [");
    for action in &cloned {
        action.spell(&mut output);
        write!(output, " internal: {}", action.internal())?;
    }
    output.push_str("]\n");
    let unique: HashSet<_> = actions.into_iter().collect();
    writeln!(output, "unique actions: {}", unique.len())?;
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
