//! Native checks of shared derived catalogs.

use super::{api::*, dc};
use umpire_catalog::{AdmissionErrors, Consequence, Kind, Rng, validate};
use umpire_temporal_api::temporal::api::failure::v1 as failure;

#[test]
fn shared_factories_admit_and_preserve_private_payload_roles() -> Result<(), AdmissionErrors> {
    let limits = dc::Limits::default();
    macro_rules! admit {
        ($($mirror:ident),* $(,)?) => {$ (let _ = $mirror::catalogs(&limits)?;)*};
    }
    admit!(
        WorkflowExecution,
        WorkflowType,
        Payloads,
        Payload,
        ActivityType,
        RetryPolicy,
        RawRetryPolicy,
        ApplicationFailureInfo,
        Failure,
        WorkerFailure,
        TaskQueue,
        TimeoutFailureInfo,
        CanceledFailureInfo,
        TerminatedFailureInfo,
        ServerFailureInfo,
        ResetWorkflowFailureInfo,
        ActivityFailureInfo,
        ChildWorkflowExecutionFailureInfo,
        NexusOperationFailureInfo,
        NexusHandlerFailureInfo,
        ProtoDuration,
        ProtoTimestamp
    );
    let payload = Payload::catalog(&limits)?;
    let metadata = payload
        .fields()
        .find(|field| field.path.ends_with(".metadata"))
        .unwrap_or_else(|| panic!("missing metadata catalog"));
    assert_eq!(metadata.kind, Kind::Secret);
    assert!(metadata.tags.iter().any(|(_, tag)| matches!(tag, Consequence::Decided)));
    let entry = Payload::__metadata_entry_catalog(&limits)?;
    assert_eq!(entry.message(), "temporal.api.common.v1.Payload.MetadataEntry");
    assert_eq!(entry.fields().count(), 2);
    Ok(())
}

#[test]
fn worker_failure_presence_probe_mutates_the_actual_oneof() -> Result<(), Box<dyn std::error::Error>>
{
    let catalog = WorkerFailure::catalog(&dc::Limits::default())?;
    let baseline = failure::Failure {
        failure_info: Some(failure::failure::FailureInfo::ApplicationFailureInfo(
            failure::ApplicationFailureInfo::default(),
        )),
        ..Default::default()
    };
    assert_eq!(validate(&baseline, &catalog), []);
    let cases = catalog.probe_cases();
    assert_eq!(cases.len(), 1);
    assert_eq!(cases[0].violation.constraint, "required");
    assert_eq!(
        cases[0].violation.fields,
        ["temporal.api.failure.v1.Failure.application_failure_info"]
    );
    let probe = catalog.probe(&baseline, &cases[0], &mut Rng::new(1))?;
    assert!(probe.message.failure_info.is_none());
    assert!(baseline.failure_info.is_some());
    Ok(())
}

#[test]
fn conditional_retry_setter_produces_a_real_invalid_coefficient()
-> Result<(), umpire_catalog::GenerationError> {
    let mut retry = umpire_temporal_api::temporal::api::common::v1::RetryPolicy::default();
    super::views::backoff_coefficient::set(&mut retry, 0)?;
    assert_eq!(super::views::backoff_coefficient::view(&retry), 0);
    Ok(())
}
