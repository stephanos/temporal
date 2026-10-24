//! Behavioral controls for the native lock and compiler drivers.
#![allow(clippy::unwrap_used, clippy::expect_used)]
#[path = "identity/compiler.rs"]
mod compiler;
#[path = "identity/current_inputs.rs"]
mod current_inputs;
#[path = "identity/lock.rs"]
mod lock;
#[path = "identity/support.rs"]
mod support;
