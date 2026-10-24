use proc_macro2::TokenStream;

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    crate::state::expand_world(input)
}
