use serde::de::{
    DeserializeOwned, IntoDeserializer, Visitor,
    value::{MapDeserializer, SeqDeserializer},
};
use serde_json::Value;

pub(super) fn typed<T: DeserializeOwned>(value: Value) -> Result<T, String> {
    T::deserialize(Fields(value)).map_err(|error| error.to_string())
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
macro_rules! null_numbers {
    ($($($method:ident)+ => $visit:ident($zero:expr);)*) => {$($(
        fn $method<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
            if self.0.is_null() { visitor.$visit($zero) } else { self.deserialize_any(visitor) }
        }
    )*)*};
}

impl<'de> serde::Deserializer<'de> for Fields {
    type Error = serde_json::Error;

    fn deserialize_any<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        match self.0 {
            Value::Object(values) => {
                let mut values = MapDeserializer::<_, Self::Error>::new(
                    values.into_iter().map(|(key, value)| (key, Fields(value))),
                );
                let result = visitor.visit_map(&mut values)?;
                values.end()?;
                Ok(result)
            }
            Value::Array(values) => {
                let mut values =
                    SeqDeserializer::<_, Self::Error>::new(values.into_iter().map(Fields));
                let result = visitor.visit_seq(&mut values)?;
                values.end()?;
                Ok(result)
            }
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
            Value::Null => serde_json::Map::new(),
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
        for field in fields {
            canonical.entry((*field).to_owned()).or_insert(Value::Null);
        }
        Fields(Value::Object(canonical)).deserialize_any(visitor)
    }

    fn deserialize_seq<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            Fields(Value::Array(Vec::new())).deserialize_any(visitor)
        } else {
            self.deserialize_any(visitor)
        }
    }

    fn deserialize_map<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            Fields(Value::Object(serde_json::Map::new())).deserialize_any(visitor)
        } else {
            self.deserialize_any(visitor)
        }
    }

    fn deserialize_tuple<V: Visitor<'de>>(
        self,
        length: usize,
        visitor: V,
    ) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            Fields(Value::Array(vec![Value::Null; length])).deserialize_any(visitor)
        } else {
            self.deserialize_any(visitor)
        }
    }

    fn deserialize_tuple_struct<V: Visitor<'de>>(
        self,
        _name: &'static str,
        length: usize,
        visitor: V,
    ) -> Result<V::Value, Self::Error> {
        self.deserialize_tuple(length, visitor)
    }

    fn deserialize_bool<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            visitor.visit_bool(false)
        } else {
            self.deserialize_any(visitor)
        }
    }

    fn deserialize_str<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        self.deserialize_string(visitor)
    }

    fn deserialize_string<V: Visitor<'de>>(self, visitor: V) -> Result<V::Value, Self::Error> {
        if self.0.is_null() {
            visitor.visit_string(String::new())
        } else {
            self.deserialize_any(visitor)
        }
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

    null_numbers! {
        deserialize_i8 deserialize_i16 deserialize_i32 deserialize_i64 => visit_i64(0);
        deserialize_u8 deserialize_u16 deserialize_u32 deserialize_u64 => visit_u64(0);
        deserialize_f32 deserialize_f64 => visit_f64(0.0);
    }
    serde::forward_to_deserialize_any! {
        i128 u128 char bytes byte_buf unit unit_struct identifier ignored_any
    }
}
