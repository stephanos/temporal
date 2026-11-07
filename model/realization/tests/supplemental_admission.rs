//! Compile-time admission for context-specific request expectations.

#![forbid(unsafe_code)]

use crate::compiler;

#[test]
fn supplemental_requests_require_expectations_and_reject_retired_helpers() {
    compiler::run(
        "supplemental_admission::supplemental_requests_require_expectations_and_reject_retired_helpers",
        || {
            let tests = trybuild::TestCases::new();
            tests.compile_fail("tests/ui/supplemental_*.rs");
        },
    );
}

/// The generated API consumer compiles as a module of this binary (see `main.rs`).
#[test]
fn generated_api_catalog_compiles_and_validates_actual_messages()
-> Result<(), Box<dyn std::error::Error>> {
    crate::generated_api::main()
}
