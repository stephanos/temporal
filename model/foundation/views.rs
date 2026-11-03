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

/// `TaskQueue.normal_name` is defined only for sticky queues in the pinned protocol.
pub mod poll_normal_name {
    use umpire_catalog::GenerationError;
    use umpire_temporal_api::temporal::api::{
        enums::v1::TaskQueueKind, workflowservice::v1::PollActivityTaskQueueRequest,
    };

    /// Projects the nonsticky queue name whose only admitted length is zero.
    pub fn view(value: &PollActivityTaskQueueRequest) -> String {
        value
            .task_queue
            .as_ref()
            .filter(|queue| queue.kind != TaskQueueKind::Sticky as i32)
            .map_or_else(String::new, |queue| queue.normal_name.clone())
    }

    /// Writes the projection without changing the declared queue kind.
    pub fn set(
        value: &mut PollActivityTaskQueueRequest,
        name: String,
    ) -> Result<(), GenerationError> {
        let queue = value.task_queue.as_mut().ok_or(GenerationError::UnsupportedType)?;
        if queue.kind == TaskQueueKind::Sticky as i32 && !name.is_empty() {
            return Err(GenerationError::InapplicableMutation);
        }
        queue.normal_name = name;
        Ok(())
    }
}

/// The documented public retry contract applies to explicit enabled policy intervals.
pub mod start_retry_order {
    use umpire_catalog::{FieldValue, GenerationError, Kind, View, kinds};
    use umpire_runtime::prost_types::Duration;
    use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;

    fn projected(value: Option<Duration>) -> Option<(umpire_catalog::Duration, i128)> {
        let duration = <_ as View<kinds::Duration>>::view(&value)?;
        duration
            .violations(&Kind::Duration(umpire_catalog::NumberRules::default()))
            .is_empty()
            .then_some((
                duration,
                i128::from(duration.seconds) * 1_000_000_000 + i128::from(duration.nanos),
            ))
    }

    /// Compares explicit positive intervals, preserving default and disabled policies.
    pub fn view(value: &StartActivityExecutionRequest) -> i32 {
        let Some(policy) = &value.retry_policy else { return 1 };
        if policy.maximum_attempts == 1 {
            return 1;
        }
        let (Some(initial), Some(maximum)) =
            (projected(policy.initial_interval), projected(policy.maximum_interval))
        else {
            return 1;
        };
        i32::from(initial.1 <= 0 || maximum.1 <= 0 || initial.1 <= maximum.1)
    }

    /// Writes an ordered or inverted initial interval under the actual explicit maximum.
    pub fn set(
        value: &mut StartActivityExecutionRequest,
        valid: i32,
    ) -> Result<(), GenerationError> {
        if valid >= 1 && view(value) >= 1 {
            return Ok(());
        }
        let policy = value.retry_policy.as_mut().ok_or(GenerationError::UnsupportedType)?;
        let (maximum, total) =
            projected(policy.maximum_interval).ok_or(GenerationError::InapplicableMutation)?;
        if valid < 1 {
            if policy.maximum_attempts == 1 || total <= 0 {
                return Err(GenerationError::InapplicableMutation);
            }
            let mut candidate = value.clone();
            candidate
                .retry_policy
                .as_mut()
                .ok_or(GenerationError::UnsupportedType)?
                .initial_interval = Some(Duration {
                seconds: maximum.seconds.checked_add(1).ok_or(GenerationError::SizeLimit)?,
                nanos: 0,
            });
            if view(&candidate) != 0 {
                return Err(GenerationError::InapplicableMutation);
            }
            *value = candidate;
        } else {
            <_ as View<kinds::Duration>>::set(&mut policy.initial_interval, Some(maximum))?;
        }
        Ok(())
    }
}
