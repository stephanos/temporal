//! Every pre-admission case has an exact retained or claim-explained disposition.
#![forbid(unsafe_code)]

use prost::Message;
use std::collections::BTreeMap;
use umpire_runtime::annotations::{ProbeApplicability, ProbeResponse};
use umpire_temporal::probes::probe;
use umpire_temporal_api::{WorkflowServiceMethod, workflow_field_claim, workflow_request_fields};

#[test]
fn every_previous_case_is_reconciled_without_prediction_or_control_loss()
-> Result<(), Box<dyn std::error::Error>> {
    let previous =
        ProbeResponse::decode(include_bytes!("data/probes-before-claims.binpb").as_slice())?;
    let current = probe(&umpire_temporal_api::umpire::temporal::v1::ProbeRequest::default())?;
    let mut retained: BTreeMap<_, _> =
        current.cases.iter().map(|case| (case.id.as_str(), case)).collect();
    let mut removed = 0;
    let mut strengthened = 0;
    for old in &previous.cases {
        if let Some(new) = retained.remove(old.id.as_str()) {
            if old != new {
                verify_strengthening(old, new)?;
                strengthened += 1;
            }
            continue;
        }
        assert!(old.generic && old.predictions.is_empty(), "lost explicit claim: {}", old.id);
        let (_, rest) = old.id.split_once("generic::").ok_or("missing generic identity")?;
        let (method, path) = rest.split_once("::").ok_or("missing method")?;
        let (path, mutation) = path.split_once(":mutation.").ok_or("missing mutation")?;
        let method = WorkflowServiceMethod::parse(method).ok_or("unknown method")?;
        let mut justified = false;
        for field in workflow_request_fields(method) {
            if path != field.path
                && !path.strip_prefix(&field.path).is_some_and(|rest| rest.starts_with('.'))
            {
                continue;
            }
            if let Some(claim) = workflow_field_claim(&field.declaration) {
                if let Some(reason) = claim.ignored {
                    assert!(reason.contains(claim.declaration));
                    justified = true;
                } else if path == field.path
                    && claim.kind == "Secret"
                    && mutation.starts_with("limit_plus_one(")
                {
                    assert!(matches!(
                        ProbeApplicability::try_from(old.applicability)?,
                        ProbeApplicability::Applicable | ProbeApplicability::Refused
                    ));
                    justified = true;
                }
            }
        }
        assert!(justified, "unexplained removal: {}", old.id);
        removed += 1;
    }
    assert_eq!(removed, 1178);
    assert_eq!(strengthened, 4);
    assert_eq!(retained.len(), 2);
    for (id, pattern) in [
        (
            "poll_queue.normal_name::temporal.api.workflowservice.v1.PollActivityTaskQueueRequest.task_queue:string.max_len",
            "only sticky queues can have normal name",
        ),
        (
            "start.retry_order::temporal.api.workflowservice.v1.StartActivityExecutionRequest.retry_policy:int32.gte",
            "MaximumInterval cannot be less than InitialInterval",
        ),
    ] {
        let added = retained.remove(id).ok_or("missing strict semantic rule")?;
        assert!(!added.generic && !added.accepts);
        assert_eq!(added.code, 3);
        assert_eq!(added.message_pattern, pattern);
    }
    Ok(())
}

fn verify_strengthening(
    old: &umpire_runtime::annotations::ProbeCase,
    new: &umpire_runtime::annotations::ProbeCase,
) -> Result<(), Box<dyn std::error::Error>> {
    for (method, field, target, pattern) in [
        (
            "PollActivityTaskQueue",
            "task_queue.normal_name",
            "task_queue:string.max_len:",
            "only sticky queues can have normal name",
        ),
        (
            "StartActivityExecution",
            "retry_policy.initial_interval.seconds",
            "retry_policy:int32.gte:",
            "MaximumInterval cannot be less than InitialInterval",
        ),
    ] {
        for bound in [1000, 2_097_152] {
            if old.id
                != format!(
                    "generic::{method}::temporal.api.workflowservice.v1.{method}Request.{field}:mutation.limit_plus_one({bound})"
                )
            {
                continue;
            }
            assert_eq!(old.predictions, []);
            assert_eq!(new.predictions.len(), 1);
            let prediction = &new.predictions[0];
            assert!(!prediction.accepts);
            assert_eq!(prediction.code, 3);
            assert_eq!(prediction.message_pattern, pattern);
            assert!(prediction.id.contains(&format!("{method}Request.{target}")));
            let mut expected = old.clone();
            expected.predictions.clone_from(&new.predictions);
            assert_eq!(&expected, new, "unexplained retained metadata change: {}", old.id);
            return Ok(());
        }
    }
    Err(format!("unexplained strengthening: {}", old.id).into())
}
