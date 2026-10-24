//! Native realization controls against the independently fixed activity Model.

use umpire_runtime::realization::Error;
use umpire_runtime::realization::Observation;
use umpire_runtime::realization::observation::Result as ObservationResult;

use prost::Message;
use umpire_dsl::{Machine, Outcome as StepOutcome, Rejection};
use umpire_runtime::{
    Dispatch, Guest, Projection, Realize,
    google::protobuf::Any,
    protocol::{self, response::Outcome},
};
use umpire_temporal::{Realization, activity, runtime};
use umpire_temporal_api::temporal::api::{
    activity::v1::ActivityExecutionInfo,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        RequestCancelActivityExecutionRequest,
    },
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
        request: None,
    });
    explained(judge(&mut guest, cancel), &["Client::Cancel"]);
    explained(judge(&mut guest, observation(4, 0)), &[]);
}

fn any(name: &str, value: &impl Message) -> Any {
    Any { type_url: format!("type.googleapis.com/{name}"), value: value.encode_to_vec() }
}

fn observation(status: i32, run_state: i32) -> Any {
    any(
        "umpire.realization.v1.Observation",
        &Observation {
            method: DESCRIBE.into(),
            result: Some(ObservationResult::Response(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse {
                    info: Some(ActivityExecutionInfo {
                        status,
                        run_state,
                        attempt: 1,
                        ..Default::default()
                    }),
                    ..Default::default()
                },
            ))),
            request: None,
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
        request: None,
    })
}

#[test]
fn cancel_after_closure_is_a_rejected_step_and_success_contradicts_it() {
    for (terminal, step) in [
        (2, "Worker::RespondCompleted"),
        (3, "Worker::RespondFailed { retryable: false }"),
        (4, "Client::Cancel"),
    ] {
        let mut guest = Guest::new(runtime());
        if terminal == 4 {
            explained(judge(&mut guest, cancel(None)), &[step]);
        } else {
            explained(judge(&mut guest, observation(1, 2)), &["Worker::Poll(1)"]);
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
        let before = activity::Model { phase, attempt: 1 };
        let Projection::Candidates(paths) = Realization::default()
            .observe(&cancel(Some(5)), &before)
            .unwrap_or_else(|error| panic!("{error:?}"))
        else {
            panic!("cancel must replay")
        };
        assert_eq!(paths.len(), 1);
        assert_eq!(paths[0].path.len(), 1);
        let (outcome, _, after) = activity::World::new(3)
            .checked_step(&before, paths[0].path[0].action.clone())
            .unwrap_or_else(|error| panic!("{error:?}"))
            .into_iter()
            .next()
            .unwrap_or_else(|| panic!("cancel must be enabled"));
        let closed = umpire_dsl::Phase::closed(&phase);
        assert_eq!(paths[0].outcome.as_ref() == Some(&outcome), closed);
        assert_eq!(paths[0].outcome, Some(StepOutcome::Rejected(Rejection::NotFound)));
        let Projection::Candidates(success) = Realization::default()
            .observe(&cancel(None), &before)
            .unwrap_or_else(|error| panic!("{error:?}"))
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
                explained(judge(&mut guest, observation(1, 2)), &["Worker::Poll(1)"]);
                vec![
                    "Client::Cancel",
                    "Worker::Poll(2)",
                    "Worker::Poll(3)",
                    "Worker::RespondCompleted",
                    "Worker::RespondFailed { retryable: false }",
                    "Worker::RespondFailed { retryable: true }",
                    "Deadline::StartToClose",
                ]
            } else {
                vec![
                    "Client::Cancel",
                    "Worker::Poll(1)",
                    "Worker::Poll(2)",
                    "Worker::Poll(3)",
                    "Deadline::ScheduleToStart",
                ]
            };
            assert_eq!(unexplained(judge(&mut guest, cancel(Some(code)))), expected);
            assert_eq!(unexplained(judge(&mut guest, cancel(None))), expected);
            call(&mut guest, 3, &protocol::ResetRequest::default());
            explained(judge(&mut guest, cancel(None)), &["Client::Cancel"]);
        }
    }
}

