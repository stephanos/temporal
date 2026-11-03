//! Protocol-local construction from exact neutral effective parameters.

use super::{
    parameters::{ParameterError, Parameters},
    protocol::{Ticks, World},
};

/// Construct this protocol's independent World without raw-request defaulting.
///
/// # Errors
/// Refuses invalid effective values or any adapter/schema disagreement.
pub fn world(parameters: Parameters) -> Result<World, ParameterError> {
    let exact = Parameters::new(
        parameters.maximum_attempts,
        [
            parameters.schedule_to_start,
            parameters.start_to_close,
            parameters.schedule_to_close,
            parameters.heartbeat_timeout,
        ],
        parameters.asynchronous_cancel,
    )?;
    let timeout = |raw: Option<u8>| -> Result<Option<Ticks>, ParameterError> {
        raw.map(|ticks| {
            Ticks::try_new(ticks)
                .ok_or_else(|| ParameterError("configured timeout outside declared domain".into()))
        })
        .transpose()
    };
    let world = World::new(
        exact.maximum_attempts,
        timeout(exact.schedule_to_start)?,
        timeout(exact.start_to_close)?,
        timeout(exact.schedule_to_close)?,
        timeout(exact.heartbeat_timeout)?,
        exact.asynchronous_cancel,
    );
    #[cfg(not(target_arch = "wasm32"))]
    exact.validate_world(&world)?;
    Ok(world)
}

/// Validate every independently declared full-state field, including latent timers.
///
/// # Errors
/// Refuses erased, renamed, reordered or differently typed fields.
#[cfg(not(target_arch = "wasm32"))]
pub fn validate_state_schema(model: &umpire_dsl::__lifted::Lifted) -> Result<(), ParameterError> {
    use umpire_dsl::__lifted::{Bounded, Type};
    let enumeration = |name: &str, variants: &[&str]| Type::Enum {
        name: name.into(),
        variants: variants.iter().map(|variant| format!("{name}::{variant}")).collect(),
    };
    let remaining = Type::Optional(Bounded {
        name: "Remaining".into(),
        field: Some("ticks".into()),
        backing: "u8".into(),
        min: 0,
        max: 255,
        low: 0,
        high: 3,
    });
    let mut expected = vec![
        ("lifecycle", enumeration("Lifecycle", &["Queued", "Executing", "Closed"])),
        ("attempt", Type::Int { min: 0, max: 255 }),
        ("cancellation_pending", Type::Bool),
        (
            "terminal_outcome",
            enumeration("Terminal", &["None", "Completed", "Failed", "Canceled", "TimedOut"]),
        ),
        (
            "retained_failure",
            enumeration(
                "Failure",
                &[
                    "None",
                    "RetryableWorker",
                    "PermanentWorker",
                    "ScheduleToStart",
                    "StartToClose",
                    "ScheduleToClose",
                    "Heartbeat",
                ],
            ),
        ),
    ];
    expected.extend(
        ["queued_budget", "attempt_budget", "execution_budget", "heartbeat_budget"]
            .into_iter()
            .map(|name| (name, remaining.clone())),
    );
    if model.fields.len() != expected.len()
        || model
            .fields
            .iter()
            .zip(expected)
            .any(|(actual, (name, ty))| actual.name != name || actual.ty != ty)
    {
        return Err(ParameterError(
            "full protocol state schema differs from its exact field contract".into(),
        ));
    }
    Ok(())
}
