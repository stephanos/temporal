//! Typed projections from wire carriers to annotation values.

pub use crate::kinds;
use crate::{EnumValidator, FieldValue, GenerationError, Kind};
use prost::Message;
use umpire_runtime::google::protobuf::{Duration, Timestamp};

/// The protobuf entry carrier used by generated metadata map catalogs.
#[derive(Clone, PartialEq, Message)]
pub struct MetadataEntry {
    /// Entry key.
    #[prost(string, tag = "1")]
    pub key: String,
    /// Entry value; its catalog kind decides whether it is secret.
    #[prost(bytes = "vec", tag = "2")]
    pub value: Vec<u8>,
}

/// Enumerates map entries in deterministic key order.
pub fn metadata_entries<'a>(
    values: impl IntoIterator<Item = (&'a String, &'a Vec<u8>)>,
) -> Vec<MetadataEntry> {
    let mut entries: Vec<_> = values
        .into_iter()
        .map(|(key, value)| MetadataEntry { key: key.clone(), value: value.clone() })
        .collect();
    entries.sort_by(|a, b| a.key.cmp(&b.key));
    entries
}

impl<K: 'static, V: 'static> View<kinds::Map<K, V>> for std::collections::BTreeMap<String, Vec<u8>>
where
    String: View<K>,
    Vec<u8>: View<V>,
{
    type Out = Vec<Vec<u8>>;
    fn view(&self) -> Self::Out {
        metadata_entries(self).iter().map(Message::encode_to_vec).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        let entries: Vec<MetadataEntry> =
            value.iter().map(|bytes| decode_message(bytes)).collect::<Result<_, _>>()?;
        *self = entries.into_iter().map(|entry| (entry.key, entry.value)).collect();
        Ok(())
    }
}

impl<K: 'static, V: 'static, S: std::hash::BuildHasher> View<kinds::Map<K, V>>
    for std::collections::HashMap<String, Vec<u8>, S>
where
    String: View<K>,
    Vec<u8>: View<V>,
{
    type Out = Vec<Vec<u8>>;
    fn view(&self) -> Self::Out {
        metadata_entries(self).iter().map(Message::encode_to_vec).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        let entries: Vec<MetadataEntry> =
            value.iter().map(|bytes| decode_message(bytes)).collect::<Result<_, _>>()?;
        self.clear();
        self.extend(entries.into_iter().map(|entry| (entry.key, entry.value)));
        Ok(())
    }
}

/// Typed alternative access used by generated oneof catalogs.
pub struct OneofProjection<M, T> {
    get: fn(&M) -> Option<&T>,
    set: fn(&mut M, Option<T>),
}
impl<M, T> OneofProjection<M, T> {
    /// Records exhaustive access to one selected alternative.
    pub fn new(get: fn(&M) -> Option<&T>, set: fn(&mut M, Option<T>)) -> Self {
        Self { get, set }
    }
}

/// Message fallback; scalar alternatives use their native field value instead.
pub trait MessageOneof<M> {
    /// Builds a projected message field with transactional decoding.
    fn field(
        self,
        path: &str,
        text: crate::StringRules,
        bytes: crate::BytesRules,
    ) -> crate::Field<M, Option<Vec<u8>>>;
}
impl<M: 'static, T: Message + Default + 'static> MessageOneof<M> for OneofProjection<M, T> {
    fn field(
        self,
        path: &str,
        _: crate::StringRules,
        bytes: crate::BytesRules,
    ) -> crate::Field<M, Option<Vec<u8>>> {
        crate::Field::try_projected(
            path,
            Kind::Bytes(bytes),
            move |message| (self.get)(message).map(Message::encode_to_vec),
            move |message, value: Option<Vec<u8>>| {
                let value = value.as_deref().map(decode_message).transpose()?;
                (self.set)(message, value);
                Ok(())
            },
        )
    }
}
macro_rules! scalar_oneof {
    ($ty:ty, $kind:ident, $rules:tt) => {
        impl<M: 'static> OneofProjection<M, $ty> {
            /// Builds a field retaining the scalar's native value and selected-default presence.
            pub fn field(
                self,
                path: &str,
                text: crate::StringRules,
                bytes: crate::BytesRules,
            ) -> crate::Field<M, Option<$ty>> {
                let _ = (&text, &bytes);
                crate::Field::try_projected(
                    path,
                    Kind::$kind((text, bytes).$rules),
                    move |message| (self.get)(message).cloned(),
                    move |message, value| {
                        (self.set)(message, value);
                        Ok(())
                    },
                )
            }
        }
    };
}
scalar_oneof!(String, Text, 0);
scalar_oneof!(Vec<u8>, Bytes, 1);
impl<C> crate::Inspect<C> for String {
    fn inspect(
        &self,
        _: &C,
        _: bool,
        _: &mut Vec<crate::Violation>,
    ) -> Result<(), crate::AdmissionErrors> {
        Ok(())
    }
}
impl<C> crate::Inspect<C> for u8 {
    fn inspect(
        &self,
        _: &C,
        _: bool,
        _: &mut Vec<crate::Violation>,
    ) -> Result<(), crate::AdmissionErrors> {
        Ok(())
    }
}

