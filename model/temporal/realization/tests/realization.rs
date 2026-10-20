//! Native realization controls against the independently fixed activity Model.

use std::collections::BTreeMap;

use prost::Message;
use umpire_dsl::{Machine, Outcome as StepOutcome, Rejection};
use umpire_temporal::{Realization, activity, runtime};
use umpire_temporal_api::{
    temporal::api::{
        activity::v1::ActivityExecutionInfo,
        common::v1::{Payload, Payloads},
        workflowservice::v1::{DescribeActivityExecutionResponse, PollActivityTaskQueueResponse},
    },
    umpire::temporal::v1::{
        Callback, Error, Observation, observation::Result as ObservationResult,
    },
};
use umpire_wasm::{
    Dispatch, Guest, Projection, Realize,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
};

const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";
const CANCEL: &str =
    "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution";

#[test]
fn successful_cancel_is_an_explained_step_and_reaches_canceled() {
    let mut guest = Guest::new(runtime());
    let response = umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse {};
    let cancel = Any::pack(&Observation {
        method: CANCEL.into(),
        result: Some(ObservationResult::Response(Any::pack(&response))),
    });
    explained(judge(&mut guest, cancel), &["Client::Cancel"]);
    explained(judge(&mut guest, observation(4, 0)), &[]);
}

fn any(name: &str, value: &impl Message) -> Any {
    Any { type_url: format!("type.googleapis.com/{name}"), value: value.encode_to_vec() }
}

fn observation(status: i32, run_state: i32) -> Any {
    any(
        "umpire.temporal.v1.Observation",
        &Observation {
            method: DESCRIBE.into(),
            result: Some(ObservationResult::Response(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse {
                    info: Some(ActivityExecutionInfo { status, run_state, ..Default::default() }),
                    ..Default::default()
                },
            ))),
        },
    )
}

fn callback(policy: &[u8]) -> Any {
    any(
        "umpire.temporal.v1.Callback",
        &Callback {
            task: Some(any(
                "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse",
                &PollActivityTaskQueueResponse {
                    input: Some(Payloads {
                        payloads: vec![Payload {
                            metadata: BTreeMap::from([(
                                "encoding".into(),
                                b"binary/plain".to_vec(),
                            )]),
                            data: policy.into(),
                            ..Default::default()
                        }],
                    }),
                    ..Default::default()
                },
            )),
        },
    )
}

fn call(
    guest: &mut Guest<impl Dispatch>,
    method: u32,
    request: &impl Message,
) -> protocol::Response {
    let mut response = None;
    assert!(guest.dispatch_to(method, &request.encode_to_vec(), |bytes| {
        response =
            Some(protocol::Response::decode(bytes).unwrap_or_else(|e| panic!("decode: {e}")));
        true
    }));
    response.unwrap_or_else(|| panic!("no response"))
}

fn judge(guest: &mut Guest<impl Dispatch>, value: Any) -> protocol::Response {
    call(guest, 4, &protocol::JudgeRequest { observation: Some(value) })
}

fn answer(guest: &mut Guest<impl Dispatch>, value: Any) -> protocol::Response {
    call(guest, 5, &protocol::AnswerRequest { callback: Some(value) })
}

fn explanation(response: protocol::Response) -> protocol::Explanation {
    let Some(Outcome::Judge(value)) = response.outcome else { panic!("not Judge") };
    let Some(protocol::judge_response::Judgment::Explained(value)) = value.judgment else {
        panic!("not Explained")
    };
    value
}

fn explained(response: protocol::Response, steps: &[&str]) {
    let value = explanation(response);
    assert_eq!(value.steps, steps);
    assert_eq!(value.outcomes, vec!["Accepted"; steps.len()]);
}

fn cancel(error: Option<i32>) -> Any {
    Any::pack(&Observation {
        method: CANCEL.into(),
        result: Some(error.map_or_else(
            || ObservationResult::Response(Any::pack(
                &umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionResponse {},
            )),
            |code| ObservationResult::Error(Error { code, message: "actual server status".into() }),
        )),
    })
}

