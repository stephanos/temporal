//! Machine states and actions must provide their canonical spelling.

#[test]
fn missing_spell() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/missing_spell_*.rs");
}
