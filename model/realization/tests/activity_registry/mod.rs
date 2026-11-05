//! Activity registry rows, typed fillers and evidence predicates on the actual Model.
use super::{
    Action, Any, Client, Failure, FillInputs, Payloads, PollActivityTaskQueueRequest,
    PollActivityTaskQueueResponse, ProposalContext, Realization,
    RecordActivityTaskHeartbeatRequest, RecordActivityTaskHeartbeatResponse,
    RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
    RespondActivityTaskCanceledRequest, RespondActivityTaskCanceledResponse,
    RespondActivityTaskCompletedRequest, RespondActivityTaskCompletedResponse,
    RespondActivityTaskFailedRequest, RespondActivityTaskFailedResponse, TaskQueue, Worker,
    activity,
};
use crate::registry::{ActionKind, Direction, Registry};
use crate::support::ApplicationFailure as _;
use umpire_dsl::Machine as _;
use umpire_runtime::{
    AnyMessage as _, Evidence, EvidenceMatch,
    capability::{Fill, Identity, Request as _},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal_api::temporal::api::common::v1::{ActivityType, RetryPolicy};
use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;

const SERVICE: &str = "/temporal.api.workflowservice.v1.WorkflowService/";
const RUN: &str = "run";

fn start() -> StartActivityExecutionRequest {
    StartActivityExecutionRequest {
        namespace: "ns".into(),
        activity_id: "activity".into(),
        request_id: "request".into(),
        activity_type: Some(ActivityType { name: "ModelSlice".into() }),
        task_queue: Some(TaskQueue { name: "queue".into(), ..Default::default() }),
        retry_policy: Some(RetryPolicy { maximum_attempts: 3, ..Default::default() }),
        ..Default::default()
    }
}

fn binding() -> Realization {
    Realization::configured(ProposalContext { start: start(), run_id: RUN.into() }, None)
}

fn world() -> activity::World {
    activity::World::new(3, None, None, None, None)
}

fn observed<Q: prost::Name, S: prost::Name>(
    method: &str,
    request: &Q,
    response: Result<&S, i32>,
) -> Any {
    let result = match response {
        Ok(response) => ResultValue::Response(Any::pack(response)),
        Err(code) => {
            ResultValue::Error(umpire_runtime::realization::Error { code, message: String::new() })
        }
    };
    Any::pack(&Observation {
        method: format!("{SERVICE}{method}"),
        request: Some(Any::pack(request)),
        result: Some(result),
    })
}

fn edge(
    source: activity::Model,
    action: Action,
) -> Result<Evidence<activity::World>, Box<dyn std::error::Error>> {
    let world = world();
    let (outcome, facts, target) = world
        .checked_step(&source, action.clone())?
        .into_iter()
        .next()
        .ok_or("action has no checked step")?;
    Ok(Evidence { source, action, choice: 0, outcome, facts: facts.into_iter().collect(), target })
}

fn scheduled() -> activity::Model {
    world().starts().into_iter().next().unwrap_or_else(|| unreachable!("one start"))
}

fn started() -> Result<activity::Model, Box<dyn std::error::Error>> {
    Ok(edge(scheduled(), Worker::Poll(1).into())?.target)
}

#[test]
fn every_model_action_has_one_declared_row() -> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let world = activity::World::new(
        3,
        Some(activity::Ticks(1)),
        Some(activity::Ticks(1)),
        Some(activity::Ticks(1)),
        Some(activity::Ticks(1)),
    );
    let catalog = world.checked_catalog(umpire_dsl::CATALOG_LIMIT)?;
    let mut rows = std::collections::BTreeSet::new();
    for action in catalog.actions() {
        let row = binding.row(action);
        let expected = match action {
            Action::Client(Client::Cancel) => ("RequestCancelActivityExecution", Direction::Client),
            Action::Worker(Worker::Poll(_)) => ("PollActivityTaskQueue", Direction::Worker),
            Action::Worker(Worker::RespondCompleted) => {
                ("RespondActivityTaskCompleted", Direction::Worker)
            }
            Action::Worker(Worker::RespondFailed { .. }) => {
                ("RespondActivityTaskFailed", Direction::Worker)
            }
            Action::Worker(Worker::Heartbeat) => ("RecordActivityTaskHeartbeat", Direction::Worker),
            Action::Worker(Worker::RespondCanceled(_)) => {
                ("RespondActivityTaskCanceled", Direction::Worker)
            }
            Action::Deadline(activity::Deadline::Tick) => {
                assert_eq!(row.kind, ActionKind::Event { reason: "abstract deadline tick" });
                rows.insert(row.variant);
                continue;
            }
        };
        let ActionKind::Rpc { method, direction } = row.kind else {
            return Err(format!("{action:?} is not an RPC row").into());
        };
        assert_eq!((method, direction), (format!("{SERVICE}{}", expected.0).as_str(), expected.1));
        rows.insert(row.variant);
    }
    assert_eq!(rows.len(), 7);
    Ok(())
}

