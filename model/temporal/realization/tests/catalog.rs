//! Full SDK wire annotation controls.

use prost::Message;
use umpire_temporal::activity_api::{AnnotationResult, annotate as annotate_wire};
use umpire_temporal_api::temporal::api::{
    activity::v1::ActivityExecutionInfo,
    common::v1::{ActivityType, Payload, Payloads, RetryPolicy},
    failure::v1::{ApplicationFailureInfo, Failure, ServerFailureInfo, failure},
    taskqueue::v1::TaskQueue,
    workflowservice::v1::{
        DescribeActivityExecutionRequest, DescribeActivityExecutionResponse,
        PollActivityExecutionRequest, PollActivityTaskQueueResponse,
        RespondActivityTaskCompletedRequest, RespondActivityTaskFailedRequest,
        StartActivityExecutionRequest,
    },
};
use umpire_wasm::google::protobuf::Duration;

fn annotate(
    method: &str,
    response: bool,
    bytes: &[u8],
    seed: u64,
) -> Result<AnnotationResult, Box<dyn std::error::Error>> {
    annotate_wire(method, response, bytes, seed).map_err(|refusal| refusal.message.into())
}

fn start() -> StartActivityExecutionRequest {
    StartActivityExecutionRequest {
        namespace: "external-namespace".into(),
        activity_id: "external-activity".into(),
        activity_type: Some(ActivityType { name: "ModelSlice".into() }),
        task_queue: Some(TaskQueue { name: "external-queue".into(), ..Default::default() }),
        start_to_close_timeout: Some(Duration { seconds: 20, nanos: 0 }),
        retry_policy: Some(RetryPolicy { maximum_attempts: 1, ..Default::default() }),
        input: Some(Payloads {
            payloads: vec![Payload {
                metadata: std::collections::BTreeMap::from([(
                    "encoding".into(),
                    b"binary/plain".to_vec(),
                )]),
                data: b"Complete".to_vec(),
                ..Default::default()
            }],
        }),
        ..Default::default()
    }
}

#[test]
fn fills_only_absent_request_ids_and_keeps_retry_bytes_stable()
-> Result<(), Box<dyn std::error::Error>> {
    let original = start();
    let result = annotate(
        "/temporal.api.workflowservice.v1.WorkflowService/StartActivityExecution",
        false,
        &original.encode_to_vec(),
        17,
    )?;
    assert!(result.violations.is_empty(), "{:?}", result.violations);
    let filled = StartActivityExecutionRequest::decode(result.bytes.as_slice())?;
    assert_eq!(filled.request_id.len(), 36);
    let mut expected = original;
    expected.request_id.clone_from(&filled.request_id);
    assert!(filled.eq(&expected));
    assert!(annotate("StartActivityExecution", false, &result.bytes, 99)?.bytes.eq(&result.bytes));
    let mut supplied = start();
    supplied.request_id = "server-accepts-non-uuid".into();
    let result = annotate("StartActivityExecution", false, &supplied.encode_to_vec(), 9)?;
    assert!(result.violations.is_empty(), "{:?}", result.violations);
    assert!(result.bytes.eq(&supplied.encode_to_vec()));
    supplied.request_id = "é".repeat(501);
    assert_ne!(
        annotate("StartActivityExecution", false, &supplied.encode_to_vec(), 9)?.violations,
        [] as [String; 0]
    );
    Ok(())
}

