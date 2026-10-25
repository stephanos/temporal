//! Generated handler conversion and contextual reply controls.

#![forbid(unsafe_code)]

use umpire_runtime::AnyMessage as _;
use umpire_runtime::realization::Observation;
use umpire_runtime::realization::observation::Result as ObservationResult;

use umpire_dsl::{Machine, Outcome};
use umpire_runtime::{
    Projection, Realize,
    prost_types::Any,
    protocol::{Refusal, RefusalCode},
};
use umpire_temporal::{Step, activity, realization};
use umpire_temporal_api::temporal::api::{
    activity::v1::{ActivityExecutionOutcome, activity_execution_outcome::Value},
    common::v1::Payloads,
    workflowservice::v1::{
        DescribeActivityExecutionResponse, RequestCancelActivityExecutionResponse,
        RespondActivityTaskCompletedRequest,
    },
};

macro_rules! observation_only {
    ($($binding:ty),+ $(,)?) => { $(
        impl umpire_runtime::capability::Identity for $binding {
            type Id = ();
            fn identify(&self, _: &Any) -> Result<(), Refusal> {
                Err(Refusal::unsupported("identity unsupported"))
            }
        }
        impl umpire_runtime::capability::Fill<<activity::World as Machine>::Action> for $binding {
            type World = activity::World;
            type Inputs = ();
            type Request = umpire_runtime::capability::NoRequest;
        }
    )+};
}
observation_only!(StepBinding, PresenceBinding, OpaqueBinding);

struct StepBinding;

#[realization(activity)]
impl StepBinding {
    #[observe(WorkflowService::RequestCancelActivityExecution)]
    fn cancel(
        _response: RequestCancelActivityExecutionResponse,
    ) -> Result<Step<activity::Client>, Refusal> {
        Ok(Step { action: activity::Client::Cancel, expected: Some(Outcome::Accepted) })
    }
}

#[test]
fn generated_step_converts_actor_action_and_retains_expected_outcome() {
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    let observation = Any::pack(&Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        result: Some(ObservationResult::Response(Any::pack(
            &RequestCancelActivityExecutionResponse {},
        ))),
        request: None,
    });
    let Projection::Candidates(candidates) = StepBinding
        .observe(&world, &observation, &state)
        .unwrap_or_else(|error| panic!("{error:?}"))
    else {
        panic!("expected one candidate");
    };
    assert_eq!(candidates.len(), 1);
    assert_eq!(candidates[0].path.len(), 1);
    assert_eq!(candidates[0].path[0].action.to_string(), "Client::Cancel");
    assert_eq!(candidates[0].outcome, Some(Outcome::Accepted));
}

struct PresenceBinding;

struct OpaqueBinding;

#[realization(activity)]
impl OpaqueBinding {
    #[observe(WorkflowService::DescribeActivityExecution, reads(info.status, info.run_state))]
    fn describe(
        _response: DescribeActivityExecutionResponse,
    ) -> Result<Projection<<activity::World as Machine>::Action>, Refusal> {
        Ok(Projection::Current)
    }
}

#[realization(activity)]
impl PresenceBinding {
    #[observe(WorkflowService::DescribeActivityExecution,
        reads(info, outcome.failure),
        ignores(run_id = "identity", long_poll_token = "transport", callbacks = "registration")
    )]
    fn describe(
        _response: DescribeActivityExecutionResponse,
    ) -> Result<Projection<<activity::World as Machine>::Action>, Refusal> {
        Ok(Projection::Current)
    }
}

#[test]
fn unsupported_optional_and_oneof_fields_use_decoded_presence() {
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    for (input, outcome, refused) in [
        (None, None, false),
        (Some(Payloads::default()), None, true),
        (None, Some(ActivityExecutionOutcome::default()), false),
        (
            None,
            Some(ActivityExecutionOutcome {
                value: Some(Value::Result(Payloads::default())),
                ..Default::default()
            }),
            true,
        ),
    ] {
        let observation = Any::pack(&Observation {
            method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
                .into(),
            result: Some(ObservationResult::Response(Any::pack(
                &DescribeActivityExecutionResponse { input, outcome, ..Default::default() },
            ))),
            request: None,
        });
        let result = PresenceBinding.observe(&world, &observation, &state);
        if refused {
            assert_eq!(result.err().map(|error| error.code()), Some(RefusalCode::Unsupported));
        } else {
            assert!(matches!(result, Ok(Projection::Current)));
        }
    }
}

