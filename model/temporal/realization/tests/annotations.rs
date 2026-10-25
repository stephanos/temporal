//! Typed transport annotation controls preserve the independent Model run.
use umpire_runtime::protocol::{Method, RefusalCode};
use umpire_runtime::testing::GuestExt;

use umpire_runtime::AnyMessage as _;
use umpire_runtime::annotations::AnnotationRequest;
use umpire_runtime::annotations::AnnotationResponse;
use umpire_runtime::annotations::ProbeResponse;

mod common;

use prost::Message;
use umpire_runtime::{Dispatch, Guest, prost_types::Any, protocol};
use umpire_temporal::annotation_runtime;
use umpire_temporal_api::{
    temporal::api::{
        activity::v1::ActivityExecutionInfo, workflowservice::v1::DescribeActivityExecutionResponse,
    },
    umpire::temporal::v1::ProbeRequest,
};

#[test]
fn annotation_route_admits_empty_successful_worker_polls() {
    let mut guest = Guest::new(annotation_runtime());
    let output = annotate(&mut guest, "PollActivityTaskQueue", true, &[]);
    assert_eq!(output.message, []);
    assert_eq!(output.violations, Vec::<String>::new());
    assert_eq!(output.out_of_scope, "");
}

#[test]
fn generated_dispatch_selects_declared_directions_and_exact_service_paths() {
    let mut guest = Guest::new(annotation_runtime());
    for prefix in ["", "/temporal.api.workflowservice.v1.WorkflowService/"] {
        for response in [false, true] {
            let output =
                annotate(&mut guest, &format!("{prefix}PollActivityTaskQueue"), response, &[]);
            assert_eq!(output.violations.is_empty(), response);
            assert_eq!(output.out_of_scope, "");
            let opaque = [255, 1, 2, 3];
            let outside =
                annotate(&mut guest, &format!("{prefix}GetSystemInfo"), response, &opaque);
            assert_eq!(outside.message, opaque);
            assert_eq!(outside.violations, [] as [String; 0]);
            assert_eq!(outside.out_of_scope, "Cluster and service information");
        }
    }
}

fn request(method: &str, response: bool, bytes: &[u8]) -> protocol::AnnotateRequest {
    protocol::AnnotateRequest {
        input: Some(Any::pack(&AnnotationRequest {
            method: method.into(),
            response,
            message: bytes.into(),
            seed: 17,
        })),
    }
}

fn annotate(
    guest: &mut Guest<impl Dispatch>,
    method: &str,
    response: bool,
    bytes: &[u8],
) -> AnnotationResponse {
    let value = GuestExt::call(guest, Method::Annotate, &request(method, response, bytes));
    let Some(protocol::response::Outcome::Annotate(value)) = value.outcome else {
        panic!("expected annotation response: {value:?}");
    };
    value
        .output
        .unwrap_or_else(|| panic!("missing Any"))
        .unpack()
        .unwrap_or_else(|e| panic!("{e:?}"))
}

#[test]
fn annotation_findings_preserve_server_bytes_and_do_not_change_frozen_judgment() {
    let mut guest = Guest::new(annotation_runtime());
    GuestExt::call(
        &mut guest,
        Method::Judge,
        &protocol::JudgeRequest { observation: Some(Any::pack(&common::start(3))) },
    );
    GuestExt::call(
        &mut guest,
        Method::Judge,
        &protocol::JudgeRequest { observation: Some(Any::pack(&common::describe(1, 2, 0))) },
    );
    let observation =
        protocol::JudgeRequest { observation: Some(Any::pack(&common::describe(1, 3, 0))) };
    let frozen = GuestExt::call(&mut guest, Method::Judge, &observation);
    assert!(
        matches!(&frozen.outcome, Some(protocol::response::Outcome::Judge(value)) if matches!(&value.judgment, Some(protocol::judge_response::Judgment::Unexplained(_))))
    );
    let invalid_response = DescribeActivityExecutionResponse {
        info: Some(ActivityExecutionInfo { status: 999, ..Default::default() }),
        ..Default::default()
    }
    .encode_to_vec();
    let output = annotate(&mut guest, "DescribeActivityExecution", true, &invalid_response);
    assert!(output.message.eq(&invalid_response));
    assert_ne!(output.violations, [] as [String; 0]);
    annotate(&mut guest, "PollActivityTaskQueue", true, &[]);
    let probes = GuestExt::call(
        &mut guest,
        Method::Annotate,
        &protocol::AnnotateRequest { input: Some(Any::pack(&ProbeRequest::default())) },
    );
    let Some(protocol::response::Outcome::Annotate(probes)) = probes.outcome else {
        panic!("expected probe inventory")
    };
    let inventory: ProbeResponse = probes
        .output
        .unwrap_or_else(|| panic!("missing inventory"))
        .unpack()
        .unwrap_or_else(|error| panic!("{error:?}"));
    assert_ne!(inventory.cases.len(), 0);
    assert_eq!(GuestExt::call(&mut guest, Method::Judge, &observation), frozen);
    let bytes = request("PollActivityTaskQueue", true, &[]).encode_to_vec();
    assert!(!guest.dispatch_to(7, &bytes, |_| false));
    assert_eq!(GuestExt::call(&mut guest, Method::Judge, &observation), frozen);
    guest.reset(None);
    annotate(&mut guest, "PollActivityTaskQueue", true, &[]);
}

#[test]
fn annotation_envelope_and_unknown_traffic_refuse_without_payload_diagnostics() {
    let mut guest = Guest::new(annotation_runtime());
    let cases = [
        (protocol::AnnotateRequest::default(), RefusalCode::MalformedRequest),
        (protocol::AnnotateRequest { input: Some(Any::default()) }, RefusalCode::MalformedRequest),
        (
            protocol::AnnotateRequest { input: Some(Any::pack(&AnnotationResponse::default())) },
            RefusalCode::Unsupported,
        ),
        (
            protocol::AnnotateRequest {
                input: Some(Any {
                    type_url: "type.googleapis.com/umpire.annotations.v1.AnnotationRequest".into(),
                    value: vec![255],
                }),
            },
            RefusalCode::MalformedRequest,
        ),
        (request("UnknownRpc", false, b"private-task-token"), RefusalCode::Unsupported),
        (request("/other.Service/PollActivityTaskQueue", false, &[]), RefusalCode::Unsupported),
        (
            request(
                "temporal.api.workflowservice.v1.WorkflowService.PollActivityTaskQueue",
                false,
                &[],
            ),
            RefusalCode::Unsupported,
        ),
        (
            request(
                "temporal.api.workflowservice.v1.WorkflowService/PollActivityTaskQueue",
                false,
                &[],
            ),
            RefusalCode::Unsupported,
        ),
        (
            request("/temporal.api.workflowservice.v1.WorkflowService/UnknownRpc", false, &[]),
            RefusalCode::Unsupported,
        ),
        (
            request(
                "/temporal.api.workflowservice.v1.WorkflowService/PollActivityTaskQueue/extra",
                false,
                &[],
            ),
            RefusalCode::Unsupported,
        ),
        (request("PollActivityTaskQueue", true, &[255]), RefusalCode::MalformedRequest),
    ];
    for (request, code) in cases {
        let response = GuestExt::call(&mut guest, Method::Annotate, &request);
        let Some(protocol::response::Outcome::Refusal(refusal)) = response.outcome else {
            panic!("expected refusal")
        };
        assert_eq!(refusal.code(), code);
        assert!(!refusal.message.contains("private-task-token"));
    }
    annotate(&mut guest, "PollActivityTaskQueue", true, &[]);
}
