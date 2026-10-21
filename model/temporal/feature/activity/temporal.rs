//! Temporal protobuf binding for the independently authored mini activity Model.

// The fixture runner compiles the Model independently of this realization.
#[path = "model.rs"]
mod included;
mod phase;

pub use included::activity;

use activity::{Client, Phase, Worker};
use umpire_dsl::{Machine, Outcome};
use umpire_temporal_api::temporal::api::{
    common::v1::Payloads,
    failure::v1::Failure,
    workflowservice::v1::{
        DescribeActivityExecutionResponse, PollActivityTaskQueueResponse,
        RequestCancelActivityExecutionResponse, RespondActivityTaskCompletedRequest,
        RespondActivityTaskFailedRequest,
    },
};
use umpire_wasm::{
    AnswerPlan, Candidate, Projection, Realize, google::protobuf::Any, protocol::Refusal,
};

use crate::support::{
    ApplicationFailure, Observed, WorkflowService, callback, method, plain_text, rejection, reply,
};

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
        let observed = Observed::new(observation)?;
        if observed.method == method::<WorkflowService>("RequestCancelActivityExecution") {
            let outcome = if let Some(code) = observed.code()? {
                let Some(rejection) = rejection(code) else {
                    return Ok(Projection::steps([]));
                };
                Outcome::Rejected(rejection)
            } else {
                observed
                    .response::<RequestCancelActivityExecutionResponse>()
                    .map_err(|error| Refusal::malformed(error.message))?;
                Outcome::Accepted
            };
            return Ok(Projection::Candidates(vec![Candidate {
                path: vec![Action::from(Client::Cancel).into()],
                outcome: Some(outcome),
            }]));
        }
        if observed.method != method::<WorkflowService>("DescribeActivityExecution") {
            return Err(Refusal::unsupported("unsupported observation method"));
        }
        let response: DescribeActivityExecutionResponse = observed.response()?;
        let phase = Phase::try_from(
            response.info.ok_or_else(|| Refusal::malformed("activity info is missing"))?,
        )?;
        Ok(if state.phase == phase {
            Projection::Current
        } else {
            match phase {
                Phase::Completed => Projection::step(Worker::RespondCompleted.into()),
                Phase::Failed => Projection::step(Worker::RespondFailed.into()),
                _ => Projection::steps([]),
            }
        })
    }

    fn answer(&self, value: &Any, _: &activity::Model) -> Result<AnswerPlan<Action>, Refusal> {
        let task: PollActivityTaskQueueResponse = callback(value)?;
        let input = task.input.ok_or_else(|| Refusal::malformed("callback input is missing"))?;
        let [payload] = input.payloads.as_slice() else {
            return Err(Refusal::malformed("callback requires exactly one input payload"));
        };
        let (action, response) = match plain_text(payload)? {
            b"Complete" => {
                let mut payload = payload.clone();
                payload.external_payloads.clear();
                (
                    Worker::RespondCompleted,
                    reply(&RespondActivityTaskCompletedRequest {
                        result: Some(Payloads { payloads: vec![payload] }),
                        ..Default::default()
                    }),
                )
            }
            b"Fail" => (
                Worker::RespondFailed,
                reply(&RespondActivityTaskFailedRequest {
                    failure: Some(Failure::application("MiniActivityFailure", "Fail", true)),
                    ..Default::default()
                }),
            ),
            _ => return Err(Refusal::unsupported("unsupported callback policy")),
        };
        Ok(AnswerPlan::new(vec![Action::from(Worker::Poll).into()], action.into(), response))
    }
}
