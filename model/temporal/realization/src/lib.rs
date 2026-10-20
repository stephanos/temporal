//! Temporal realization facade for the mini activity guest.
#![cfg(not(test))]

pub mod support;

#[path = "../../features/activity/temporal.rs"]
mod temporal;

pub use temporal::{Realization, activity, runtime};
