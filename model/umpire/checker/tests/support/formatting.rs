use std::fmt::{self, Write};

use umpire_checker::{Checker, Path, PathStep};

use super::support::{Mode, Tiny};

struct RefusingWriter;

impl Write for RefusingWriter {
    fn write_str(&mut self, _: &str) -> fmt::Result {
        Err(fmt::Error)
    }
}

struct RefusingValue;

impl fmt::Display for RefusingValue {
    fn fmt(&self, _: &mut fmt::Formatter<'_>) -> fmt::Result {
        Err(fmt::Error)
    }
}

#[test]
fn report_and_path_propagate_formatter_errors() -> umpire_testing::TestResult {
    let result = Checker::explore(&Tiny(Mode::Fails), 3)?;
    let report = result.verify_all()?;
    assert_eq!(write!(RefusingWriter, "{report}"), Err(fmt::Error));
    let path = report.claims[0].path.as_ref().ok_or("missing refutation")?;
    assert_eq!(write!(RefusingWriter, "{path}"), Err(fmt::Error));
    let path = Path {
        start: RefusingValue,
        steps: Vec::<PathStep<RefusingValue, RefusingValue>>::new(),
        cycle_start: None,
        failure: None,
    };
    assert_eq!(write!(String::new(), "{path}"), Err(fmt::Error));
    Ok(())
}
