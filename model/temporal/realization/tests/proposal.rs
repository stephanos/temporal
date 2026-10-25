//! Configured bootstrap, read correlation, and completion polling controls.
#![forbid(unsafe_code)]

use prost::Message;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::protocol::{Method, RefusalCode};
use umpire_runtime::testing::{GuestExt, ResponseExt};
use umpire_runtime::{
    Dispatch, Guest,
    prost_types::Any,
    protocol::{self, response::Outcome},
    realization::Observation,
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::{
    temporal::api::workflowservice::v1::DescribeActivityExecutionRequest,
    umpire::temporal::v1::{ObserveActivity, TemporalInstruction},
};
mod common;

fn configure(guest: &mut Guest<impl Dispatch>) -> protocol::ResetRequest {
    let request = protocol::ResetRequest { configuration: common::start(3).request };
    assert!(matches!(
        GuestExt::call(guest, Method::Reset, &request).outcome,
        Some(Outcome::Reset(_))
    ));
    request
}
fn admit(guest: &mut Guest<impl Dispatch>) {
    admit_seed(guest, 0);
}
fn admit_seed(guest: &mut Guest<impl Dispatch>, seed: u64) {
    let bootstrap = guest.propose(seed);
    assert_eq!(bootstrap.kind(), protocol::ProposalKind::Bootstrap);
    assert_eq!(bootstrap.action, "");
    let mut start = common::start(3);
    start.request = bootstrap.request;
    let response = guest.judge(Any::pack(&start));
    assert!(matches!(response.outcome, Some(Outcome::Judge(_))), "{response:?}");
}
#[test]
fn configured_bootstrap_and_reset_retain_configuration_but_clear_admission() {
    let mut guest = Guest::new(annotation_runtime());
    let reset = configure(&mut guest);
    GuestExt::call(&mut guest, Method::Answer, &protocol::AnswerRequest::default())
        .refusal(RefusalCode::Unsupported);
    let bootstrap = guest.propose(17);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 17 })
        .refusal(RefusalCode::State);
    let mut crossed = common::start(3);
    crossed.request = common::start(1).request;
    guest.judge(Any::pack(&crossed)).refusal(RefusalCode::State);
    let mut start = common::start(3);
    start.request = bootstrap.request.clone();
    let bytes = protocol::JudgeRequest { observation: Some(Any::pack(&start)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &bytes, |_| false));
    GuestExt::call(&mut guest, Method::Answer, &protocol::AnswerRequest::default())
        .refusal(RefusalCode::Unsupported);
    guest.judge(Any::pack(&start));
    assert!(!guest.dispatch_to(3, &reset.encode_to_vec(), |_| false));
    assert_eq!(guest.propose(1).kind(), protocol::ProposalKind::Read);
    guest.reset(None);
    assert_eq!(guest.propose(17), bootstrap);
}
#[test]
fn rejected_cancel_after_completion_is_still_an_enabled_model_action() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    guest.judge(Any::pack(&common::describe(1, 2, 0)));
    let cancel = guest.propose(0);
    assert_eq!(cancel.action, "Client::Cancel");
    let rejected = Observation {
        method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
            .into(),
        request: cancel.request.map(|request| {
            let instruction: TemporalInstruction =
                request.unpack().unwrap_or_else(|e| panic!("{e:?}"));
            instruction.request.unwrap_or_else(|| panic!("cancel request"))
        }),
        result: Some(umpire_runtime::realization::observation::Result::Error(
            umpire_runtime::realization::Error { code: 5, message: String::new() },
        )),
    };
    let Some(Outcome::Judge(result)) = guest.judge(Any::pack(&rejected)).outcome else {
        panic!("rejected cancellation")
    };
    assert!(!result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = result.judgment else {
        panic!("rejection is explained")
    };
    assert_eq!(
        explanation.steps,
        ["Worker::Poll(1)", "Worker::RespondCompleted", "Client::Cancel"]
    );
    assert_eq!(explanation.outcomes, ["Accepted", "Accepted", "Rejected(NotFound)"]);
    assert_eq!(guest.propose(0).action, "Client::Cancel");
}

#[test]
fn failed_and_missing_start_evidence_never_admits_or_clears_pending() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    let bootstrap = guest.propose(1);
    for result in [
        None,
        Some(umpire_runtime::realization::observation::Result::Error(
            umpire_runtime::realization::Error { code: 3, message: String::new() },
        )),
    ] {
        let mut observed = common::start(3);
        observed.request = bootstrap.request.clone();
        observed.result = result;
        guest.judge(Any::pack(&observed)).refusal(RefusalCode::MalformedRequest);
        GuestExt::call(&mut guest, Method::Answer, &protocol::AnswerRequest::default())
            .refusal(RefusalCode::Unsupported);
        GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 1 })
            .refusal(RefusalCode::State);
    }
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
    GuestExt::call(
        &mut invalid_guest,
        Method::Reset,
        &protocol::ResetRequest { configuration: Some(Any::pack(&missing)) },
    );
    GuestExt::call(&mut invalid_guest, Method::Propose, &protocol::ProposeRequest { seed: 42 })
        .refusal(RefusalCode::MalformedRequest);
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
        GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 42 })
            .refusal(RefusalCode::Unsupported);
        GuestExt::call(&mut guest, Method::Reset, &reset);
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
        let published =
            GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 42 });
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
        guest.reset(None);
        assert_eq!(guest.propose(42).request, Some(Any::pack(&filled)));
    }
}

