//! Upstream enum additions must refuse the actual realization conversion source.

#![forbid(unsafe_code)]

#[test]
fn phase_conversion_requires_each_generated_enum_variant() {
    trybuild::TestCases::new().compile_fail("tests/ui/phase_enum_addition.rs");
}
