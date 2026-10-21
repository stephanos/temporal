//! Executable controls for the attribute mirror surface.

#![allow(dead_code)]

use umpire_annotations::{Kind, validate};
use umpire_temporal_annotate::{Catalog, Inspect, kinds::*};

mod foundation {
    pub(crate) mod dc {
        pub(crate) struct Limit;
        pub(crate) struct Limits(pub(crate) u64);
        pub(crate) const MAX_ID_LENGTH: Limit = Limit;
        impl Limits {
            pub(crate) fn get(&self, _: &Limit) -> u64 {
                self.0
            }
        }
    }
}

mod validators {
    pub(crate) const REQUEST: &str = "validator.request";
}

#[derive(Clone, Default)]
struct Wire {
    id: String,
    text: String,
    other: String,
    count: i64,
    flag: bool,
    coefficient: f64,
}

/// Describes a request without retaining its values.
#[derive(Catalog)]
#[api(request = "test.Wire", type = Wire, covers = validators::REQUEST)]
struct Request {
    #[api(required, idempotency_key, accepts)]
    id: Uuid,
    /// Limits text by UTF-8 byte count.
    #[api(max_bytes = foundation::dc::MAX_ID_LENGTH, rejects = "too long")]
    text: Text,
    #[api(required, rejects)]
    #[api(also(uuid, accepts))]
    other: Text,
    #[api(gte = 1, rejects(NotFound, "missing"))]
    count: Counter,
    flag: Flag,
    #[api(accepts)]
    coefficient: Number,
}

#[test]
fn factories_validate_typed_attributes_and_keep_supplemental_intents()
-> Result<(), Box<dyn std::error::Error>> {
    let catalogs = Request::catalogs(&foundation::dc::Limits(4))?;
    assert_eq!(catalogs.len(), 2);
    let wire = Wire { text: "ééé".into(), count: 0, ..Wire::default() };
    let findings = validate(&wire, &catalogs[0]);
    assert!(
        findings.iter().any(|v| v.path == "test.Wire.text" && v.constraint == "string.max_bytes")
    );
    assert!(catalogs[0].fields().any(|field| matches!(field.kind, Kind::Flag)));
    assert!(catalogs[0].fields().any(|field| matches!(field.kind, Kind::Number(_))));
    let cases = catalogs[0].probe_cases();
    assert!(cases.iter().any(|case| case.description.contains("UTF-8")));
    Ok(())
}

#[test]
fn generated_identifier_fill_is_stable_and_preserves_supplied_values()
-> Result<(), Box<dyn std::error::Error>> {
    let mut wire = Wire::default();
    assert!(Inspect::<foundation::dc::Limits>::fill_ids(&mut wire, 7)?);
    let supplied = wire.id.clone();
    assert!(!Inspect::<foundation::dc::Limits>::fill_ids(&mut wire, 9)?);
    assert_eq!(wire.id, supplied);
    Ok(())
}

#[derive(Clone, PartialEq, prost::Message)]
struct Child {
    #[prost(string, tag = "1")]
    value: String,
}

#[derive(Clone, PartialEq, prost::Oneof)]
enum Choice {
    #[prost(message, tag = "1")]
    Result(Child),
    #[prost(message, tag = "2")]
    Failure(Child),
}

#[derive(Clone, Default)]
struct NestedWire {
    child: Option<Child>,
    children: Vec<Child>,
    choice: Option<Choice>,
    metadata: std::collections::BTreeMap<String, Vec<u8>>,
}

#[derive(Catalog)]
#[api(response = "test.Child", type = Child)]
struct ChildCatalog {
    #[api(max_bytes = 2)]
    value: Text,
}

#[derive(Catalog)]
#[api(response = "test.Nested", type = NestedWire)]
#[api(any_of(result, failure))]
struct NestedCatalog {
    child: Message<Child>,
    children: Vec<Message<Child>>,
    #[api(oneof(Result = "result", Failure = "failure"))]
    choice: OneOf<Choice>,
    metadata: Map<Text, Secret>,
}

#[test]
fn nested_inspection_visits_repeated_and_oneof_messages() -> Result<(), Box<dyn std::error::Error>>
{
    let child = Child { value: "too long".into() };
    let wire = NestedWire {
        child: Some(child.clone()),
        children: vec![child.clone()],
        choice: Some(Choice::Failure(child)),
        ..NestedWire::default()
    };
    let mut findings = Vec::new();
    wire.inspect(&foundation::dc::Limits(4), true, &mut findings)?;
    assert_eq!(findings.len(), 3);
    assert!(findings.iter().all(|violation| violation.path == "test.Child.value"));
    let catalog = NestedCatalog::catalog(&foundation::dc::Limits(4))?;
    assert!(catalog.fields().any(|field| field.path == "test.Nested.result"));
    assert!(catalog.fields().any(|field| field.path == "test.Nested.failure"));
    let mut wire = wire;
    wire.metadata.insert("encoding".into(), b"private".to_vec());
    catalog.redact(&mut wire)?;
    assert!(wire.metadata.is_empty());
    assert!(wire.child.is_some());
    Ok(())
}

#[derive(Clone, Copy, Debug, PartialEq, Eq, prost::Enumeration)]
enum State {
    Ready = 0,
    Done = 2,
}

#[derive(Clone, Default)]
struct TypedWire {
    state: i32,
    raw: Option<umpire_wasm::google::protobuf::Duration>,
    labels: Vec<String>,
}

#[derive(Catalog)]
#[api(response = "test.Typed", type = TypedWire)]
struct TypedCatalog {
    state: Enum<State>,
    raw: RawDuration,
    #[api(max_items = 1)]
    labels: Vec<Text>,
}

#[test]
fn enum_raw_duration_and_scalar_repeated_views_keep_their_semantics()
-> Result<(), Box<dyn std::error::Error>> {
    let wire = TypedWire {
        state: 1,
        raw: Some(umpire_wasm::google::protobuf::Duration { seconds: 1, nanos: -1 }),
        labels: vec!["one".into(), "two".into()],
    };
    let mut findings = Vec::new();
    wire.inspect(&foundation::dc::Limits(4), true, &mut findings)?;
    assert_eq!(findings.len(), 3);
    assert!(findings.iter().any(|v| v.constraint == "enum.defined_only"));
    assert!(findings.iter().any(|v| v.constraint == "duration.valid"));
    assert!(findings.iter().any(|v| v.constraint == "repeated.max_items"));
    Ok(())
}

#[derive(Clone, Default)]
struct OpenRequest {
    state: i32,
}

#[derive(Clone, Default)]
struct OpenResponse {
    state: i32,
}

#[derive(Catalog)]
#[api(request = "test.OpenRequest", type = OpenRequest)]
struct OpenRequestCatalog {
    #[api(defined_only = false)]
    state: Enum<State>,
}

#[derive(Catalog)]
#[api(response = "test.OpenResponse", type = OpenResponse)]
struct OpenResponseCatalog {
    #[api(defined_only = false)]
    state: Enum<State>,
}

#[test]
fn disabled_enum_constraint_accepts_unknown_values_and_needs_no_request_expectation()
-> Result<(), Box<dyn std::error::Error>> {
    let limits = foundation::dc::Limits(4);
    let response = OpenResponseCatalog::catalog(&limits)?;
    let findings = validate(&OpenResponse { state: 99 }, &response);
    let request = OpenRequestCatalog::catalog(&limits);
    assert_eq!(
        (findings.is_empty(), request.is_ok()),
        (true, true),
        "{:?}",
        request.as_ref().err()
    );
    assert_eq!(validate(&OpenRequest { state: 99 }, &request?), []);
    Ok(())
}