#[test]
fn pending_cancel_rejects_unrequested_describe_until_its_matching_invocation_finishes() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    let cancel = guest.propose(2);
    assert_eq!(cancel.action, "Client::Cancel");
    let mut unrequested = common::describe(1, 2, 0);
    let mut request: DescribeActivityExecutionRequest = unrequested
        .request
        .as_ref()
        .unwrap_or_else(|| panic!("Describe request"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    request.include_outcome = false;
    unrequested.request = Some(Any::pack(&request));
    guest.judge(Any::pack(&unrequested)).refusal(RefusalCode::State);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 0 })
        .refusal(RefusalCode::State);
    let observed = Observation { method: "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution".into(), request: cancel.request.map(|request| { let instruction: TemporalInstruction = request.unpack().unwrap_or_else(|e| panic!("{e:?}")); instruction.request.unwrap_or_else(|| panic!("cancel request")) }),
        result: Some(umpire_runtime::realization::observation::Result::Response(Any::pack(&umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse::default()))) };
    let bytes = protocol::JudgeRequest { observation: Some(Any::pack(&observed)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &bytes, |_| false));
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 0 })
        .refusal(RefusalCode::State);
    let Some(Outcome::Judge(result)) = guest.judge(Any::pack(&observed)).outcome else {
        panic!("cancel")
    };
    assert!(!result.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = result.judgment else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Client::Cancel"]);
    assert_eq!(guest.propose(0).action, "Client::Cancel");
}

#[test]
fn read_proposals_correlate_observations_and_never_name_worker_actions() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit(&mut guest);
    let proposal = guest.propose(1);
    assert_eq!(proposal.kind(), protocol::ProposalKind::Read);
    assert_eq!(proposal.action, "");
    let instruction: ObserveActivity = proposal
        .request
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let request: DescribeActivityExecutionRequest = instruction
        .request
        .unwrap_or_else(|| panic!("Describe"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    assert!(request.include_outcome && request.include_last_failure);
    for field in ["include_outcome", "include_last_failure"] {
        let identity =
            format!("temporal.api.workflowservice.v1.DescribeActivityExecutionRequest.{field}");
        let claim = umpire_temporal_api::workflow_field_claim(&identity)
            .unwrap_or_else(|| panic!("missing {identity}"));
        assert_eq!(claim.kind, "Flag");
        assert_eq!(claim.ignored, None);
    }
    assert_eq!(request.run_id, common::RUN);
    assert_eq!(
        (instruction.poll_interval_ms, instruction.read_timeout_ms, instruction.backoff_ms),
        (10, 30_000, 10)
    );
    let Some(Outcome::Judge(scheduled)) =
        guest.judge(Any::pack(&common::describe(1, 1, 1))).outcome
    else {
        panic!("scheduled")
    };
    assert!(scheduled.continue_polling);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 0 })
        .refusal(RefusalCode::State);
    let Some(Outcome::Judge(complete)) = guest.judge(Any::pack(&common::describe(1, 2, 0))).outcome
    else {
        panic!("completed")
    };
    assert!(!complete.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = complete.judgment else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Worker::Poll(1)", "Worker::RespondCompleted"]);
    assert_eq!(guest.propose(0).action, "Client::Cancel");
}

#[test]
fn read_remains_pending_through_retry_and_delivery_until_terminal_evidence() {
    let mut guest = Guest::new(annotation_runtime());
    configure(&mut guest);
    admit_seed(&mut guest, 1);
    assert_eq!(guest.propose(1).kind(), protocol::ProposalKind::Read);
    let Some(Outcome::Judge(retry)) = guest.judge(Any::pack(&common::describe(2, 1, 1))).outcome
    else {
        panic!("retry")
    };
    assert!(retry.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = retry.judgment else {
        panic!("explained")
    };
    assert_eq!(explanation.steps, ["Worker::Poll(1)", "Worker::RespondFailed { retryable: true }"]);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 1 })
        .refusal(RefusalCode::State);
    let Some(Outcome::Judge(current)) = guest.judge(Any::pack(&common::describe(2, 1, 1))).outcome
    else {
        panic!("current")
    };
    assert!(current.continue_polling);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 0 })
        .refusal(RefusalCode::State);
    let Some(Outcome::Judge(started)) = guest.judge(Any::pack(&common::describe(2, 1, 2))).outcome
    else {
        panic!("started")
    };
    assert!(started.continue_polling);
    GuestExt::call(&mut guest, Method::Propose, &protocol::ProposeRequest { seed: 1 })
        .refusal(RefusalCode::State);
    let Some(Outcome::Judge(current)) = guest.judge(Any::pack(&common::describe(2, 1, 2))).outcome
    else {
        panic!("current started")
    };
    assert!(current.continue_polling);
    let Some(Outcome::Judge(completed)) =
        guest.judge(Any::pack(&common::describe(2, 2, 0))).outcome
    else {
        panic!("completed")
    };
    assert!(!completed.continue_polling);
    let Some(protocol::judge_response::Judgment::Explained(explanation)) = completed.judgment
    else {
        panic!("terminal explanation")
    };
    assert_eq!(
        explanation.steps,
        [
            "Worker::Poll(1)",
            "Worker::RespondFailed { retryable: true }",
            "Worker::Poll(2)",
            "Worker::RespondCompleted"
        ]
    );
    assert_eq!(guest.propose(0).action, "Client::Cancel");
}
