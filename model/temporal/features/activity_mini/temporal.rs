//! Temporal protobuf binding for the independently authored mini activity Model.

#[path = "model.rs"]
mod included;

pub use included::activity;

use prost::Message;
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
use umpire_wasm::{AnswerPlan, Projection, Realize, Transition, google::protobuf::Any, protocol};

const OBSERVATION: &str = "type.googleapis.com/umpire.temporal.v1.Observation";
const CALLBACK: &str = "type.googleapis.com/umpire.temporal.v1.Callback";
const REPLY: &str = "type.googleapis.com/umpire.temporal.v1.Reply";
const DESCRIBE_RESPONSE: &str =
    "type.googleapis.com/temporal.api.workflowservice.v1.DescribeActivityExecutionResponse";
const POLL_RESPONSE: &str =
    "type.googleapis.com/temporal.api.workflowservice.v1.PollActivityTaskQueueResponse";
const COMPLETED_REQUEST: &str =
    "type.googleapis.com/temporal.api.workflowservice.v1.RespondActivityTaskCompletedRequest";
const FAILED_REQUEST: &str =
    "type.googleapis.com/temporal.api.workflowservice.v1.RespondActivityTaskFailedRequest";
const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";

/// Named action carrier for the independent Model's three actors.
#[derive(Clone, PartialEq, Eq, Hash)]
pub enum Action {
    /// Client input.
    Client(activity::Client),
    /// Worker input.
    Worker(activity::Worker),
    /// Abstract deadline input.
    Deadline(activity::Deadline),
}

impl From<activity::Client> for Action {
    fn from(value: activity::Client) -> Self {
        Self::Client(value)
    }
}

impl From<activity::Worker> for Action {
    fn from(value: activity::Worker) -> Self {
        Self::Worker(value)
    }
}

impl From<activity::Deadline> for Action {
    fn from(value: activity::Deadline) -> Self {
        Self::Deadline(value)
    }
}

impl umpire_dsl::Spell for Action {
    fn spell(&self, out: &mut String) {
        match self {
            Self::Client(value) => value.spell(out),
            Self::Worker(value) => value.spell(out),
            Self::Deadline(value) => value.spell(out),
        }
    }
}

/// Named guest world delegating transitions to the independent Model.
pub struct World;

impl Machine for World {
    type State = activity::Model;
    type Action = Action;
    type Outcome = ();
    type Fact = ();

    fn starts(&self) -> Vec<Self::State> {
        activity::world().starts()
    }

    fn actions(&self) -> Vec<Action> {
        vec![
            activity::Client::Cancel.into(),
            activity::Worker::Poll.into(),
            activity::Worker::RespondCompleted.into(),
            activity::Worker::RespondFailed.into(),
            activity::Deadline::ScheduleToStart.into(),
            activity::Deadline::StartToClose.into(),
        ]
    }

    fn internal(&self, action: &Action) -> bool {
        let world = activity::world();
        match action {
            Action::Client(value) => world.internal(&(*value).into()),
            Action::Worker(value) => world.internal(&(*value).into()),
            Action::Deadline(value) => world.internal(&(*value).into()),
        }
    }

    fn step(&self, state: &Self::State, action: Action) -> umpire_dsl::Steps<Self::State, (), ()> {
        let world = activity::world();
        match action {
            Action::Client(value) => world.step(state, value.into()),
            Action::Worker(value) => world.step(state, value.into()),
            Action::Deadline(value) => world.step(state, value.into()),
        }
    }

    fn postconditions(&self) -> Vec<umpire_dsl::StepClaim<Self>> {
        vec![umpire_dsl::StepClaim::<Self>::new(
            "terminal_remains",
            |_, before, after, action, (), _| {
                let world = activity::world();
                let claims = world.postconditions();
                let emitted = ((), Vec::new(), after.clone());
                match action {
                    Action::Client(value) => {
                        claims[0].holds(&world, before, &(*value).into(), &emitted)
                    }
                    Action::Worker(value) => {
                        claims[0].holds(&world, before, &(*value).into(), &emitted)
                    }
                    Action::Deadline(value) => {
                        claims[0].holds(&world, before, &(*value).into(), &emitted)
                    }
                }
            },
        )]
    }
}

/// Creates one independently owned guest run.
pub fn runtime() -> umpire_wasm::Runtime<World, Realization> {
    umpire_wasm::Runtime::new(World, Realization)
}

/// The strict mini activity observation and callback registry.
pub struct Realization;