#[test]
fn fillers_refuse_inputs_of_another_action_and_contradicting_failures()
-> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let world = world();
    let identity = binding.identify(&Any::pack(&start())).map_err(|refusal| refusal.message)?;
    let completed: Action = Worker::RespondCompleted.into();
    assert!(
        binding
            .fill(&completed, &world, &FillInputs::Cancel { request_id: String::new() }, &identity)
            .is_err()
    );
    let retry = FillInputs::Failed {
        attempt: 1,
        failure: Box::new(Failure::application("MiniActivityFailure", "Retry", false)),
    };
    assert!(
        binding
            .fill(&Worker::RespondFailed { retryable: true }.into(), &world, &retry, &identity)
            .is_ok()
    );
    assert!(
        binding
            .fill(&Worker::RespondFailed { retryable: false }.into(), &world, &retry, &identity)
            .is_err()
    );
    let poll = binding
        .fill(
            &Worker::Poll(2).into(),
            &world,
            &FillInputs::Worker { attempt: 2, details: None },
            &identity,
        )
        .map_err(|refusal| refusal.message)?
        .pack_request()
        .unpack::<PollActivityTaskQueueRequest>()
        .map_err(|refusal| refusal.message)?;
    assert_eq!(
        (poll.namespace.as_str(), poll.task_queue.map(|queue| queue.name)),
        ("ns", Some("queue".into()))
    );
    assert!(
        binding
            .fill(
                &Worker::Poll(1).into(),
                &world,
                &FillInputs::Worker { attempt: 2, details: None },
                &identity
            )
            .is_err()
    );
    assert!(
        binding
            .fill(
                &Action::from(activity::Deadline::Tick),
                &world,
                &FillInputs::Cancel { request_id: String::new() },
                &identity
            )
            .is_err()
    );
    let cancel = binding
        .fill(
            &Client::Cancel.into(),
            &world,
            &FillInputs::Cancel { request_id: String::new() },
            &identity,
        )
        .map_err(|refusal| refusal.message)?
        .pack_request();
    assert_eq!(
        cancel,
        Any::pack(&RequestCancelActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            ..Default::default()
        })
    );
    Ok(())
}

