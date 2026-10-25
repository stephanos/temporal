//! Read authored `#[machine]` Models once, for Umpire's native generation and its lifted back ends.
//!
//! [`declare`] reads the declarations both consumers need; its refusals are structural, and the
//! macros report them as compile errors. [`admit`] then admits the bodies into the checked
//! `umpire-ir` tree, which the macro embeds in the compiled Model; its refusals are soft, since a
//! Model outside the subset still checks natively.
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
use proc_macro2::{Span, TokenStream};
use quote::ToTokens;
mod body;
pub mod declare;
mod errors;
mod library;
pub mod path;
mod shape;
mod source;
mod syntax;
use errors::Errors;
pub use source::source;
#[doc(hidden)]
pub use umpire_ir::{Bounded, Diagnostic, Embedded, Lifted, REJECTIONS, Source, Type, expr, tree};

/// Locating a front-end refusal at a syntax node or span.
pub(crate) trait Locate: Sized {
    fn at(node: &impl ToTokens, message: impl Into<String>) -> Self;
    fn span(span: Span, message: impl Into<String>) -> Self;
    fn syn(error: syn::Error) -> Self;
}
impl Locate for Diagnostic {
    fn at(node: &impl ToTokens, message: impl Into<String>) -> Self {
        Self::span(
            node.to_token_stream()
                .into_iter()
                .next()
                .map_or(Span::call_site(), |token| token.span()),
            message,
        )
    }
    fn span(span: Span, message: impl Into<String>) -> Self {
        let location = span.start();
        Self { line: location.line, column: location.column + 1, message: message.into() }
    }
    fn syn(error: syn::Error) -> Self {
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

/// Admit a `#[machine]` module, as the macro receives it, into the tree, given its declarations
/// as [`declare::machine`] read them; a refusal here is a subset refusal.
pub fn admit(declared: &declare::Machine, module: &syn::ItemMod) -> Result<Lifted, Diagnostic> {
    shape::parse(declared, module).map(|(model, _)| model)
}

/// [`admit`], with the argument tokens of each `check!` the tree's `ExprKind::Check` indexes,
/// so the macro can spell their failures as `check!` does.
pub fn admit_checks(
    declared: &declare::Machine,
    module: &syn::ItemMod,
) -> Result<(Lifted, Vec<TokenStream>), Diagnostic> {
    shape::parse(declared, module)
}

/// Add the negative claims of golden hooks (`fixture!`) to an admitted Model, returning where
/// each added claim's name is written; a refusal names the failing hook's index.
pub fn supplement(
    model: &mut Lifted,
    hooks: &[&syn::ItemMacro],
) -> Result<Vec<Diagnostic>, (usize, Diagnostic)> {
    let environment = body::Environment::of(model);
    let initial = environment.field_env(false);
    let mut locations = Vec::new();
    for (index, hook) in hooks.iter().enumerate() {
        let claims =
            environment.supplemental(&hook.mac, &initial).map_err(|refusal| (index, refusal))?;
        for (claim, location) in claims {
            model.negative.push(claim);
            locations.push(location);
        }
    }
    Ok(locations)
}

/// Read a source file holding one `#[machine]` module as the macro would, with the golden hooks
/// whose negative claims it carries, and where those claims' names are written.
pub fn read(
    ast: &syn::File,
    hooks: &[&syn::ItemMacro],
) -> Result<(Lifted, Source, Vec<Diagnostic>), Refusal> {
    let modules: Vec<_> = ast
        .items
        .iter()
        .filter_map(|item| match item {
            syn::Item::Mod(module) if syntax::has(&module.attrs, "machine") => Some(module),
            _ => None,
        })
        .collect();
    let [module] = modules[..] else {
        return Err(Refusal::Model(if modules.is_empty() {
            unlifted(ast)
        } else {
            Diagnostic::at(ast, "expected exactly one inline #[machine] module")
        }));
    };
    let mut module = module.clone();
    let marker = module.attrs.iter().position(|attr| attr.path().is_ident("machine"));
    let args = match marker.map(|index| module.attrs.remove(index).meta) {
        Some(syn::Meta::List(list)) => list.tokens,
        _ => TokenStream::new(),
    };
    let declared =
        declare::machine(&args, &module).map_err(|error| Refusal::Model(Diagnostic::syn(error)))?;
    let mut model = admit(&declared, &module).map_err(Refusal::Model)?;
    let claims =
        supplement(&mut model, hooks).map_err(|(index, refusal)| Refusal::Hook(index, refusal))?;
    Ok((model, source(&module), claims))
}

/// The located refusal for source without a `#[machine]` Model: at a hand-written
/// `impl Machine` when there is one, otherwise at the start of the file.
#[must_use]
pub fn unlifted(ast: &syn::File) -> Diagnostic {
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

#[cfg(test)]
mod error_tests {
    #[test]
    fn errors_preserve_order_and_success_value() {
        let mut errors = super::Errors::default();
        for message in ["first", "second"] {
            errors.push(syn::Error::new(proc_macro2::Span::call_site(), message));
        }
        let Err(error) = errors.finish(42) else { panic!("errors were discarded") };
        assert_eq!(
            error.into_iter().map(|error| error.to_string()).collect::<Vec<_>>(),
            ["first", "second"]
        );
        assert_eq!(super::Errors::default().finish(42).ok(), Some(42));
    }
}
