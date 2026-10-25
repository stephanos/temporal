//! Actual-request admission and publication controls for the Temporal facade.
use umpire_runtime::protocol::{Method, RefusalCode};
use umpire_runtime::testing::{GuestExt, ResponseExt};

use prost::Message;
use std::panic::AssertUnwindSafe;
use umpire_runtime::AnyMessage as _;
use umpire_runtime::{
    Dispatch, Guest,
    prost_types::Any,
    protocol::{self, response::Outcome},
    realization::{Observation, observation::Result as ResultValue},
};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::temporal::api::workflowservice::v1::{
    DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
    StartActivityExecutionRequest, StartActivityExecutionResponse,
};

mod common;
use common::{RUN, describe, start};

fn explained(response: protocol::Response) {
    assert_eq!(response.explanation().steps, [] as [String; 0]);
}

#[test]
fn start_selects_world_and_original_replay_never_resets_progress() {
    let mut guest = Guest::new(annotation_runtime());
    guest.judge(Any::pack(&describe(1, 1, 1))).refusal(RefusalCode::State);
    let captured = start(1);
    explained(guest.judge(Any::pack(&captured)));
    let Some(Outcome::Explore(explore)) =
        GuestExt::call(&mut guest, Method::Explore, &protocol::ExploreRequest { budget: 64 })
            .outcome
    else {
        panic!("explore")
    };
    assert_eq!(explore.reached, 5);
    advanced(guest.judge(Any::pack(&describe(1, 1, 2))), &["Worker::Poll(1)"]);
    explained(guest.judge(Any::pack(&captured)));
    explained(guest.judge(Any::pack(&describe(1, 1, 2))));
    let mut replay = captured.clone();
    replay.result = Some(ResultValue::Response(Any::pack(&StartActivityExecutionResponse {
        run_id: RUN.into(),
        started: false,
        ..Default::default()
    })));
    explained(guest.judge(Any::pack(&replay)));
    guest.reset(None);
    explained(guest.judge(Any::pack(&replay)));
    explained(guest.judge(Any::pack(&describe(1, 1, 1))));
}

#[test]
fn admission_and_first_domain_freeze_publish_atomically() {
    for attempts in [1, 4] {
        for panic in [false, true] {
            let mut guest = Guest::new(annotation_runtime());
            GuestExt::call(&mut guest, Method::Explore, &protocol::ExploreRequest { budget: 64 });
            let before =
                GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest { state_id: 0 });
            let captured = start(attempts);
            let bytes =
                protocol::JudgeRequest { observation: Some(Any::pack(&captured)) }.encode_to_vec();
            let result = std::panic::catch_unwind(AssertUnwindSafe(|| {
                guest.dispatch_to(4, &bytes, |_| {
                    assert!(!panic, "publication failed");
                    false
                })
            }));
            assert_eq!(result.is_err(), panic);
            assert_eq!(
                before,
                GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest { state_id: 0 })
            );
            guest.judge(Any::pack(&describe(1, 1, 1))).refusal(RefusalCode::State);
            if attempts == 1 {
                explained(guest.judge(Any::pack(&captured)));
                GuestExt::call(&mut guest, Method::Rows, &protocol::RowsRequest { state_id: 0 })
                    .refusal(RefusalCode::NotExplored);
            } else {
                let first = guest.judge(Any::pack(&captured)).refusal(RefusalCode::Unsupported);
                assert!(first.message.contains("maximum_attempts"));
                assert_eq!(
                    before,
                    GuestExt::call(
                        &mut guest,
                        Method::Rows,
                        &protocol::RowsRequest { state_id: 0 }
                    )
                );
                assert_eq!(
                    first,
                    guest.judge(Any::pack(&captured)).refusal(RefusalCode::Unsupported)
                );
                guest.reset(None);
                guest.judge(Any::pack(&describe(1, 1, 1))).refusal(RefusalCode::State);
                guest.judge(Any::pack(&start(1))).refusal(RefusalCode::State);
            }
        }
    }
}

