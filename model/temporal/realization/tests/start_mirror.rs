//! The first real derive mirror retains the reviewed Start catalog.

#![forbid(unsafe_code)]

use umpire_catalog::validate;
use umpire_temporal::{activity_api, foundation::dc};
use umpire_temporal_api::temporal::api::workflowservice::v1::StartActivityExecutionRequest;

#[test]
fn start_mirror_retains_fields_reasons_limits_and_expected_errors()
-> Result<(), Box<dyn std::error::Error>> {
    let limits = dc::Limits::default();
    let catalog = activity_api::StartActivityExecutionRequest::catalog(&limits)?;
    assert_eq!(catalog.fields().count(), 22);
    let ignored: Vec<_> = catalog
        .fields()
        .filter(|field| {
            field.tags.iter().any(|(_, tag)| matches!(tag, umpire_catalog::Consequence::Ignored(_)))
        })
        .collect();
    assert_eq!(ignored.len(), 7);
    let request = StartActivityExecutionRequest {
        identity: "x".repeat(1001),
        activity_id: "valid-id".into(),
        ..Default::default()
    };
    let violations = validate(&request, &catalog);
    assert!(
        violations
            .iter()
            .any(|v| v.path.ends_with(".identity") && v.constraint == "string.max_bytes")
    );
    assert!(violations.iter().any(|v| v.constraint == "required"));
    let mut limits = limits;
    limits.set(&dc::MAX_ID_LENGTH, 4);
    limits.set(&dc::BLOB_SIZE_ERROR, 2);
    let catalog = activity_api::StartActivityExecutionRequest::catalog(&limits)?;
    let request = StartActivityExecutionRequest { identity: "five!".into(), ..Default::default() };
    assert!(validate(&request, &catalog).iter().any(|v| v.path.ends_with(".identity")));
    let cancel = umpire_temporal_api::temporal::api::workflowservice::v1::RequestCancelActivityExecutionRequest {
        reason: "big".into(),
        ..Default::default()
    };
    let catalog = activity_api::RequestCancelActivityExecutionRequest::catalog(&limits)?;
    assert!(validate(&cancel, &catalog).iter().any(|v| v.path.ends_with(".reason")));
    Ok(())
}

#[test]
fn shared_token_dependencies_remain_independent() -> Result<(), Box<dyn std::error::Error>> {
    use umpire_temporal_api::temporal::api::{
        common::v1::{Payload, Payloads},
        workflowservice::v1::PollActivityTaskQueueResponse,
    };
    let catalog = activity_api::PollActivityTaskQueueResponse::catalog(&dc::Limits::default())?;
    for (trigger, mut response) in [
        (
            "activity_id",
            PollActivityTaskQueueResponse { activity_id: "activity".into(), ..Default::default() },
        ),
        (
            "input",
            PollActivityTaskQueueResponse {
                input: Some(Payloads {
                    payloads: vec![Payload { data: vec![1], ..Default::default() }],
                }),
                ..Default::default()
            },
        ),
    ] {
        let findings: Vec<_> = validate(&response, &catalog)
            .into_iter()
            .filter(|v| v.constraint == "requires")
            .collect();
        assert_eq!(findings.len(), 1, "{trigger}");
        let prefix = "temporal.api.workflowservice.v1.PollActivityTaskQueueResponse";
        assert_eq!(
            findings[0].fields,
            [format!("{prefix}.{trigger}"), format!("{prefix}.task_token")]
        );
        response.task_token = vec![1];
        assert!(validate(&response, &catalog).into_iter().all(|v| v.constraint != "requires"));
    }
    Ok(())
}
