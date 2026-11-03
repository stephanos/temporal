//! Exact effective budgets and the separate pinned Start timeout-admission boundary.

use std::fmt;

/// Effective selected budgets; absent deadlines remain absent, including abstract checker Worlds.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct Parameters {
    /// Explicit finite retry cap.
    pub maximum_attempts: u8,
    /// Effective waiting budget for each attempt.
    pub schedule_to_start: Option<u8>,
    /// Effective execution budget for each attempt.
    pub start_to_close: Option<u8>,
    /// Effective budget spanning all attempts.
    pub schedule_to_close: Option<u8>,
    /// Effective heartbeat budget.
    pub heartbeat_timeout: Option<u8>,
    /// Whether running cancellation waits for an attempt boundary.
    pub asynchronous_cancel: bool,
}

/// A parameter, schema or adapter disagreement.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct ParameterError(pub String);

impl fmt::Display for ParameterError {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        out.write_str(&self.0)
    }
}

impl std::error::Error for ParameterError {}

impl Parameters {
    /// Validate effective values without applying raw-request defaults.
    ///
    /// # Errors
    /// Refuses attempts outside one through three and present timeouts outside one through three.
    pub fn new(
        maximum_attempts: u8,
        timeouts: [Option<u8>; 4],
        asynchronous_cancel: bool,
    ) -> Result<Self, ParameterError> {
        if !(1..=3).contains(&maximum_attempts) {
            return Err(ParameterError("maximum_attempts must be in 1..=3".into()));
        }
        let [schedule_to_start, start_to_close, schedule_to_close, heartbeat_timeout] = timeouts;
        for (name, value) in TIMEOUT_NAMES.into_iter().zip(timeouts) {
            if value.is_some_and(|ticks| !(1..=3).contains(&ticks)) {
                return Err(ParameterError(format!("{name} must be absent or in 1..=3")));
            }
        }
        Ok(Self {
            maximum_attempts,
            schedule_to_start,
            start_to_close,
            schedule_to_close,
            heartbeat_timeout,
            asynchronous_cancel,
        })
    }

    /// Check an adapter's exact effective result and retain its independently declared schema.
    ///
    /// # Errors
    /// Refuses an invalid schema, invalid values or any difference from this parameter record.
    #[cfg(not(target_arch = "wasm32"))]
    pub fn validate_world<M: umpire_dsl::Values + umpire_dsl::__lifted::Lifts>(
        &self,
        world: &M,
    ) -> Result<WorldProvenance, ParameterError> {
        let (actual, provenance) = inspect_world(world)?;
        if actual != *self {
            return Err(ParameterError(format!(
                "effective World differs from exact parameters: expected {self:?}, got {actual:?}"
            )));
        }
        Ok(provenance)
    }
}

const TIMEOUT_NAMES: [&str; 4] =
    ["schedule_to_start", "start_to_close", "schedule_to_close", "heartbeat_timeout"];

/// Raw timeout selections at the pinned standalone Start boundary, before defaults.
/// This record covers the bounded timeout/retry scope, not other Start request admission.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct RawStart {
    /// Explicit selected finite retry cap.
    pub maximum_attempts: u8,
    /// Caller-selected waiting timeout, before its documented default.
    pub schedule_to_start: Option<u8>,
    /// Caller-selected attempt timeout.
    pub start_to_close: Option<u8>,
    /// Caller-selected spanning timeout.
    pub schedule_to_close: Option<u8>,
    /// Caller-selected heartbeat timeout.
    pub heartbeat_timeout: Option<u8>,
    /// Selected cancellation mode.
    pub asynchronous_cancel: bool,
}

impl RawStart {
    /// Apply the pinned API's waiting-timeout default at this named boundary only.
    ///
    /// # Errors
    /// Refuses out-of-scope supplied values and absence of both execution and spanning timeouts.
    pub fn into_effective(self) -> Result<Parameters, ParameterError> {
        let supplied = Parameters::new(
            self.maximum_attempts,
            [
                self.schedule_to_start,
                self.start_to_close,
                self.schedule_to_close,
                self.heartbeat_timeout,
            ],
            self.asynchronous_cancel,
        )?;
        if supplied.start_to_close.is_none() && supplied.schedule_to_close.is_none() {
            return Err(ParameterError(
                "raw Start requires start_to_close or schedule_to_close".into(),
            ));
        }
        Ok(Parameters {
            schedule_to_start: supplied.schedule_to_start.or(supplied.schedule_to_close),
            ..supplied
        })
    }
}

