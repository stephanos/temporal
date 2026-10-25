use crate::AnyMessage as _;
use prost::{Message, Name};

use crate::annotations::{AnnotationRequest, AnnotationResponse};

use crate::{Dispatch, prost_types::Any, protocol};

/// A dispatcher with an independent, stateless annotation extension.
pub struct AnnotationRuntime<D, P, R> {
    model: D,
    probe: fn(&P) -> Result<R, protocol::Refusal>,
    annotate: fn(&AnnotationRequest) -> Result<AnnotationResponse, protocol::Refusal>,
}

impl<D, P, R> AnnotationRuntime<D, P, R> {
    /// Attach typed, stateless probe and annotation handlers to an existing dispatcher.
    pub fn with_probes(
        model: D,
        probe: fn(&P) -> Result<R, protocol::Refusal>,
        annotate: fn(&AnnotationRequest) -> Result<AnnotationResponse, protocol::Refusal>,
    ) -> Self {
        Self { model, probe, annotate }
    }
}

impl<D, P, R> Dispatch for AnnotationRuntime<D, P, R>
where
    D: Dispatch,
    P: Name + Default,
    R: Name,
{
    type Transaction = D::Transaction;

    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>) {
        if method != protocol::Method::Annotate as u32 {
            return self.model.prepare(method, request);
        }
        let result = protocol::AnnotateRequest::decode(request)
            .map_err(|_| protocol::Refusal::malformed("malformed Annotate request"))
            .and_then(|request| Any::required(request.input))
            .and_then(|input| {
                if input.type_url == P::type_url() {
                    (self.probe)(&input.unpack()?).map(|response| Any::pack(&response))
                } else {
                    (self.annotate)(&input.unpack()?).map(|response| Any::pack(&response))
                }
            });
        let outcome = match result {
            Ok(output) => protocol::response::Outcome::Annotate(protocol::AnnotateResponse {
                output: Some(output),
            }),
            Err(refusal) => protocol::response::Outcome::Refusal(refusal),
        };
        (protocol::Response { outcome: Some(outcome) }, None)
    }

    fn commit(&mut self, transaction: Self::Transaction) {
        self.model.commit(transaction);
    }
}

use crate::annotations::{ProbeRecipe, ProbeSegment};

/// Encode a bounded probe payload as literal and repeated-byte transport segments.
pub fn probe_recipe(bytes: &[u8]) -> Result<ProbeRecipe, Box<dyn std::error::Error>> {
    if bytes.len() > 16 * 1024 * 1024 {
        return Err("probe exceeds recipe bound".into());
    }
    let mut segments = Vec::new();
    let mut literal = Vec::new();
    let mut index = 0;
    while index < bytes.len() {
        let mut end = index + 1;
        while end < bytes.len() && bytes[end] == bytes[index] {
            end += 1;
        }
        if end - index >= 16 {
            if !literal.is_empty() {
                segments.push(ProbeSegment { literal: std::mem::take(&mut literal), repeat: 1 });
            }
            segments.push(ProbeSegment {
                literal: vec![bytes[index]],
                repeat: u32::try_from(end - index)?,
            });
        } else {
            literal.extend_from_slice(&bytes[index..end]);
        }
        index = end;
    }
    if !literal.is_empty() {
        segments.push(ProbeSegment { literal, repeat: 1 });
    }
    let fnv64 = bytes.iter().fold(0xcbf2_9ce4_8422_2325_u64, |hash, byte| {
        (hash ^ u64::from(*byte)).wrapping_mul(0x0000_0100_0000_01b3)
    });
    Ok(ProbeRecipe { segments, decoded_length: u32::try_from(bytes.len())?, fnv64 })
}