impl<M> Realize<M> for Realization
where
    M: Machine<State = activity::Model>,
    M::Action: From<activity::Client> + From<activity::Worker> + From<activity::Deadline>,
{
    fn observe(
        &self,
        observation: &Any,
        state: &activity::Model,
    ) -> Result<Projection<M::Action>, protocol::Refusal> {
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

    fn matches(
        &self,
        observation: &Any,
        state: &activity::Model,
    ) -> Result<bool, protocol::Refusal> {
        Ok(project(observation)? == state.phase)
    }

    fn answer(
        &self,
        callback: &Any,
        state: &activity::Model,
    ) -> Result<AnswerPlan<M::Action>, protocol::Refusal> {
        let callback: Callback = unpack(callback, CALLBACK)?;
        let task: PollActivityTaskQueueResponse = unpack(&required(callback.task)?, POLL_RESPONSE)?;
        let payloads = task.input.ok_or_else(|| malformed("callback input is missing"))?;
        let [payload] = payloads.payloads.as_slice() else {
            return Err(malformed("callback requires exactly one input payload"));
        };
        let encoding = payload
            .metadata
            .get("encoding")
            .ok_or_else(|| malformed("payload encoding is missing"))?;
        if payload.metadata.len() != 1 {
            return Err(malformed("payload metadata must contain only encoding"));
        }
        if encoding != b"binary/plain" {
            return Err(unsupported("unsupported payload encoding"));
        }
        let (response_action, request) = match payload.data.as_slice() {
            b"Complete" => (
                activity::Worker::RespondCompleted,
                pack(
                    COMPLETED_REQUEST,
                    &RespondActivityTaskCompletedRequest {
                        result: Some(Payloads {
                            payloads: vec![Payload {
                                metadata: payload.metadata.clone(),
                                data: b"Complete".to_vec(),
                                ..Default::default()
                            }],
                        }),
                        ..Default::default()
                    },
                ),
            ),
            b"Fail" => (
                activity::Worker::RespondFailed,
                pack(
                    FAILED_REQUEST,
                    &RespondActivityTaskFailedRequest {
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
                    },
                ),
            ),
            _ => return Err(unsupported("unsupported callback policy")),
        };
        if state.phase != activity::Phase::Scheduled {
            return Err(protocol::Refusal {
                code: protocol::RefusalCode::State as i32,
                message: "callback requires Scheduled delivery".into(),
            });
        }
        Ok(AnswerPlan {
            delivery: vec![Transition { action: activity::Worker::Poll.into(), choice: 0 }],
            response_action: response_action.into(),
            reply: pack(REPLY, &Reply { request: Some(request) }),
        })
    }
}

fn project(observation: &Any) -> Result<activity::Phase, protocol::Refusal> {
    let observation: Observation = unpack(observation, OBSERVATION)?;
    if observation.method.is_empty() {
        return Err(malformed("observation method is missing"));
    }
    if observation.method != DESCRIBE {
        return Err(unsupported("unsupported observation method"));
    }
    let response: DescribeActivityExecutionResponse =
        unpack(&required(observation.response)?, DESCRIBE_RESPONSE)?;
    let info = response.info.ok_or_else(|| malformed("activity info is missing"))?;
    let status = ActivityExecutionStatus::try_from(info.status)
        .map_err(|_| unsupported("unknown activity execution status"))?;
    let run_state = PendingActivityState::try_from(info.run_state)
        .map_err(|_| unsupported("unknown activity run state"))?;
    let phase = match status {
        ActivityExecutionStatus::Unspecified => {
            return Err(malformed("activity status is missing"));
        }
        ActivityExecutionStatus::Running => {
            return match run_state {
                PendingActivityState::Scheduled => Ok(activity::Phase::Scheduled),
                PendingActivityState::Started => Ok(activity::Phase::Started),
                PendingActivityState::Unspecified => {
                    Err(malformed("running activity run state is missing"))
                }
                _ => Err(unsupported("unsupported running activity state")),
            };
        }
        ActivityExecutionStatus::Completed => activity::Phase::Completed,
        ActivityExecutionStatus::Failed => activity::Phase::Failed,
        ActivityExecutionStatus::Canceled => activity::Phase::Canceled,
        ActivityExecutionStatus::TimedOut => activity::Phase::TimedOut,
        _ => return Err(unsupported("unsupported activity execution status")),
    };
    if run_state != PendingActivityState::Unspecified {
        return Err(malformed("terminal activity must have unspecified run state"));
    }
    Ok(phase)
}

fn unpack<T: Message + Default>(value: &Any, expected: &str) -> Result<T, protocol::Refusal> {
    if value.type_url.is_empty() {
        return Err(malformed("Any type URL is missing"));
    }
    if value.type_url != expected {
        return Err(unsupported("unsupported Any type URL"));
    }
    T::decode(value.value.as_slice()).map_err(|error| malformed(error.to_string()))
}

fn required(value: Option<Any>) -> Result<Any, protocol::Refusal> {
    value.ok_or_else(|| malformed("required Any payload is missing"))
}

fn pack(type_url: &str, value: &impl Message) -> Any {
    Any { type_url: type_url.into(), value: value.encode_to_vec() }
}

fn malformed(message: impl Into<String>) -> protocol::Refusal {
    protocol::Refusal {
        code: protocol::RefusalCode::MalformedRequest as i32,
        message: message.into(),
    }
}

fn unsupported(message: impl Into<String>) -> protocol::Refusal {
    protocol::Refusal { code: protocol::RefusalCode::Unsupported as i32, message: message.into() }
}
