//! Integration controls for the Temporal realization, consolidated into one test binary.

#![forbid(unsafe_code)]

#[path = "activity_policy.rs"]
mod activity_policy;
#[path = "admission.rs"]
mod admission;
#[path = "admission_contracts.rs"]
mod admission_contracts;
#[path = "annotations.rs"]
mod annotations;
#[path = "catalog.rs"]
mod catalog;
#[allow(
    dead_code,
    unreachable_pub,
    reason = "each former test binary uses a different subset of the shared fixtures"
)]
mod common;
mod compiler;
#[path = "declarative.rs"]
mod declarative;
#[allow(
    dead_code,
    unreachable_pub,
    reason = "each former test binary uses a different subset of the shared oracles"
)]
mod oracles;
#[path = "phase_exhaustiveness.rs"]
mod phase_exhaustiveness;
#[path = "probe_accounting.rs"]
mod probe_accounting;
#[path = "probes.rs"]
mod probes;
#[path = "program.rs"]
mod program;
#[path = "program_history.rs"]
mod program_history;
#[path = "proposal.rs"]
mod proposal;
#[path = "realization.rs"]
mod realization;
#[path = "realization_admission.rs"]
mod realization_admission;
#[path = "registry.rs"]
mod registry;
#[path = "start_mirror.rs"]
mod start_mirror;
#[path = "supplemental_admission.rs"]
mod supplemental_admission;
#[path = "support.rs"]
mod support;

// The generated API consumer's `Catalog` derive resolves its defaults at the crate root.
#[path = "ui-pass/generated_api.rs"]
#[allow(
    dead_code,
    unreachable_pub,
    clippy::all,
    clippy::pedantic,
    reason = "valid consumer shapes follow the macro contracts and were never linted as trybuild cases"
)]
mod generated_api;
use generated_api::{CatalogContext, temporal};
