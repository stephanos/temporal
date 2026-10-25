//! Native realization controls against the independently fixed activity Model.
use umpire_runtime::protocol::{Method, RefusalCode};
use umpire_runtime::testing::{GuestExt, ResponseExt};

use umpire_runtime::AnyMessage as _;
use umpire_runtime::realization::Error;
use umpire_runtime::realization::Observation;
use umpire_runtime::realization::observation::Result as ObservationResult;

use prost::Message;
use umpire_dsl::{Machine, Outcome as StepOutcome, Rejection};
use umpire_runtime::{
    Guest, Projection, Realize,
    prost_types::Any,
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
    explained(guest.judge(cancel), &["Client::Cancel"]);
    explained(guest.judge(observation(4, 0)), &[]);
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

fn explained(response: protocol::Response, steps: &[&str]) {
    let value = response.explanation();
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
            explained(guest.judge(cancel(None)), &[step]);
        } else {
            explained(guest.judge(observation(1, 2)), &["Worker::Poll(1)"]);
            explained(guest.judge(observation(terminal, 0)), &[step]);
        }
        for _ in 0..2 {
            let result = guest.judge(cancel(Some(5))).explanation();
            assert_eq!(result.steps, ["Client::Cancel"]);
            assert_eq!(result.outcomes, ["Rejected(NotFound)"]);
            explained(guest.judge(observation(terminal, 0)), &[]);
        }
        assert_eq!(guest.judge(cancel(None)).unexplained(), ["Client::Cancel"]);
        assert_eq!(guest.judge(cancel(Some(5))).unexplained(), ["Client::Cancel"]);
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
        let before = activity::Model {
            phase,
            attempt: 1,
            cancel_requested: false,
            schedule_to_start_remaining: None,
            start_to_close_remaining: None,
            schedule_to_close_remaining: None,
            heartbeat_remaining: None,
        };
        let Projection::Candidates(paths) = Realization::default()
            .observe(&cancel(Some(5)), &before)
            .unwrap_or_else(|error| panic!("{error:?}"))
        else {
            panic!("cancel must replay")
        };
        assert_eq!(paths.len(), 1);
        assert_eq!(paths[0].path.len(), 1);
        let (outcome, _, after) = activity::World::new(3, None, None, None, None, false)
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
                explained(guest.judge(cancel(None)), &["Client::Cancel"]);
                vec!["Client::Cancel"]
            } else if phase == activity::Phase::Started {
                explained(guest.judge(observation(1, 2)), &["Worker::Poll(1)"]);
                vec![
                    "Client::Cancel",
                    "Worker::Poll(2)",
                    "Worker::Poll(3)",
                    "Worker::RespondCompleted",
                    "Worker::RespondFailed { retryable: false }",
                    "Worker::RespondFailed { retryable: true }",
                    "Worker::Heartbeat",
                    "Worker::RespondCanceled(2)",
                    "Worker::RespondCanceled(3)",
                ]
            } else {
                vec![
                    "Client::Cancel",
                    "Worker::Poll(1)",
                    "Worker::Poll(2)",
                    "Worker::Poll(3)",
                    "Worker::RespondCanceled(2)",
                    "Worker::RespondCanceled(3)",
                ]
            };
            assert_eq!(guest.judge(cancel(Some(code))).unexplained(), expected);
            assert_eq!(guest.judge(cancel(None)).unexplained(), expected);
            guest.reset(None);
            explained(guest.judge(cancel(None)), &["Client::Cancel"]);
        }
    }
}

#[test]
fn cancel_envelope_refusals_preserve_state_and_frozen_evidence() {
    let malformed = RefusalCode::MalformedRequest;
    let cases = [
        (Observation { method: "unknown".into(), result: Some(ObservationResult::Error(Error { code: 5, message: String::new() })), request: None }, RefusalCode::Unsupported),
        (Observation { method: String::new(), result: Some(ObservationResult::Error(Error::default())), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: None, request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::default())), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any::pack(&DescribeActivityExecutionResponse::default()))), request: None }, malformed),
        (Observation { method: CANCEL.into(), result: Some(ObservationResult::Response(Any { type_url: "type.googleapis.com/temporal.api.workflowservice.v1.RequestCancelActivityExecutionResponse".into(), value: vec![255] })), request: None }, malformed),
    ];
    let mut guest = Guest::new(runtime());
    for (value, code) in &cases {
        guest.judge(Any::pack(value)).refusal(*code);
        explained(guest.judge(observation(1, 1)), &[]);
    }
    let frozen = guest.judge(cancel(Some(5))).unexplained();
    for (value, code) in cases {
        guest.judge(Any::pack(&value)).refusal(code);
    }
    assert_eq!(guest.judge(cancel(None)).unexplained(), frozen);
}

