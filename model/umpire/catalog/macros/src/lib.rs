//! Procedural entry point for checked annotation declarations.

#![forbid(unsafe_code)]

mod roots;

/// Derives an exhaustive typed catalog from an attribute mirror struct.
#[proc_macro_derive(Catalog, attributes(rule, mirror))]
pub fn derive_catalog(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let input = syn::parse_macro_input!(input as syn::DeriveInput);
    umpire_catalog_syntax::DeriveCatalog::parse(&input)
        .and_then(|catalog| umpire_catalog_syntax::expand_derive(&catalog))
        .map_or_else(syn::Error::into_compile_error, roots::resolve)
        .into()
}

/// Generate typed catalogs and exhaustive enum and model action bindings.
#[proc_macro]
pub fn annotate(input: proc_macro::TokenStream) -> proc_macro::TokenStream {
    let catalog = syn::parse_macro_input!(input as umpire_catalog_syntax::Catalog);
    umpire_catalog_syntax::expand(&catalog)
        .map_or_else(syn::Error::into_compile_error, roots::resolve)
        .into()
}
