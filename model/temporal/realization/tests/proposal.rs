//! Configured bootstrap, callback context, and completion polling controls.
use prost::Message;
use umpire_runtime::{
    Dispatch, Guest,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
    realization::{Callback, Observation},
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::{
    temporal::api::{
        common::v1::{Payload, Payloads},
        workflowservice::v1::{DescribeActivityExecutionRequest, PollActivityTaskQueueResponse},
    },
    umpire::temporal::v1::ObserveActivity,
};
mod common;

fn call(
    guest: &mut Guest<impl Dispatch>,
    method: u32,
    request: &impl Message,
) -> protocol::Response {
    let mut response = None;
    assert!(guest.dispatch_to(method, &request.encode_to_vec(), |bytes| {
        response =
            Some(protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")));
        true
    }));
    response.unwrap_or_else(|| panic!("response missing"))
}
fn propose(guest: &mut Guest<impl Dispatch>, seed: u64) -> protocol::ProposeResponse {
    let response = call(guest, 6, &protocol::ProposeRequest { seed });
    let Some(Outcome::Propose(proposal)) = response.outcome else {
        panic!("proposal: {response:?}")
    };
    proposal
}
fn judge(guest: &mut Guest<impl Dispatch>, observation: &Observation) -> protocol::Response {
    call(guest, 4, &protocol::JudgeRequest { observation: Some(Any::pack(observation)) })
}
fn refused(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(value)) = response.outcome else { panic!("refusal: {response:?}") };
    assert_eq!(value.code(), code);
}
fn configure(guest: &mut Guest<impl Dispatch>) -> protocol::ResetRequest {
    let request = protocol::ResetRequest { configuration: common::start(3).request };
    assert!(matches!(call(guest, 3, &request).outcome, Some(Outcome::Reset(_))));
    request
}
fn admit(guest: &mut Guest<impl Dispatch>) {
    let bootstrap = propose(guest, 1);
    assert_eq!(bootstrap.kind(), protocol::ProposalKind::Bootstrap);
    assert_eq!(bootstrap.action, "");
    let mut start = common::start(3);
    start.request = bootstrap.request;
    let response = judge(guest, &start);
    assert!(matches!(response.outcome, Some(Outcome::Judge(_))), "{response:?}");
}
fn callback_policy(policy: &str) -> protocol::AnswerRequest {
    protocol::AnswerRequest {
        callback: Some(Any::pack(&Callback {
            task: Some(Any::pack(&PollActivityTaskQueueResponse {
                workflow_namespace: "model-slice".into(),
                activity_id: "activity".into(),
                activity_run_id: common::RUN.into(),
                attempt: 1,
                task_token: vec![1, 2, 3],
                input: Some(Payloads {
                    payloads: vec![Payload {
                        metadata: std::collections::BTreeMap::from([(
                            "encoding".into(),
                            b"binary/plain".to_vec(),
                        )]),
                        data: policy.as_bytes().to_vec(),
                        ..Default::default()
                    }],
                }),
                ..Default::default()
            })),
        })),
    }
}
fn callback() -> protocol::AnswerRequest {
    callback_policy("Complete")
}

#[test]
fn configured_bootstrap_and_reset_retain_configuration_but_clear_admission() {
    let mut guest = Guest::new(annotation_runtime());
    let reset = configure(&mut guest);
    refused(call(&mut guest, 5, &callback()), protocol::RefusalCode::State);
    let bootstrap = propose(&mut guest, 17);
    refused(
        call(&mut guest, 6, &protocol::ProposeRequest { seed: 17 }),
        protocol::RefusalCode::State,
    );
    let mut crossed = common::start(3);
    crossed.request = common::start(1).request;
    refused(judge(&mut guest, &crossed), protocol::RefusalCode::State);
    let mut start = common::start(3);
    start.request = bootstrap.request.clone();
    let bytes = protocol::JudgeRequest { observation: Some(Any::pack(&start)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &bytes, |_| false));
    refused(call(&mut guest, 5, &callback()), protocol::RefusalCode::State);
    judge(&mut guest, &start);
    assert!(!guest.dispatch_to(3, &reset.encode_to_vec(), |_| false));
    assert_eq!(propose(&mut guest, 1).kind(), protocol::ProposalKind::Action);
    call(&mut guest, 3, &protocol::ResetRequest::default());
    assert_eq!(propose(&mut guest, 17), bootstrap);
}
#[test]
fn published_callback_context_survives_intermediate_reads_and_proposes_no_duplicate_response() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    assert!(matches!(call(&mut guest, 5, &callback()).outcome, Some(Outcome::Answer(_))));
    let Some(Outcome::Judge(intermediate)) = judge(&mut guest, &common::describe(1, 1, 2)).outcome
    else {
        panic!("intermediate")
    };
    assert!(!intermediate.continue_polling);
    let proposal = propose(&mut guest, 1);
    assert_eq!(proposal.action, "Worker::RespondCompleted");
    let instruction: ObserveActivity = proposal
        .request
        .unwrap_or_else(|| panic!("request missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let read: DescribeActivityExecutionRequest = instruction
        .request
        .unwrap_or_else(|| panic!("request missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert_eq!(read.run_id, common::RUN);
    assert!(
        instruction.poll_interval_ms > 0
            && instruction.read_timeout_ms > 0
            && instruction.backoff_ms > 0
    );
    assert_eq!(instruction.worker_tasks, 0, "a response read admits no worker task");
    let Some(Outcome::Judge(final_result)) = judge(&mut guest, &common::describe(1, 2, 0)).outcome
    else {
        panic!("final")
    };
    assert!(!final_result.continue_polling);
    assert_eq!(propose(&mut guest, 1).action, "Client::Cancel");
}
#[test]
fn pending_read_accepts_callback_between_polls_and_finally_clears() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    let proposal = propose(&mut guest, 1);
    assert_eq!(proposal.action, "Worker::Poll(1)");
    let instruction: ObserveActivity = proposal
        .request
        .unwrap_or_else(|| panic!("request missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert_eq!(instruction.worker_tasks, 1, "a worker poll admits one worker task");
    let Some(Outcome::Judge(poll)) = judge(&mut guest, &common::describe(1, 1, 1)).outcome else {
        panic!("poll")
    };
    assert!(poll.continue_polling);
    call(&mut guest, 5, &callback());
    refused(
        call(&mut guest, 6, &protocol::ProposeRequest { seed: 0 }),
        protocol::RefusalCode::State,
    );
    let Some(Outcome::Judge(final_result)) = judge(&mut guest, &common::describe(1, 2, 0)).outcome
    else {
        panic!("final")
    };
    assert!(!final_result.continue_polling);
    assert_eq!(propose(&mut guest, 0).action, "Client::Cancel");
}

#[test]
fn delivery_proposal_completes_at_started_before_worker_response() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(1)");
    call(&mut guest, 5, &callback());
    let Some(Outcome::Judge(delivery)) = judge(&mut guest, &common::describe(1, 1, 2)).outcome
    else {
        panic!("delivery")
    };
    assert!(
        !delivery.continue_polling,
        "observed Started finishes Poll, without waiting for worker response"
    );
    assert_eq!(propose(&mut guest, 1).action, "Worker::RespondCompleted");
}

#[test]
fn rejected_cancel_after_completion_is_still_an_enabled_model_action() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    call(&mut guest, 5, &callback());
    judge(&mut guest, &common::describe(1, 2, 0));
    let cancel = propose(&mut guest, 0);
    assert_eq!(cancel.action, "Client::Cancel");
    let rejected = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: cancel.request,
        result: Some(umpire_runtime::realization::observation::Result::Error(
            umpire_runtime::realization::Error { code: 5, message: String::new() },
        )),
    };
    let Some(Outcome::Judge(result)) = judge(&mut guest, &rejected).outcome else {
        panic!("rejected cancellation")
    };
    assert!(!result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = result.judgment else {
        panic!("rejection is explained")
    };
    assert_eq!(explanation.outcomes, ["Rejected(NotFound)"]);
    assert_eq!(propose(&mut guest, 0).action, "Client::Cancel");
}

#[test]
fn failed_and_missing_start_evidence_never_admits_or_clears_pending() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    let bootstrap = propose(&mut guest, 1);
    for result in [
        None,
        Some(umpire_runtime::realization::observation::Result::Error(
            umpire_runtime::realization::Error { code: 3, message: String::new() },
        )),
    ] {
        let mut observed = common::start(3);
        observed.request = bootstrap.request.clone();
        observed.result = result;
        refused(judge(&mut guest, &observed), protocol::RefusalCode::MalformedRequest);
        refused(call(&mut guest, 5, &callback()), protocol::RefusalCode::State);
        refused(
            call(&mut guest, 6, &protocol::ProposeRequest { seed: 1 }),
            protocol::RefusalCode::State,
        );
    }
}

#[test]
fn retry_response_completes_at_next_scheduled_attempt() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    let request = callback_policy("RetryThenComplete");
    call(&mut guest, 5, &request);
    assert_eq!(propose(&mut guest, 1).action, "Worker::RespondFailed { retryable: true }");
    let Some(Outcome::Judge(result)) = judge(&mut guest, &common::describe(2, 1, 1)).outcome else {
        panic!("retry")
    };
    assert!(!result.continue_polling);
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(2)");
}

#[test]
fn failed_answer_publication_never_caches_context_or_delivery() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    assert!(!guest.dispatch_to(5, &callback().encode_to_vec(), |_| false));
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(1)");
    call(&mut guest, 3, &protocol::ResetRequest::default());
    admit(&mut guest);
    let mut envelope: Callback = callback()
        .callback
        .unwrap_or_else(|| panic!("callback"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let mut task: PollActivityTaskQueueResponse = envelope
        .task
        .take()
        .unwrap_or_else(|| panic!("task"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    task.attempt = 2;
    envelope.task = Some(Any::pack(&task));
    refused(
        call(&mut guest, 5, &protocol::AnswerRequest { callback: Some(Any::pack(&envelope)) }),
        protocol::RefusalCode::State,
    );
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(1)");
}

#[test]
fn filled_bootstrap_is_seeded_from_explicit_configuration_and_failed_configuration_is_atomic() {
    use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;
    let mut start: StartActivityExecutionRequest = common::start(3)
        .request
        .unwrap_or_else(|| panic!("start"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    start.request_id.clear();
    let mut missing = start.clone();
    missing.activity_id.clear();
    let mut invalid_guest = Guest::new(annotation_runtime());
    call(
        &mut invalid_guest,
        3,
        &protocol::ResetRequest { configuration: Some(Any::pack(&missing)) },
    );
    refused(
        call(&mut invalid_guest, 6, &protocol::ProposeRequest { seed: 42 }),
        protocol::RefusalCode::MalformedRequest,
    );
    let reset = protocol::ResetRequest { configuration: Some(Any::pack(&start)) };
    for panic in [false, true] {
        let mut guest = Guest::new(annotation_runtime());
        let failed = std::panic::catch_unwind(std::panic::AssertUnwindSafe(|| {
            guest.dispatch_to(3, &reset.encode_to_vec(), |_| {
                assert!(!panic, "publisher failed");
                false
            })
        }));
        assert_eq!(failed.is_err(), panic);
        refused(
            call(&mut guest, 6, &protocol::ProposeRequest { seed: 42 }),
            protocol::RefusalCode::Unsupported,
        );
        call(&mut guest, 3, &reset);
        let mut unpublished = None;
        assert!(!guest.dispatch_to(
            6,
            &protocol::ProposeRequest { seed: 42 }.encode_to_vec(),
            |bytes| {
                unpublished = Some(
                    protocol::Response::decode(bytes).unwrap_or_else(|error| panic!("{error}")),
                );
                false
            }
        ));
        let published = call(&mut guest, 6, &protocol::ProposeRequest { seed: 42 });
        assert_eq!(Some(published.clone()), unpublished);
        let Some(Outcome::Propose(proposal)) = published.outcome else {
            panic!("bootstrap: {published:?}")
        };
        let filled: StartActivityExecutionRequest = proposal
            .request
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|error| panic!("{error:?}"));
        assert_eq!(filled.activity_id, start.activity_id);
        assert_ne!(filled.request_id, "");
        assert_eq!(filled.namespace, start.namespace);
        assert_eq!(filled.retry_policy, start.retry_policy);
        call(&mut guest, 3, &protocol::ResetRequest::default());
        assert_eq!(propose(&mut guest, 42).request, Some(Any::pack(&filled)));
    }
}

#[test]
fn delivery_completes_when_retry_precedes_the_first_started_read() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(1)");
    call(&mut guest, 5, &callback_policy("RetryThenComplete"));
    let Some(Outcome::Judge(result)) = judge(&mut guest, &common::describe(2, 1, 1)).outcome else {
        panic!("retry")
    };
    assert!(!result.continue_polling, "validated retry proves prior delivery completed");
    assert_eq!(propose(&mut guest, 1).action, "Worker::Poll(2)");
}

#[test]
fn pending_cancel_rejects_describe_until_its_matching_invocation_finishes() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    call(&mut guest, 5, &callback());
    let cancel = propose(&mut guest, 0);
    assert_eq!(cancel.action, "Client::Cancel");
    refused(judge(&mut guest, &common::describe(1, 2, 0)), protocol::RefusalCode::State);
    refused(
        call(&mut guest, 6, &protocol::ProposeRequest { seed: 0 }),
        protocol::RefusalCode::State,
    );
    let observed = Observation { method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution".into(), request: cancel.request,
        result: Some(umpire_runtime::realization::observation::Result::Response(Any::pack(&umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse::default()))) };
    let bytes = protocol::JudgeRequest { observation: Some(Any::pack(&observed)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &bytes, |_| false));
    refused(
        call(&mut guest, 6, &protocol::ProposeRequest { seed: 0 }),
        protocol::RefusalCode::State,
    );
    let Some(Outcome::Judge(result)) = judge(&mut guest, &observed).outcome else {
        panic!("cancel")
    };
    assert!(!result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = result.judgment else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Client::Cancel"]);
    assert_eq!(propose(&mut guest, 0).action, "Client::Cancel");
}