#[test]
fn cancel_publication_failure_preserves_state_and_freeze() {
    let mut guest = Guest::new(runtime());
    for value in [cancel(None), cancel(Some(5))] {
        let request = protocol::JudgeRequest { observation: Some(value) }.encode_to_vec();
        assert!(!guest.dispatch_to(4, &request, |_| false));
        explained(guest.judge(observation(1, 1)), &[]);
    }
    explained(guest.judge(cancel(None)), &["Client::Cancel"]);
    let request = protocol::JudgeRequest { observation: Some(cancel(Some(5))) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &request, |_| false));
    let result = guest.judge(cancel(Some(5))).explanation();
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
            let response = present.judge(carried);
            assert_eq!(response, absent.judge(value));
            response
        })
        .collect();
    let [scheduled, canceled, closed, after, impossible] =
        judged.try_into().unwrap_or_else(|_| panic!("five judgments"));
    explained(scheduled, &[]);
    explained(canceled, &["Client::Cancel"]);
    explained(closed, &[]);
    assert_eq!(after.explanation().outcomes, ["Rejected(NotFound)"]);
    assert_eq!(impossible.unexplained(), ["Client::Cancel"]);
}

#[test]
fn named_guest_adapter_preserves_the_independent_report_and_ordered_rows() {
    let world = activity::World::new(
        3,
        Some(activity::Ticks(1)),
        Some(activity::Ticks(1)),
        None,
        None,
        false,
    );
    let mut guest = Guest::new(umpire_runtime::Runtime::new(world, Realization::default()));
    let Some(Outcome::Explore(exploration)) =
        GuestExt::call(&mut guest, Method::Explore, &protocol::ExploreRequest { budget: 64 })
            .outcome
    else {
        panic!("missing exploration")
    };
    let mut settings = insta::Settings::clone_current();
    settings.set_snapshot_path("../../feature/activity/expected");
    settings.set_prepend_module_to_snapshot(false);
    settings.set_omit_expression(true);
    let _guard = settings.bind_to_scope();
    insta::assert_snapshot!("report.txt", exploration.report);
    let mut rows = String::new();
    for state_id in 0..exploration.reached {
        let Some(Outcome::Rows(value)) =
            GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest { state_id }).outcome
        else {
            panic!("missing rows")
        };
        for row in value.rows {
            rows.push_str(&row);
        }
    }
    insta::assert_snapshot!("rows.txt", rows);
}

#[test]
fn malformed_and_unsupported_observations_preserve_run_and_frozen_evidence() {
    let malformed = RefusalCode::MalformedRequest;
    let unsupported = RefusalCode::Unsupported;
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
        guest.judge(value.clone()).refusal(*code);
        explained(guest.judge(observation(1, 1)), &[]);
    }
    explained(guest.judge(cancel(None)), &["Client::Cancel"]);
    let first = guest.judge(observation(2, 0)).unexplained();
    for (value, code) in cases {
        guest.judge(value).refusal(code);
    }
    assert_eq!(guest.judge(observation(1, 1)).unexplained(), first);
}

#[test]
fn rejected_publication_rolls_back_delivery_terminal_freeze_and_reset() {
    let mut guest = Guest::new(runtime());
    let deliver = protocol::JudgeRequest { observation: Some(observation(1, 2)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &deliver, |_| false));
    explained(guest.judge(observation(1, 1)), &[]);
    explained(guest.judge(observation(1, 2)), &["Worker::Poll(1)"]);
    let close = protocol::JudgeRequest { observation: Some(observation(2, 0)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &close, |_| false));
    explained(guest.judge(observation(1, 2)), &[]);
    let stale = protocol::JudgeRequest { observation: Some(observation(1, 1)) }.encode_to_vec();
    assert!(!guest.dispatch_to(4, &stale, |_| false));
    explained(guest.judge(observation(2, 0)), &["Worker::RespondCompleted"]);
    assert!(!guest.dispatch_to(3, &[], |_| false));
    explained(guest.judge(observation(2, 0)), &[]);
}

#[test]
fn server_observations_evidence_delivery_and_fast_terminal_paths() {
    for (terminal, action) in
        [(2, "Worker::RespondCompleted"), (3, "Worker::RespondFailed { retryable: false }")]
    {
        let mut guest = Guest::new(runtime());
        explained(guest.judge(observation(terminal, 0)), &["Worker::Poll(1)", action]);
        explained(guest.judge(observation(terminal, 0)), &[]);
        assert_eq!(guest.judge(observation(5 - terminal, 0)).unexplained(), ["Client::Cancel"]);
        guest.reset(None);
        explained(guest.judge(observation(1, 1)), &[]);
    }
}

#[test]
fn temporal_has_no_live_answer_policy() {
    let mut guest = Guest::new(runtime());
    GuestExt::call(
        &mut guest,
        Method::Answer,
        &protocol::AnswerRequest { callback: Some(Any::default()) },
    )
    .refusal(RefusalCode::Unsupported);
    explained(guest.judge(observation(1, 1)), &[]);
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
        let state = activity::Model {
            phase: activity::Phase::Scheduled,
            attempt: 1,
            cancel_requested: false,
            schedule_to_start_remaining: None,
            start_to_close_remaining: None,
            schedule_to_close_remaining: None,
            heartbeat_remaining: None,
        };
        let Projection::Candidates(paths) = Realization::default()
            .observe(&Any::pack(&observed), &state)
            .unwrap_or_else(|e| panic!("{e:?}"))
        else {
            panic!("candidates")
        };
        assert_eq!(paths.len(), 0);
    }
}
