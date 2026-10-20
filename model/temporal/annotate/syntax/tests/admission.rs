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

#[test]
fn string_byte_constraints_require_typed_literals_and_request_expectations() {
    for (kind, bound, expected, valid) in [
        ("Text", "min_bytes = 2, max_bytes = 4", "=> Accepts covers \"validator\"", true),
        ("Identifier", "max_bytes = 4", "=> Accepts covers \"validator\"", true),
        ("Text", "max_bytes = 4", "", false),
        ("Text", "max_bytes = true", "=> Accepts covers \"validator\"", false),
        ("Bytes", "max_bytes = 4", "=> Accepts covers \"validator\"", false),
    ] {
        let source = format!(
            "catalog C; message request request: Request = \"test.Request\" {{ value: {kind}[{bound}] {expected}; }}"
        );
        assert_eq!(
            syn::parse_str::<umpire_temporal_annotate_syntax::Catalog>(&source).is_ok(),
            valid,
            "{source}"
        );
    }
}

#[test]
fn oneof_rules_require_known_nonignored_source_aliases_and_projection() {
    for (declaration, rule, expected) in [
        ("Bytes via Option<Vec<u8>> (|m| None, |m, value| ())", "one_of(result, failure)", None),
        ("Bytes", "", Some("typed projection")),
        (
            "Bytes via Option<Vec<u8>> (|m| None, |m, value| ())",
            "required(unknown)",
            Some("unknown rule field"),
        ),
        (
            "Bytes via Option<Vec<u8>> (|m| None, |m, value| ())",
            "required(value)",
            Some("unknown rule field"),
        ),
        ("Bytes + Ignored(\"opaque oneof\") opaque", "required(result)", Some("ignored fields")),
    ] {
        let rules = if rule.is_empty() { String::new() } else { format!("rules {{ {rule}; }}") };
        let source = format!(
            "catalog C; message response response: Response = \"example.Response\" {{ value: {declaration} variants {{ Choice::Result = \"result\", Choice::Failure = \"failure\", }}; {rules} }}"
        );
        let result = syn::parse_str::<umpire_temporal_annotate_syntax::Catalog>(&source);
        if let Some(expected) = expected {
            assert!(
                result.err().is_some_and(|error| error.to_string().contains(expected)),
                "{source}"
            );
        } else {
            assert!(result.is_ok(), "{source}: {result:?}");
        }
    }
}