/// Protobuf map entry whose value is text.
#[derive(Clone, PartialEq, Message)]
pub struct TextEntry {
    /// Entry key.
    #[prost(string, tag = "1")]
    pub key: String,
    /// Entry text value.
    #[prost(string, tag = "2")]
    pub value: String,
}

/// Enumerates text map entries in deterministic key order.
pub fn text_entries<'a>(
    values: impl IntoIterator<Item = (&'a String, &'a String)>,
) -> Vec<TextEntry> {
    let mut entries: Vec<_> = values
        .into_iter()
        .map(|(key, value)| TextEntry { key: key.clone(), value: value.clone() })
        .collect();
    entries.sort_by(|a, b| a.key.cmp(&b.key));
    entries
}

impl<K: 'static, V: 'static> View<kinds::Map<K, V>> for std::collections::BTreeMap<String, String>
where
    String: View<K> + View<V>,
{
    type Out = Vec<Vec<u8>>;
    fn view(&self) -> Self::Out {
        text_entries(self).iter().map(Message::encode_to_vec).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        let entries: Vec<TextEntry> =
            value.iter().map(|bytes| decode_message(bytes)).collect::<Result<_, _>>()?;
        *self = entries.into_iter().map(|entry| (entry.key, entry.value)).collect();
        Ok(())
    }
}
impl<K: 'static, V: 'static, S: std::hash::BuildHasher> View<kinds::Map<K, V>>
    for std::collections::HashMap<String, String, S>
where
    String: View<K> + View<V>,
{
    type Out = Vec<Vec<u8>>;
    fn view(&self) -> Self::Out {
        text_entries(self).iter().map(Message::encode_to_vec).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        let entries: Vec<TextEntry> =
            value.iter().map(|bytes| decode_message(bytes)).collect::<Result<_, _>>()?;
        self.clear();
        self.extend(entries.into_iter().map(|entry| (entry.key, entry.value)));
        Ok(())
    }
}

/// Encodes a nested message for a generated projection.
pub fn encode_message<T: Message>(value: &T) -> Vec<u8> {
    value.encode_to_vec()
}
/// Decodes a nested message; generated oneof setters retain their value on malformed input.
pub fn decode_message<T: Message + Default>(bytes: &[u8]) -> Result<T, GenerationError> {
    T::decode(bytes).map_err(|_| GenerationError::InvalidEncoding)
}

/// A semantic projection with a real mutation back to the wire carrier.
pub trait View<K> {
    /// Annotation value validated and generated by the generic runtime.
    type Out: FieldValue;
    /// Projects a wire value without retaining it in diagnostics.
    fn view(&self) -> Self::Out;
    /// Replaces the wire value from its projection.
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError>;
}

/// Checks a generated enum using its fallible wire conversion.
pub fn enum_is_valid<T: TryFrom<i32>>(value: i32) -> bool {
    T::try_from(value).is_ok()
}

/// Builds a defined-only kind without assuming a prost enumeration trait.
pub fn defined_enum<T: TryFrom<i32> + Default + Into<i32> + 'static>() -> Kind {
    Kind::DefinedEnum(EnumValidator {
        name: std::any::type_name::<T>(),
        default: T::default().into(),
        is_valid: enum_is_valid::<T>,
    })
}

macro_rules! identity {
    ($ty:ty; $($kind:ty),*) => {$ (
        impl View<$kind> for $ty {
            type Out = Self;
            fn view(&self) -> Self::Out { self.clone() }
            fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> { *self = value; Ok(()) }
        }
    )*};
}
identity!(String; kinds::Text, kinds::Identifier, kinds::Secret);
identity!(Vec<u8>; kinds::Bytes, kinds::Secret);
identity!(bool; kinds::Flag);
identity!(f64; kinds::Number);
identity!(i32; kinds::Counter);
identity!(i64; kinds::Counter);
identity!(u32; kinds::Counter);
identity!(u64; kinds::Counter);

