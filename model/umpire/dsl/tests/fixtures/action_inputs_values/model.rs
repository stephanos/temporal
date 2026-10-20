use std::collections::HashSet;
use std::fmt::Write;

use umpire_dsl::Action;

#[derive(Action)]
enum User {
    Idle,
    Set {
        #[inputs(0..=1)]
        value: u8,
        #[inputs([false, true])]
        enabled: bool,
    },
    Label(#[inputs([String::from("a"), String::from("b")])] String),
}

#[derive(Action)]
#[internal]
#[fault]
enum Clock {
    Tick(#[inputs(1..3)] u8),
}

#[derive(Action)]
#[fault]
enum Network {
    Drop,
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let mut output = String::new();
    let actions = User::actions();
    for action in &actions {
        writeln!(output, "{action:?} | {action}")?;
    }
    let unique: HashSet<_> = actions.iter().chain(actions.iter()).cloned().collect();
    writeln!(output, "unique cloned actions: {}", unique.len())?;
    let outside =
        User::__umpire_path_Set().__umpire_input_value(Some(umpire_dsl::InputDomain::values(vec![7]))).finish();
    assert_eq!(outside.count(), Ok(2));
    assert_eq!(outside.get(0), User::Set { value: 7, enabled: false });
    assert_eq!(User::__umpire_path_actions().count(), Ok(7));
    for action in Clock::actions() {
        writeln!(output, "{action}: internal={} fault={}", action.internal(), action.fault())?;
    }
    let action = Network::Drop;
    writeln!(output, "{action}: internal={} fault={}", action.internal(), action.fault())?;
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
