//! Compile-time admission for context-specific request expectations.

#[test]
fn supplemental_requests_require_expected_behavior_and_provenance() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/supplemental_*.rs");
}
