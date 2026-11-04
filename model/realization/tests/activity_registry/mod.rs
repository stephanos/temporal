//! Activity registry rows, typed fillers and evidence predicates on the actual Model.
use super::*;
use crate::registry::{ActionKind, Direction, Registry};
use umpire_dsl::Machine as _;
use umpire_runtime::{
    Evidence, EvidenceMatch,
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
    Realization::configured(ProposalContext { start: start(), run_id: RUN.into() })
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
    use umpire_temporal_api::temporal::api::common::v1::{Payload, Payloads, WorkflowExecution};
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
