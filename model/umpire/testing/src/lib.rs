//! Shared fixture discovery and golden runner for Umpire's tests.

pub mod backend;
pub mod boundary;
pub mod discovery;

mod render;
mod runner;

pub use render::{add_paths, render, render_rows};
pub use runner::{CHECKER, Consumer, SUITE, run, run_at, run_at_skipping};

/// Deterministic relative golden paths and their payloads.
pub type Files = std::collections::BTreeMap<String, String>;
/// A fixture or test failure.
pub type TestResult = Result<(), Box<dyn std::error::Error>>;
/// The files produced by one fixture.
pub type RunResult = Result<Files, Box<dyn std::error::Error>>;
/// A named fixture producer discovered by its owning crate's build script.
pub type Fixture = (&'static str, fn() -> RunResult);

/// Define a golden producer, optionally with a negative claim and additional files.
/// `fixture!(world, budget, negative, extra = files)` evaluates `files` once after
/// the checker reports and refuses paths that would replace an existing output.
#[macro_export]
macro_rules! fixture {
    ($world:expr, $budget:expr) => {
        pub(crate) fn run() -> $crate::RunResult {
            Ok($crate::render(&$crate::Checker::explore(&$world, $budget)?)?)
        }
    };
    ($world:expr, $budget:expr, $negative:expr) => {
        $crate::fixture!($world, $budget, $negative, extra = $crate::Files::new());
    };
    ($world:expr, $budget:expr, $negative:expr, extra = $extra:expr) => {
        pub(crate) fn run() -> $crate::RunResult {
            let mut files = $crate::render(&$crate::Checker::explore(&$world, $budget)?)?;
            let negative = $crate::Checker::refuted(&$world, $negative)?;
            let report = negative.verify_all()?;
            files.insert("negative_report.txt".into(), report.to_string());
            $crate::add_paths(&mut files, report);
            let extra: $crate::Files = $extra;
            for (file, content) in extra {
                if files.contains_key(&file) {
                    return Err(format!("duplicate fixture output: {file}").into());
                }
                files.insert(file, content);
            }
            Ok(files)
        }
    };
}

#[doc(hidden)]
pub use umpire_checker::Checker;