#[test]
fn cancel_envelope_refusals_preserve_state_and_frozen_evidence() {
    let malformed = protocol::RefusalCode::MalformedRequest;
    let cases = [
        (Observation { method: "unknown".into(), result: Some(ObservationResult::Error(Error { code: 5, message: String::new() })), request: None }, protocol::RefusalCode::Unsupported),
        (Observation { method: String::new(), result: Some(ObservationResult::Error(Error::default())), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: None, request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::default())), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::pack(&DescribeActivityExecutionResponse::default()))), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any { type_url: "type.googleapis.com/temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse".into(), value: vec![255] })), request: None }, malformed),
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
        let observation = Observation { method: "m".into(), result: Some(result), request: None };
        assert_eq!(observation.encode_to_vec(), bytes);
        assert_eq!(Observation::decode(bytes.as_slice()), Ok(observation));
    }
}

#[test]
fn observation_request_uses_tag_four_after_the_unchanged_result() {
    let observation = Observation {
        method: "m".into(),
        result: Some(ObservationResult::Response(Any { type_url: "x".into(), value: vec![1, 2] })),
        request: Some(Any { type_url: "r".into(), value: vec![3] }),
    };
    let bytes = vec![10, 1, b'm', 18, 7, 10, 1, b'x', 18, 2, 1, 2, 34, 6, 10, 1, b'r', 18, 1, 3];
    assert_eq!(observation.encode_to_vec(), bytes);
    assert_eq!(Observation::decode(bytes.as_slice()), Ok(observation.clone()));
    let go = [10, 1, b'm', 34, 6, 10, 1, b'r', 18, 1, 3, 18, 7, 10, 1, b'x', 18, 2, 1, 2];
    assert_eq!(Observation::decode(go.as_slice()), Ok(observation));
}

#[test]
fn actual_requests_leave_current_judgments_unchanged() {
    let describe = DescribeActivityExecutionRequest {
        namespace: "model-slice".into(),
        activity_id: "activity".into(),
        run_id: "run".into(),
        include_outcome: true,
        ..Default::default()
    };
    let cancel_request = RequestCancelActivityExecutionRequest {
        namespace: "model-slice".into(),
        activity_id: "activity".into(),
        run_id: "run".into(),
        request_id: "request".into(),
        ..Default::default()
    };
    let pair = |value: Any, request: Any| {
        let mut carried: Observation = value.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        carried.request = Some(request);
        (value, Any::pack(&carried))
    };
    let script = [
        pair(observation(1, 1), Any::pack(&describe)),
        pair(cancel(None), Any::pack(&cancel_request)),
        pair(observation(4, 0), Any::pack(&describe)),
        pair(cancel(Some(5)), Any::pack(&cancel_request)),
        pair(observation(2, 0), Any::pack(&describe)),
    ];
    let mut absent = Guest::new(runtime());
    let mut present = Guest::new(runtime());
    let judged: Vec<_> = script
        .into_iter()
        .map(|(value, carried)| {
            let response = judge(&mut present, carried);
            assert_eq!(response, judge(&mut absent, value));
            response
        })
        .collect();
    let [scheduled, canceled, closed, after, impossible] =
        judged.try_into().unwrap_or_else(|_| panic!("five judgments"));
    explained(scheduled, &[]);
    explained(canceled, &["Client::Cancel"]);
    explained(closed, &[]);
    assert_eq!(explanation(after).outcomes, ["Rejected(NotFound)"]);
    assert_eq!(unexplained(impossible), ["Client::Cancel"]);
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
    assert_eq!(exploration.report, include_str!("../../feature/activity/expected/report.snap.txt"));
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
    assert_eq!(rows, include_str!("../../feature/activity/expected/rows.snap.txt"));
}

