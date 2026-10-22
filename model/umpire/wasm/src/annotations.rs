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
