//! Grammar admission controls independent of generated API compilation.

#[test]
fn derive_catalog_uses_shared_admission_and_discovery() -> Result<(), Box<dyn std::error::Error>> {
    let source = r#"
        #[derive(Catalog)]
        #[api(request = "temporal.api.example.v1.Request", covers = validators::REQUEST)]
        struct Request {
            #[api(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity", required)]
            identity: Text,
            #[api(ignored = "outside this model")]
            extra: Ignored,
        }
    "#;
    let catalogs = umpire_temporal_annotate_syntax::catalogs(source)?;
    assert_eq!(catalogs.len(), 1);
    assert_eq!(catalogs[0].coverage().messages[0].fields, ["identity", "extra"]);
    for replacement in ["#[api(ignored = \"\")]", "#[api(observed, decided)]"] {
        let invalid = source.replace("#[api(ignored = \"outside this model\")]", replacement);
        assert!(umpire_temporal_annotate_syntax::catalogs(&invalid).is_err());
    }
    let missing = source.replace(
        "#[api(max_bytes = dc::MAX_ID_LENGTH, rejects = \"identity\", required)]",
        "#[api(max_bytes = dc::MAX_ID_LENGTH)]",
    );
    assert!(umpire_temporal_annotate_syntax::catalogs(&missing).is_err());
    Ok(())
}

#[test]
fn derive_attributes_preserve_admission_errors() {
    for (declaration, expected) in [
        ("#[api(ignored = \"\")] value: Ignored", "valid literal reason"),
        ("value: Ignored", "explicit ignored reason"),
        ("#[api(observed)] value: Secret", "contradictory"),
        ("#[api(decided, observed)] value: Text", "contradictory"),
        ("#[api(monotone)] value: Text", "incompatible"),
        ("#[api(required)] value: Text", "expectation"),
        ("#[api(requires = absent, accepts)] value: Text", "unknown rule field"),
        ("#[api(max_bytes = true, accepts)] value: Text", "incompatible"),
        ("#[api(unknown)] value: Text", "unknown api attribute"),
    ] {
        let source = format!(
            "#[derive(Catalog)] #[api(request = \"test.Request\", covers = validators::REQUEST)] struct Request {{ {declaration}, }}"
        );
        let error = umpire_temporal_annotate_syntax::catalogs(&source).err();
        let Some(error) = error else { panic!("expected admission failure: {source}") };
        assert!(error.to_string().contains(expected), "{source}: {error}");
    }
}

#[test]
fn supplemental_mirrors_do_not_claim_descriptor_coverage() -> Result<(), Box<dyn std::error::Error>>
{
    let source = r#"
        mod nested {
            #[derive(Catalog)]
            #[api(response = "test.Response", supplemental)]
            struct Raw { value: Counter }
        }
    "#;
    assert!(umpire_temporal_annotate_syntax::catalogs(source)?.is_empty());
    Ok(())
}

#[test]
fn derive_validates_supplemental_intents_and_nested_map_coverage()
-> Result<(), Box<dyn std::error::Error>> {
    let invalid = r#"#[derive(Catalog)] #[api(request = "test.Request")]
        struct Request { #[api(also(uuid))] id: Text }"#;
    assert!(umpire_temporal_annotate_syntax::catalogs(invalid).is_err());
    let source = r#"#[derive(Catalog)] #[api(response = "test.Payload")]
        struct Payload { metadata: Map<Text, Secret> }"#;
    let coverage = umpire_temporal_annotate_syntax::catalogs(source)?[0].coverage();
    assert_eq!(coverage.messages.len(), 2);
    assert_eq!(coverage.messages[1].name, "test.Payload.MetadataEntry");
    assert_eq!(coverage.messages[1].fields, ["key", "value"]);
    Ok(())
}

#[test]
fn stacked_presence_expectations_may_differ_from_field_constraints()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r#"#[derive(Catalog)]
        #[api(request = "test.Request", covers = validators::REQUEST)]
        struct Request {
            #[api(max_bytes = 4, rejects = "too long")]
            #[api(required, rejects = "missing")]
            name: Text,
        }"#;
    let input: syn::DeriveInput = syn::parse_str(source)?;
    let declaration = umpire_temporal_annotate_syntax::DeriveCatalog::parse(&input)?;
    assert!(format!("{:?}", declaration.fields[0].attributes.expected).contains("too long"));
    assert!(format!("{:?}", declaration.rules[0].expected).contains("missing"));
    Ok(())
}

#[test]
fn disabled_derive_enum_constraint_needs_no_expectation() {
    let source = r#"#[derive(Catalog)] #[api(request = "test.Request")]
        struct Request { #[api(defined_only = false)] state: Enum<State> }"#;
    assert!(umpire_temporal_annotate_syntax::catalogs(source).is_ok());
}

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
