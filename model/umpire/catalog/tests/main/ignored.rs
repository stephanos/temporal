//! Opaque ignored fields retain typed access and secret redaction.

#![forbid(unsafe_code)]

use std::collections::BTreeMap;
use umpire_catalog::*;

#[derive(Clone, Debug, Default, PartialEq, Eq)]
struct Envelope<T> {
    value: T,
}

#[derive(Clone, Debug, Default, PartialEq, Eq)]
struct OpaqueMessage {
    value: String,
}

#[derive(Clone, Debug, PartialEq, Eq)]
enum Choice {
    Opaque(OpaqueMessage),
    Text(String),
}

#[test]
fn ignored_projection_retains_payload_type_and_redacts_oneof()
-> Result<(), Box<dyn std::error::Error>> {
    let catalog = Catalog::response("example.Envelope")
        .ignored(IgnoredField::projected(
            "opaque",
            Kind::Secret,
            |message: &Envelope<Option<Choice>>| match &message.value {
                Some(Choice::Opaque(value)) => Some(value.clone()),
                _ => None,
            },
            |message, value| message.value = value.map(Choice::Opaque),
            "outside this model",
        ))
        .build()?;
    assert_eq!(
        catalog.fields().next().ok_or("missing projected field")?.rust_type,
        std::any::type_name::<Option<OpaqueMessage>>()
    );
    for choice in [
        Choice::Opaque(OpaqueMessage { value: "secret".into() }),
        Choice::Text("other secret".into()),
    ] {
        let mut message = Envelope { value: Some(choice) };
        let before = message.clone();
        fill(&mut message, &catalog, &mut Rng::new(7))?;
        assert_eq!(message, before);
        assert_eq!(catalog.redact(&mut message), Ok(()));
        assert_eq!(message.value, None);
    }
    Ok(())
}

fn ignored_catalog<T: Clone + Default + 'static>(
    kind: Kind,
    reason: &str,
) -> CatalogBuilder<Envelope<T>> {
    Catalog::request("example.Envelope").ignored(IgnoredField::new(
        "value",
        kind,
        |message: &Envelope<T>| &message.value,
        |message| &mut message.value,
        reason,
    ))
}

fn opaque_control<T>(value: &T) -> Result<(), Box<dyn std::error::Error>>
where
    T: Clone + Default + std::fmt::Debug + PartialEq + 'static,
{
    for kind in [Kind::Bytes(BytesRules::default()), Kind::Secret] {
        let catalog = ignored_catalog::<T>(kind.clone(), "outside this model").build()?;
        let descriptor = catalog.fields().next().ok_or("missing ignored field")?;
        assert_eq!(descriptor.path, "example.Envelope.value");
        assert_eq!(descriptor.rust_type, std::any::type_name::<T>());
        assert_eq!(descriptor.kind, kind);
        assert_eq!(
            descriptor.tags,
            [("Ignored".into(), Consequence::Ignored("outside this model".into()))]
        );
        let mut message = Envelope { value: value.clone() };
        assert_eq!(validate(&message, &catalog), []);
        let mut rng = Rng::new(7);
        let mut untouched = rng.clone();
        fill(&mut message, &catalog, &mut rng)?;
        assert_eq!(&message.value, value);
        assert_eq!(rng.next_u64(), untouched.next_u64());
        assert_eq!(catalog.redact(&mut message), Ok(()));
        assert_eq!(message.value, if kind == Kind::Secret { T::default() } else { value.clone() });
    }
    Ok(())
}

#[test]
fn ignored_opaque_fields_preserve_types_and_redact_secrets()
-> Result<(), Box<dyn std::error::Error>> {
    opaque_control(&true)?;
    opaque_control(&BTreeMap::from([("key".to_owned(), "value".to_owned())]))?;
    opaque_control(&OpaqueMessage { value: "opaque".into() })?;
    Ok(())
}

#[test]
fn ignored_opaque_fields_require_reasons_and_refuse_presence_dependencies() {
    for reason in ["", " \t\n"] {
        assert_eq!(
            ignored_catalog::<bool>(Kind::Bytes(BytesRules::default()), reason).build().err(),
            Some(AdmissionErrors(vec![AdmissionError::MissingIgnoreReason {
                path: "example.Envelope.value".into(),
            }]))
        );
    }
    assert_eq!(
        ignored_catalog::<bool>(Kind::Bytes(BytesRules::default()), "outside this model")
            .rule(Rule {
                presence: Presence::Required(vec!["value".into()]),
                expected: Some(Expected { expectation: Expectation::Accepts }),
            })
            .build()
            .err(),
        Some(AdmissionErrors(vec![AdmissionError::IgnoredRuleField(
            "example.Envelope.value".into()
        )]))
    );
}

#[test]
fn ignored_opaque_fields_refuse_semantic_kinds_and_byte_constraints() {
    for kind in [
        Kind::Uuid,
        Kind::Bytes(BytesRules { min_len: Some(1), max_len: None }),
        Kind::Bytes(BytesRules { min_len: None, max_len: Some(10) }),
    ] {
        assert_eq!(
            ignored_catalog::<bool>(kind.clone(), "outside this model").build().err(),
            Some(AdmissionErrors(vec![AdmissionError::UnsupportedKind {
                path: "value".into(),
                kind,
                rust_type: std::any::type_name::<bool>(),
            }]))
        );
    }
}