#[test]
fn cancel_after_closure_is_a_rejected_step_and_success_contradicts_it() {
    for (policy, terminal, step) in [
        (b"Complete".as_slice(), 2, "Worker::RespondCompleted"),
        (b"Fail".as_slice(), 3, "Worker::RespondFailed"),
        (b"".as_slice(), 4, "Client::Cancel"),
    ] {
        let mut guest = Guest::new(runtime());
        if terminal == 4 {
            explained(judge(&mut guest, cancel(None)), &[step]);
        } else {
            answer(&mut guest, callback(policy));
            explained(judge(&mut guest, observation(terminal, 0)), &[step]);
        }
        for _ in 0..2 {
            let result = explanation(judge(&mut guest, cancel(Some(5))));
            assert_eq!(result.steps, ["Client::Cancel"]);
            assert_eq!(result.outcomes, ["Rejected(NotFound)"]);
            explained(judge(&mut guest, observation(terminal, 0)), &[]);
        }
        assert_eq!(unexplained(judge(&mut guest, cancel(None))), ["Client::Cancel"]);
        assert_eq!(unexplained(judge(&mut guest, cancel(Some(5)))), ["Client::Cancel"]);
        refused(answer(&mut guest, callback(b"Complete")), protocol::RefusalCode::State);
    }
}

#[test]
fn cancel_projection_checks_the_actual_outcome_in_every_phase() {
    for phase in [
        activity::Phase::Scheduled,
        activity::Phase::Started,
        activity::Phase::Completed,
        activity::Phase::Failed,
        activity::Phase::Canceled,
        activity::Phase::TimedOut,
    ] {
        let before = activity::Model { phase };
        let Projection::Candidates(paths) = Realization
            .observe(&cancel(Some(5)), &before)
            .unwrap_or_else(|error| panic!("{error:?}"))
        else {
            panic!("cancel must replay")
        };
        assert_eq!(paths.len(), 1);
        assert_eq!(paths[0].path.len(), 1);
        let (outcome, _, after) = activity::World
            .checked_step(&before, paths[0].path[0].action.clone())
            .unwrap_or_else(|error| panic!("{error:?}"))
            .into_iter()
            .next()
            .unwrap_or_else(|| panic!("cancel must be enabled"));
        let closed = umpire_dsl::Phase::closed(&phase);
        assert_eq!(paths[0].outcome.as_ref() == Some(&outcome), closed);
        assert_eq!(paths[0].outcome, Some(StepOutcome::Rejected(Rejection::NotFound)));
        let Projection::Candidates(success) =
            Realization.observe(&cancel(None), &before).unwrap_or_else(|error| panic!("{error:?}"))
        else {
            panic!("cancel must replay")
        };
        assert_eq!(success[0].outcome.as_ref() == Some(&outcome), !closed);
        assert_eq!(success[0].outcome, Some(StepOutcome::Accepted));
        assert_eq!(after.phase, if closed { phase } else { activity::Phase::Canceled });
        assert_eq!(
            outcome,
            if closed { StepOutcome::Rejected(Rejection::NotFound) } else { StepOutcome::Accepted }
        );
    }
}

#[test]
fn contradictory_and_unknown_cancel_statuses_freeze_the_current_expected_actions() {
    for phase in [activity::Phase::Scheduled, activity::Phase::Started, activity::Phase::Canceled] {
        for code in [5, 6, 9, 3, 7, 14, 0, 999] {
            if phase == activity::Phase::Canceled && code == 5 {
                continue;
            }
            let mut guest = Guest::new(runtime());
            let expected = if phase == activity::Phase::Canceled {
                explained(judge(&mut guest, cancel(None)), &["Client::Cancel"]);
                vec!["Client::Cancel"]
            } else if phase == activity::Phase::Started {
                answer(&mut guest, callback(b"Complete"));
                vec![
                    "Client::Cancel",
                    "Worker::RespondCompleted",
                    "Worker::RespondFailed",
                    "Deadline::StartToClose",
                ]
            } else {
                vec!["Client::Cancel", "Worker::Poll", "Deadline::ScheduleToStart"]
            };
            assert_eq!(unexplained(judge(&mut guest, cancel(Some(code)))), expected);
            assert_eq!(unexplained(judge(&mut guest, cancel(None))), expected);
            refused(answer(&mut guest, callback(b"Complete")), protocol::RefusalCode::State);
            call(&mut guest, 3, &protocol::ResetRequest {});
            explained(judge(&mut guest, cancel(None)), &["Client::Cancel"]);
        }
    }
}