#[test]
fn identity_and_result_validation_precede_domains_and_never_replace_first_freeze() {
    let request = |value: &mut Observation, edit: fn(&mut StartActivityExecutionRequest)| {
        let mut message: StartActivityExecutionRequest = value
            .request
            .as_ref()
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        edit(&mut message);
        value.request = Some(Any::pack(&message));
    };
    let response = |value: &mut Observation, edit: fn(&mut StartActivityExecutionResponse)| {
        let Some(ResultValue::Response(raw)) = &value.result else { panic!("response") };
        let mut message: StartActivityExecutionResponse =
            raw.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        edit(&mut message);
        value.result = Some(ResultValue::Response(Any::pack(&message)));
    };
    let malformed = malformed_starts(request, response);
    for initial in [false, true] {
        let mut guest = Guest::new(annotation_runtime());
        let freeze = if initial {
            Some(guest.judge(Any::pack(&start(4))).refusal(RefusalCode::Unsupported))
        } else {
            None
        };
        for value in &malformed {
            let response = guest.judge(Any::pack(value));
            let Some(Outcome::Refusal(refusal)) = response.outcome else {
                panic!("invalid evidence admitted")
            };
            assert!(
                !refusal.message.contains("maximum_attempts"),
                "identity/result validation must precede domains"
            );
            assert!(!format!("{refusal:?}").contains("private-error"));
            if let Some(first) = &freeze {
                assert_eq!(
                    *first,
                    guest.judge(Any::pack(&start(4))).refusal(RefusalCode::Unsupported)
                );
            }
        }
        if !initial {
            explained(guest.judge(Any::pack(&start(1))));
        }
    }
    let mut guest = Guest::new(annotation_runtime());
    let captured = start(2);
    explained(guest.judge(Any::pack(&captured)));
    for edit in [
        |r: &mut StartActivityExecutionRequest| r.namespace = "crossed".into(),
        |r: &mut StartActivityExecutionRequest| r.activity_id = "crossed".into(),
        |r: &mut StartActivityExecutionRequest| {
            r.request_id = "33333333-3333-4333-8333-333333333333".into();
        },
        |r: &mut StartActivityExecutionRequest| r.identity = "private-changed-bytes".into(),
    ] {
        let mut crossed = captured.clone();
        request(&mut crossed, edit);
        guest.judge(Any::pack(&crossed)).refusal(RefusalCode::State);
    }
    let mut unknown = captured.clone();
    unknown
        .request
        .as_mut()
        .unwrap_or_else(|| panic!("request"))
        .value
        .extend_from_slice(&[0xf8, 0x07, 0x01]);
    guest.judge(Any::pack(&unknown)).refusal(RefusalCode::State);
    let mut crossed = captured.clone();
    response(&mut crossed, |r| r.run_id = "33333333-3333-4333-8333-333333333333".into());
    guest.judge(Any::pack(&crossed)).refusal(RefusalCode::State);
    explained(guest.judge(Any::pack(&describe(1, 1, 1))));
}

fn malformed_starts(
    request: impl Fn(&mut Observation, fn(&mut StartActivityExecutionRequest)),
    response: impl Fn(&mut Observation, fn(&mut StartActivityExecutionResponse)),
) -> Vec<Observation> {
    use umpire_temporal_api::temporal::api::common::v1::{Link, link};
    let mut malformed = Vec::new();
    for field in 0..4 {
        let mut value = start(4);
        match field {
            0 => request(&mut value, |r| r.namespace.clear()),
            1 => request(&mut value, |r| r.activity_id.clear()),
            2 => request(&mut value, |r| r.request_id.clear()),
            3 => response(&mut value, |r| r.run_id.clear()),
            _ => unreachable!(),
        }
        malformed.push(value);
    }
    let mut value = start(4);
    value.request = None;
    malformed.push(value);
    let mut value = start(4);
    value.result = None;
    malformed.push(value);
    let mut value = start(4);
    value.result = Some(ResultValue::Error(umpire_runtime::realization::Error {
        code: 3,
        message: "private-error".into(),
    }));
    malformed.push(value);
    let mut value = start(4);
    response(&mut value, |r| r.started = false);
    malformed.push(value);
    let mut value = start(4);
    response(&mut value, |r| {
        r.link = Some(Link {
            variant: Some(link::Variant::Activity(link::Activity {
                namespace: "crossed".into(),
                activity_id: "activity".into(),
                run_id: RUN.into(),
            })),
        });
    });
    malformed.push(value);
    let mut value = start(4);
    value.request.as_mut().unwrap_or_else(|| panic!("request")).type_url =
        Any::pack(&DescribeActivityExecutionRequest::default()).type_url;
    malformed.push(value);
    let mut value = start(4);
    value.request.as_mut().unwrap_or_else(|| panic!("request")).value = vec![255];
    malformed.push(value);
    let mut value = start(4);
    value.result =
        Some(ResultValue::Response(Any::pack(&DescribeActivityExecutionResponse::default())));
    malformed.push(value);
    let mut value = start(4);
    response(&mut value, |r| r.link = Some(Link { variant: None }));
    malformed.push(value);
    malformed
}

