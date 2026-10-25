//! Descriptor-driven catalog coverage and Rust API generation.
pub mod codegen;
pub mod coverage;
pub mod services;
mod system;
pub mod wire_schema;

pub use system::System;