#[test]
fn cancel_envelope_refusals_preserve_state_and_frozen_evidence() {
    let malformed = protocol::RefusalCode::MalformedRequest;
    let cases = [
        (Observation { method: "unknown".into(), result: Some(ObservationResult::Error(Error { code: 5, message: String::new() })) }, protocol::RefusalCode::Unsupported),
        (Observation { method: String::new(), result: Some(ObservationResult::Error(Error::default())) }, malformed),
        (Observation { method: CANCEL.into(), result: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::default())) }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::pack(&DescribeActivityExecutionResponse::default()))) }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any { type_url: "type.googleapis.com/temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse".into(), value: vec![255] })) }, malformed),
    ];
    let mut guest = Guest::new(runtime());
    for (value, code) in &cases {
        refused(judge(&mut guest, Any::pack(value)), *code);
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
    let frozen = unexplained(judge(&mut guest, cancel(Some(5))));
    for (value, code) in cases {
        refused(judge(&mut guest, Any::pack(&value)), code);
    }
    assert_eq!(unexplained(judge(&mut guest, cancel(None))), frozen);
}

#[test]
fn cancel_publication_failure_preserves_state_and_freeze() {
    let mut guest = Guest::new(runtime());
    for value in [cancel(None), cancel(Some(5))] {
        let request = protocol::JudgeRequest { observation: Some(value) }.encode_to_vec();
        assert!(!guest.dispatch_to(4, &request, |_| false));
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
    explained(judge(&mut guest, cancel(None)), &["Client::Cancel"]);
    let request = protocol::JudgeRequest { observation: Some(cancel(Some(5))) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &request, |_| false));
    let result = explanation(judge(&mut guest, cancel(Some(5))));
    assert_eq!(result.steps, ["Client::Cancel"]);
    assert_eq!(result.outcomes, ["Rejected(NotFound)"]);
}

#[test]
fn observation_response_tag_stays_pinned_and_error_uses_tag_three() {
    for (result, bytes) in [
        (
            ObservationResult::Response(Any { type_url: "x".into(), value: vec![1, 2] }),
            vec![10, 1, b'm', 18, 7, 10, 1, b'x', 18, 2, 1, 2],
        ),
        (
            ObservationResult::Error(Error { code: 5, message: "x".into() }),
            vec![10, 1, b'm', 26, 5, 8, 5, 18, 1, b'x'],
        ),
    ] {
        let observation = Observation { method: "m".into(), result: Some(result) };
        assert_eq!(observation.encode_to_vec(), bytes);
        assert_eq!(Observation::decode(bytes.as_slice()), Ok(observation));
    }
}

fn unexplained(response: protocol::Response) -> Vec<String> {
    let Some(Outcome::Judge(value)) = response.outcome else { panic!("not Judge") };
    let Some(protocol::judge_response::Judgment::Unexplained(value)) = value.judgment else {
        panic!("not Unexplained")
    };
    value.expected_actions
}

fn refused(response: protocol::Response, code: protocol::RefusalCode) {
    let Some(Outcome::Refusal(value)) = response.outcome else { panic!("not Refusal") };
    assert_eq!(value.code(), code);
}

#[test]
fn named_guest_adapter_preserves_the_independent_report_and_ordered_rows() {
    let mut guest = Guest::new(runtime());
    let Some(Outcome::Explore(exploration)) =
        call(&mut guest, 1, &protocol::ExploreRequest { budget: 64 }).outcome
    else {
        panic!("missing exploration")
    };
    assert_eq!(
        exploration.report,
        include_str!("../../features/activity/expected/report.snap.txt")
    );
    let mut rows = String::new();
    for state_id in 0..exploration.reached {
        let Some(Outcome::Rows(value)) =
            call(&mut guest, 2, &protocol::RowsRequest { state_id }).outcome
        else {
            panic!("missing rows")
        };
        for row in value.rows {
            rows.push_str(&row);
        }
    }
    assert_eq!(rows, include_str!("../../features/activity/expected/rows.snap.txt"));
}

#[test]
fn callback_delivery_precedes_terminal_replay_and_repeated_matches_are_zero_step() {
    for (policy, terminal, step) in [
        (b"Complete".as_slice(), 2, "Worker::RespondCompleted"),
        (b"Fail".as_slice(), 3, "Worker::RespondFailed"),
    ] {
        let mut guest = Guest::new(runtime());
        explained(judge(&mut guest, observation(1, 1)), &[]);
        explained(judge(&mut guest, observation(1, 1)), &[]);
        let Some(Outcome::Answer(value)) = answer(&mut guest, callback(policy)).outcome else {
            panic!("not Answer")
        };
        assert_eq!(
            value.reply.unwrap_or_else(|| panic!("reply missing")).type_url,
            "type.googleapis.com/umpire.temporal.v1.Reply"
        );
        explained(judge(&mut guest, observation(1, 2)), &[]);
        explained(judge(&mut guest, observation(1, 2)), &[]);
        refused(answer(&mut guest, callback(policy)), protocol::RefusalCode::State);
        explained(judge(&mut guest, observation(terminal, 0)), &[step]);
        explained(judge(&mut guest, observation(terminal, 0)), &[]);
        refused(answer(&mut guest, callback(policy)), protocol::RefusalCode::State);
        assert_eq!(
            unexplained(judge(&mut guest, observation(5 - terminal, 0))),
            ["Client::Cancel"]
        );
    }
}

