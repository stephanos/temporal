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

pub(crate) fn backend_publication(
    observed: &Execution,
    limits: &crate::evidence::record::Limits,
) -> Result<Vec<u8>, String> {
    use crate::evidence::{backend_outcome, canonical};
    if observed.outcome.fuel_remaining > limits.fuel {
        return Err("backend outcome fuel exceeds admitted limit".into());
    }
    let Some(raw) = observed.evidence.get("native-output-accounting") else {
        if observed.timing.is_some() {
            return Err("backend outcome output accounting is missing".into());
        }
        return backend_observation(&observed.outcome).map(|value| value.0);
    };
    let retained = ["stdout", "stderr"]
        .into_iter()
        .try_fold(0_u64, |total, name| {
            total
                .checked_add(
                    observed
                        .evidence
                        .get(name)
                        .map_or(0, |bytes| bytes.len() as u64),
                )
                .ok_or("backend outcome output size overflow")
        })?;
    backend_outcome::output(
        raw,
        observed.outcome.output_bytes,
        retained,
        limits.output_bytes,
    )?;
    let mut backend = serde_json::to_value(observed.outcome.backend_evidence())
        .map_err(|error| error.to_string())?;
    backend["schema"] = "gomad.backend-outcome/v2".into();
    backend["output_accounting_sha256"] = canonical::digest(raw).into();
    canonical::encode(&backend)
}
