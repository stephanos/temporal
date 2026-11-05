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
    common::v1::{ActivityType, Header, Link, Payload, Payloads, RetryPolicy, link},
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
            header: Some(Header {
                fields: std::collections::BTreeMap::from([(
                    "umpire-scenario".into(),
                    Payload {
                        metadata: std::collections::BTreeMap::from([(
                            "encoding".into(),
                            b"binary/plain".to_vec(),
                        )]),
                        data: b"activity-v1_1h=c".to_vec(),
                        ..Default::default()
                    },
                )]),
            }),
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

#[allow(dead_code)]
pub(super) fn policy_witness(
    program: &umpire_temporal_api::umpire::temporal::v1::WorkerProgram,
    choices: &[umpire_temporal::policy::Choice],
) -> (String, u64) {
    use umpire_temporal::{activity, policy::TaskKey};
    use umpire_temporal_api::umpire::temporal::v1::ActivityTaskKey;
    let start: StartActivityExecutionRequest = start(3)
        .request
        .unwrap_or_else(|| panic!("start"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let world = activity::World::new(3, None, None, None, None);
    let expected: Vec<_> = choices
        .iter()
        .enumerate()
        .map(|(index, choice)| {
            let attempt = u8::try_from(index + 1).unwrap_or_else(|_| panic!("attempt"));
            let key = TaskKey { attempt, heartbeat_history: vec![] };
            (
                key.clone(),
                choice.fill(&world, &start, &key).unwrap_or_else(|error| panic!("{error:?}")),
            )
        })
        .collect();
    let mut groups: Vec<(String, Vec<_>)> = vec![];
    for rule in &program.rules {
        let key: ActivityTaskKey = rule
            .key
            .as_ref()
            .unwrap_or_else(|| panic!("key"))
            .unpack()
            .unwrap_or_else(|error| panic!("{error:?}"));
        if groups.last().is_none_or(|(name, _)| *name != key.scenario) {
            assert!(!groups.iter().any(|(name, _)| *name == key.scenario), "noncontiguous policy");
            groups.push((key.scenario.clone(), vec![]));
        }
        let [response] = rule.respond.as_slice() else { panic!("one response") };
        groups.last_mut().unwrap_or_else(|| panic!("group")).1.push((
            TaskKey {
                attempt: u8::try_from(key.attempt).unwrap_or_else(|_| panic!("attempt")),
                heartbeat_history: key.heartbeat_history,
            },
            response.clone(),
        ));
    }
    let matches: Vec<_> =
        groups.iter().enumerate().filter(|(_, (_, rows))| *rows == expected).collect();
    let [(seed, (name, _))] = matches.as_slice() else { panic!("unique exact policy witness") };
    (name.clone(), u64::try_from(*seed).unwrap_or_else(|_| panic!("seed")))
}

#[allow(dead_code)]
pub(super) fn terminal_witness(
    program: &umpire_temporal_api::umpire::temporal::v1::WorkerProgram,
    label: &str,
) -> (String, u64) {
    use umpire_temporal::policy::Choice::{Complete, Fail, Retry};
    let choices: &[_] = match label {
        "Complete" => &[Complete],
        "RetryThenComplete" => &[Retry, Complete],
        "RetryTwiceThenComplete" => &[Retry, Retry, Complete],
        "RetryAtCap" => &[Retry, Retry, Retry],
        "Fail" => &[Fail],
        _ => panic!("unknown semantic witness {label}"),
    };
    policy_witness(program, choices)
}
