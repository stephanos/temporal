//! Generated RPC dispatch with value-free catalog findings.

use crate::foundation::dc;
use prost::Message;
use umpire_runtime::protocol::Refusal;
use umpire_temporal_api::WorkflowServiceMethod;
use umpire_temporal_api::temporal::api::workflowservice::v1::RespondActivityTaskFailedRequest;

pub use umpire_catalog::AnnotationResult;
use umpire_catalog::annotation_findings as findings;

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
    let mut result =
        umpire_temporal_api::annotate_workflow!(method, response, bytes, seed, |_, _, _| Ok(
            dc::Limits::default()
        ))?;
    if !response
        && WorkflowServiceMethod::parse_rpc(method)
            == Some(WorkflowServiceMethod::RespondActivityTaskFailed)
    {
        let request = RespondActivityTaskFailedRequest::decode(result.bytes.as_slice())
            .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
        if let Some(failure) = request.failure {
            let catalog = crate::foundation::api::WorkerFailure::catalog(&dc::Limits::default())
                .map_err(|_| {
                    Refusal::unsupported("activity annotation catalog admission failed")
                })?;
            result.violations.extend(findings(umpire_catalog::validate(&failure, &catalog), false));
        }
    }
    result.violations.sort();
    result.violations.dedup();
    Ok(result)
}
