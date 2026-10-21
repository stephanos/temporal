//! Typed exhaustive annotation catalogs.

#[cfg(test)]
extern crate self as umpire_temporal_annotate;

pub use umpire_temporal_annotate_macros::{Catalog, annotate};

pub mod kinds;
mod provenance;
pub mod views;
pub use provenance::validator_has_name;
pub use views::*;

/// Recursive catalog inspection, parameterized by the caller's limit context.
pub trait Inspect<C> {
    /// Collects value-free findings, including nested messages.
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<umpire_annotations::Violation>,
    ) -> Result<(), umpire_annotations::AdmissionErrors>;
    /// Fills declared request identifiers without replacing supplied values.
    fn fill_ids(&mut self, _seed: u64) -> Result<bool, umpire_annotations::GenerationError> {
        Ok(false)
    }
}

impl<C, T: Inspect<C>> Inspect<C> for Option<T> {
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<umpire_annotations::Violation>,
    ) -> Result<(), umpire_annotations::AdmissionErrors> {
        if let Some(value) = self {
            value.inspect(context, response, output)?;
        }
        Ok(())
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, umpire_annotations::GenerationError> {
        self.as_mut().map_or(Ok(false), |value| value.fill_ids(seed))
    }
}
impl<C, T: Inspect<C>> Inspect<C> for Vec<T> {
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<umpire_annotations::Violation>,
    ) -> Result<(), umpire_annotations::AdmissionErrors> {
        for value in self {
            value.inspect(context, response, output)?;
        }
        Ok(())
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, umpire_annotations::GenerationError> {
        let mut changed = false;
        for value in self {
            changed |= value.fill_ids(seed)?;
        }
        Ok(changed)
    }
}
impl<C, T: Inspect<C>> Inspect<C> for Box<T> {
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<umpire_annotations::Violation>,
    ) -> Result<(), umpire_annotations::AdmissionErrors> {
        self.as_ref().inspect(context, response, output)
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, umpire_annotations::GenerationError> {
        self.as_mut().fill_ids(seed)
    }
}

#[cfg(test)]
#[path = "../../foundation/mod.rs"]
pub mod foundation;

#[cfg(test)]
mod view_tests {
    use super::{View, kinds};
    use prost::Message;
    use umpire_temporal_api::temporal::api::common::v1::{Payload, Payloads};

    use super::foundation::views as conditional;

    #[test]
    fn validator_provenance_requires_a_non_whitespace_name() {
        for value in [
            "",
            " \t\n\r",
            "\u{85}\u{a0}\u{1680}",
            "\u{2000}\u{2001}\u{2002}\u{2003}\u{2004}\u{2005}\u{2006}\u{2007}\u{2008}\u{2009}\u{200a}",
            "\u{2028}\u{2029}\u{202f}\u{205f}\u{3000}",
        ] {
            assert!(!super::validator_has_name(value));
        }
        for value in ["validator", " \u{200a}validator\u{3000}", "\u{200b}"] {
            assert!(super::validator_has_name(value));
        }
    }

    #[test]
    fn malformed_message_projection_is_a_value_free_refusal_without_mutation() {
        let original = Payload { data: vec![1], ..Default::default() };
        let mut value = Some(original.clone());
        let result = <_ as View<kinds::Message<Payload>>>::set(&mut value, Some(vec![0xff]));
        assert_eq!(result, Err(umpire_annotations::GenerationError::InvalidEncoding));
        assert_eq!(value, Some(original));
    }

    #[test]
    fn malformed_generated_message_bytes_propagate_through_catalog_fill()
    -> Result<(), Box<dyn std::error::Error>> {
        use umpire_annotations::{
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
                .expects(&Expected {
                    expectation: Expectation::Accepts,
                    covers: "validator".into(),
                }),
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
    fn conditional_retry_setter_produces_a_real_invalid_coefficient()
    -> Result<(), umpire_annotations::GenerationError> {
        let mut retry = umpire_temporal_api::temporal::api::common::v1::RetryPolicy::default();
        conditional::backoff_coefficient::set(&mut retry, 0)?;
        assert_eq!(conditional::backoff_coefficient::view(&retry), 0);
        Ok(())
    }

    #[test]
    fn message_setters_decode_projected_bytes() -> Result<(), umpire_annotations::GenerationError> {
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
    fn time_views_preserve_raw_values_and_write_back()
    -> Result<(), umpire_annotations::GenerationError> {
        use umpire_wasm::google::protobuf::Duration;
        let mut value = Some(Duration { seconds: i64::MAX, nanos: 0 });
        let raw = <_ as View<kinds::RawDuration>>::view(&value);
        assert_eq!(raw.map(|v| v.seconds), Some(i64::MAX));
        let normal = <_ as View<kinds::Duration>>::view(&value);
        assert_eq!(normal.map(|v| v.seconds), Some(3_153_600_000));
        <_ as View<kinds::Duration>>::set(
            &mut value,
            Some(umpire_annotations::Duration { seconds: 5, nanos: 12 }),
        )?;
        assert_eq!(value, Some(Duration { seconds: 5, nanos: 12 }));
        assert!(<_ as View<kinds::Duration>>::view(&Some(Duration::default())).is_none());
        Ok(())
    }

    #[test]
    fn map_setters_decode_entries_and_projection_is_order_independent()
    -> Result<(), umpire_annotations::GenerationError> {
        use std::collections::HashMap;
        type Metadata = kinds::Map<kinds::Text, kinds::Bytes>;
        let mut first = HashMap::from([("b".to_owned(), vec![2]), ("a".to_owned(), vec![1])]);
        let second = HashMap::from([("a".to_owned(), vec![1]), ("b".to_owned(), vec![2])]);
        assert_eq!(<_ as View<Metadata>>::view(&first), <_ as View<Metadata>>::view(&second));
        let entry = super::MetadataEntry { key: "new".into(), value: vec![4, 5] };
        <_ as View<Metadata>>::set(&mut first, vec![entry.encode_to_vec()])?;
        assert_eq!(first, HashMap::from([("new".to_owned(), vec![4, 5])]));
        Ok(())
    }

    #[test]
    fn generated_payload_map_round_trips_through_its_actual_carrier()
    -> Result<(), umpire_annotations::GenerationError> {
        type Metadata = kinds::Map<kinds::Text, kinds::Secret>;
        let mut payload = Payload::default();
        let entry =
            super::MetadataEntry { key: "encoding".into(), value: b"binary/plain".to_vec() };
        <_ as View<Metadata>>::set(&mut payload.metadata, vec![entry.encode_to_vec()])?;
        assert_eq!(payload.metadata.get("encoding"), Some(&b"binary/plain".to_vec()));
        assert_eq!(<_ as View<Metadata>>::view(&payload.metadata), vec![entry.encode_to_vec()]);
        Ok(())
    }
}

/// An action's declared runtime binding.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum Binding {
    /// Fully qualified service method.
    Rpc(&'static str),
    /// Named revealing observation.
    Observed(&'static str),
    /// Explained environmental action.
    Environmental(&'static str),
}

/// A wire integer that is not a known protobuf enum discriminant.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct UnknownEnum {
    /// Unknown wire value.
    pub value: i32,
    /// Declared protobuf enum name.
    pub enum_name: &'static str,
}

impl std::fmt::Display for UnknownEnum {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "unknown {} enum integer {}", self.enum_name, self.value)
    }
}

impl std::error::Error for UnknownEnum {}
