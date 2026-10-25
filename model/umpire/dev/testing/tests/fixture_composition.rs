//! Fixture producers retain checker outputs when adding suite goldens.

#![forbid(unsafe_code)]

use std::sync::atomic::{AtomicUsize, Ordering};
use umpire_dsl::machine;
use umpire_testing::Files;

#[machine]
mod switch {
    use umpire_dsl::{Action, State};

    #[derive(State)]
    pub(crate) struct Model {
        pub on: bool,
    }

    #[derive(Action, Copy)]
    pub(crate) enum User {
        TurnOn,
    }

    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::TurnOn => self.on = true,
            }
        }
    }
}

mod composed {
    use super::{AtomicUsize, Files, Ordering, switch};
    use umpire_dsl::Claim;

    pub(crate) static CALLS: AtomicUsize = AtomicUsize::new(0);

    fn files(name: &str, content: impl FnOnce() -> String) -> Files {
        CALLS.fetch_add(1, Ordering::Relaxed);
        Files::from([(name.into(), content())])
    }

    umpire_testing::fixture!(
        switch::world(),
        2,
        Claim::new("never_on", |_, state: &switch::Model| (!state.on).into()),
        extra = files("coverage.txt", || "covered\n".into())
    );
}

mod collision {
    use super::{Files, switch};
    use umpire_dsl::Claim;

    umpire_testing::fixture!(
        switch::world(),
        2,
        Claim::new("never_on", |_, state: &switch::Model| (!state.on).into()),
        extra = Files::from([("negative_report.txt".into(), "overwritten\n".into())])
    );
}

#[test]
fn fixture_appends_suite_files_after_retaining_negative_outputs() -> umpire_testing::TestResult {
    let files = composed::run()?;
    assert_eq!(composed::CALLS.load(Ordering::Relaxed), 1);
    assert_eq!(files.get("coverage.txt").map(String::as_str), Some("covered\n"));
    assert!(files.get("negative_report.txt").is_some_and(|report| report.contains("never_on")));
    assert!(files.contains_key("paths/never_on.txt"));
    assert!(files.contains_key("report.txt"));
    Ok(())
}

#[test]
fn fixture_refuses_extra_files_that_replace_checker_outputs() -> umpire_testing::TestResult {
    let error = collision::run().err().ok_or("extra file replaced the negative report")?;
    assert!(error.to_string().contains("duplicate fixture output: negative_report.txt"), "{error}");
    Ok(())
}
