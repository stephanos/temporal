//! Admission rules exposed by the independent raw-schema probe.

#![forbid(unsafe_code)]

use umpire_catalog::{Kind, validate};
use umpire_temporal::{activity_api, foundation::dc};
use umpire_temporal_api::temporal::api::workflowservice::v1 as wire;

#[test]
fn namespace_and_worker_token_admission_preserves_opaque_kinds()
-> Result<(), Box<dyn std::error::Error>> {
    let limits = dc::Limits::default();
    let mut actual = Vec::new();
    macro_rules! namespace {
        ($name:ident, $required:expr) => {{
            let catalog = activity_api::$name::catalog(&limits)?;
            let path = concat!("temporal.api.workflowservice.v1.", stringify!($name), ".namespace");
            assert!(
                catalog
                    .fields()
                    .any(|field| field.path == path && matches!(field.kind, Kind::Text(_)))
            );
            let missing_findings = validate(&wire::$name::default(), &catalog);
            let missing = missing_findings
                .iter()
                .find(|v| v.constraint == "required" && v.fields.iter().any(|field| field == path));
            let missing_matches = missing.is_some() == $required && missing.is_none_or(|finding| {
                matches!(finding.expected.as_ref().map(|expected| &expected.expectation),
                    Some(umpire_catalog::Expectation::Rejects { code: 3, message_pattern, .. })
                        if message_pattern == "Namespace is empty")
            });
            let oversized = wire::$name { namespace: "x".repeat(1001), ..Default::default() };
            let bounded = validate(&oversized, &catalog).iter().any(|v| {
                v.path == path && v.constraint == "string.max_bytes" && v.expected.is_some()
            });
            let supplied = wire::$name { namespace: "x".repeat(1000), ..Default::default() };
            let valid = validate(&supplied, &catalog).iter().all(|v| v.path != path);
            actual.push((stringify!($name), bounded, missing_matches, valid));
        }};
    }
    namespace!(StartActivityExecutionRequest, true);
    namespace!(DescribeActivityExecutionRequest, true);
    namespace!(PollActivityExecutionRequest, true);
    namespace!(PollActivityTaskQueueRequest, true);
    namespace!(RequestCancelActivityExecutionRequest, true);
    namespace!(RespondActivityTaskCompletedRequest, false);
    namespace!(RespondActivityTaskFailedRequest, false);
    namespace!(RespondActivityTaskCanceledRequest, false);
    macro_rules! token {
        ($name:ident) => {{
            let catalog = activity_api::$name::catalog(&limits)?;
            let path =
                concat!("temporal.api.workflowservice.v1.", stringify!($name), ".task_token");
            assert!(catalog.fields().any(|field| field.path == path && field.kind == Kind::Secret));
            let missing = validate(&wire::$name::default(), &catalog)
                .iter()
                .any(|v| v.constraint == "required" && v.fields.iter().any(|field| field == path));
            let supplied = wire::$name { task_token: vec![1], ..Default::default() };
            let opaque = validate(&supplied, &catalog).iter().all(|v| {
                v.path != path
                    && !(v.constraint == "required" && v.fields.iter().any(|field| field == path))
            });
            actual.push((concat!(stringify!($name), ".task_token"), missing, true, opaque));
        }};
    }
    token!(RespondActivityTaskCompletedRequest);
    token!(RespondActivityTaskFailedRequest);
    token!(RespondActivityTaskCanceledRequest);
    let expected: Vec<_> = actual.iter().map(|(name, _, _, _)| (*name, true, true, true)).collect();
    assert_eq!(actual, expected);
    Ok(())
}
