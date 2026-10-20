//! Grammar admission controls independent of generated API compilation.

#[test]
fn request_constraints_need_expectations() -> Result<(), Box<dyn std::error::Error>> {
    let error = syn::parse_str::<umpire_temporal_annotate_syntax::Catalog>(
        r#"pub catalog C; message request request: Request = "test.Request" {
            id: Uuid;
        }"#,
    )
    .err()
    .ok_or("missing expectation must refuse")?;
    assert!(error.to_string().contains("expectation"), "{error}");
    Ok(())
}

#[test]
fn rpc_classification_names_the_bound_action_variant() {
    let declaration = r#"catalog C;
        service "test.Service" { Start: Action("Client::Stale"); }
        actions bind: Action { Action::Client => { Client::Start => Rpc("test.Service.Start"), }, }
    "#;
    assert!(syn::parse_str::<umpire_temporal_annotate_syntax::Catalog>(declaration).is_err());
}

#[test]
fn disabled_repeated_constraint_needs_no_request_expectation() {
    assert!(
        syn::parse_str::<umpire_temporal_annotate_syntax::Catalog>(
            r#"catalog C; message request request: Request = "test.Request" {
            values: Text[unique = false];
        }"#,
        )
        .is_ok()
    );
}
