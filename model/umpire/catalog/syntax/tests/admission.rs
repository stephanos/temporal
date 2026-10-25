//! Grammar admission controls independent of generated API compilation.

#[test]
fn derive_catalog_uses_shared_admission_and_discovery() -> Result<(), Box<dyn std::error::Error>> {
    let source = r#"
        #[derive(Catalog)]
        #[mirror(request = "temporal.api.example.v1.Request")]
        #[rule(required(identity), rejects = "identity")]
        struct Request {
            #[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = "identity")]
            identity: Text,
            #[mirror(ignored = "outside this model")]
            extra: Ignored,
        }
    "#;
    let catalogs = umpire_catalog_syntax::catalogs(source)?;
    assert_eq!(catalogs.len(), 1);
    assert_eq!(catalogs[0].coverage().messages[0].fields, ["identity", "extra"]);
    for replacement in ["#[mirror(ignored = \"\")]", "#[mirror(observed, decided)]"] {
        let invalid = source.replace("#[mirror(ignored = \"outside this model\")]", replacement);
        assert!(umpire_catalog_syntax::catalogs(&invalid).is_err());
    }
    let missing = source.replace(
        "#[rule(max_bytes = dc::MAX_ID_LENGTH, rejects = \"identity\")]",
        "#[rule(max_bytes = dc::MAX_ID_LENGTH)]",
    );
    assert!(umpire_catalog_syntax::catalogs(&missing).is_err());
    Ok(())
}

#[test]
fn derive_attributes_preserve_admission_errors() {
    for (declaration, expected) in [
        ("#[mirror(ignored = \"\")] value: Ignored", "valid literal reason"),
        ("value: Ignored", "explicit ignored reason"),
        ("#[mirror(observed)] value: Secret", "contradictory"),
        ("#[mirror(decided, observed)] value: Text", "contradictory"),
        ("#[mirror(monotone)] value: Text", "incompatible"),
        ("#[rule(required)] value: Text", "expected parentheses"),
        ("#[rule(required(value), accepts)] value: Text", "presence rules belong"),
        ("#[rule(max_bytes = true, accepts)] value: Text", "incompatible"),
        ("#[rule(unknown)] value: Text", "unknown catalog attribute"),
    ] {
        let source = format!(
            "#[derive(Catalog)] #[mirror(request = \"test.Request\")] struct Request {{ {declaration}, }}"
        );
        let error = umpire_catalog_syntax::catalogs(&source).err();
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
            #[mirror(response = "test.Response", supplemental)]
            struct Raw { value: Counter }
        }
    "#;
    assert!(umpire_catalog_syntax::catalogs(source)?.is_empty());
    Ok(())
}

