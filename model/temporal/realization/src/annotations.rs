use crate::{Realization, activity, activity_api, runtime};
use umpire_temporal_api::umpire::temporal::v1::ProbeRequest;
use umpire_wasm::{
    Runtime,
    annotations::{AnnotationRequest, AnnotationResponse},
    google::protobuf::Any,
    protocol,
};

/// A Model runtime with independent, stateless transport annotation dispatch.
pub type AnnotationRuntime = umpire_wasm::AnnotationRuntime<
    Runtime<activity::World, Realization>,
    fn(&Any) -> Result<Any, protocol::Refusal>,
>;

/// Creates one independently owned run with typed transport annotation support.
pub fn annotation_runtime() -> AnnotationRuntime {
    umpire_wasm::AnnotationRuntime::new(runtime(), annotate)
}

fn annotate(input: &Any) -> Result<Any, protocol::Refusal> {
    if input.type_url == Any::pack(&ProbeRequest::default()).type_url {
        let input: ProbeRequest = input.unpack()?;
        let response = crate::probes::probe(&input)
            .map_err(|error| protocol::Refusal::malformed(error.to_string()))?;
        return Ok(Any::pack(&response));
    }
    let input: AnnotationRequest = input.unpack()?;
    let result = activity_api::annotate(&input.method, input.response, &input.message, input.seed)?;
    Ok(Any::pack(&AnnotationResponse {
        message: result.bytes,
        violations: result.violations,
        out_of_scope: result.out_of_scope.unwrap_or_default(),
    }))
}
