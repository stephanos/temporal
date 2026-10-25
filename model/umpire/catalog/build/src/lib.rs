//! Descriptor-driven catalog coverage and Rust API generation.
#![forbid(unsafe_code)]

pub mod claims;
pub mod codegen;
pub mod coverage;
pub mod mirrors;
pub mod services;
mod system;
pub mod wire_schema;

pub use system::System;