#[test]
fn malformed_and_unsupported_observations_preserve_run_and_frozen_evidence() {
    let malformed = protocol::RefusalCode::MalformedRequest;
    let unsupported = protocol::RefusalCode::Unsupported;
    let missing_info = any(
        "umpire.realization.v1.Observation",
        &Observation {
            method: DESCRIBE.into(),
            result: Some(ObservationResult::Response(any(
                "temporal.api.workflowservice.v1.DescribeActivityExecutionResponse",
                &DescribeActivityExecutionResponse::default(),
            ))),
            request: None,
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
                type_url: "type.googleapis.com/umpire.realization.v1.Observation".into(),
                value: vec![255],
            },
            malformed,
        ),
        (
            any(
                "umpire.realization.v1.Observation",
                &Observation { method: "wrong".into(), result: None, request: None },
            ),
            unsupported,
        ),
        (
            any(
                "umpire.realization.v1.Observation",
                &Observation { method: DESCRIBE.into(), result: None, request: None },
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
    explained(judge(&mut guest, cancel(None)), &["Client::Cancel"]);
    let first = unexplained(judge(&mut guest, observation(2, 0)));
    for (value, code) in cases {
        refused(judge(&mut guest, value), code);
    }
    assert_eq!(unexplained(judge(&mut guest, observation(1, 1))), first);
}

#[test]
fn rejected_publication_rolls_back_delivery_terminal_freeze_and_reset() {
    let mut guest = Guest::new(runtime());
    let deliver = protocol::JudgeRequest { observation: Some(observation(1, 2)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &deliver, |_| false));
    explained(judge(&mut guest, observation(1, 1)), &[]);
    explained(judge(&mut guest, observation(1, 2)), &["Worker::Poll(1)"]);
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
fn server_observations_evidence_delivery_and_fast_terminal_paths() {
    for (terminal, action) in
        [(2, "Worker::RespondCompleted"), (3, "Worker::RespondFailed { retryable: false }")]
    {
        let mut guest = Guest::new(runtime());
        explained(judge(&mut guest, observation(terminal, 0)), &["Worker::Poll(1)", action]);
        explained(judge(&mut guest, observation(terminal, 0)), &[]);
        assert_eq!(
            unexplained(judge(&mut guest, observation(5 - terminal, 0))),
            ["Client::Cancel"]
        );
        call(&mut guest, 3, &protocol::ResetRequest::default());
        explained(judge(&mut guest, observation(1, 1)), &[]);
    }
}

#[test]
fn temporal_has_no_live_answer_policy() {
    let mut guest = Guest::new(runtime());
    refused(
        call(&mut guest, 5, &protocol::AnswerRequest { callback: Some(Any::default()) }),
        protocol::RefusalCode::Unsupported,
    );
    explained(judge(&mut guest, observation(1, 1)), &[]);
}

#[test]
fn skipped_retry_projection_never_accepts_a_path_that_stops_before_the_observed_phase() {
    for status in [4, 6] {
        let mut observed: Observation =
            observation(status, 0).unpack().unwrap_or_else(|e| panic!("{e:?}"));
        let Some(ObservationResult::Response(response)) = observed.result else {
            panic!("response")
        };
        let mut response: DescribeActivityExecutionResponse =
            response.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        response.info.as_mut().unwrap_or_else(|| panic!("info")).attempt = 2;
        observed.result = Some(ObservationResult::Response(Any::pack(&response)));
        let state = activity::Model { phase: activity::Phase::Scheduled, attempt: 1 };
        let Projection::Candidates(paths) = Realization::default()
            .observe(&Any::pack(&observed), &state)
            .unwrap_or_else(|e| panic!("{e:?}"))
        else {
            panic!("candidates")
        };
        assert_eq!(paths.len(), 0);
    }
}
