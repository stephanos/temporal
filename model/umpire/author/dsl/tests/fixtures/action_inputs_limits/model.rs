use std::fmt::Write;

use umpire_dsl::{Action, InputDomain, PatternError};

#[derive(Action)]
enum User {
    Set {
        #[inputs(0..=u64::MAX)]
        first: u64,
        #[inputs(0..=u64::MAX)]
        second: u64,
    },
}

pub(crate) fn run() -> umpire_testing::RunResult {
    let overflow = User::__umpire_path_actions().count();
    assert_eq!(overflow, Err(PatternError::ExpansionOverflow));
    let empty = User::__umpire_path_Set().__umpire_input_second(Some(InputDomain::values(vec![]))).finish();
    assert_eq!(empty.count(), Ok(0));
    let finite = User::__umpire_path_Set()
        .__umpire_input_first(Some(InputDomain::range(2..=3)))
        .__umpire_input_second(Some(InputDomain::values(vec![7, 9])))
        .finish();
    assert_eq!(finite.count(), Ok(4));
    assert_eq!(finite.get(0), User::Set { first: 2, second: 7 });
    assert_eq!(finite.get(1), User::Set { first: 2, second: 9 });
    assert_eq!(finite.get(2), User::Set { first: 3, second: 7 });
    assert_eq!(finite.get(3), User::Set { first: 3, second: 9 });
    let mut output = String::new();
    writeln!(output, "overflow: {overflow:?}; empty: {:?}; finite: {:?}", empty.count(), finite.count())?;
    Ok(umpire_testing::Files::from([("values.txt".into(), output)]))
}
