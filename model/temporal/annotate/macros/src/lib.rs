//! Procedural entry point for checked annotation declarations.

/// Derives an exhaustive typed catalog from an attribute mirror struct.
#[proc_macro_derive(Catalog, attributes(api))]
pub fn derive_catalog(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let input = syn::parse_macro_input!(input as syn::DeriveInput);
    umpire_temporal_annotate_syntax::DeriveCatalog::parse(&input)
        .and_then(|catalog| umpire_temporal_annotate_syntax::expand_derive(&catalog))
        .unwrap_or_else(syn::Error::into_compile_error)
        .into()
}

/// Generate typed catalogs and exhaustive enum and model action bindings.
#[proc_macro]
pub fn annotate(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let catalog = syn::parse_macro_input!(input as umpire_temporal_annotate_syntax::Catalog);
    umpire_temporal_annotate_syntax::expand(&catalog)
        .unwrap_or_else(syn::Error::into_compile_error)
        .into()
}
