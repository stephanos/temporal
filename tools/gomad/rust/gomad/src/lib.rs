#[cfg(all(feature = "helper-diagnostic-profile", feature = "determinism-fixture"))]
compile_error!("diagnostic profiling cannot be combined with host fixture controls");

pub mod backstop;
pub mod campaign;
pub mod cli;
pub mod evidence;
pub mod execute;
#[cfg(feature = "determinism-fixture")]
pub mod fixture;
#[cfg(feature = "determinism-fixture")]
mod fixture_cluster;
#[cfg(feature = "determinism-fixture")]
mod fixture_execute;
#[cfg(feature = "determinism-fixture")]
mod fixture_replay;
#[cfg(feature = "determinism-fixture")]
mod fixture_role;
#[cfg(feature = "determinism-fixture")]
mod fixture_storage;
pub mod install;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod live_backend;
#[cfg(feature = "determinism-fixture")]
mod live_broker;
#[cfg(feature = "determinism-fixture")]
mod live_cluster;
#[cfg(feature = "determinism-fixture")]
mod live_runtime;
pub(crate) mod minimize;
pub(crate) mod qualify;
pub mod replay;
pub mod slots;
pub(crate) mod stress;

#[cfg(feature = "helper-diagnostic-profile")]
pub mod profile;
