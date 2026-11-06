//! Temporal protobuf realization for the independently authored mini activity Model.

// The fixture runner compiles the Model independently of this realization.
#[path = "model.rs"]
mod included;
mod phase;
pub mod policy;
pub mod projection;
mod registry;

pub use projection::ActivityTarget;
pub use registry::{ActivityIdentity, FillInputs};

pub use included::activity;

use activity::{Client, Phase, Worker};
use umpire_dsl::Machine;
use umpire_runtime::{Proposal, prost_types::Any, protocol::Refusal};
use umpire_temporal_api::temporal::api::{
    failure::v1::Failure,
    workflowservice::v1::{DescribeActivityExecutionRequest, StartActivityExecutionRequest},
};

use crate::realization;

type Action = <activity::World as Machine>::Action;

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_runtime::Runtime<activity::World, Realization> {
    umpire_runtime::Runtime::new(
        activity::World::new(3, None, None, None, None),
        Realization::default(),
    )
}

/// The strict mini activity observation and worker program policy.
#[derive(Default)]
pub struct Realization {
    context: Option<ProposalContext>,
    manifest: Option<std::rc::Rc<policy::Manifest>>,
}

/// The admitted Start receipt: the actual bootstrap request and its run.
#[derive(Clone)]
pub struct ProposalContext {
    /// Actual admitted Start request.
    pub start: StartActivityExecutionRequest,
    /// Run created by the admitted Start.
    pub run_id: String,
}

impl ProposalContext {
    /// The admitted finite retry cap, absent when outside the selected domain.
    pub(crate) fn maximum_attempts(&self) -> Option<u8> {
        self.start
            .retry_policy
            .as_ref()
            .map(|policy| policy.maximum_attempts)
            .filter(|attempts| (1..=3).contains(attempts))
            .and_then(|attempts| u8::try_from(attempts).ok())
    }

    /// Refuse evidence whose nonempty namespace, activity and run differ from this receipt.
    pub(crate) fn matches(
        &self,
        namespace: &str,
        activity: &str,
        run: &str,
    ) -> Result<(), Refusal> {
        for (matches, component) in [
            (!namespace.is_empty() && namespace == self.start.namespace, "namespace"),
            (!activity.is_empty() && activity == self.start.activity_id, "activity_id"),
            (!run.is_empty() && run == self.run_id, "run_id"),
        ] {
            if !matches {
                return Err(Refusal::new(
                    umpire_runtime::protocol::RefusalCode::State,
                    format!("evidence {component} does not match Start receipt"),
                ));
            }
        }
        Ok(())
    }
}

impl Realization {
    pub(crate) fn configured(
        context: ProposalContext,
        manifest: Option<std::rc::Rc<policy::Manifest>>,
    ) -> Self {
        Self { context: Some(context), manifest }
    }

    /// The admitted Start receipt, absent before admission.
    #[cfg(test)]
    pub(crate) fn context(&self) -> Option<&ProposalContext> {
        self.context.as_ref()
    }
}

/// Header carrying the worker's scenario independently of application input.
pub const SCENARIO_HEADER: &str = "umpire-scenario";

