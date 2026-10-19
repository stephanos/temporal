//! Temporal protobuf binding for the independently authored mini activity Model.

#[path = "model.rs"]
mod included;

pub use included::activity;

use umpire_dsl::Machine;
use umpire_temporal_api::{
    temporal::api::{
        common::v1::{Payload, Payloads},
        enums::v1::{ActivityExecutionStatus, PendingActivityState},
        failure::v1::{ApplicationFailureInfo, Failure, failure::FailureInfo},
        workflowservice::v1::{
            DescribeActivityExecutionResponse, PollActivityTaskQueueResponse,
            RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        },
    },
    umpire::temporal::v1::{Callback, Observation, Reply},
};
use umpire_wasm::protocol::Refusal;
use umpire_wasm::{AnswerPlan, Projection, Realize, Transition, google::protobuf::Any};

const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";
type Action = <activity::World as Machine>::Action;

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_wasm::Runtime<activity::World, Realization> {
    umpire_wasm::Runtime::new(activity::World, Realization)
}

/// The strict mini activity observation and callback registry.
pub struct Realization;

impl Realize<activity::World> for Realization {
    fn observe(
        &self,
        observation: &Any,
        state: &activity::Model,
    ) -> Result<Projection<Action>, Refusal> {
        let phase = project(observation)?;
        if state.phase == phase {
            return Ok(Projection::Current);
        }
        let paths = match phase {
            activity::Phase::Completed => vec![vec![Transition {
                action: activity::Worker::RespondCompleted.into(),
                choice: 0,
            }]],
            activity::Phase::Failed => {
                vec![vec![Transition { action: activity::Worker::RespondFailed.into(), choice: 0 }]]
            }
            _ => Vec::new(),
        };
        Ok(Projection::Candidates(paths))
    }

    fn matches(&self, observation: &Any, state: &activity::Model) -> Result<bool, Refusal> {
        Ok(project(observation)? == state.phase)
    }

    fn answer(&self, callback: &Any, _: &activity::Model) -> Result<AnswerPlan<Action>, Refusal> {
        let callback: Callback = callback.unpack()?;
        let task: PollActivityTaskQueueResponse = Any::required(callback.task)?.unpack()?;
        let policy = Policy::parse(task.input)?;
        let (response_action, request) = policy.reply();
        Ok(AnswerPlan {
            delivery: vec![Transition { action: activity::Worker::Poll.into(), choice: 0 }],
            response_action: response_action.into(),
            reply: Any::pack(&Reply { request: Some(request) }),
        })
    }
}

enum Policy {
    Complete(Payload),
    Fail,
}

impl Policy {
    fn parse(input: Option<Payloads>) -> Result<Self, Refusal> {
        let payloads = input.ok_or_else(|| Refusal::malformed("callback input is missing"))?;
        let [payload] = payloads.payloads.as_slice() else {
            return Err(Refusal::malformed("callback requires exactly one input payload"));
        };
        let encoding = payload
            .metadata
            .get("encoding")
            .ok_or_else(|| Refusal::malformed("payload encoding is missing"))?;
        if payload.metadata.len() != 1 {
            return Err(Refusal::malformed("payload metadata must contain only encoding"));
        }
        if encoding != b"binary/plain" {
            return Err(Refusal::unsupported("unsupported payload encoding"));
        }
        match payload.data.as_slice() {
            b"Complete" => Ok(Self::Complete(Payload {
                metadata: payload.metadata.clone(),
                data: b"Complete".to_vec(),
                ..Default::default()
            })),
            b"Fail" => Ok(Self::Fail),
            _ => Err(Refusal::unsupported("unsupported callback policy")),
        }
    }

    fn reply(self) -> (activity::Worker, Any) {
        match self {
            Self::Complete(payload) => (
                activity::Worker::RespondCompleted,
                Any::pack(&RespondActivityTaskCompletedRequest {
                    result: Some(Payloads { payloads: vec![payload] }),
                    ..Default::default()
                }),
            ),
            Self::Fail => (
                activity::Worker::RespondFailed,
                Any::pack(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure {
                        message: "Fail".into(),
                        failure_info: Some(FailureInfo::ApplicationFailureInfo(
                            ApplicationFailureInfo {
                                r#type: "MiniActivityFailure".into(),
                                non_retryable: true,
                                ..Default::default()
                            },
                        )),
                        ..Default::default()
                    }),
                    ..Default::default()
                }),
            ),
        }
    }
}

fn project(observation: &Any) -> Result<activity::Phase, Refusal> {
    use ActivityExecutionStatus as Status;
    use PendingActivityState as Run;

    let observation: Observation = observation.unpack()?;
    if observation.method.is_empty() {
        return Err(Refusal::malformed("observation method is missing"));
    }
    if observation.method != DESCRIBE {
        return Err(Refusal::unsupported("unsupported observation method"));
    }
    let response: DescribeActivityExecutionResponse =
        Any::required(observation.response)?.unpack()?;
    let info = response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?;
    let status = ActivityExecutionStatus::try_from(info.status)
        .map_err(|_| Refusal::unsupported("unknown activity execution status"))?;
    let run_state = PendingActivityState::try_from(info.run_state)
        .map_err(|_| Refusal::unsupported("unknown activity run state"))?;
    match (status, run_state) {
        (Status::Unspecified, _) => Err(Refusal::malformed("activity status is missing")),
        (Status::Running, Run::Scheduled) => Ok(activity::Phase::Scheduled),
        (Status::Running, Run::Started) => Ok(activity::Phase::Started),
        (Status::Running, Run::Unspecified) => {
            Err(Refusal::malformed("running activity run state is missing"))
        }
        (Status::Running, _) => Err(Refusal::unsupported("unsupported running activity state")),
        (Status::Completed, Run::Unspecified) => Ok(activity::Phase::Completed),
        (Status::Failed, Run::Unspecified) => Ok(activity::Phase::Failed),
        (Status::Canceled, Run::Unspecified) => Ok(activity::Phase::Canceled),
        (Status::TimedOut, Run::Unspecified) => Ok(activity::Phase::TimedOut),
        (Status::Completed | Status::Failed | Status::Canceled | Status::TimedOut, _) => {
            Err(Refusal::malformed("terminal activity must have unspecified run state"))
        }
        _ => Err(Refusal::unsupported("unsupported activity execution status")),
    }
}
