pub use gomad_wasmhost::linked::{Execution, Limits, ModelInput, Outcome, Request};

/// # Errors
/// Preserves the helper's typed refusal, including replay preflight metadata.
pub fn execute(request: Request) -> Result<Execution, Box<Outcome>> {
    crate::backstop::within(request.limits.wall_timeout, || {
        gomad_wasmhost::linked::execute(request)
    })
    .map_err(|error| Box::new(Outcome::refusal(&error)))
}