/// An independently declared World field, its type/domain source and exact selected value.
#[cfg(not(target_arch = "wasm32"))]
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FieldProvenance {
    /// Authored field name.
    pub name: String,
    /// Full nominal/domain/backing descriptor.
    pub ty: umpire_dsl::__lifted::Type,
    /// Source coordinate of the independent declaration.
    pub location: umpire_dsl::__lifted::Diagnostic,
    /// Exact native exported value, including optional presence.
    pub value: umpire_dsl::Value,
}

/// Complete source and field accounting for one selected local World.
#[cfg(not(target_arch = "wasm32"))]
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct WorldProvenance {
    /// Source file of the independently admitted machine.
    pub source: String,
    /// Every World field in declaration order.
    pub fields: Vec<FieldProvenance>,
}

/// Read exact effective parameters only after validating every independent field descriptor.
///
/// # Errors
/// Refuses source admission failures, omissions, extras, changed names/types/domains or invalid values.
#[cfg(not(target_arch = "wasm32"))]
pub fn validate_effective_world<M: umpire_dsl::Values + umpire_dsl::__lifted::Lifts>(
    world: &M,
) -> Result<Parameters, ParameterError> {
    inspect_world(world).map(|(parameters, _)| parameters)
}

#[cfg(not(target_arch = "wasm32"))]
fn expected_schema() -> Vec<(&'static str, umpire_dsl::__lifted::Type)> {
    use umpire_dsl::__lifted::{Bounded, Type};
    let ticks = Type::Optional(Bounded {
        name: "Ticks".into(),
        field: None,
        backing: "u8".into(),
        min: 0,
        max: 255,
        low: 1,
        high: 3,
    });
    let mut fields = vec![("maximum_attempts", Type::Int { min: 0, max: 255 })];
    fields.extend(TIMEOUT_NAMES.map(|name| (name, ticks.clone())));
    fields.push(("asynchronous_cancel", Type::Bool));
    fields
}

#[cfg(not(target_arch = "wasm32"))]
fn inspect_world<M: umpire_dsl::Values + umpire_dsl::__lifted::Lifts>(
    world: &M,
) -> Result<(Parameters, WorldProvenance), ParameterError> {
    use umpire_dsl::Value;
    let embedded = M::__umpire_lifted();
    let model = embedded.model.map_err(|error| {
        ParameterError(format!(
            "{}:{}:{}: {}",
            embedded.file, error.line, error.column, error.message
        ))
    })?;
    let expected = expected_schema();
    let values = world.values();
    if !model.declared_world
        || model.world_fields.len() != expected.len()
        || values.len() != expected.len()
    {
        return Err(ParameterError(
            "effective World must declare and export exactly six fields".into(),
        ));
    }
    let mut provenance = WorldProvenance { source: embedded.file, fields: Vec::new() };
    for ((field, (name, ty)), (exported_name, value)) in
        model.world_fields.iter().zip(&expected).zip(&values)
    {
        if field.name != *name || field.ty != *ty || exported_name != name {
            return Err(ParameterError(format!(
                "effective World field {name} differs: declaration {} {:?}, export {exported_name}",
                field.name, field.ty
            )));
        }
        provenance.fields.push(FieldProvenance {
            name: field.name.clone(),
            ty: field.ty.clone(),
            location: field.location.clone(),
            value: value.clone(),
        });
    }
    let Value::Int(maximum) = values[0].1 else {
        return Err(ParameterError("maximum_attempts must export an integer".into()));
    };
    let maximum = u8::try_from(maximum)
        .map_err(|_| ParameterError("maximum_attempts is outside its backing type".into()))?;
    let mut timeouts = [None; 4];
    for (slot, (name, value)) in values[1..5].iter().enumerate() {
        timeouts[slot] =
            match value {
                Value::Optional(None) => None,
                Value::Optional(Some(payload)) => {
                    let Value::Bounded { name: "Ticks", raw } = payload.as_ref() else {
                        return Err(ParameterError(format!("{name} must export optional Ticks")));
                    };
                    Some(u8::try_from(*raw).map_err(|_| {
                        ParameterError(format!("{name} is outside its backing type"))
                    })?)
                }
                _ => return Err(ParameterError(format!("{name} must export an optional value"))),
            };
    }
    let Value::Bool(asynchronous) = values[5].1 else {
        return Err(ParameterError("asynchronous_cancel must export a boolean".into()));
    };
    Ok((Parameters::new(maximum, timeouts, asynchronous)?, provenance))
}
