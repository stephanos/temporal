//! Semantic views preserve wire values and refuse malformed encodings.

#![forbid(unsafe_code)]

use prost::Message;
use umpire_catalog::{View, kinds};

#[test]
fn malformed_message_projection_is_a_value_free_refusal_without_mutation() {
    let original = Payload { data: vec![1], ..Default::default() };
    let mut value = Some(original.clone());
    let result = <_ as View<kinds::Message<Payload>>>::set(&mut value, Some(vec![0xff]));
    assert_eq!(result, Err(umpire_catalog::GenerationError::InvalidEncoding));
    assert_eq!(value, Some(original));
}

#[test]
fn malformed_generated_message_bytes_propagate_through_catalog_fill()
-> Result<(), Box<dyn std::error::Error>> {
    use umpire_catalog::{
        BytesRules, Catalog, Expectation, Expected, Field, FillError, GenerationError, Kind,
        ProbeError, Rng, fill,
    };
    let catalog = Catalog::response("example.Payload")
        .field(Field::try_projected(
            "value",
            Kind::Bytes(BytesRules { min_len: Some(1), max_len: Some(1) }),
            |value: &Option<Payload>| <_ as View<kinds::Message<Payload>>>::view(value),
            <Option<Payload> as View<kinds::Message<Payload>>>::set,
        ))
        .build()?;
    let mut value = None;
    assert!(matches!(
        fill(&mut value, &catalog, &mut Rng::new(1)),
        Err(FillError::Field { reason: GenerationError::InvalidEncoding, .. })
    ));
    assert!(value.is_none());
    let catalog = Catalog::request("example.Payload")
        .field(
            Field::try_projected(
                "value",
                Kind::Bytes(BytesRules { max_len: Some(0), ..Default::default() }),
                |value: &Option<Payload>| <_ as View<kinds::Message<Payload>>>::view(value),
                <Option<Payload> as View<kinds::Message<Payload>>>::set,
            )
            .expects(&Expected { expectation: Expectation::Accepts }),
        )
        .build()?;
    let baseline = Some(Payload::default());
    let case = catalog.probe_cases().remove(0);
    assert!(matches!(
        catalog.probe(&baseline, &case, &mut Rng::new(1)),
        Err(ProbeError::Generation(GenerationError::InvalidEncoding))
    ));
    assert_eq!(baseline, Some(Payload::default()));
    Ok(())
}

#[test]
fn message_setters_decode_projected_bytes() -> Result<(), umpire_catalog::GenerationError> {
    let payload = Payload { data: vec![1, 2, 3], ..Default::default() };
    let mut optional = None;
    <Option<Payload> as View<kinds::Message<Payload>>>::set(
        &mut optional,
        Some(payload.encode_to_vec()),
    )?;
    assert_eq!(optional, Some(payload.clone()));
    let mut repeated = Vec::new();
    <Vec<Payload> as View<kinds::Message<Payload>>>::set(
        &mut repeated,
        vec![payload.encode_to_vec()],
    )?;
    assert_eq!(repeated.as_slice(), std::slice::from_ref(&payload));
    let mut boxed = Some(Box::new(Payloads::default()));
    let value = Payloads { payloads: vec![payload] };
    <Option<Box<Payloads>> as View<kinds::Message<Payloads>>>::set(
        &mut boxed,
        Some(value.encode_to_vec()),
    )?;
    assert_eq!(boxed.as_deref(), Some(&value));
    Ok(())
}

#[test]
fn time_views_preserve_raw_values_and_write_back() -> Result<(), umpire_catalog::GenerationError> {
    use umpire_runtime::prost_types::Duration;
    let mut value = Some(Duration { seconds: i64::MAX, nanos: 0 });
    let raw = <_ as View<kinds::RawDuration>>::view(&value);
    assert_eq!(raw.map(|v| v.seconds), Some(i64::MAX));
    let normal = <_ as View<kinds::Duration>>::view(&value);
    assert_eq!(normal.map(|v| v.seconds), Some(3_153_600_000));
    <_ as View<kinds::Duration>>::set(
        &mut value,
        Some(umpire_catalog::Duration { seconds: 5, nanos: 12 }),
    )?;
    assert_eq!(value, Some(Duration { seconds: 5, nanos: 12 }));
    assert!(<_ as View<kinds::Duration>>::view(&Some(Duration::default())).is_none());
    Ok(())
}

#[test]
fn map_setters_decode_entries_and_projection_is_order_independent()
-> Result<(), umpire_catalog::GenerationError> {
    use std::collections::HashMap;
    type Metadata = kinds::Map<kinds::Text, kinds::Bytes>;
    let mut first = HashMap::from([("b".to_owned(), vec![2]), ("a".to_owned(), vec![1])]);
    let second = HashMap::from([("a".to_owned(), vec![1]), ("b".to_owned(), vec![2])]);
    assert_eq!(<_ as View<Metadata>>::view(&first), <_ as View<Metadata>>::view(&second));
    let entry = umpire_catalog::MetadataEntry { key: "new".into(), value: vec![4, 5] };
    <_ as View<Metadata>>::set(&mut first, vec![entry.encode_to_vec()])?;
    assert_eq!(first, HashMap::from([("new".to_owned(), vec![4, 5])]));
    Ok(())
}

#[test]
fn payload_map_round_trips_through_its_actual_carrier()
-> Result<(), umpire_catalog::GenerationError> {
    type Metadata = kinds::Map<kinds::Text, kinds::Secret>;
    let mut payload = Payload::default();
    let entry =
        umpire_catalog::MetadataEntry { key: "encoding".into(), value: b"binary/plain".to_vec() };
    <_ as View<Metadata>>::set(&mut payload.metadata, vec![entry.encode_to_vec()])?;
    assert_eq!(payload.metadata.get("encoding"), Some(&b"binary/plain".to_vec()));
    assert_eq!(<_ as View<Metadata>>::view(&payload.metadata), vec![entry.encode_to_vec()]);
    Ok(())
}
#[derive(Clone, PartialEq, prost::Message)]
struct Payload {
    #[prost(btree_map = "string, bytes", tag = "1")]
    metadata: std::collections::BTreeMap<String, Vec<u8>>,
    #[prost(bytes = "vec", tag = "2")]
    data: Vec<u8>,
}

#[derive(Clone, PartialEq, prost::Message)]
struct Payloads {
    #[prost(message, repeated, tag = "1")]
    payloads: Vec<Payload>,
}