#[test]
fn complete_reply_drops_external_payload_references() {
    use umpire_temporal_api::temporal::api::{
        common::v1::payload::ExternalPayloadDetails,
        workflowservice::v1::RespondActivityTaskCompletedRequest,
    };
    use umpire_temporal_api::umpire::temporal::v1::Reply;

    let mut task: PollActivityTaskQueueResponse =
        umpire_temporal::support::callback(&callback(b"Complete"))
            .unwrap_or_else(|error| panic!("{error:?}"));
    let payload = &mut task.input.as_mut().unwrap_or_else(|| panic!("input")).payloads[0];
    payload.external_payloads.push(ExternalPayloadDetails { size_bytes: 123 });
    let expected = Payload {
        metadata: payload.metadata.clone(),
        data: payload.data.clone(),
        ..Default::default()
    };
    let mut guest = Guest::new(runtime());
    let Some(Outcome::Answer(value)) =
        answer(&mut guest, Any::pack(&Callback { task: Some(Any::pack(&task)) })).outcome
    else {
        panic!("callback must be admitted")
    };
    let reply: Reply = value
        .reply
        .unwrap_or_else(|| panic!("reply missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let request: RespondActivityTaskCompletedRequest = reply
        .request
        .unwrap_or_else(|| panic!("request missing"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    let result = request.result.unwrap_or_else(|| panic!("result missing"));
    assert_eq!(result.payloads, [expected]);
    explained(judge(&mut guest, observation(1, 2)), &[]);
}

#[test]
fn observations_never_invent_poll_and_first_impossible_projection_freezes_until_reset() {
    for premature in [observation(1, 2), observation(2, 0), observation(3, 0)] {
        let mut guest = Guest::new(runtime());
        let first = unexplained(judge(&mut guest, premature));
        assert_eq!(first, ["Client::Cancel", "Worker::Poll", "Deadline::ScheduleToStart"]);
        assert_eq!(unexplained(judge(&mut guest, observation(1, 1))), first);
        refused(answer(&mut guest, callback(b"Complete")), protocol::RefusalCode::State);
        call(&mut guest, 3, &protocol::ResetRequest {});
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
    let mut guest = Guest::new(runtime());
    answer(&mut guest, callback(b"Complete"));
    assert_eq!(
        unexplained(judge(&mut guest, observation(1, 1))),
        [
            "Client::Cancel",
            "Worker::RespondCompleted",
            "Worker::RespondFailed",
            "Deadline::StartToClose"
        ]
    );
    assert_eq!(
        unexplained(judge(&mut guest, observation(2, 0))),
        [
            "Client::Cancel",
            "Worker::RespondCompleted",
            "Worker::RespondFailed",
            "Deadline::StartToClose"
        ]
    );
    let mut other = Guest::new(runtime());
    explained(judge(&mut other, observation(1, 1)), &[]);
}

#[test]
fn malformed_and_unsupported_observations_preserve_run_and_frozen_evidence() {
    let malformed = protocol::RefusalCode::MalformedRequest;
    let unsupported = protocol::RefusalCode::Unsupported;
    let missing_info = any(
        "umpire.temporal.v1.Observation",
        &Observation {
            method: DESCRIBE.into(),
            result: Some(ObservationResult::Response(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse::default(),
            ))),
        },
    );
    let cases = [
        (Any { type_url: String::new(), value: vec![] }, malformed),
        (
            Any {
                type_url: "type.googleapis.com/umpire.oracle.v1.Observation".into(),
                value: vec![],
            },
            unsupported,
        ),
        (
            Any {
                type_url: "type.googleapis.com/umpire.temporal.v1.Observation".into(),
                value: vec![255],
            },
            malformed,
        ),
        (
            any(
                "umpire.temporal.v1.Observation",
                &Observation { method: "wrong".into(), result: None },
            ),
            unsupported,
        ),
        (
            any(
                "umpire.temporal.v1.Observation",
                &Observation { method: DESCRIBE.into(), result: None },
            ),
            malformed,
        ),
        (missing_info, malformed),
        (observation(0, 1), malformed),
        (observation(1, 0), malformed),
        (observation(999, 1), unsupported),
        (observation(1, 999), unsupported),
    ];
    let mut guest = Guest::new(runtime());
    for (value, code) in &cases {
        refused(judge(&mut guest, value.clone()), *code);
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
    let first = unexplained(judge(&mut guest, observation(2, 0)));
    for (value, code) in cases {
        refused(judge(&mut guest, value), code);
    }
    assert_eq!(unexplained(judge(&mut guest, observation(1, 1))), first);
}

#[test]
fn rejected_publication_rolls_back_delivery_terminal_freeze_and_reset() {
    let mut guest = Guest::new(runtime());
    let deliver = protocol::AnswerRequest { callback: Some(callback(b"Complete")) }.encode_to_vec();
    assert!(!guest.dispatch_to(5, &deliver, |_| false));
    explained(judge(&mut guest, observation(1, 1)), &[]);
    answer(&mut guest, callback(b"Complete"));
    let close = protocol::JudgeRequest { observation: Some(observation(2, 0)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &close, |_| false));
    explained(judge(&mut guest, observation(1, 2)), &[]);
    let stale = protocol::JudgeRequest { observation: Some(observation(1, 1)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &stale, |_| false));
    explained(judge(&mut guest, observation(2, 0)), &["Worker::RespondCompleted"]);
    assert!(!guest.dispatch_to(3, &[], |_| false));
    explained(judge(&mut guest, observation(2, 0)), &[]);
}

#[test]
fn invalid_callback_projection_cannot_partially_commit_poll() {
    let malformed = protocol::RefusalCode::MalformedRequest;
    let unsupported = protocol::RefusalCode::Unsupported;
    let mut cases = vec![
        (Any { type_url: String::new(), value: vec![] }, malformed),
        (Any { type_url: "wrong".into(), value: vec![] }, unsupported),
        (Any { type_url: "type.googleapis.com/umpire.temporal.v1.Callback".into(), value: vec![255] }, malformed),
        (any("umpire.temporal.v1.Callback", &Callback { task: None }), malformed),
        (any("umpire.temporal.v1.Callback", &Callback { task: Some(Any { type_url: String::new(), value: vec![] }) }), malformed),
        (any("umpire.temporal.v1.Callback", &Callback { task: Some(Any { type_url: "wrong".into(), value: vec![] }) }), unsupported),
        (any("umpire.temporal.v1.Callback", &Callback { task: Some(Any { type_url: "type.googleapis.com/temporal.api.workflowservice.v1.PollActivityTaskQueueResponse".into(), value: vec![255] }) }), malformed),
        (callback(b"Unknown"), unsupported),
    ];
    let valid = PollActivityTaskQueueResponse {
        input: Some(Payloads {
            payloads: vec![Payload {
                metadata: BTreeMap::from([("encoding".into(), b"binary/plain".to_vec())]),
                data: b"Complete".to_vec(),
                ..Default::default()
            }],
        }),
        ..Default::default()
    };
    for variant in 0..6 {
        let mut invalid = valid.clone();
        match variant {
            0 => invalid.input = None,
            1 => invalid.input = Some(Payloads { payloads: vec![] }),
            2 => {
                let payloads = invalid.input.as_mut().unwrap_or_else(|| panic!("input"));
                payloads.payloads.push(payloads.payloads[0].clone());
            }
            3 => invalid.input.as_mut().unwrap_or_else(|| panic!("input")).payloads[0]
                .metadata
                .clear(),
            4 => {
                invalid.input.as_mut().unwrap_or_else(|| panic!("input")).payloads[0]
                    .metadata
                    .insert("encoding".into(), b"json/plain".to_vec());
            }
            5 => invalid.input.as_mut().unwrap_or_else(|| panic!("input")).payloads[0].data.clear(),
            _ => unreachable!(),
        }
        cases.push((
            any(
                "umpire.temporal.v1.Callback",
                &Callback {
                    task: Some(any(
                        "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse",
                        &invalid,
                    )),
                },
            ),
            if variant < 4 { malformed } else { unsupported },
        ));
    }
    let mut guest = Guest::new(runtime());
    for (value, code) in cases {
        refused(answer(&mut guest, value), code);
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
    assert!(matches!(answer(&mut guest, callback(b"Complete")).outcome, Some(Outcome::Answer(_))));
    explained(judge(&mut guest, observation(1, 2)), &[]);
}
