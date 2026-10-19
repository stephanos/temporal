use std::fmt::Write;

use umpire_suite::Run;

use crate::kettle::{at, kettle::{Heater, User}};

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut out = String::new();
    let heated = Run::from(at(97, false, false)).step(User::PressButton)?.repeat(3, Heater::Heat)?;
    write!(out, "pressed, then heated three times:\n{heated}")?;
    let Err(pressed) = heated.step(User::PressButton) else { return Err("press while on".into()) };
    writeln!(out, "pressed while on: {pressed}")?;
    let Err(depressed) = Run::from(at(20, false, false)).step(User::DepressButton) else {
        return Err("depress while off".into());
    };
    writeln!(out, "depressed while off: {depressed}")?;
    let Err(boiled) = Run::from(at(98, true, false)).repeat(3, Heater::Heat) else {
        return Err("heat past boiling".into());
    };
    writeln!(out, "heated three times from 98: {boiled}")?;
    Ok(umpire_testing::Files::from([("messages.txt".into(), out)]))
}
