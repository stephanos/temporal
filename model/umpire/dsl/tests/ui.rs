//! Machine states and actions must implement Display.

#[test]
fn missing_display() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/missing_display_*.rs");
}