#[test]
fn bound_rpc_identities_are_required_before_projection() {
    let mut guest = Guest::new(annotation_runtime());
    explained(guest.judge(Any::pack(&start(3))));
    for field in 0..8 {
        let mut value = describe(1, 1, 1);
        let mut request: DescribeActivityExecutionRequest = value
            .request
            .as_ref()
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        let Some(ResultValue::Response(raw)) = &value.result else { panic!("result") };
        let mut response: DescribeActivityExecutionResponse =
            raw.unpack().unwrap_or_else(|e| panic!("{e:?}"));
        match field {
            0 => request.namespace = "crossed".into(),
            1 => request.activity_id = "crossed".into(),
            2 => request.run_id.clear(),
            3 => response.run_id = "crossed".into(),
            4 => {
                response.info.as_mut().unwrap_or_else(|| panic!("info")).activity_id =
                    "crossed".into();
            }
            5 => response.info.as_mut().unwrap_or_else(|| panic!("info")).run_id = "crossed".into(),
            6 => response.info.as_mut().unwrap_or_else(|| panic!("info")).attempt = 0,
            7 => response.info.as_mut().unwrap_or_else(|| panic!("info")).attempt = 4,
            _ => unreachable!(),
        }
        value.request = Some(Any::pack(&request));
        value.result = Some(ResultValue::Response(Any::pack(&response)));
        guest.judge(Any::pack(&value)).refusal(RefusalCode::State);
        explained(guest.judge(Any::pack(&describe(1, 1, 1))));
    }
    for field in 0..3 {
        use umpire_temporal_api::temporal::api::workflowservice::v1::{
            RequestCancelActivityExecutionRequest, RequestCancelActivityExecutionResponse,
        };
        let mut request = RequestCancelActivityExecutionRequest {
            namespace: "model-slice".into(),
            activity_id: "activity".into(),
            run_id: RUN.into(),
            ..Default::default()
        };
        match field {
            0 => request.namespace = "crossed".into(),
            1 => request.activity_id = "crossed".into(),
            2 => request.run_id.clear(),
            _ => unreachable!(),
        }
        let value = Observation {
            method:
                "/temporal.api.workflowservice.v1.WorkflowService/RequestCancelActivityExecution"
                    .into(),
            request: Some(Any::pack(&request)),
            result: Some(ResultValue::Response(Any::pack(
                &RequestCancelActivityExecutionResponse {},
            ))),
        };
        guest.judge(Any::pack(&value)).refusal(RefusalCode::State);
        explained(guest.judge(Any::pack(&describe(1, 1, 1))));
    }
}

#[test]
fn actual_retry_describe_advances_attempt_without_worker_callbacks() {
    let mut guest = Guest::new(annotation_runtime());
    explained(guest.judge(Any::pack(&start(2))));
    advanced(
        guest.judge(Any::pack(&describe(2, 1, 1))),
        &["Worker::Poll(1)", "Worker::RespondFailed { retryable: true }"],
    );
    advanced(guest.judge(Any::pack(&describe(2, 1, 2))), &["Worker::Poll(2)"]);
}

#[path = "admission/publication.rs"]
mod publication;

