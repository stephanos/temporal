//! Native realization controls against the independently fixed activity Model.

use std::collections::BTreeMap;

use prost::Message;
use umpire_temporal::runtime;
use umpire_temporal_api::{
    temporal::api::{
        activity::v1::ActivityExecutionInfo,
        common::v1::{Payload, Payloads},
        workflowservice::v1::{DescribeActivityExecutionResponse, PollActivityTaskQueueResponse},
    },
    umpire::temporal::v1::{Callback, Observation},
};
use umpire_wasm::{
    Dispatch, Guest,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
};

const DESCRIBE: &str = "/temporal.api.workflowservice.v1.WorkflowService/DescribeActivityExecution";

fn any(name: &str, value: &impl Message) -> Any {
    Any { type_url: format!("type.googleapis.com/{name}"), value: value.encode_to_vec() }
}

fn observation(status: i32, run_state: i32) -> Any {
    any(
        "umpire.temporal.v1.Observation",
        &Observation {
            method: DESCRIBE.into(),
            response: Some(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse {
                    info: Some(ActivityExecutionInfo { status, run_state, ..Default::default() }),
                    ..Default::default()
                },
            )),
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

fn explained(response: protocol::Response, steps: &[&str]) {
    let Some(Outcome::Judge(value)) = response.outcome else { panic!("not Judge") };
    let Some(protocol::judge_response::Judgment::Explained(value)) = value.judgment else {
        panic!("not Explained")
    };
    assert_eq!(value.steps, steps);
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
            response: Some(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse::default(),
            )),
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
                &Observation { method: "wrong".into(), response: None },
            ),
            unsupported,
        ),
        (
            any(
                "umpire.temporal.v1.Observation",
                &Observation { method: DESCRIBE.into(), response: None },
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
