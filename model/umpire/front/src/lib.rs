//! Read authored `#[machine]` Models once, for Umpire's native generation and its lifted back ends.
//!
//! [`declare`] reads the declarations both consumers need; its refusals are structural, and the
//! macros report them as compile errors. [`read`] then admits the bodies into a checked internal
//! tree; its refusals are soft, since a Model outside the subset still checks natively.
//!
//! The admitted integers are signed 32-bit. The collection subset admits canonical
//! `imbl::Vector<T>` fields whose elements are scalar booleans, admitted integers, or fieldless
//! enums. Defaults and `Vector::new/default()` are empty; `len`, `is_empty`, and ordered
//! `push_back` read as sequence length, emptiness and append. Growth needs an authored finite
//! length guard. Reads admit `get(index).copied().unwrap_or(default)` (including eager default
//! evaluation), `get(index).is_some()`, and `get(index).is_none()`. Indices must fit the
//! nonnegative signed 32-bit domain. Other collection operations, aliases, and unchecked unwraps
//! refuse by source location; this is a method table rather than general Rust collection
//! interpretation.
use proc_macro2::Span;
use quote::ToTokens;
mod body;
pub mod declare;
#[doc(hidden)]
pub mod expr;
pub mod path;
mod shape;
mod syntax;
#[doc(hidden)]
pub mod tree;
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
/// A located refusal of unsupported source behavior; the consumer names the file.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Diagnostic {
    /// One-based line.
    pub line: usize,
    /// One-based column.
    pub column: usize,
    /// Admission failure.
    pub message: String,
}
impl Diagnostic {
    pub(crate) fn at(node: &impl ToTokens, message: impl Into<String>) -> Self {
        Self::span(
            node.to_token_stream()
                .into_iter()
                .next()
                .map_or(Span::call_site(), |token| token.span()),
            message,
        )
    }
    pub(crate) fn span(span: Span, message: impl Into<String>) -> Self {
        let location = span.start();
        Self { line: location.line, column: location.column + 1, message: message.into() }
    }
}
impl From<syn::Error> for Diagnostic {
    fn from(error: syn::Error) -> Self {
        Self::span(error.span(), error.to_string())
    }
}
/// A refusal and the input it locates into.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Refusal {
    /// In the Model source.
    Model(Diagnostic),
    /// In the hook at this index of the hooks passed to [`read`].
    Hook(usize, Diagnostic),
}
/// The default concrete action catalog limit, equal to the native checker's.
pub const CATALOG_LIMIT: usize = 4096;

/// Admit a parsed source root and the golden hooks (`fixture!`) whose negative claims it carries.
pub fn read(ast: &syn::File, hooks: &[&syn::ItemMacro]) -> Result<Lifted, Refusal> {
    if !ast
        .items
        .iter()
        .any(|item| matches!(item, syn::Item::Mod(module) if syntax::has(&module.attrs, "machine")))
    {
        return Err(Refusal::Model(not_attribute(ast)));
    }
    let mut failed_hook = None;
    shape::parse(ast, hooks, &mut failed_hook).map_err(|diagnostic| match failed_hook {
        Some(index) => Refusal::Hook(index, diagnostic),
        None => Refusal::Model(diagnostic),
    })
}
/// The located refusal for source without a `#[machine]` Model: at a hand-written
/// `impl Machine` when there is one, otherwise at the start of the file.
fn not_attribute(ast: &syn::File) -> Diagnostic {
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
            implementation,
            format!(
                "the lifter reads #[machine] Models; `impl Machine for {}` implements Machine by hand",
                rust_type(&implementation.self_ty)
            ),
        ),
        None => Diagnostic::span(
            Span::call_site(),
            "the lifter reads #[machine] Models; this source declares none",
        ),
    }
}
/// A type as rustfmt would print it; token streams space every token apart.
fn rust_type(ty: &syn::Type) -> String {
    let alias = prettyplease::unparse(&syn::parse_quote!(type T = #ty;));
    alias.trim().trim_start_matches("type T = ").trim_end_matches(';').to_owned()
}
