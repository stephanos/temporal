//! The public parameterized example is compiled and kept identical to its README block.

#[path = "readme/parameterized.rs"]
mod parameterized;

#[test]
fn readme_matches_compiled_parameterized_example() {
    let readme = include_str!("../../../../README.md");
    let section = readme.split("### Fixed parameters and computed starts\n").nth(1).unwrap_or("");
    let example = section.split("```rust\n").nth(1).unwrap_or("").split("\n```").next();
    assert_eq!(example, Some(include_str!("readme/parameterized.rs").trim_end()));
}
