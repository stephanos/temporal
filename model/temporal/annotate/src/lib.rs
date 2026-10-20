//! Typed exhaustive annotation catalogs.

pub use umpire_temporal_annotate_macros::annotate;

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
