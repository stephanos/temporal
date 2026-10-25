//! Compiler diagnostics for invalid realization declarations.

#![forbid(unsafe_code)]

#[test]
fn realization_declarations_are_checked_before_expansion() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/realization_*.rs");
}
