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
mod body;
pub mod build;
mod emit;
mod expr;
mod names;
mod shape;
mod syntax;
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
/// A literal projected from compiler-checked authored fields.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Literal {
    /// Boolean value.
    Bool(bool),
    /// Integer checked against the independently admitted field type.
    Int(i128),
    /// Qualified variant of an independently admitted fieldless enum.
    Enum(String),
    /// Ordered elements of an admitted state sequence.
    Sequence(Vec<Literal>),
}
/// One selected World and its complete authored start states.
#[derive(Clone, Debug)]
pub struct Binding {
    /// Named World fields, validated against the original source declaration.
    pub world: Vec<(String, Literal)>,
    /// Complete named state fields for each authored start, in enumeration order.
    pub starts: Vec<Vec<(String, Literal)>>,
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
    /// Authored state struct name.
    pub state_name: String,
    /// Unit World constructor or declared World type path.
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
    /// Read a source root, with the golden hook from a `tests.rs` beside it.
    pub fn source(path: impl AsRef<Path>) -> Result<Lifted, Diagnostic> {
        let path = path.as_ref();
        let file = path.to_string_lossy();
        let read = |path: &Path, file: &str| {
            let source = std::fs::read_to_string(path)
                .map_err(|error| Diagnostic::span(file, Span::call_site(), error.to_string()))?;
            syn::parse_file(&source)
                .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))
        };
        let ast = read(path, &file)?;
        let tests_path = path.with_file_name("tests.rs");
        let tests_file = tests_path.to_string_lossy();
        let tests = if tests_path.is_file() { Some(read(&tests_path, &tests_file)?) } else { None };
        let mut hooks = Hook::all(&file, &ast);
        if let Some(tests) = &tests {
            hooks.extend(Hook::all(&tests_file, tests));
        }
        Self::lift(&file, &ast, &hooks)
    }
    /// Parse a source root without executing its Rust transitions.
    pub fn parse(file: &str, source: &str) -> Result<Lifted, Diagnostic> {
        let ast = syn::parse_file(source)
            .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))?;
        Self::lift(file, &ast, &Hook::all(file, &ast))
    }
    fn lift(file: &str, ast: &syn::File, hooks: &[Hook<'_>]) -> Result<Lifted, Diagnostic> {
        if !ast.items.iter().any(
            |item| matches!(item, syn::Item::Mod(module) if syntax::has(&module.attrs,"machine")),
        ) {
            return Err(Self::not_attribute(file, ast));
        }
        let lifted = shape::parse(file, ast, hooks)?;
        names::validate(file, ast, &lifted)?;
        Ok(lifted)
    }
    /// The located refusal for source without a `#[machine]` Model: at a hand-written
    /// `impl Machine` when there is one, otherwise at the start of the file.
    fn not_attribute(file: &str, ast: &syn::File) -> Diagnostic {
        let by_hand = ast.items.iter().find_map(|item| match item {
            syn::Item::Impl(implementation)
                if implementation
                    .trait_
                    .as_ref()
                    .and_then(|(path, _)| path.segments.last())
                    .is_some_and(|segment| segment.ident == "Machine") =>
            {
                Some(implementation)
            }
            _ => None,
        });
        match by_hand {
            Some(implementation) => Diagnostic::at(
                file,
                implementation,
                format!(
                    "the lifter reads #[machine] Models; `impl Machine for {}` implements Machine by hand",
                    rust_type(&implementation.self_ty)
                ),
            ),
            None => Diagnostic::span(
                file,
                Span::call_site(),
                "the lifter reads #[machine] Models; this source declares none",
            ),
        }
    }
}
/// A type as rustfmt would print it; token streams space every token apart.
fn rust_type(ty: &syn::Type) -> String {
    let alias = prettyplease::unparse(&syn::parse_quote!(type T = #ty;));
    alias.trim().trim_start_matches("type T = ").trim_end_matches(';').to_owned()
}
/// A golden hook (`fixture!`) and the file it is written in.
pub(crate) struct Hook<'a> {
    file: &'a str,
    mac: &'a syn::ItemMacro,
}
impl<'a> Hook<'a> {
    fn all(file: &'a str, ast: &'a syn::File) -> Vec<Self> {
        ast.items
            .iter()
            .filter_map(|item| match item {
                syn::Item::Macro(mac)
                    if mac.mac.path.segments.last().is_some_and(|s| s.ident == "fixture") =>
                {
                    Some(Self { file, mac })
                }
                _ => None,
            })
            .collect()
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
