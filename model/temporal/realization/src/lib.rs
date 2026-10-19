//! Temporal realization facade for the mini activity guest.
#![cfg(not(test))]

#[path = "../../features/activity/temporal.rs"]
mod temporal;

pub use temporal::{Action, Realization, World, activity, runtime};