#[test]
fn one_rpc_evidences_one_edge_and_unmapped_statuses_refuse()
-> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let cancel = RequestCancelActivityExecutionRequest {
        namespace: "ns".into(),
        activity_id: "activity".into(),
        run_id: RUN.into(),
        request_id: "cancel".into(),
        ..Default::default()
    };
    let started = started()?;
    let pending = edge(started.clone(), Client::Cancel.into())?;
    let not_found = observed::<_, RequestCancelActivityExecutionResponse>(
        "RequestCancelActivityExecution",
        &cancel,
        Err(5),
    );
    // One actual RPC evidences exactly one edge of a witness.
    let closed = edge(started.clone(), Worker::RespondCompleted.into())?.target;
    let rejected = edge(closed, Client::Cancel.into())?;
    assert!(matches!(
        binding.check_registered(&not_found, &[&rejected]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    assert!(matches!(
        binding.check_registered(&not_found, &[&rejected, &rejected]),
        Ok(EvidenceMatch::Reject)
    ));
    // A status outside the declared rejections refuses instead of rejecting the witness.
    for code in [0, 1, 2, 4, 8, 13, 999] {
        let unmapped = observed::<_, RequestCancelActivityExecutionResponse>(
            "RequestCancelActivityExecution",
            &cancel,
            Err(code),
        );
        let refusal = binding
            .check_registered(&unmapped, &[&pending])
            .err()
            .ok_or("unmapped status was judged")?;
        assert_eq!(refusal.code(), umpire_runtime::protocol::RefusalCode::Unsupported, "{code}");
    }
    Ok(())
}

#[test]
fn rpc_evidence_matches_only_the_actual_last_edge_and_outcome()
-> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let cancel = RequestCancelActivityExecutionRequest {
        namespace: "ns".into(),
        activity_id: "activity".into(),
        run_id: RUN.into(),
        request_id: "cancel".into(),
        ..Default::default()
    };
    let accepted = observed(
        "RequestCancelActivityExecution",
        &cancel,
        Ok(&RequestCancelActivityExecutionResponse::default()),
    );
    let started = started()?;
    let pending = edge(started.clone(), Client::Cancel.into())?;
    assert!(matches!(binding.check_registered(&accepted, &[]), Ok(EvidenceMatch::Reject)));
    assert!(matches!(
        binding.check_registered(&accepted, &[&pending]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    let not_found = observed::<_, RequestCancelActivityExecutionResponse>(
        "RequestCancelActivityExecution",
        &cancel,
        Err(5),
    );
    assert!(matches!(binding.check_registered(&not_found, &[&pending]), Ok(EvidenceMatch::Reject)));
    let crossed = RequestCancelActivityExecutionRequest { run_id: "other".into(), ..cancel };
    let crossed = observed(
        "RequestCancelActivityExecution",
        &crossed,
        Ok(&RequestCancelActivityExecutionResponse::default()),
    );
    assert!(binding.check_registered(&crossed, &[&pending]).is_err());

    let worker = |failure| RespondActivityTaskFailedRequest {
        namespace: "ns".into(),
        task_token: vec![1],
        failure: Some(failure),
        ..Default::default()
    };
    let retry = observed(
        "RespondActivityTaskFailed",
        &worker(Failure::application("MiniActivityFailure", "Retry", false)),
        Ok(&RespondActivityTaskFailedResponse::default()),
    );
    let retried = edge(started.clone(), Worker::RespondFailed { retryable: true }.into())?;
    let failed = edge(started.clone(), Worker::RespondFailed { retryable: false }.into())?;
    assert!(matches!(
        binding.check_registered(&retry, &[&retried]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    assert!(matches!(binding.check_registered(&retry, &[&failed]), Ok(EvidenceMatch::Reject)));

    let heartbeat = RecordActivityTaskHeartbeatRequest {
        namespace: "ns".into(),
        task_token: vec![1],
        ..Default::default()
    };
    let beat = edge(started, Worker::Heartbeat.into())?;
    let quiet = observed(
        "RecordActivityTaskHeartbeat",
        &heartbeat,
        Ok(&RecordActivityTaskHeartbeatResponse::default()),
    );
    let reported = observed(
        "RecordActivityTaskHeartbeat",
        &heartbeat,
        Ok(&RecordActivityTaskHeartbeatResponse { cancel_requested: true, ..Default::default() }),
    );
    assert!(matches!(
        binding.check_registered(&quiet, &[&beat]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    assert!(matches!(binding.check_registered(&reported, &[&beat]), Ok(EvidenceMatch::Reject)));
    let paused = observed(
        "RecordActivityTaskHeartbeat",
        &heartbeat,
        Ok(&RecordActivityTaskHeartbeatResponse { activity_paused: true, ..Default::default() }),
    );
    assert!(binding.check_registered(&paused, &[&beat]).is_err());
    let unconfigured = Realization::default();
    assert!(unconfigured.check_registered(&quiet, &[&beat]).is_err());
    Ok(())
}

#[test]
fn worker_fillers_refuse_attempt_and_detail_mismatches() -> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let world = world();
    let identity = binding.identify(&Any::pack(&start())).map_err(|refusal| refusal.message)?;
    let worker = |attempt, details| FillInputs::Worker { attempt, details };
    let details = Some(Payloads::default());
    for (action, inputs) in [
        (Action::from(Worker::RespondCanceled(2)), worker(1, None)),
        (Action::from(Worker::Heartbeat), worker(4, None)),
        (Action::from(Worker::Poll(1)), worker(1, details.clone())),
        (
            Action::from(Worker::RespondCompleted),
            FillInputs::Completed { attempt: 0, result: None },
        ),
    ] {
        assert!(binding.fill(&action, &world, &inputs, &identity).is_err(), "{action:?}");
    }
    assert!(
        binding.fill(&Worker::Heartbeat.into(), &world, &worker(3, details), &identity).is_ok()
    );
    Ok(())
}

#[test]
fn worker_evidence_maps_actual_results_and_correlates_delivery()
-> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let poll = PollActivityTaskQueueRequest { namespace: "ns".into(), ..Default::default() };
    let task = |activity_id: &str, attempt| PollActivityTaskQueueResponse {
        task_token: vec![1],
        activity_id: activity_id.into(),
        attempt,
        ..Default::default()
    };
    let delivered = edge(scheduled(), Worker::Poll(1).into())?;
    let first = observed("PollActivityTaskQueue", &poll, Ok(&task("activity", 1)));
    assert!(matches!(
        binding.check_registered(&first, &[&delivered]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    let second = observed("PollActivityTaskQueue", &poll, Ok(&task("activity", 2)));
    assert!(matches!(binding.check_registered(&second, &[&delivered]), Ok(EvidenceMatch::Reject)));
    let crossed = observed("PollActivityTaskQueue", &poll, Ok(&task("other", 1)));
    assert!(binding.check_registered(&crossed, &[&delivered]).is_err());
    let stale = edge(scheduled(), Worker::Poll(2).into())?;
    let refused =
        observed::<_, PollActivityTaskQueueResponse>("PollActivityTaskQueue", &poll, Err(9));
    assert!(matches!(
        binding.check_registered(&refused, &[&stale]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    assert!(matches!(binding.check_registered(&refused, &[&delivered]), Ok(EvidenceMatch::Reject)));

    let started = started()?;
    let token = |namespace: &str| RespondActivityTaskCompletedRequest {
        namespace: namespace.into(),
        task_token: vec![1],
        ..Default::default()
    };
    let completed = edge(started.clone(), Worker::RespondCompleted.into())?;
    let ok = observed(
        "RespondActivityTaskCompleted",
        &token("ns"),
        Ok(&RespondActivityTaskCompletedResponse::default()),
    );
    assert!(matches!(
        binding.check_registered(&ok, &[&completed]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    let elsewhere = observed(
        "RespondActivityTaskCompleted",
        &token("other"),
        Ok(&RespondActivityTaskCompletedResponse::default()),
    );
    assert!(binding.check_registered(&elsewhere, &[&completed]).is_err());

    let pending = edge(started, Client::Cancel.into())?.target;
    let acknowledgement = RespondActivityTaskCanceledRequest {
        namespace: "ns".into(),
        task_token: vec![1],
        ..Default::default()
    };
    let acknowledged = edge(pending.clone(), Worker::RespondCanceled(1).into())?;
    let stale = edge(pending, Worker::RespondCanceled(2).into())?;
    let accepted = observed(
        "RespondActivityTaskCanceled",
        &acknowledgement,
        Ok(&RespondActivityTaskCanceledResponse::default()),
    );
    let not_found = observed::<_, RespondActivityTaskCanceledResponse>(
        "RespondActivityTaskCanceled",
        &acknowledgement,
        Err(5),
    );
    assert!(matches!(
        binding.check_registered(&accepted, &[&acknowledged]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    assert!(matches!(binding.check_registered(&accepted, &[&stale]), Ok(EvidenceMatch::Reject)));
    assert!(matches!(
        binding.check_registered(&not_found, &[&stale]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    Ok(())
}

#[test]
fn evidence_ignores_unclaimed_fields_of_realistic_worker_traffic()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::common::v1::{Payload, WorkflowExecution};
    let binding = binding();
    let payloads =
        || Payloads { payloads: vec![Payload { data: b"x".to_vec(), ..Default::default() }] };
    let poll = PollActivityTaskQueueRequest {
        namespace: "ns".into(),
        task_queue: Some(TaskQueue { name: "queue".into(), ..Default::default() }),
        identity: "worker".into(),
        ..Default::default()
    };
    // A server delivery carries type, input, schedule and workflow fields the Model does not claim.
    let task = PollActivityTaskQueueResponse {
        task_token: vec![1],
        activity_id: "activity".into(),
        attempt: 1,
        workflow_namespace: "ns".into(),
        workflow_execution: Some(WorkflowExecution::default()),
        activity_type: Some(ActivityType { name: "ModelSlice".into() }),
        input: Some(payloads()),
        heartbeat_timeout: Some(umpire_runtime::prost_types::Duration { seconds: 10, nanos: 0 }),
        ..Default::default()
    };
    let delivered = edge(scheduled(), Worker::Poll(1).into())?;
    // An empty long-poll delivers nothing: rejected, not refused as crossed identity.
    let empty =
        observed("PollActivityTaskQueue", &poll, Ok(&PollActivityTaskQueueResponse::default()));
    assert!(matches!(binding.check_registered(&empty, &[&delivered]), Ok(EvidenceMatch::Reject)));
    let evidence = observed("PollActivityTaskQueue", &poll, Ok(&task));
    assert!(matches!(
        binding.check_registered(&evidence, &[&delivered]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    let heartbeat = RecordActivityTaskHeartbeatRequest {
        namespace: "ns".into(),
        task_token: vec![1],
        details: Some(payloads()),
        ..Default::default()
    };
    let beat = edge(started()?, Worker::Heartbeat.into())?;
    let reply = RecordActivityTaskHeartbeatResponse::default();
    let evidence = observed("RecordActivityTaskHeartbeat", &heartbeat, Ok(&reply));
    assert!(matches!(
        binding.check_registered(&evidence, &[&beat]),
        Ok(EvidenceMatch::Match { cursor: 1 })
    ));
    // A claimed field still decides: a paused activity stays outside the Model.
    let paused = RecordActivityTaskHeartbeatResponse { activity_paused: true, ..reply };
    let evidence = observed("RecordActivityTaskHeartbeat", &heartbeat, Ok(&paused));
    assert!(binding.check_registered(&evidence, &[&beat]).is_err());
    Ok(())
}

fn programmed(scenario: &str, maximum: i32) -> Realization {
    use umpire_temporal_api::temporal::api::common::v1::{Header, Payload};
    let mut start = start();
    start.retry_policy = Some(RetryPolicy { maximum_attempts: maximum, ..Default::default() });
    let manifest = std::rc::Rc::new(
        crate::policy::Manifest::production(&start).unwrap_or_else(|error| panic!("{error:?}")),
    );
    let name = if scenario == "Unknown" {
        "activity-v1_1h=c_2h=c"
    } else if scenario == "LegacyComplete" {
        "Complete"
    } else {
        use crate::policy::{Choice, TaskKey};
        let choices: &[Choice] = match scenario {
            "Complete" => &[Choice::Complete],
            "Fail" => &[Choice::Fail],
            "HeartbeatThenComplete" => &[Choice::Heartbeat],
            "RetryThenComplete" if maximum == 1 => &[Choice::Retry],
            "RetryThenComplete" => &[Choice::Retry, Choice::Complete],
            _ => panic!("unknown witness"),
        };
        let mut rows: Vec<_> = choices
            .iter()
            .enumerate()
            .map(|(index, choice)| {
                (
                    TaskKey {
                        attempt: u8::try_from(index + 1).unwrap_or_else(|_| panic!("attempt")),
                        heartbeat_history: vec![],
                    },
                    *choice,
                )
            })
            .collect();
        if scenario == "HeartbeatThenComplete" {
            rows.extend([false, true].map(|bit| {
                (TaskKey { attempt: 1, heartbeat_history: vec![bit] }, Choice::Complete)
            }));
        }
        manifest
            .policies()
            .iter()
            .find(|policy| policy.rows() == rows)
            .unwrap_or_else(|| panic!("exact policy witness"))
            .name()
    };
    start.header = Some(Header {
        fields: std::collections::BTreeMap::from([(
            super::super::SCENARIO_HEADER.into(),
            Payload { data: name.as_bytes().to_vec(), ..Default::default() },
        )]),
    });
    Realization::configured(ProposalContext { start, run_id: RUN.into() }, Some(manifest))
}

#[test]
fn a_pruned_exact_attempt_key_is_unexplained_without_fabricating_a_row() {
    use umpire_runtime::{
        Guest, Runtime,
        testing::{GuestExt as _, ResponseExt as _},
    };
    let mut guest = Guest::new(Runtime::new(world(), programmed("Complete", 3)));
    guest
        .judge(describe(2, 1, 1, Some(Failure::application("MiniActivityFailure", "Retry", false))))
        .unexplained();
}

#[test]
fn snapshot_completion_cannot_supply_unobserved_heartbeat_history() {
    use umpire_runtime::{
        Guest, Runtime,
        testing::{GuestExt as _, ResponseExt as _},
    };
    use umpire_temporal_api::temporal::api::{
        activity::v1::{
            ActivityExecutionInfo, ActivityExecutionOutcome, activity_execution_outcome::Value,
        },
        workflowservice::v1::{
            DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        },
    };
    let start = start();
    let completed: RespondActivityTaskCompletedRequest = crate::policy::Choice::Complete
        .fill(&world(), &start, &crate::policy::TaskKey { attempt: 1, heartbeat_history: vec![] })
        .unwrap_or_else(|e| panic!("{e:?}"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let observation = observed(
        "DescribeActivityExecution",
        &DescribeActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            ..Default::default()
        },
        Ok(&DescribeActivityExecutionResponse {
            run_id: RUN.into(),
            info: Some(ActivityExecutionInfo {
                activity_id: "activity".into(),
                run_id: RUN.into(),
                status: 2,
                attempt: 1,
                ..Default::default()
            }),
            outcome: Some(ActivityExecutionOutcome {
                value: Some(Value::Result(completed.result.unwrap_or_else(|| panic!("result")))),
                ..Default::default()
            }),
            ..Default::default()
        }),
    );
    let mut guest = Guest::new(Runtime::new(world(), programmed("HeartbeatThenComplete", 3)));
    guest.judge(observation).unexplained();
}

fn describe(attempt: i32, status: i32, run_state: i32, last_failure: Option<Failure>) -> Any {
    use umpire_temporal_api::temporal::api::{
        activity::v1::ActivityExecutionInfo,
        workflowservice::v1::{
            DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        },
    };
    observed(
        "DescribeActivityExecution",
        &DescribeActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            include_outcome: true,
            include_last_failure: true,
            ..Default::default()
        },
        Ok(&DescribeActivityExecutionResponse {
            run_id: RUN.into(),
            info: Some(ActivityExecutionInfo {
                activity_id: "activity".into(),
                run_id: RUN.into(),
                status,
                run_state,
                attempt,
                last_failure,
                ..Default::default()
            }),
            ..Default::default()
        }),
    )
}

/// A program the Describe evidence must consult refuses before search, as the pre-Reach
/// projection did, instead of turning into an Unexplained judgment.
#[test]
fn describe_program_selection_refuses_before_search() {
    use umpire_runtime::{Realize as _, protocol::RefusalCode};
    let start = world().starts().remove(0);
    let refused = |binding: &Realization, world: &activity::World, observation: &Any| {
        binding.observe(world, observation, &start).err().map(|refusal| refusal.code())
    };
    let running = describe(1, 1, 1, None);
    let retried = Some(Failure::application("MiniActivityFailure", "Retry", false));
    assert_eq!(
        refused(&programmed("Unknown", 3), &world(), &running),
        Some(RefusalCode::Unsupported),
        "an unknown scenario header refuses"
    );
    assert_eq!(
        refused(&programmed("LegacyComplete", 3), &world(), &running),
        Some(RefusalCode::Unsupported),
        "a legacy alias is not a canonical policy identity"
    );
    assert!(refused(&programmed("Complete", 3), &world(), &running).is_none());
    let capped = activity::World::new(1, None, None, None, None);
    assert_eq!(
        refused(&programmed("RetryThenComplete", 1), &capped, &describe(2, 1, 1, retried.clone())),
        Some(RefusalCode::Unsupported),
        "an attempt beyond World.maximum_attempts refuses"
    );
    // A program contradicted before the over-cap attempt stays Unexplained, as before.
    assert!(refused(&programmed("Complete", 1), &capped, &describe(2, 1, 1, retried)).is_none());
    assert!(
        refused(&programmed("RetryThenComplete", 1), &capped, &describe(2, 1, 1, None)).is_none()
    );
    assert!(
        refused(&programmed("RetryThenComplete", 1), &capped, &describe(1, 1, 1, None)).is_none()
    );
}

#[test]
fn non_application_worker_failure_refuses() -> Result<(), Box<dyn std::error::Error>> {
    let binding = binding();
    let started = started()?;
    let retried = edge(started, Worker::RespondFailed { retryable: true }.into())?;
    let request = RespondActivityTaskFailedRequest {
        namespace: "ns".into(),
        task_token: vec![1],
        failure: Some(Failure::default()),
        ..Default::default()
    };
    let observation = observed(
        "RespondActivityTaskFailed",
        &request,
        Ok(&RespondActivityTaskFailedResponse::default()),
    );
    let refusal =
        binding.check_registered(&observation, &[&retried]).err().ok_or("failure was judged")?;
    assert_eq!(refusal.code(), umpire_runtime::protocol::RefusalCode::Unsupported);
    Ok(())
}

/// Recorded R4 exception: a programmed retryable failure observed as the terminal Failed status
/// at attempt one is explained only when the World's cap leaves no retry. The pre-Reach
/// candidate replay explained it under every cap while landing in a queued retry.
#[test]
fn terminal_retryable_failure_is_explained_only_at_the_cap() {
    use umpire_runtime::{
        Guest, Runtime,
        testing::{GuestExt as _, ResponseExt as _},
    };
    use umpire_temporal_api::temporal::api::{
        activity::v1::{
            ActivityExecutionInfo, ActivityExecutionOutcome, activity_execution_outcome::Value,
        },
        workflowservice::v1::{
            DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        },
    };
    let retry = Failure::application("MiniActivityFailure", "Retry", false);
    let failed = observed(
        "DescribeActivityExecution",
        &DescribeActivityExecutionRequest {
            namespace: "ns".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            ..Default::default()
        },
        Ok(&DescribeActivityExecutionResponse {
            run_id: RUN.into(),
            info: Some(ActivityExecutionInfo {
                activity_id: "activity".into(),
                run_id: RUN.into(),
                status: 3,
                attempt: 1,
                last_failure: Some(retry.clone()),
                ..Default::default()
            }),
            outcome: Some(ActivityExecutionOutcome {
                value: Some(Value::Failure(retry)),
                ..Default::default()
            }),
            ..Default::default()
        }),
    );
    for maximum in [1_u8, 2, 3] {
        let world = activity::World::new(maximum, None, None, None, None);
        let mut guest =
            Guest::new(Runtime::new(world, programmed("RetryThenComplete", i32::from(maximum))));
        let response = guest.judge(failed.clone());
        if maximum == 1 {
            assert_eq!(
                response.explanation().steps,
                ["Worker::Poll(1)", "Worker::RespondFailed { retryable: true }"]
            );
        } else {
            response.unexplained();
        }
    }
}
