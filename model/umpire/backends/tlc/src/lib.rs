//! Pinned TLC execution, complete reachable sets and typed counterexamples.
//!
//! Each existing main, progress or negative configuration explores without claims, then checks
//! its properties together. Failed batches recover canonical singleton traces in cfg order.
//! Executions use fingerprint polynomial zero (`-fp 0`) in both strategies: TLC otherwise chooses
//! a random polynomial, which can change the lasso even between identical singleton checks.
//!
//! `Tlc::property_workers` bounds singleton recovery within a check (default: available cores,
//! capped at two). Gates must additionally bound test processes and concurrent gate invocations.
//! With two test threads or two nextest JVM-group slots, the default permits at most four model
//! JVMs and 2 GiB of configured Java heap, plus native JVM memory. Run only one required-backend
//! gate invocation per resource allocation; this limit does not coordinate separate processes.
//!
//! The development-only `strategy-equivalence` feature also runs the sequential singleton strategy
//! beneath every `check`/`check_module` caller, including shared fixtures and source mutants. It
//! asserts exact states, property verdicts, trace assignments/private observers, lasso destinations
//! and typed evaluation fields. Raw subprocess output is excluded; both strategies must refuse
//! execution failures. It adds comparison work and is not enabled in the routine fast gate.

#![forbid(unsafe_code)]

mod decode;
mod module;
mod runner;

pub use decode::*;
pub use runner::*;
pub use umpire_backend_process::{Limits, Refusal};
pub use umpire_tla::values::*;

/// TLC's own evaluation of the emitted module failed, such as an integer overflow past TLC's
/// 32-bit integers or a failed `Assert`: a located finding, never a property verdict.
#[derive(Clone, Debug)]
pub struct Evaluation {
    /// The selected property TLC was evaluating, or `None` for the specification's initial
    /// predicate or next-state relation.
    pub property: Option<String>,
    /// TLC's report, with its nested messages and expression positions.
    pub message: String,
    /// The behavior up to the state TLC failed in, which is its last; `None` when TLC failed
    /// in an initial state, which it does not print.
    pub trace: Option<Trace>,
    /// That last state as TLC printed it.
    pub state: Option<String>,
}

impl std::fmt::Display for Evaluation {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        let property = self.property.as_ref().map_or_else(
            || {
                if self.trace.is_some() {
                    "the next-state relation"
                } else {
                    "the initial predicate"
                }
                .to_owned()
            },
            |property| format!("property {property}"),
        );
        write!(out, "TLC evaluation error in {property}: {}", self.message)?;
        match (&self.trace, &self.state) {
            (Some(trace), Some(state)) => {
                write!(out, "\nat state {} of the behavior:\n{state}", trace.states.len())?;
            }
            _ if self.property.is_some() => write!(out, "\nin an initial state")?,
            _ => {}
        }
        Ok(())
    }
}

impl std::error::Error for Evaluation {}

/// A TLC check that produced no verdict.
#[derive(Debug, thiserror::Error)]
pub enum Error {
    /// The backend refused the module, its execution or its output.
    #[error(transparent)]
    Refusal(#[from] Refusal),
    /// TLC failed to evaluate the module in a reached state.
    #[error(transparent)]
    Evaluation(#[from] Box<Evaluation>),
}

impl From<std::io::Error> for Error {
    fn from(error: std::io::Error) -> Self {
        Self::Refusal(error.into())
    }
}
