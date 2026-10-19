#![allow(clippy::reversed_empty_ranges, reason = "the empty-domain fixture intentionally has no starts")]

use std::collections::HashSet;
use std::fmt::Write;

use umpire_dsl::{Action, State};

#[derive(State)]
struct Kettle {
    #[starts(-1..=1)]
    temperature: i32,
    heating_on: bool,
    beeping: bool,
}

#[derive(State)]
struct Product {
    #[starts(1..3)]
    first: u8,
    #[starts(4..=5)]
    second: u8,
    label: String,
}

#[derive(State)]
struct Defaults<T> {
    r#type: T,
}

#[derive(State)]
struct Empty {
    #[starts(0..0)]
    value: u8,
}

#[derive(Action)]
enum User {
    PressButton,
    DepressButton,
}

#[derive(Action)]
#[internal]
enum Heater {
    Heat,
    Failover,
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut output = String::new();
    for state in Kettle::starts() {
        write!(output, "{state}")?;
        output.push('\n');
    }
    let starts = Product::starts();
    for state in &starts {
        write!(output, "{state}")?;
        output.push('\n');
    }
    for state in Defaults::<String>::starts() {
        write!(output, "{state}")?;
        output.push('\n');
    }
    writeln!(output, "empty starts: {}", Empty::starts().len())?;
    let mut unique = HashSet::new();
    for state in starts {
        unique.insert(state.clone());
        unique.insert(state);
    }
    writeln!(output, "unique cloned states: {}", unique.len())?;
    let users = User::actions();
    for action in &users {
        write!(output, "{action}")?;
        writeln!(output, " internal: {}", action.internal())?;
    }
    let mut unique = HashSet::new();
    for action in users {
        unique.insert(action.clone());
        unique.insert(action);
    }
    writeln!(output, "unique cloned actions: {}", unique.len())?;
    for action in Heater::actions() {
        write!(output, "{action}")?;
        writeln!(output, " internal: {}", action.internal())?;
    }
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