#[test]
fn opaque_ignored_catalog_fields_still_have_real_decoded_presence() {
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    for present in [false, true] {
        let observation = Any::pack(&Observation {
            method: "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution"
                .into(),
            result: Some(ObservationResult::Response(Any::pack(
                &DescribeActivityExecutionResponse {
                    info: Some(
                        umpire_temporal_api::temporal::api::activity::v1::ActivityExecutionInfo {
                            priority: present.then(Default::default),
                            ..Default::default()
                        },
                    ),
                    ..Default::default()
                },
            ))),
            request: None,
        });
        let result = OpaqueBinding.observe(&world, &observation, &state);
        if present {
            let error =
                result.err().unwrap_or_else(|| panic!("opaque priority presence must refuse"));
            assert_eq!(error.code(), RefusalCode::Unsupported);
            assert!(error.message.ends_with("info.priority"));
        } else {
            assert!(matches!(result, Ok(Projection::Current)));
        }
    }
}

struct CallbackBinding;

impl umpire_runtime::capability::Identity for CallbackBinding {
    type Id = Vec<u8>;
    fn identify(&self, value: &Any) -> Result<Self::Id, Refusal> {
        let task: umpire_temporal_api::temporal::api::workflowservice::v1::PollActivityTaskQueueResponse = umpire_runtime::callback(value)?;
        if task.task_token.is_empty() {
            return Err(Refusal::malformed("callback task identity is missing"));
        }
        Ok(task.task_token)
    }
}

impl umpire_runtime::capability::Fill<<activity::World as Machine>::Action> for CallbackBinding {
    type World = activity::World;
    type Inputs = ();
    type Request = RespondActivityTaskCompletedRequest;
    fn fill(
        &self,
        _: &<activity::World as Machine>::Action,
        _: &activity::World,
        (): &(),
        identity: &Vec<u8>,
    ) -> Result<Self::Request, Refusal> {
        Ok(Self::Request { task_token: identity.clone(), ..Default::default() })
    }
}

#[realization(activity)]
impl CallbackBinding {
    #[answer(
        PollActivityTaskQueueResponse,
        reads(task_token),
        produces(RespondActivityTaskCompletedRequest, fields(task_token))
    )]
    fn respond(
        _: umpire_temporal_api::temporal::api::workflowservice::v1::PollActivityTaskQueueResponse,
    ) -> Result<umpire_temporal::Reply<activity::Worker, ()>, Refusal> {
        Ok(umpire_temporal::Reply {
            delivery: vec![],
            action: activity::Worker::RespondCompleted,
            inputs: (),
        })
    }
}

#[test]
fn generated_callback_fills_once_from_the_admitted_actual_identity() {
    use umpire_temporal_api::temporal::api::workflowservice::v1::{
        PollActivityTaskQueueResponse, RespondActivityTaskCompletedRequest,
    };
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    let callback = |task_token| {
        Any::pack(&umpire_runtime::realization::Callback {
            task: Some(Any::pack(&PollActivityTaskQueueResponse {
                task_token,
                ..Default::default()
            })),
        })
    };
    let plan = CallbackBinding
        .answer(&world, &callback(vec![7, 8]), &state)
        .unwrap_or_else(|error| panic!("{error:?}"));
    let reply: umpire_runtime::realization::Reply =
        plan.reply.unpack().unwrap_or_else(|error| panic!("{error:?}"));
    let request: RespondActivityTaskCompletedRequest = reply
        .request
        .unwrap_or_else(|| panic!("filled request missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert_eq!(request.task_token, [7, 8]);
    assert!(plan.delivery.is_empty());
    assert_eq!(plan.response_action.to_string(), "Worker::RespondCompleted");
    assert_eq!(
        CallbackBinding.answer(&world, &callback(vec![]), &state).err().map(|error| error.code()),
        Some(RefusalCode::MalformedRequest)
    );
}

#[test]
fn generated_observation_validates_actual_request_before_feature_delegation() {
    use umpire_temporal_api::temporal::api::workflowservice::v1::DescribeActivityExecutionRequest;
    let world = activity::World::new(3, None, None, None, None, false);
    let state = world.starts().remove(0);
    let valid = Any::pack(&RequestCancelActivityExecutionResponse {});
    let wrong = Any::pack(&DescribeActivityExecutionResponse::default());
    for (response, request, expected) in [
        (
            valid.clone(),
            Any::pack(&DescribeActivityExecutionRequest::default()),
            RefusalCode::Unsupported,
        ),
        (valid, Any::default(), RefusalCode::MalformedRequest),
        (wrong, Any::default(), RefusalCode::Unsupported),
    ] {
        let observation = Any::pack(&Observation {
            method:
                "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
                    .into(),
            result: Some(ObservationResult::Response(response)),
            request: Some(request),
        });
        assert_eq!(
            StepBinding.observe(&world, &observation, &state).err().map(|error| error.code()),
            Some(expected),
        );
    }
}
