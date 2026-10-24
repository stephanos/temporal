//! Pinned TLC execution, complete reachable sets and typed counterexamples.

mod decode;
mod module;
mod runner;

pub use decode::*;
pub use runner::*;
pub use umpire_tla::{Limits, Refusal, values::*};

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
