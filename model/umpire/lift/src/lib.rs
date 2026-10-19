//! Independently lift authored Rust Models into a checked internal tree and TLA+.
//!
//! The collection subset admits canonical `imbl::Vector<T>` fields whose elements are scalar
//! booleans, admitted integers, or fieldless enums. Defaults and `Vector::new/default()` are empty;
//! `len`, `is_empty`, and ordered `push_back` map to TLA+ sequences. Growth needs an authored finite
//! length guard. Reads admit `get(index).copied().unwrap_or(default)` (including eager default
//! evaluation), `get(index).is_some()`, and `get(index).is_none()`. Indices must fit the nonnegative
//! signed 32-bit domain. Other collection operations, aliases, and unchecked unwraps refuse by
//! source location; this is a method table rather than general Rust collection interpretation.
use proc_macro2::Span;
use quote::ToTokens;
use std::{collections::BTreeMap, fmt, path::Path};
pub mod build;
mod emit;
mod leaf;
mod manual;
mod names;
mod parse;
mod tree;
pub use tree::Lifted;

/// Admitted value types for typed backend decoding.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Type {
    /// Boolean.
    Bool,
    /// Checked Rust integer.
    Int {
        /// Inclusive minimum.
        min: i128,
        /// Inclusive maximum.
        max: i128,
    },
    /// Fieldless enum encoded with qualified variant names.
    Enum {
        /// Authored type name.
        name: String,
        /// Qualified variant names in declaration order.
        variants: Vec<String>,
    },
    /// String.
    Text,
    /// Unit.
    Unit,
    /// Tuple.
    Tuple(Vec<Type>),
    /// Sequence.
    Seq(Box<Type>),
}
/// A located warning refusing unsupported source behavior.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Diagnostic {
    /// Source filename.
    pub file: String,
    /// One-based line.
    pub line: usize,
    /// One-based column.
    pub column: usize,
    /// Admission failure.
    pub message: String,
}
impl Diagnostic {
    pub(crate) fn at(file: &str, node: &impl ToTokens, message: impl Into<String>) -> Self {
        Self::span(
            file,
            node.to_token_stream()
                .into_iter()
                .next()
                .map_or(Span::call_site(), |token| token.span()),
            message,
        )
    }
    pub(crate) fn span(file: &str, span: Span, message: impl Into<String>) -> Self {
        let location = span.start();
        Self {
            file: file.into(),
            line: location.line,
            column: location.column + 1,
            message: message.into(),
        }
    }
}
impl fmt::Display for Diagnostic {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            out,
            "{}:{}:{}: warning: {}; TLA+ emission refused",
            self.file, self.line, self.column, self.message
        )
    }
}
impl std::error::Error for Diagnostic {}
/// An authored state field.
#[derive(Clone, Debug)]
pub struct Field {
    /// Authored name.
    pub name: String,
    /// Declared type.
    pub ty: Type,
}
/// An authored action.
#[derive(Clone, Debug)]
pub struct Action {
    /// Qualified authored name.
    pub name: String,
    /// TLA+ source relation operator.
    pub operator: String,
    /// Internal marker, which adds no fairness.
    pub internal: bool,
}
/// A property with its isolated TLC configuration.
#[derive(Clone, Debug)]
pub struct Property {
    /// Authored name.
    pub name: String,
    /// TLA+ operator.
    pub operator: String,
    /// Isolated configuration.
    pub cfg: String,
}
/// A progress property with its authored scheduling assumptions.
#[derive(Clone, Debug)]
pub struct Progress {
    /// Authored name.
    pub name: String,
    /// TLA+ property operator.
    pub operator: String,
    /// Scoped observer specification configuration.
    pub cfg: String,
    /// Admitted suffix actions.
    pub allowed: Vec<String>,
    /// Explicit fair actions only.
    pub fair: Vec<String>,
    /// Optional Model-transition bound.
    pub bound: Option<u64>,
}
/// Exact decoding and property metadata.
#[derive(Clone, Debug)]
pub struct Metadata {
    /// Whether the authored state has unnamed tuple fields.
    pub state_tuple: bool,
    /// Every authored source dependency read by the lifter.
    pub inputs: Vec<String>,
    /// Authored state struct name.
    pub state_name: String,
    /// Authored world constructor.
    pub world_path: String,
    /// State fields in declaration order.
    pub fields: Vec<Field>,
    /// Actions in catalog order.
    pub actions: Vec<Action>,
    /// Registered state claims.
    pub invariants: Vec<Property>,
    /// Registered action claims.
    pub postconditions: Vec<Property>,
    /// Progress claims.
    pub progress: Vec<Progress>,
    /// Supplemental negative fixture claims.
    pub negative: Vec<Property>,
    /// Exactly the fields projected out of state agreement.
    pub observer_fields: Vec<String>,
    /// Exact private field types.
    pub observer_types: BTreeMap<String, Type>,
    /// Outcome type.
    pub outcome_type: Type,
    /// Fact type.
    pub fact_type: Type,
}
/// Generated module and decoding contract.
#[derive(Clone, Debug)]
pub struct Module {
    /// Module name.
    pub name: String,
    /// TLA+ source.
    pub tla: String,
    /// Full-graph specification and safety checks.
    pub cfg: String,
    /// Projection metadata.
    pub metadata: Metadata,
}
/// Independent authored-source admission.
pub struct Lift;
impl Lift {
    /// Read a source root.
    pub fn source(path: impl AsRef<Path>) -> Result<Lifted, Diagnostic> {
        let path = path.as_ref();
        let file = path.to_string_lossy();
        let source = std::fs::read_to_string(path)
            .map_err(|error| Diagnostic::span(&file, Span::call_site(), error.to_string()))?;
        Self::parse(&file, &source)
    }
    /// Parse a source root without executing its Rust transitions.
    pub fn parse(file: &str, source: &str) -> Result<Lifted, Diagnostic> {
        let ast = syn::parse_file(source)
            .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))?;
        let lifted = if ast.items.iter().any(
            |item| matches!(item, syn::Item::Mod(module) if parse::has(&module.attrs,"machine")),
        ) {
            parse::parse(file, &ast)
        } else {
            manual::parse(file, &ast)
        }?;
        names::validate(file, &ast, &lifted)?;
        Ok(lifted)
    }
}
/// TLA+ emission from the internal typed tree.
pub struct Tla;
impl Tla {
    /// Emit a module with isolated property configurations.
    pub fn emit(lifted: &Lifted) -> Result<Module, Diagnostic> {
        emit::emit(lifted)
    }
}
