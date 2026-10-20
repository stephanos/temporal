//! Procedural entry point for checked annotation declarations.

/// Generate typed catalogs and exhaustive enum and model action bindings.
#[proc_macro]
pub fn annotate(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let catalog = syn::parse_macro_input!(input as umpire_temporal_annotate_syntax::Catalog);
    umpire_temporal_annotate_syntax::expand(&catalog)
        .unwrap_or_else(syn::Error::into_compile_error)
        .into()
}
