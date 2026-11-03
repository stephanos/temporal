use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    prost_types::{Any, Duration},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal::support::ApplicationFailure;
use umpire_temporal_api::temporal::api::{
    activity::v1::{
        ActivityExecutionInfo, ActivityExecutionOutcome, activity_execution_outcome::Value,
    },
    common::v1::{ActivityType, Link, Payload, Payloads, RetryPolicy, link},
    failure::v1::Failure,
    taskqueue::v1::TaskQueue,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        StartActivityExecutionRequest, StartActivityExecutionResponse,
    },
};
const START: &str = "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution";
const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";
pub(super) const RUN: &str = "22222222-2222-4222-8222-222222222222";
pub(super) fn start(attempts: i32) -> Observation {
    Observation {
        method: START.into(),
        request: Some(Any::pack(&StartActivityExecutionRequest {
            namespace: "model-slice".into(),
            activity_id: "activity".into(),
            request_id: "11111111-1111-4111-8111-111111111111".into(),
            activity_type: Some(ActivityType { name: "ModelSlice".into() }),
            task_queue: Some(TaskQueue { name: "queue".into(), ..Default::default() }),
            start_to_close_timeout: Some(Duration { seconds: 20, nanos: 123 }),
            retry_policy: Some(RetryPolicy { maximum_attempts: attempts, ..Default::default() }),
            ..Default::default()
        })),
        result: Some(ResultValue::Response(Any::pack(&StartActivityExecutionResponse {
            run_id: RUN.into(),
            started: true,
            link: Some(Link {
                variant: Some(link::Variant::Activity(link::Activity {
                    namespace: "model-slice".into(),
                    activity_id: "activity".into(),
                    run_id: RUN.into(),
                })),
            }),
        }))),
    }
}

pub(super) fn describe(attempt: i32, status: i32, run_state: i32) -> Observation {
    Observation {
        method: DESCRIBE.into(),
        request: Some(Any::pack(&DescribeActivityExecutionRequest {
            namespace: "model-slice".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            include_outcome: true,
            include_last_failure: true,
            ..Default::default()
        })),
        result: Some(ResultValue::Response(Any::pack(&DescribeActivityExecutionResponse {
            run_id: RUN.into(),
            outcome: (status == 2).then(|| ActivityExecutionOutcome {
                value: Some(Value::Result(Payloads {
                    payloads: vec![Payload {
                        metadata: std::collections::BTreeMap::from([(
                            "encoding".into(),
                            b"binary/plain".to_vec(),
                        )]),
                        data: b"Complete".to_vec(),
                        ..Default::default()
                    }],
                })),
                ..Default::default()
            }),
            info: Some(ActivityExecutionInfo {
                activity_id: "activity".into(),
                run_id: RUN.into(),
                attempt,
                status,
                run_state,
                last_failure: (attempt > 1)
                    .then(|| Failure::application("MiniActivityFailure", "Retry", false)),
                ..Default::default()
            }),
            ..Default::default()
        }))),
    }
}