#[test]
fn enforces_start_shape_and_server_timeout_retry_normalization()
-> Result<(), Box<dyn std::error::Error>> {
    let mut request = start();
    request.activity_id.clear();
    assert_ne!(
        annotate("StartActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request = start();
    request.start_to_close_timeout = Some(Duration { seconds: -1, nanos: 0 });
    assert_ne!(
        annotate("StartActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request = start();
    request.start_to_close_timeout = None;
    assert_ne!(
        annotate("StartActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request = start();
    request.schedule_to_close_timeout = Some(Duration { seconds: 10, nanos: 0 });
    request.heartbeat_timeout = Some(Duration { seconds: 100, nanos: 0 });
    request.retry_policy = Some(RetryPolicy {
        maximum_attempts: 1,
        backoff_coefficient: -2.0,
        initial_interval: Some(Duration { seconds: -1, nanos: 0 }),
        ..Default::default()
    });
    assert_eq!(
        annotate("StartActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request.retry_policy.as_mut().ok_or("retry policy")?.maximum_attempts = 2;
    assert_ne!(
        annotate("StartActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    Ok(())
}

#[test]
fn response_findings_never_repair_bytes_and_empty_worker_polls_are_valid()
-> Result<(), Box<dyn std::error::Error>> {
    let poll = PollActivityTaskQueueResponse::default().encode_to_vec();
    assert_eq!(annotate("PollActivityTaskQueue", true, &poll, 1)?.violations, [] as [String; 0]);
    let response = DescribeActivityExecutionResponse {
        info: Some(ActivityExecutionInfo { status: 999, ..Default::default() }),
        ..Default::default()
    }
    .encode_to_vec();
    let result = annotate("DescribeActivityExecution", true, &response, 1)?;
    assert_ne!(result.violations, [] as [String; 0]);
    assert!(result.bytes.eq(&response));
    let malformed_duration = PollActivityTaskQueueResponse {
        start_to_close_timeout: Some(Duration { seconds: 1, nanos: 1_000_000_000 }),
        ..Default::default()
    }
    .encode_to_vec();
    let result = annotate("PollActivityTaskQueue", true, &malformed_duration, 1)?;
    assert!(result.violations.iter().any(|violation| violation.contains("duration.valid")));
    assert!(result.bytes.eq(&malformed_duration));
    Ok(())
}

#[test]
fn diagnostics_exclude_tokens_and_payloads_and_unknown_traffic_refuses()
-> Result<(), Box<dyn std::error::Error>> {
    let request = RespondActivityTaskCompletedRequest {
        task_token: b"private-task-token".to_vec(),
        identity: "é".repeat(501),
        ..Default::default()
    };
    let result = annotate("RespondActivityTaskCompleted", false, &request.encode_to_vec(), 1)?;
    assert_ne!(result.violations, [] as [String; 0]);
    assert!(!format!("{:?}", result.violations).contains("private-task-token"));
    assert!(annotate("UnknownRpc", false, &[], 1).is_err());
    assert!(annotate("/another.Service/StartActivityExecution", false, &[], 1).is_err());
    let infrastructure = annotate("GetSystemInfo", false, &[], 1)?;
    assert!(infrastructure.out_of_scope.is_some_and(|reason| reason.contains("capabilit")));
    Ok(())
}

#[test]
fn optional_run_ids_follow_server_uuid_forms_and_long_poll_dependency()
-> Result<(), Box<dyn std::error::Error>> {
    for run_id in [
        "",
        "a13cd9da-521c-48d4-ab9c-6ab174876f25",
        "a13cd9da521c48d4ab9c6ab174876f25",
        "{a13cd9da-521c-48d4-ab9c-6ab174876f25}",
        "urn:uuid:a13cd9da-521c-48d4-ab9c-6ab174876f25",
        "invalid",
    ] {
        let request = PollActivityExecutionRequest {
            namespace: "external-namespace".into(),
            activity_id: "activity".into(),
            run_id: run_id.into(),
        };
        let result = annotate("PollActivityExecution", false, &request.encode_to_vec(), 1)?;
        assert_eq!(result.violations.is_empty(), run_id != "invalid", "{run_id}");
        assert!(result.bytes.eq(&request.encode_to_vec()));
    }
    let mut request = DescribeActivityExecutionRequest {
        namespace: "external-namespace".into(),
        activity_id: "activity".into(),
        long_poll_token: b"private-token".to_vec(),
        ..Default::default()
    };
    assert_ne!(
        annotate("DescribeActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request.run_id = "a13cd9da-521c-48d4-ab9c-6ab174876f25".into();
    assert_eq!(
        annotate("DescribeActivityExecution", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    Ok(())
}

#[test]
fn secret_fields_redact_report_copies_without_changing_transport()
-> Result<(), Box<dyn std::error::Error>> {
    let original = Payload {
        metadata: std::collections::BTreeMap::from([(
            "private-key".into(),
            b"private-value".to_vec(),
        )]),
        data: b"private-payload".to_vec(),
        ..Default::default()
    };
    let before = original.clone();
    let mut report = original.clone();
    umpire_temporal::activity_api::catalog::payload()?.redact(&mut report);
    assert!(report.metadata.is_empty());
    assert_eq!(report.data.len(), 0);
    assert!(original.eq(&before));
    let original = RespondActivityTaskCompletedRequest {
        task_token: b"private-task-token".to_vec(),
        ..Default::default()
    };
    let before = original.clone();
    let mut report = original.clone();
    umpire_temporal::activity_api::catalog::respond_activity_task_completed_request()?
        .redact(&mut report);
    assert_eq!(report.task_token.len(), 0);
    assert!(original.eq(&before));
    Ok(())
}

#[test]
fn request_duration_validation_mirrors_server_normalization_without_repair()
-> Result<(), Box<dyn std::error::Error>> {
    for duration in
        [Duration { seconds: 0, nanos: 1_000_000_001 }, Duration { seconds: i64::MAX, nanos: 0 }]
    {
        let mut request = start();
        request.request_id = "supplied".into();
        request.start_to_close_timeout = Some(duration);
        let bytes = request.encode_to_vec();
        let result = annotate("StartActivityExecution", false, &bytes, 1)?;
        assert!(result.violations.is_empty(), "{:?}", result.violations);
        assert!(result.bytes.eq(&bytes));
    }
    let catalog = umpire_temporal::activity_api::start_request_id_intent()?;
    let mut request = start();
    request.request_id = "supplied".into();
    let violations = umpire_annotations::validate(&request, &catalog);
    assert!(matches!(
        violations[0].expected.as_ref().map(|expected| &expected.expectation),
        Some(umpire_annotations::Expectation::Accepts)
    ));
    Ok(())
}

#[test]
fn worker_failure_requires_application_info_only_when_failure_is_supplied()
-> Result<(), Box<dyn std::error::Error>> {
    let mut request = RespondActivityTaskFailedRequest::default();
    assert_eq!(
        annotate("RespondActivityTaskFailed", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request.failure = Some(Failure {
        failure_info: Some(failure::FailureInfo::ServerFailureInfo(ServerFailureInfo::default())),
        ..Default::default()
    });
    assert_ne!(
        annotate("RespondActivityTaskFailed", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    request.failure.as_mut().ok_or("failure")?.failure_info =
        Some(failure::FailureInfo::ApplicationFailureInfo(ApplicationFailureInfo::default()));
    assert_eq!(
        annotate("RespondActivityTaskFailed", false, &request.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    let response = umpire_temporal_api::temporal::api::workflowservice::v1::RespondActivityTaskFailedResponse { failures: vec![Failure { failure_info: Some(failure::FailureInfo::ServerFailureInfo(ServerFailureInfo::default())), ..Default::default() }] };
    assert_eq!(
        annotate("RespondActivityTaskFailed", true, &response.encode_to_vec(), 1)?.violations,
        [] as [String; 0]
    );
    Ok(())
}