#[test]
fn only_explicit_attempt_domain_is_admitted_while_timeouts_are_retained_exactly() {
    for attempts in [None, Some(-1), Some(0), Some(4), Some(i32::MAX)] {
        let mut captured = start(1);
        let mut request: StartActivityExecutionRequest = captured
            .request
            .as_ref()
            .unwrap_or_else(|| panic!("request"))
            .unpack()
            .unwrap_or_else(|e| panic!("{e:?}"));
        request.retry_policy = attempts.map(|maximum_attempts| {
            umpire_temporal_api::temporal::api::common::v1::RetryPolicy {
                maximum_attempts,
                ..Default::default()
            }
        });
        captured.request = Some(Any::pack(&request));
        let mut guest = Guest::new(annotation_runtime());
        let refusal = guest.judge(Any::pack(&captured)).refusal(RefusalCode::Unsupported);
        assert!(refusal.message.contains("maximum_attempts"));
        if attempts == Some(4) {
            let annotate = protocol::AnnotateRequest {
                input: Some(Any::pack(&umpire_runtime::annotations::AnnotationRequest {
                    method: "PollActivityTaskQueue".into(),
                    response: true,
                    ..Default::default()
                })),
            };
            assert!(matches!(
                GuestExt::call(&mut guest, Method::Annotate, &annotate).outcome,
                Some(Outcome::Annotate(_))
            ));
            let probe = protocol::AnnotateRequest {
                input: Some(Any::pack(
                    &umpire_temporal_api::umpire::temporal::v1::ProbeRequest::default(),
                )),
            };
            assert!(matches!(
                GuestExt::call(&mut guest, Method::Annotate, &probe).outcome,
                Some(Outcome::Annotate(_))
            ));
            assert_eq!(
                refusal,
                guest.judge(Any::pack(&captured)).refusal(RefusalCode::Unsupported)
            );
        }
    }
    let mut captured = start(1);
    let mut request: StartActivityExecutionRequest = captured
        .request
        .as_ref()
        .unwrap_or_else(|| panic!("request"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"));
    let duration = umpire_runtime::prost_types::Duration { seconds: 123, nanos: 456_789 };
    request.schedule_to_start_timeout = Some(duration);
    request.schedule_to_close_timeout = Some(duration);
    request.heartbeat_timeout =
        Some(umpire_runtime::prost_types::Duration { seconds: 1, nanos: 456_789 });
    captured.request = Some(Any::pack(&request));
    let mut guest = Guest::new(annotation_runtime());
    explained(guest.judge(Any::pack(&captured)));
    let original = captured.clone();
    request.heartbeat_timeout.as_mut().unwrap_or_else(|| panic!("heartbeat")).nanos += 1;
    captured.request = Some(Any::pack(&request));
    guest.judge(Any::pack(&captured)).refusal(RefusalCode::State);
    explained(guest.judge(Any::pack(&original)));
    let Some(Outcome::Explore(result)) =
        GuestExt::call(&mut guest, Method::Explore, &protocol::ExploreRequest { budget: 64 })
            .outcome
    else {
        panic!("explore")
    };
    assert_eq!(result.reached, 5);
}

#[test]
fn selected_world_cap_rejects_describe_attempts_without_freezing_or_advancing() {
    let mut guest = Guest::new(annotation_runtime());
    explained(guest.judge(Any::pack(&start(1))));
    for attempt in [2, 3] {
        guest.judge(Any::pack(&describe(attempt, 1, 1))).refusal(RefusalCode::State);
        explained(guest.judge(Any::pack(&describe(1, 1, 1))));
    }
    advanced(guest.judge(Any::pack(&describe(1, 1, 2))), &["Worker::Poll(1)"]);
    for attempt in [2, 3] {
        guest.judge(Any::pack(&describe(attempt, 1, 1))).refusal(RefusalCode::State);
        explained(guest.judge(Any::pack(&describe(1, 1, 2))));
    }
}

fn advanced(response: protocol::Response, steps: &[&str]) {
    let value = response.explanation();
    assert_eq!(value.steps, steps);
    assert_eq!(value.outcomes, vec!["Accepted"; steps.len()]);
}
