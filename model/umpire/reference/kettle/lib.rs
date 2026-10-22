//! Framework-owned Kettle reference system using the canonical independent Model.

#[path = "../../tests/fixtures/sample_kettle/model.rs"]
mod model;
pub use model::kettle;

/// Generated reference service messages.
#[allow(missing_docs, clippy::all, clippy::pedantic)]
pub mod api {
    include!(concat!(env!("OUT_DIR"), "/umpire.reference.kettle.v1.rs"));
}
/// Evidence projection and callback binding.
pub mod realization;
pub use realization::Realization;

/// Transactional live Kettle run, independent of full-space exploration.
pub type KettleRuntime = umpire_wasm::Runtime<kettle::World, Realization>;

/// Select the declared room-temperature start for one live Kettle instance.
pub fn runtime() -> KettleRuntime {
    KettleRuntime::with_initial(
        kettle::world(),
        Realization,
        kettle::Model { temperature: 20, heating_on: false, beeping: false },
    )
    .unwrap_or_else(|error| panic!("canonical Kettle start is missing: {}", error.message))
}
