//! Typed annotation catalogs, validation, generation and inspection.

mod black_box;
mod catalog;
mod gap;
mod kind;
mod negative;
pub mod types;
mod value;

pub use black_box::*;
pub use catalog::*;
pub use gap::*;
pub use kind::*;
pub use value::*;

pub use umpire_catalog_macros::{Catalog, annotate};

pub mod kinds;
pub mod limits;
pub mod views;
pub use views::*;

/// Recursive catalog inspection, parameterized by the caller's limit context.
pub trait Inspect<C> {
    /// Collects value-free findings, including nested messages.
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<Violation>,
    ) -> Result<(), AdmissionErrors>;
    /// Fills declared request identifiers without replacing supplied values.
    fn fill_ids(&mut self, _seed: u64) -> Result<bool, GenerationError> {
        Ok(false)
    }
}

impl<C, T: Inspect<C>> Inspect<C> for Option<T> {
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<Violation>,
    ) -> Result<(), AdmissionErrors> {
        if let Some(value) = self {
            value.inspect(context, response, output)?;
        }
        Ok(())
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, GenerationError> {
        self.as_mut().map_or(Ok(false), |value| value.fill_ids(seed))
    }
}
impl<C, T: Inspect<C>> Inspect<C> for Vec<T> {
    fn inspect(
        &self,
        context: &C,
        response: bool,
        output: &mut Vec<Violation>,
    ) -> Result<(), AdmissionErrors> {
        for value in self {
            value.inspect(context, response, output)?;
        }
        Ok(())
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, GenerationError> {
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
        output: &mut Vec<Violation>,
    ) -> Result<(), AdmissionErrors> {
        self.as_ref().inspect(context, response, output)
    }
    fn fill_ids(&mut self, seed: u64) -> Result<bool, GenerationError> {
        self.as_mut().fill_ids(seed)
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

mod transport;
pub use transport::{AnnotationResult, annotate_message, annotation_findings};
