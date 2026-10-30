//! Compile-time admission for context-specific request expectations.

#![forbid(unsafe_code)]

#[test]
fn supplemental_requests_require_expectations_and_reject_retired_helpers() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/supplemental_*.rs");
}

#[test]
fn generated_api_catalog_compiles_and_validates_actual_messages() {
    let tests = trybuild::TestCases::new();
    tests.pass("tests/ui-pass/generated_api.rs");
}
