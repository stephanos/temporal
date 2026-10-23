//! Generated RPC dispatch with value-free catalog findings.

use super::{RpcRole, classify};
use crate::foundation::dc;
use prost::Message;
use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::WorkflowServiceMethod;
use umpire_temporal_api::temporal::api::workflowservice::v1::RespondActivityTaskFailedRequest;

pub use umpire_catalog::AnnotationResult;
use umpire_catalog::{annotate_message, annotation_findings as findings};

/// Fills absent declared request IDs and validates one classified activity RPC.
/// Request constraints declared Accepts remain probe metadata; responses retain
/// their original bytes and every finding.
///
/// # Errors
/// Refuses unknown methods, malformed bytes, failed mutations and catalog admission.
pub fn annotate(
    method: &str,
    response: bool,
    bytes: &[u8],
    seed: u64,
) -> Result<AnnotationResult, Refusal> {
    let method = method
        .strip_prefix("/temporal.api.workflowservice.v1.WorkflowService/")
        .or_else(|| (!method.contains('/') && !method.contains('.')).then_some(method))
        .and_then(WorkflowServiceMethod::parse)
        .ok_or_else(|| Refusal::unsupported("annotation RPC is not classified"))?;
    if let RpcRole::OutOfScope(reason) = classify(method) {
        return Ok(AnnotationResult {
            bytes: bytes.to_vec(),
            violations: Vec::new(),
            out_of_scope: Some(reason.into()),
        });
    }
    let limits = dc::Limits::default();
    let mut result = umpire_temporal_api::dispatch_workflow_annotations!(
        method,
        response,
        M,
        { annotate_message::<M, _>(bytes, response, seed, &limits) },
        Err(Refusal::unsupported("classified RPC has no annotation message"))
    )?;
    if !response && method == WorkflowServiceMethod::RespondActivityTaskFailed {
        let request = RespondActivityTaskFailedRequest::decode(result.bytes.as_slice())
            .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
        if let Some(failure) = request.failure {
            let catalog =
                crate::foundation::api::WorkerFailure::catalog(&limits).map_err(|_| {
                    Refusal::unsupported("activity annotation catalog admission failed")
                })?;
            result
                .violations
                .extend(findings(umpire_annotations::validate(&failure, &catalog), false));
        }
    }
    result.violations.sort();
    result.violations.dedup();
    Ok(result)
}
