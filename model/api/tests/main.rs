//! Integration controls for the Temporal API closure, consolidated into one test binary.

#![forbid(unsafe_code)]

#[path = "build_inputs.rs"]
mod build_inputs;
#[path = "codegen.rs"]
mod codegen;
#[path = "descriptor.rs"]
mod descriptor;
#[path = "services.rs"]
#[expect(
    clippy::module_inception,
    reason = "the former services binary keeps its inner generator module"
)]
mod services;
#[path = "../system.rs"]
#[allow(dead_code, reason = "tests read only the system identity")]
mod system;
#[path = "wire_schema.rs"]
mod wire_schema;
