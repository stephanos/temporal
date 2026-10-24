//! Compile-time kind/type witnesses for exhaustive catalog macros.

/// Implemented only for Rust types that can represent the marker's kind.
/// Generated wrapper types can use a local newtype or a typed projected accessor.
pub trait SupportsKind<K>: crate::FieldValue {}

macro_rules! kinds {
    ($($name:ident),*) => {$ (
        #[doc = concat!("Compile-time witness for the `", stringify!($name), "` kind.")]
        pub struct $name;
    )*};
}
kinds!(Uuid, Identifier, Timestamp, Duration, Counter, Random, Secret, Enum, Text, Bytes);

macro_rules! supports {
    ($ty:ty: $($kind:ty),*) => {$ (impl SupportsKind<$kind> for $ty {})*};
}
supports!(String: Uuid, Identifier, Random, Secret, Text);
supports!(Vec<u8>: Bytes, Random, Secret);
supports!(i32: Counter, Enum);
supports!(i64: Counter);
supports!(u32: Counter);
supports!(u64: Counter);
supports!(crate::Timestamp: Timestamp);
supports!(crate::Duration: Duration);

impl<T: SupportsKind<K>, K> SupportsKind<K> for Option<T> {}
impl<T: SupportsKind<K>, K> SupportsKind<K> for Vec<T> {}

/// Infers the field type and enforces its marker bound without reflecting on a message.
///
/// ```
/// use umpire_catalog::types::{self, Uuid};
/// types::check::<Uuid, _>(&String::new());
/// ```
///
/// ```compile_fail
/// use umpire_catalog::types::{self, Uuid};
/// types::check::<Uuid, _>(&42_i32);
/// ```
pub fn check<K, T: SupportsKind<K>>(_field: &T) {}

/// A carrier with repeated element semantics, excluding scalar byte strings.
pub trait SupportsRepeated: crate::FieldValue {}

impl<T: crate::FieldValue> SupportsRepeated for Vec<T> {}

/// Enforces repeated constraint applicability on a concrete typed field.
pub fn check_repeated<T: SupportsRepeated>(_field: &T) {}
