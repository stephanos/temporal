//! Shared fixture discovery and golden runner for Umpire's tests.

pub mod boundary;
pub mod discovery;

mod render;
mod runner;

pub use render::{add_paths, render, render_rows};
pub use runner::{CHECKER, Consumer, run, run_at};

/// Deterministic relative golden paths and their payloads.
pub type Files = std::collections::BTreeMap<String, String>;
/// A fixture or test failure.
pub type TestResult = Result<(), Box<dyn std::error::Error>>;
/// The files produced by one fixture.
pub type RunResult = Result<Files, Box<dyn std::error::Error>>;
/// A named fixture producer discovered by its owning crate's build script.
pub type Fixture = (&'static str, fn() -> RunResult);

/// Define the golden producer for a world and budget, optionally with a negative claim.
#[macro_export]
macro_rules! fixture {
    ($world:expr, $budget:expr) => {
        pub(crate) fn run() -> $crate::RunResult {
            Ok($crate::render(&$crate::Checker::explore(&$world, $budget)?)?)
        }
    };
    ($world:expr, $budget:expr, $negative:expr) => {
        pub(crate) fn run() -> $crate::RunResult {
            let mut files = $crate::render(&$crate::Checker::explore(&$world, $budget)?)?;
            let negative = $crate::Checker::refuted(&$world, $negative)?;
            let report = negative.verify_all()?;
            files.insert("negative_report.txt".into(), report.to_string());
            $crate::add_paths(&mut files, report);
            Ok(files)
        }
    };
}

#[doc(hidden)]
pub use umpire_checker::Checker;
