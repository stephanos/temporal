//! Typed annotation processing shared by service dispatchers.

use crate::Inspect;
use prost::Message;
use umpire_annotations::{Expectation, Violation};
use umpire_runtime::protocol::Refusal;

/// Filled request bytes or untouched response bytes, with value-free findings.
pub struct AnnotationResult {
    /// Protobuf bytes to transport; response bytes are always unchanged.
    pub bytes: Vec<u8>,
    /// Field paths and violated constraint names, never field contents.
    pub violations: Vec<String>,
    /// Explicit limitation for a classified RPC outside this Model.
    pub out_of_scope: Option<String>,
}

/// Decode, fill absent request identifiers and inspect a typed wire message.
/// Response bytes remain unchanged, including unknown fields.
pub fn annotate_message<M: Message + Default + Inspect<C>, C>(
    bytes: &[u8],
    response: bool,
    seed: u64,
    limits: &C,
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
        .map_err(|_| Refusal::unsupported("annotation catalog admission failed"))?;
    let violations = annotation_findings(violations, response);
    Ok(AnnotationResult {
        bytes: if changed { message.encode_to_vec() } else { bytes.to_vec() },
        violations,
        out_of_scope: None,
    })
}

/// Render value-free findings, retaining Accepts predictions only for responses.
pub fn annotation_findings(violations: Vec<Violation>, response: bool) -> Vec<String> {
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
