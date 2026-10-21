//! Temporal realization facade for the mini activity guest.
#![cfg(not(test))]

pub mod support;

#[path = "../../features/activity/api.rs"]
pub mod activity_api;

mod annotations;

pub mod probes;

#[path = "../../features/activity/temporal.rs"]
mod temporal;

pub use annotations::{AnnotationRuntime, annotation_runtime};
pub use temporal::{Realization, activity, runtime};
