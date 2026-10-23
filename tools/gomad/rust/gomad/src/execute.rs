pub use gomad_wasmhost::linked::{Execution, Limits, ModelInput, Outcome, Request};

/// # Errors
/// Preserves the helper's typed refusal, including replay preflight metadata.
pub fn execute(request: Request) -> Result<Execution, Box<Outcome>> {
    crate::backstop::within(request.limits.wall_timeout, || {
        gomad_wasmhost::linked::execute(request)
    })
    .map_err(|error| Box::new(Outcome::refusal(&error)))
}

pub(crate) fn cache_failure(outcome: &Outcome) -> bool {
    outcome.termination == "infrastructure" && outcome.message.starts_with("compile cache:")
        || outcome.termination == "capacity" && outcome.message == "compile cache artifact capacity"
}

pub(crate) fn cache_error(message: &str) -> bool {
    message.starts_with("infrastructure: compile cache:")
        || message == "capacity: compile cache artifact capacity"
}

pub(crate) fn backend_observation(outcome: &Outcome) -> Result<(Vec<u8>, Vec<u8>), String> {
    Ok((
        crate::evidence::canonical::encode(&outcome.backend_evidence())?,
        crate::evidence::canonical::encode(&outcome.metrics)?,
    ))
}
