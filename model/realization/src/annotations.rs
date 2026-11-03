use crate::activity_api;

#[path = "live.rs"]
mod live;
use umpire_runtime::{annotations::ProbeResponse, protocol};
use umpire_temporal_api::umpire::temporal::v1::ProbeRequest;

/// A Model runtime with independent, stateless transport annotation dispatch.
pub type AnnotationRuntime =
    umpire_runtime::AnnotationRuntime<live::LiveRuntime, ProbeRequest, ProbeResponse>;

/// Creates one independently owned run with typed transport annotation support.
pub fn annotation_runtime() -> AnnotationRuntime {
    umpire_runtime::AnnotationRuntime::with_probes(
        live::LiveRuntime::new(),
        |request| {
            crate::probes::probe(request)
                .map_err(|error| protocol::Refusal::malformed(error.to_string()))
        },
        |request| {
            activity_api::annotate(
                &request.method,
                request.response,
                &request.message,
                request.seed,
            )
            .map(Into::into)
        },
    )
}
