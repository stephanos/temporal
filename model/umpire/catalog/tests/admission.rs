//! Compiler controls for exhaustive sidecar declarations.

#[test]
fn declarations_reject_unclassified_or_incompatible_shapes() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/admission-ui/*.rs");
}
