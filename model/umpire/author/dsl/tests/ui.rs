//! Compile-time refusals of the DSL's declarative macros and traits.

#[test]
fn missing_display() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/missing_display_*.rs");
}

#[test]
fn malformed_paths() {
    trybuild::TestCases::new().compile_fail("tests/ui/path_*.rs");
}