#[test]
fn derive_validates_supplemental_intents_and_nested_map_coverage()
-> Result<(), Box<dyn std::error::Error>> {
    let invalid = r#"#[derive(Catalog)] #[mirror(request = "test.Request")]
        struct Request { #[mirror(also(uuid))] id: Text }"#;
    assert!(umpire_catalog_syntax::catalogs(invalid).is_err());
    let source = r#"#[derive(Catalog)] #[mirror(response = "test.Payload")]
        struct Payload { metadata: Map<Text, Secret> }"#;
    let coverage = umpire_catalog_syntax::catalogs(source)?[0].coverage();
    assert_eq!(coverage.messages.len(), 2);
    assert_eq!(coverage.messages[1].name, "test.Payload.MetadataEntry");
    assert_eq!(coverage.messages[1].fields, ["key", "value"]);
    Ok(())
}

#[test]
fn stacked_presence_expectations_may_differ_from_field_constraints()
-> Result<(), Box<dyn std::error::Error>> {
    let source = r#"#[derive(Catalog)]
        #[mirror(request = "test.Request")]
        #[rule(required(name), rejects = "missing")]
        struct Request {
            #[rule(max_bytes = 4, rejects = "too long")]
            name: Text,
        }"#;
    let input: syn::DeriveInput = syn::parse_str(source)?;
    let declaration = umpire_catalog_syntax::DeriveCatalog::parse(&input)?;
    assert!(format!("{:?}", declaration.fields[0].attributes.expected).contains("too long"));
    assert!(format!("{:?}", declaration.rules[0].expected).contains("missing"));
    Ok(())
}

#[test]
fn disabled_derive_enum_constraint_needs_no_expectation() {
    let source = r#"#[derive(Catalog)] #[mirror(request = "test.Request")]
        struct Request { #[rule(defined_only = false)] state: Enum<State> }"#;
    assert!(umpire_catalog_syntax::catalogs(source).is_ok());
}

#[test]
fn item_presence_algebra_and_helper_boundaries_are_explicit() {
    for rule in
        ["required(a, b)", "any_of(a, b)", "one_of(a, b)", "requires(a, b)", "excludes(a, b)"]
    {
        let source = format!(
            "#[derive(Catalog)] #[mirror(request = \"test.Request\")] #[rule({rule}, rejects)] struct Request {{ a: Text, b: Text }}"
        );
        assert!(umpire_catalog_syntax::catalogs(&source).is_ok(), "{source}");
    }
    for attribute in [
        "#[rule(covers = retired::VALIDATOR)]",
        "#[mirror(max_bytes = 1, accepts)]",
        "#[rule(observed)]",
        "#[rule(required(a))]",
        "#[rule(required(absent), accepts)]",
        "#[rule(conflicts_with(a, b), accepts)]",
    ] {
        let source = format!(
            "#[derive(Catalog)] #[mirror(request = \"test.Request\")] {attribute} struct Request {{ a: Text, b: Text }}"
        );
        assert!(umpire_catalog_syntax::catalogs(&source).is_err(), "{source}");
    }
}

#[test]
fn request_constraints_need_expectations() -> Result<(), Box<dyn std::error::Error>> {
    let error = syn::parse_str::<umpire_catalog_syntax::Catalog>(
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
    assert!(syn::parse_str::<umpire_catalog_syntax::Catalog>(declaration).is_err());
}

#[test]
fn disabled_repeated_constraint_needs_no_request_expectation() {
    assert!(
        syn::parse_str::<umpire_catalog_syntax::Catalog>(
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
        ("Text", "min_bytes = 2, max_bytes = 4", "=> Accepts", true),
        ("Identifier", "max_bytes = 4", "=> Accepts", true),
        ("Text", "max_bytes = 4", "", false),
        ("Text", "max_bytes = true", "=> Accepts", false),
        ("Bytes", "max_bytes = 4", "=> Accepts", false),
    ] {
        let source = format!(
            "catalog C; message request request: Request = \"test.Request\" {{ value: {kind}[{bound}] {expected}; }}"
        );
        assert_eq!(
            syn::parse_str::<umpire_catalog_syntax::Catalog>(&source).is_ok(),
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
        let result = syn::parse_str::<umpire_catalog_syntax::Catalog>(&source);
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

#[test]
fn catalog_expansion_accepts_a_caller_owned_context_and_api_root()
-> Result<(), Box<dyn std::error::Error>> {
    let input = syn::parse_str::<syn::DeriveInput>(
        r#"
        #[mirror(response = "example.v1.Response", context = crate::Limits, api = crate::api)]
        struct Response { value: Counter }
    "#,
    )?;
    let catalog = umpire_catalog_syntax::DeriveCatalog::parse(&input)?;
    let output = umpire_catalog_syntax::expand_derive(&catalog)?.to_string();
    assert!(output.contains("crate :: Limits"));
    assert!(output.contains("crate :: api :: example :: v1 :: Response"));
    assert!(!output.contains("foundation"));
    Ok(())
}

#[test]
fn native_primitive_wkt_mirrors_refuse_named_field_access() {
    for message in ["google.protobuf.Empty", "google.protobuf.DoubleValue"] {
        let source = format!("#[mirror(response = {message:?})] struct Native {{}}");
        let input = syn::parse_str(&source).unwrap_or_else(|error| panic!("{error}"));
        let error = umpire_catalog_syntax::DeriveCatalog::parse(&input)
            .err()
            .unwrap_or_else(|| panic!("native primitive must not generate a named-field mirror"));
        assert!(error.to_string().contains("no named Rust fields"), "{error}");
    }
}

const MIGRATED_REFUSALS: &[(&str, &str, &str, &str, bool)] = &[
    (
        "contradictory_roles",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(decided, observed)] id: Uuid }"#,
        "contradictory roles or kind/tag pair",
        "id",
        false,
    ),
    (
        "empty_ignored_reason",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(ignored = "")] opaque: Ignored }"#,
        "valid literal reason or positive consequence",
        "ignored",
        false,
    ),
    (
        "ignored_kind_type",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(ignored = "outside")] value: Uuid }"#,
        "opaque fields require Bytes or Secret",
        "value",
        false,
    ),
    (
        "ignored_rule",
        r#"#[mirror(response = "test.Wire")] #[rule(required(opaque))] struct Mirror { #[mirror(ignored = "opaque boolean")] opaque: Ignored }"#,
        "ignored fields cannot appear in rules",
        "opaque",
        false,
    ),
    (
        "incompatible_custom_kind",
        r#"catalog C; tag Visibility { kinds: [Timestamp]; consequence: EventuallyConsistent(3); } message response message: Message = "test.Message" { id: Uuid + Visibility; }"#,
        "custom tag Visibility is incompatible with kind Uuid",
        "Visibility",
        true,
    ),
    (
        "incompatible_tag",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(idempotency_key)] count: Counter }"#,
        "tag IdempotencyKey is incompatible with kind Counter",
        "idempotency_key",
        false,
    ),
    (
        "missing_custom_consequence",
        "catalog C; tag Visibility { kinds: [Timestamp]; }",
        "expected identifier",
        "}",
        true,
    ),
    (
        "missing_request_expectation",
        r#"#[mirror(request = "test.Wire")] struct Mirror { id: Uuid }"#,
        "request constraint requires an expectation",
        "id",
        false,
    ),
    (
        "missing_rule_expectation",
        r#"#[mirror(request = "test.Wire")] #[rule(any_of(id))] struct Mirror { #[rule(accepts)] id: Uuid }"#,
        "request rule requires an expectation",
        "any_of",
        false,
    ),
    (
        "missing_validator",
        r#"#[mirror(request = "test.Wire")] struct Mirror { #[rule(accepts, covers = retired::VALIDATOR)] id: Uuid }"#,
        "unknown catalog attribute",
        "covers",
        false,
    ),
    (
        "opaque_semantic_kind",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(ignored = "outside")] #[rule(max_len = 2)] value: Ignored }"#,
        "opaque fields require Bytes or Secret",
        "value",
        false,
    ),
    (
        "random_decided",
        r#"catalog C; message response wire: Wire = "test.Wire" { bytes: Random + Decided; }"#,
        "contradictory roles or kind/tag pair",
        "bytes",
        true,
    ),
    (
        "secret_observed",
        r#"#[mirror(response = "test.Wire")] struct Mirror { #[mirror(observed)] secret: Secret }"#,
        "contradictory roles or kind/tag pair",
        "secret",
        false,
    ),
    (
        "wildcard_action",
        r#"catalog C; actions bind: Action { Action::Client => { _ => Rpc("test.Service.Start"), }, }"#,
        "action must name a variant; wildcard bindings hide additions",
        "_",
        true,
    ),
    (
        "duplicate_supplemental",
        r#"#[mirror(response = "test.Wire", supplemental, supplemental)] struct Mirror {}"#,
        "Duplicate field `supplemental`",
        "supplemental",
        false,
    ),
    (
        "duplicate_context",
        r#"#[mirror(response = "test.Wire", context = First, context = Second)] struct Mirror {}"#,
        "Duplicate field `context`",
        "context = Second",
        false,
    ),
    (
        "duplicate_api",
        r#"#[mirror(response = "test.Wire", api = first, api = second)] struct Mirror {}"#,
        "Duplicate field `api`",
        "api = second",
        false,
    ),
    (
        "quoted_context",
        r#"#[mirror(response = "test.Wire", context = "crate::Context")] struct Mirror {}"#,
        "expected identifier",
        r#""crate::Context""#,
        false,
    ),
    (
        "supplemental_missing_covers",
        r#"#[mirror(request = "test.Wire", supplemental)] struct Supplemental { #[rule(accepts, covers = "retired")] value: Uuid }"#,
        "unknown catalog attribute",
        "covers",
        false,
    ),
    (
        "supplemental_missing_expectation",
        r#"#[mirror(request = "test.Wire", supplemental)] struct Supplemental { value: Uuid }"#,
        "request constraint requires an expectation",
        "value",
        false,
    ),
    (
        "supplemental_presence_missing_expectation",
        r#"#[mirror(request = "test.Wire", supplemental)] #[rule(required(value))] struct Supplemental { value: Text }"#,
        "request rule requires an expectation",
        "required",
        false,
    ),
    (
        "supplemental_blank_covers",
        r#"#[mirror(request = "test.Wire", supplemental)] struct Supplemental { #[api(accepts)] value: Uuid }"#,
        "request constraint requires an expectation",
        "value",
        false,
    ),
];