#[realization(activity)]
#[reach(target = ActivityTarget, project = reach_projection, evidence = reach_evidence)]
impl Realization {
    #[propose(matches = matches_proposal, continue_polling = polling)]
    fn proposal(
        &self,
        world: &activity::World,
        state: &activity::Model,
        seed: u64,
    ) -> Result<Proposal<Action>, Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let describe = Any::pack(&(Self::read().fill)(self, world, context)?);
        let read = Any::pack(&crate::support::scheduled_read(describe, 10, 30_000, 10));
        if !umpire_dsl::Phase::closed(&state.phase) && seed % 2 == 1 {
            return Ok(Proposal::Read { request: read });
        }
        let hold = state.phase == Phase::Scheduled && seed & 2 != 0;
        if !umpire_dsl::Phase::closed(&state.phase) && !hold {
            return Err(Refusal::unsupported(
                "unheld cancellation of an open activity is outside the atomic cancellation Model",
            ));
        }
        let action = Action::from(Client::Cancel);
        let identity =
            umpire_runtime::capability::Identity::identify(self, &Any::pack(&context.start))?;
        let request = umpire_runtime::capability::Fill::fill(
            self,
            &action,
            world,
            &FillInputs::Cancel { request_id: String::new() },
            &identity,
        )?;
        let instruction = crate::support::instruction(
            umpire_runtime::capability::Request::pack_request(&request),
            hold,
            read,
        );
        Ok(Proposal::Action { action, request: Any::pack(&instruction) })
    }

    fn matches_proposal(&self, request: &Any, observation: &Any) -> Result<(), Refusal> {
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        let observed = crate::support::Observed::new(observation)?;
        let observation = observed.current();
        let actual = observation
            .request
            .as_ref()
            .ok_or_else(|| Refusal::malformed("actual request is missing"))?;
        if request.type_url
            == <umpire_temporal_api::umpire::temporal::v1::TemporalInstruction as prost::Name>::type_url()
        {
            let instruction: umpire_temporal_api::umpire::temporal::v1::TemporalInstruction = request.unpack()?;
            if observation.method == "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution" {
                if observed.acknowledged().is_some() { return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State, "compound evidence requires the reconciliation read")); }
                return if instruction.request.as_ref() == Some(actual) {
                    Ok(())
                } else {
                    Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                        "observation does not match cancellation invocation"))
                };
            }
            if observation.method == "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution" {
                let read: umpire_temporal_api::umpire::temporal::v1::ObserveActivity =
                    Any::required(instruction.not_found_read)?.unpack()?;
                let acknowledged = observed.acknowledged().ok_or_else(|| Refusal::new(umpire_runtime::protocol::RefusalCode::State, "reconciliation requires the actual cancellation acknowledgment"))?;
                if acknowledged.method != "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
                    || acknowledged.request.as_ref() != instruction.request.as_ref()
                    || !matches!(&acknowledged.result, Some(umpire_runtime::realization::observation::Result::Error(error)) if error.code == 5)
                {
                    return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State, "reconciliation acknowledgment does not match cancellation NotFound"));
                }
                return if read.request.as_ref() == Some(actual) {
                    Ok(())
                } else {
                    Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                        "Describe differs from cancellation reconciliation read"))
                };
            }
            return Err(Refusal::new(umpire_runtime::protocol::RefusalCode::State,
                "observation does not match cancellation invocation"));
        }
        if observed.acknowledged().is_some() {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "compound evidence requires a pending cancellation",
            ));
        }
        if observation.method
            != "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
        {
            return Err(Refusal::new(
                umpire_runtime::protocol::RefusalCode::State,
                "observation RPC does not correlate with proposal",
            ));
        }
        let actual: DescribeActivityExecutionRequest = actual.unpack()?;
        context.matches(&actual.namespace, &actual.activity_id, &actual.run_id)?;
        if request.type_url
            == <umpire_temporal_api::umpire::temporal::v1::ObserveActivity as prost::Name>::type_url(
            )
        {
            let instruction: umpire_temporal_api::umpire::temporal::v1::ObserveActivity =
                request.unpack()?;
            let expected: DescribeActivityExecutionRequest =
                Any::required(instruction.request)?.unpack()?;
            if expected != actual {
                return Err(Refusal::new(
                    umpire_runtime::protocol::RefusalCode::State,
                    "Describe differs from proposal",
                ));
            }
        } else {
            return Err(Refusal::unsupported("unknown proposal request"));
        }
        Ok(())
    }

    fn polling(
        &self,
        observation: &Any,
        state: &activity::Model,
        action: Option<&Action>,
    ) -> Result<bool, Refusal> {
        let observed = crate::support::Observed::new(observation)?;
        Ok(self.context.is_some()
            && observed.method
                == "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
            && (action.is_some() || !umpire_dsl::Phase::closed(&state.phase)))
    }

    pub(crate) fn programmed_failure(failure: &Failure) -> Result<(), Refusal> {
        use umpire_temporal_api::temporal::api::failure::v1::failure::FailureInfo;
        if matches!(&failure.failure_info, Some(FailureInfo::ApplicationFailureInfo(info)) if info.r#type == "umpire.unprogrammed")
        {
            return Err(Refusal::unsupported("unprogrammed worker task"));
        }
        Ok(())
    }

    pub(crate) fn same_failure(observed: Option<&Failure>, expected: Option<&Failure>) -> bool {
        fn content(failure: &Failure) -> Failure {
            let mut failure = failure.clone();
            failure.source.clear();
            failure.stack_trace.clear();
            failure.cause = failure.cause.map(|cause| Box::new(content(&cause)));
            failure
        }
        observed.map(content) == expected.map(content)
    }

    pub(crate) fn resolve_response(
        &self,
        scenario: &str,
        key: &policy::TaskKey,
    ) -> Result<Option<(Worker, Any)>, Refusal> {
        let manifest = self
            .manifest
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("admitted policy manifest is required"))?;
        manifest.admit_name(scenario)?;
        if !(1..=manifest.world().maximum_attempts).contains(&key.attempt) {
            return Err(Refusal::unsupported("worker attempt exceeds World.maximum_attempts"));
        }
        let Some(choice) = manifest.resolve(scenario, key) else {
            return Ok(None);
        };
        let context = self
            .context
            .as_ref()
            .ok_or_else(|| Refusal::unsupported("proposal configuration required"))?;
        Ok(Some((choice.action(key.attempt), choice.fill(manifest.world(), &context.start, key)?)))
    }
}

impl Realization {
    fn heartbeat_world(world: &activity::World, attempt: u8) -> Result<(), Refusal> {
        if !(1..=3).contains(&world.maximum_attempts)
            || !(1..=world.maximum_attempts).contains(&attempt)
        {
            return Err(Refusal::unsupported(
                "heartbeat attempt is outside World.maximum_attempts",
            ));
        }
        for (field, value) in [
            ("schedule_to_start", world.schedule_to_start),
            ("start_to_close", world.start_to_close),
            ("schedule_to_close", world.schedule_to_close),
            ("heartbeat_timeout", world.heartbeat_timeout),
        ] {
            if value.is_some_and(|ticks| !(1..=3).contains(&ticks.0)) {
                return Err(Refusal::unsupported(format!(
                    "heartbeat World.{field} is outside 1..=3"
                )));
            }
        }
        Ok(())
    }
}
