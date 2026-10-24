use prost::Message;

use crate::{Dispatch, google::protobuf::Any, protocol};

/// A dispatcher with an independent, stateless annotation extension.
pub struct AnnotationRuntime<D, A> {
    model: D,
    annotate: A,
}

impl<D, A> AnnotationRuntime<D, A> {
    /// Attach a stateless annotation handler to an existing dispatcher.
    pub fn new(model: D, annotate: A) -> Self {
        Self { model, annotate }
    }
}

impl<D, A> Dispatch for AnnotationRuntime<D, A>
where
    D: Dispatch,
    A: Fn(&Any) -> Result<Any, protocol::Refusal>,
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
            .and_then(|input| (self.annotate)(&input));
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
