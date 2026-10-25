//! Generated handler conversion and contextual reply controls.

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
    },
};

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
    let state = activity::World::new(3, None, None, None, None, false).starts().remove(0);
    let observation = Any::pack(&Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        result: Some(ObservationResult::Response(Any::pack(
            &RequestCancelActivityExecutionResponse {},
        ))),
        request: None,
    });
    let Projection::Candidates(candidates) =
        StepBinding.observe(&observation, &state).unwrap_or_else(|error| panic!("{error:?}"))
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
    let state = activity::World::new(3, None, None, None, None, false).starts().remove(0);
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
        let result = PresenceBinding.observe(&observation, &state);
        if refused {
            assert_eq!(result.err().map(|error| error.code()), Some(RefusalCode::Unsupported));
        } else {
            assert!(matches!(result, Ok(Projection::Current)));
        }
    }
}

#[test]
fn opaque_ignored_catalog_fields_still_have_real_decoded_presence() {
    let state = activity::World::new(3, None, None, None, None, false).starts().remove(0);
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
        let result = OpaqueBinding.observe(&observation, &state);
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
