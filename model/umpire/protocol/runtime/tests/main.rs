//! Integration controls for protocol/runtime.

#![forbid(unsafe_code)]

use umpire_runtime::protocol;

#[path = "main/abi_staging.rs"]
mod abi_staging;
#[path = "main/annotations.rs"]
mod annotations;
#[path = "main/catalog.rs"]
mod catalog;
#[path = "main/constructors.rs"]
mod constructors;
#[path = "main/core_protocol.rs"]
mod core_protocol;
#[path = "main/dispatch.rs"]
mod dispatch;
#[path = "main/prefix.rs"]
mod prefix;
#[path = "main/proposal.rs"]
mod proposal;
#[path = "main/proposal_wire.rs"]
mod proposal_wire;
#[path = "main/reach.rs"]
mod reach;
#[path = "main/rejecting.rs"]
pub mod rejecting;
#[path = "main/selected_start.rs"]
mod selected_start;
#[path = "main/stateful.rs"]
mod stateful;
#[path = "main/wire.rs"]
mod wire;

#[path = "support/control.rs"]
mod control;
#[path = "../src/refusal.rs"]
mod refusal_mapping;
#[path = "../guests/stock/src/model.rs"]
mod stock;
mod support;