#[test]
fn migrated_ui_refusals_name_the_offending_token() -> Result<(), Box<dyn std::error::Error>> {
    for &(case, source, expected, token, grammar) in MIGRATED_REFUSALS {
        let error = if grammar {
            syn::parse_str::<umpire_catalog_syntax::Catalog>(source).err()
        } else {
            umpire_catalog_syntax::DeriveCatalog::parse(&syn::parse_str(source)?).err()
        }
        .ok_or_else(|| format!("{case}: expected refusal"))?;
        assert!(error.to_string().contains(expected), "{case}: {error}");
        let positions: Vec<_> = source
            .match_indices(token)
            .filter(|(offset, _)| {
                !token.chars().next().is_some_and(char::is_alphanumeric)
                    || (source[..*offset]
                        .chars()
                        .next_back()
                        .is_none_or(|c| !c.is_alphanumeric() && c != '_')
                        && source[*offset + token.len()..]
                            .chars()
                            .next()
                            .is_none_or(|c| !c.is_alphanumeric() && c != '_'))
            })
            .map(|(offset, _)| offset)
            .collect();
        let offset = if case == "ignored_rule" { positions.first() } else { positions.last() }
            .copied()
            .ok_or("offending token missing")?;
        let start = proc_macro2::LineColumn { line: 1, column: offset };
        let end = proc_macro2::LineColumn { line: 1, column: offset + token.len() };
        assert_eq!((error.span().start(), error.span().end()), (start, end), "{case}: {error}");
    }
    Ok(())
}
