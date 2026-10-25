//! Conditional projections whose semantics depend on sibling fields.

use umpire_catalog::{Duration, GenerationError};
use umpire_catalog::{View, kinds};
use umpire_temporal_api::temporal::api::{
    common::v1::RetryPolicy, failure::v1::ActivityFailureInfo,
};

/// Initial retry interval, disabled when retries are disabled.
pub mod initial_interval {
    use super::{Duration, GenerationError, RetryPolicy, View, kinds};
    /// Projects the retry interval under the server's retry-disable convention.
    pub fn view(value: &RetryPolicy) -> Option<Duration> {
        (value.maximum_attempts != 1)
            .then(|| <_ as View<kinds::Duration>>::view(&value.initial_interval))
            .flatten()
    }
    /// Writes the underlying interval.
    pub fn set(value: &mut RetryPolicy, interval: Option<Duration>) -> Result<(), GenerationError> {
        <_ as View<kinds::Duration>>::set(&mut value.initial_interval, interval)
    }
}

/// Maximum retry interval, disabled when retries are disabled.
pub mod maximum_interval {
    use super::{Duration, GenerationError, RetryPolicy, View, kinds};
    /// Projects the retry interval under the server's retry-disable convention.
    pub fn view(value: &RetryPolicy) -> Option<Duration> {
        (value.maximum_attempts != 1)
            .then(|| <_ as View<kinds::Duration>>::view(&value.maximum_interval))
            .flatten()
    }
    /// Writes the underlying interval.
    pub fn set(value: &mut RetryPolicy, interval: Option<Duration>) -> Result<(), GenerationError> {
        <_ as View<kinds::Duration>>::set(&mut value.maximum_interval, interval)
    }
}

/// Retry backoff validation uses the server's zero/default and disabled semantics.
pub mod backoff_coefficient {
    use super::{GenerationError, RetryPolicy};
    /// Projects the coefficient used for validation.
    pub fn view(value: &RetryPolicy) -> i32 {
        i32::from(
            value.maximum_attempts == 1
                || value.backoff_coefficient == 0.0
                || value.backoff_coefficient.partial_cmp(&1.0) != Some(std::cmp::Ordering::Less),
        )
    }
    /// Writes the supplied coefficient.
    pub fn set(value: &mut RetryPolicy, coefficient: i32) -> Result<(), GenerationError> {
        value.backoff_coefficient = if coefficient < 1 { 0.5 } else { 1.0 };
        Ok(())
    }
}

/// Activity failure information carries its name through a nested message.
pub mod activity_type {
    use super::{ActivityFailureInfo, GenerationError};
    /// Projects the nested activity name.
    pub fn view(value: &ActivityFailureInfo) -> Option<String> {
        value.activity_type.as_ref().map(|value| value.name.clone())
    }
    /// Replaces the nested activity name.
    pub fn set(
        value: &mut ActivityFailureInfo,
        name: Option<String>,
    ) -> Result<(), GenerationError> {
        value.activity_type =
            name.map(|name| umpire_temporal_api::temporal::api::common::v1::ActivityType { name });
        Ok(())
    }
}
