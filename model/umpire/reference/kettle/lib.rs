//! Framework-owned Kettle reference system using the canonical independent Model.

#[path = "../../dev/fixtures/fixtures/sample_kettle/model.rs"]
mod model;
pub use model::kettle;

/// Generated reference service messages.
#[allow(missing_docs, clippy::all, clippy::pedantic)]
pub mod wire {
    pub mod umpire {
        pub mod reference {
            pub mod kettle {
                pub mod v1 {
                    include!(concat!(env!("OUT_DIR"), "/umpire.reference.kettle.v1.rs"));
                }
            }
        }
    }
}
#[allow(missing_docs, deprecated, clippy::all, clippy::pedantic)]
#[macro_use]
mod generated {
    include!(concat!(env!("OUT_DIR"), "/services.rs"));
    include!(concat!(env!("OUT_DIR"), "/wire_schema.rs"));
}
pub use generated::*;
/// Exhaustive annotation mirrors.
pub mod api;
/// Explicit static and per-kettle limits.
pub mod config;
/// Stateless generic probe controls.
pub mod probes;

/// Evidence projection and callback realization.
pub mod realization;
pub use realization::Realization;

/// Transactional live Kettle run, independent of full-space exploration.
pub type KettleRuntime = umpire_runtime::Runtime<kettle::World, Realization>;

/// Select the declared room-temperature start for one live Kettle instance.
pub fn runtime() -> KettleRuntime {
    KettleRuntime::with_initial(
        kettle::world(),
        Realization,
        kettle::Model { temperature: 20, heating_on: false, beeping: false },
    )
    .unwrap_or_else(|error| panic!("canonical Kettle start is missing: {}", error.message))
}

/// Kettle Model dispatcher with independent annotation and probe operations.
pub type AnnotationRuntime = umpire_runtime::AnnotationRuntime<
    KettleRuntime,
    wire::umpire::reference::kettle::v1::ProbeRequest,
    umpire_runtime::annotations::ProbeResponse,
>;

/// Creates the selected live run and its stateless annotation extension.
pub fn annotation_runtime() -> AnnotationRuntime {
    umpire_runtime::AnnotationRuntime::with_probes(
        runtime(),
        |request| {
            probes::run(request)
                .map_err(|error| umpire_runtime::protocol::Refusal::malformed(error.to_string()))
        },
        |request| {
            transport::annotate(&request.method, request.response, &request.message, request.seed)
                .map(Into::into)
        },
    )
}
mod transport;
