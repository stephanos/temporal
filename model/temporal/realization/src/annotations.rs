use prost::Message;
use umpire_temporal_api::umpire::temporal::v1::{
    AnnotationRequest, AnnotationResponse, ProbeRequest,
};
use umpire_wasm::{Dispatch, Runtime, google::protobuf::Any, protocol};

use crate::{Realization, activity, activity_api, runtime};

/// A Model runtime with independent, stateless transport annotation dispatch.
pub struct AnnotationRuntime {
    model: Runtime<activity::World, Realization>,
}

/// Creates one independently owned run with typed transport annotation support.
pub fn annotation_runtime() -> AnnotationRuntime {
    AnnotationRuntime { model: runtime() }
}

impl Dispatch for AnnotationRuntime {
    type Transaction = <Runtime<activity::World, Realization> as Dispatch>::Transaction;

    fn prepare(
        &self,
        method: u32,
        request: &[u8],
    ) -> (protocol::Response, Option<Self::Transaction>) {
        if method != protocol::Method::Annotate as u32 {
            return self.model.prepare(method, request);
        }
        let outcome = match annotate(request) {
            Ok(response) => protocol::response::Outcome::Annotate(response),
            Err(refusal) => protocol::response::Outcome::Refusal(refusal),
        };
        (protocol::Response { outcome: Some(outcome) }, None)
    }

    fn commit(&mut self, transaction: Self::Transaction) {
        self.model.commit(transaction);
    }
}

fn annotate(bytes: &[u8]) -> Result<protocol::AnnotateResponse, protocol::Refusal> {
    let request = protocol::AnnotateRequest::decode(bytes)
        .map_err(|_| protocol::Refusal::malformed("malformed Annotate request"))?;
    let input = Any::required(request.input)?;
    if input.type_url == Any::pack(&ProbeRequest::default()).type_url {
        let input: ProbeRequest = input.unpack()?;
        let response = crate::probes::probe(&input)
            .map_err(|error| protocol::Refusal::malformed(error.to_string()))?;
        return Ok(protocol::AnnotateResponse { output: Some(Any::pack(&response)) });
    }
    let input: AnnotationRequest = input.unpack()?;
    let result = activity_api::annotate(&input.method, input.response, &input.message, input.seed)?;
    Ok(protocol::AnnotateResponse {
        output: Some(Any::pack(&AnnotationResponse {
            message: result.bytes,
            violations: result.violations,
            out_of_scope: result.out_of_scope.unwrap_or_default(),
        })),
    })
}