impl<T: 'static> View<kinds::Enum<T>> for i32 {
    type Out = i32;
    fn view(&self) -> Self::Out {
        *self
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value;
        Ok(())
    }
}

impl View<kinds::Uuid> for String {
    type Out = Option<String>;
    fn view(&self) -> Self::Out {
        uuid(self)
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.unwrap_or_default();
        Ok(())
    }
}

/// Normalizes the UUID forms accepted by the server while retaining invalid input for validation.
pub fn uuid(value: &str) -> Option<String> {
    if value.is_empty() {
        return None;
    }
    let value = if value.len() == 45
        && value.get(..9).is_some_and(|prefix| prefix.eq_ignore_ascii_case("urn:uuid:"))
    {
        &value[9..]
    } else {
        value
    };
    let value = if value.len() == 38 { value.get(1..37).unwrap_or(value) } else { value };
    if value.len() == 32 && value.bytes().all(|byte| byte.is_ascii_hexdigit()) {
        Some(format!(
            "{}-{}-{}-{}-{}",
            &value[..8],
            &value[8..12],
            &value[12..16],
            &value[16..20],
            &value[20..]
        ))
    } else {
        Some(value.to_owned())
    }
}

impl View<kinds::RawDuration> for Option<Duration> {
    type Out = Option<crate::Duration>;
    fn view(&self) -> Self::Out {
        self.as_ref().map(|v| crate::Duration { seconds: v.seconds, nanos: v.nanos })
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.map(|v| Duration { seconds: v.seconds, nanos: v.nanos });
        Ok(())
    }
}
impl View<kinds::Duration> for Option<Duration> {
    type Out = Option<crate::Duration>;
    fn view(&self) -> Self::Out {
        let value = <Self as View<kinds::RawDuration>>::view(self)?;
        if value.seconds == 0 && value.nanos == 0 {
            return None;
        }
        if (value.seconds > 0 && value.nanos < 0) || (value.seconds < 0 && value.nanos > 0) {
            return Some(value);
        }
        let total = (i128::from(value.seconds) * 1_000_000_000 + i128::from(value.nanos))
            .clamp(i128::from(i64::MIN), 3_153_600_000_000_000_000);
        Some(crate::Duration {
            seconds: i64::try_from(total / 1_000_000_000).unwrap_or(i64::MIN),
            nanos: i32::try_from(total % 1_000_000_000).unwrap_or(0),
        })
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        <Self as View<kinds::RawDuration>>::set(self, value)
    }
}
impl View<kinds::Timestamp> for Option<Timestamp> {
    type Out = Option<crate::Timestamp>;
    fn view(&self) -> Self::Out {
        self.as_ref().map(|v| crate::Timestamp { seconds: v.seconds, nanos: v.nanos })
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.map(|v| Timestamp { seconds: v.seconds, nanos: v.nanos });
        Ok(())
    }
}

impl<T: Message + Default> View<kinds::Message<T>> for Option<T> {
    type Out = Option<Vec<u8>>;
    fn view(&self) -> Self::Out {
        self.as_ref().map(Message::encode_to_vec)
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.as_deref().map(decode_message).transpose()?;
        Ok(())
    }
}
impl<T: Message + Default> View<kinds::Message<T>> for Option<Box<T>> {
    type Out = Option<Vec<u8>>;
    fn view(&self) -> Self::Out {
        self.as_ref().map(Message::encode_to_vec)
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.as_deref().map(decode_message).transpose()?.map(Box::new);
        Ok(())
    }
}
impl<T: Message + Default> View<kinds::Message<T>> for Vec<T> {
    type Out = Vec<Vec<u8>>;
    fn view(&self) -> Self::Out {
        self.iter().map(Message::encode_to_vec).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value.iter().map(|bytes| decode_message(bytes)).collect::<Result<_, _>>()?;
        Ok(())
    }
}
impl<T: Message + Default> View<kinds::Message<T>> for T {
    type Out = Vec<u8>;
    fn view(&self) -> Self::Out {
        self.encode_to_vec()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = decode_message(&value)?;
        Ok(())
    }
}
impl<K, T: View<K> + Default> View<Vec<K>> for Vec<T> {
    type Out = Vec<T::Out>;
    fn view(&self) -> Self::Out {
        self.iter().map(View::view).collect()
    }
    fn set(&mut self, value: Self::Out) -> Result<(), GenerationError> {
        *self = value
            .into_iter()
            .map(|value| {
                let mut item = T::default();
                item.set(value)?;
                Ok(item)
            })
            .collect::<Result<_, _>>()?;
        Ok(())
    }
}
