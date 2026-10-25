//! The reviewed Rust expansion and its descriptor projection share authored declarations.

#[test]
fn executable_expansion_and_source_coverage() -> Result<(), Box<dyn std::error::Error>> {
    let declarations = umpire_temporal_annotate_syntax::catalogs(include_str!(
        "../../../../umpire/catalog/tests/controls.rs"
    ))?;
    assert_eq!(declarations.len(), 1);
    let declaration = &declarations[0];
    let coverage = declaration.coverage();
    assert_eq!(
        coverage.messages[1].fields,
        ["id", "timestamp", "flag", "labels", "opaque", "text", "choice_flag"]
    );
    assert_eq!(coverage.enums[0].values, ["READY", "DONE"]);
    assert_eq!(coverage.services[0].methods.len(), 3);
    let raw = umpire_temporal_annotate_syntax::catalogs(include_str!(
        "../../../../umpire/catalog/tests/raw_field.rs"
    ))?;
    assert_eq!(raw[0].coverage().messages[0].fields, ["type"]);
    let expansion = umpire_temporal_annotate_syntax::expand(declaration)?;
    let file = syn::parse2(expansion)?;
    insta::assert_snapshot!("typed-catalog", prettyplease::unparse(&file));
    Ok(())
}
