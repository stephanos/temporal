//! Generated RPC dispatch with value-free catalog findings.

use super::{RpcRole, classify};
use crate::foundation::dc;
use prost::Message;
use umpire_annotations::{Expectation, Violation};
use umpire_temporal_annotate::Inspect;
use umpire_temporal_api::WorkflowServiceMethod;
use umpire_temporal_api::temporal::api::workflowservice::v1::RespondActivityTaskFailedRequest;
use umpire_wasm::protocol::Refusal;

/// Filled request bytes or untouched response bytes, with value-free findings.
pub struct AnnotationResult {
    /// Protobuf bytes to transport; response bytes are always unchanged.
    pub bytes: Vec<u8>,
    /// Field paths and violated constraint names, never field contents.
    pub violations: Vec<String>,
    /// Explicit limitation for a classified RPC outside this activity Model.
    pub out_of_scope: Option<String>,
}

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
        { process::<M>(bytes, response, seed, &limits) },
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

fn process<M: Message + Default + Inspect<dc::Limits>>(
    bytes: &[u8],
    response: bool,
    seed: u64,
    limits: &dc::Limits,
) -> Result<AnnotationResult, Refusal> {
    let mut message = M::decode(bytes)
        .map_err(|_| Refusal::malformed("annotation protobuf could not be decoded"))?;
    let changed = !response
        && message
            .fill_ids(seed)
            .map_err(|_| Refusal::unsupported("annotation request ID mutation failed"))?;
    let mut violations = Vec::new();
    message
        .inspect(limits, response, &mut violations)
        .map_err(|_| Refusal::unsupported("activity annotation catalog admission failed"))?;
    let violations = findings(violations, response);
    Ok(AnnotationResult {
        bytes: if changed { message.encode_to_vec() } else { bytes.to_vec() },
        violations,
        out_of_scope: None,
    })
}

fn findings(violations: Vec<Violation>, response: bool) -> Vec<String> {
    violations
        .into_iter()
        .filter(|violation| {
            response
                || !violation
                    .expected
                    .as_ref()
                    .is_some_and(|expected| matches!(expected.expectation, Expectation::Accepts))
        })
        .map(|violation| violation.to_string())
        .collect()
}
