//! Compile-time admission for context-specific request expectations.

#[test]
fn supplemental_requests_require_expectations_and_reject_retired_helpers() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/supplemental_*.rs");
    tests.pass("tests/ui-pass/generated_api.rs");
}
