//! Generic catalog refusal controls without extension dependencies.

#[test]
fn catalog_requires_every_field_and_enum_variant() {
    let tests = trybuild::TestCases::new();
    tests.compile_fail("tests/ui/*.rs");
    tests.pass("tests/pass/prost_names.rs");
}
