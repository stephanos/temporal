use serde::de::{
    DeserializeOwned, IntoDeserializer, Visitor,
    value::{MapDeserializer, SeqDeserializer},
};
use serde_json::Value;

pub(super) fn typed<T: DeserializeOwned>(value: Value) -> Result<T, String> {
    T::deserialize(Fields(value)).map_err(|error| error.to_string())
}

pub(super) fn zero<'de, T, D>(deserializer: D) -> Result<T, D::Error>
where
    T: serde::Deserialize<'de> + Default,
    D: serde::Deserializer<'de>,
{
    <Option<T> as serde::Deserialize>::deserialize(deserializer).map(Option::unwrap_or_default)
}

// Go's pinned MatchCaseSensitiveDelimiter(true) uses EqualFold. All declared
// config fields are ASCII; Kelvin K and long S are its two non-ASCII aliases.
fn equal_field(declared: &str, supplied: &str) -> bool {
    let mut supplied = supplied.chars();
    declared.chars().all(|expected| {
        supplied.next().is_some_and(|actual| {
            let actual = match actual {
                '\u{212a}' => 'K',
                '\u{017f}' => 'S',
                value => value,
            };
            expected.eq_ignore_ascii_case(&actual)
        })
    }) && supplied.next().is_none()
}

struct Fields(Value);
impl IntoDeserializer<'_, serde_json::Error> for Fields {
    type Deserializer = Self;
    fn into_deserializer(self) -> Self {
        self
    }
}
impl<'de> serde::Deserializer<'de> for Fields {
    type Error = serde_json::Error;

    fn deserialize_any<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        match self.0 {
            Value::Object(values) => visitor.visit_map(MapDeserializer::<_, Self::Error>::new(
                values.into_iter().map(|(key, value)| (key, Fields(value))),
            )),
            Value::Array(values) => visitor.visit_seq(SeqDeserializer::<_, Self::Error>::new(
                values.into_iter().map(Fields),
            )),
            value => value.into_deserializer().deserialize_any(visitor),
        }
    }

    fn deserialize_struct<V: Visitor<'de>>(
        self,
        _name: &'static str,
        fields: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value, Self::Error> {
        let values = match self.0 {
            Value::Object(values) => values,
            value => return Fields(value).deserialize_any(visitor),
        };
        let mut canonical = serde_json::Map::new();
        for (key, value) in values {
            let name = fields
                .iter()
                .find(|field| equal_field(field, &key))
                .map_or(key.as_str(), |field| *field);
            if canonical.insert(name.to_owned(), value).is_some() {
                return Err(<Self::Error as serde::de::Error>::custom(format!(
                    "duplicate configuration field {name}"
                )));
            }
        }
        Fields(Value::Object(canonical)).deserialize_any(visitor)
    }

    fn deserialize_option<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            visitor.visit_none()
        } else {
            visitor.visit_some(self)
        }
    }

    fn deserialize_newtype_struct<V: Visitor<'de>>(
        self,
        _name: &'static str,
        visitor: V,
    ) -> Result<V::Value, Self::Error> {
        visitor.visit_newtype_struct(self)
    }

    fn deserialize_enum<V: Visitor<'de>>(
        self,
        name: &'static str,
        variants: &'static [&'static str],
        visitor: V,
    ) -> Result<V::Value, Self::Error> {
        self.0
            .into_deserializer()
            .deserialize_enum(name, variants, visitor)
    }

    serde::forward_to_deserialize_any! {
        bool i8 i16 i32 i64 u8 u16 u32 u64 f32 f64 char str string bytes byte_buf
        unit unit_struct seq tuple tuple_struct map identifier ignored_any
    }
}
