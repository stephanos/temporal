#[cfg(all(feature = "helper-diagnostic-profile", feature = "determinism-fixture"))]
compile_error!("diagnostic profiling cannot be combined with host fixture controls");

pub mod backstop;
pub mod campaign;
pub mod cli;
pub(crate) mod config;
pub(crate) mod cycle;
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
pub(crate) mod fork_order;
pub mod install;
#[cfg(test)]
pub(crate) mod installed_lifecycle_tests;
#[cfg(feature = "determinism-fixture")]
pub(crate) mod live_backend;
#[cfg(feature = "determinism-fixture")]
mod live_broker;
#[cfg(feature = "determinism-fixture")]
mod live_cluster;
#[cfg(feature = "determinism-fixture")]
mod live_runtime;
pub mod measurement;
pub(crate) mod minimize;
pub mod profile;
pub(crate) mod qualify;
pub mod replay;
pub mod runtime_qualification;
pub mod shim;
pub mod slots;
pub(crate) mod stress;
pub mod sweep;

#[doc(hidden)]
pub use qualify::support::run as support_tool;
