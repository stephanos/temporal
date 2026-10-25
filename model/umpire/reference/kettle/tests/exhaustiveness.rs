//! Compile refusals complement the real descriptor-addition checks.
#![forbid(unsafe_code)]

#[test]
fn kettle_exhaustiveness_names_every_missing_declaration() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/*.rs");
}
