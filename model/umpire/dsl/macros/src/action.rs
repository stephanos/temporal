use darling::FromMeta;
use proc_macro2::TokenStream;
use quote::quote;

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let syn::Data::Enum(data) = &input.data else {
        manyhow::bail!(input, "Action requires an enum");
    };
    let variants = &data.variants;
    let mut errors = manyhow::Emitter::new();
    let attrs: Vec<_> =
        input.attrs.iter().filter(|attr| attr.path().is_ident("internal")).collect();
    for attr in &attrs {
        if let Err(error) = darling::util::Flag::from_meta(&attr.meta) {
            errors.emit(error);
        }
    }
    let internal = !attrs.is_empty();
    for variant in variants {
        for attr in variant.attrs.iter().filter(|attr| attr.path().is_ident("internal")) {
            errors.emit(syn::Error::new_spanned(
                attr,
                "#[internal] belongs on the action enum, not a variant",
            ));
        }
        if !matches!(variant.fields, syn::Fields::Unit) {
            let message = if internal {
                "#[internal] requires a unit action variant"
            } else {
                "Action currently requires unit variants"
            };
            errors.emit(syn::Error::new_spanned(&variant.ident, message));
        }
    }
    errors.into_result()?;
    let name = &input.ident;
    let names: Vec<_> = variants.iter().map(|variant| &variant.ident).collect();
    let labels = names.iter().map(ToString::to_string);
    let spellings = names.iter().map(|variant| format!("{name}::{variant}"));
    let (impl_generics, type_generics, where_clause) = input.generics.split_for_impl();
    Ok(quote! {
        #[automatically_derived]
        impl #impl_generics ::core::clone::Clone for #name #type_generics #where_clause {
            fn clone(&self) -> Self { match *self { #(Self::#names => Self::#names),* } }
        }
        #[automatically_derived]
        impl #impl_generics ::core::cmp::PartialEq for #name #type_generics #where_clause {
            fn eq(&self, other: &Self) -> bool {
                ::core::mem::discriminant(self) == ::core::mem::discriminant(other)
            }
        }
        #[automatically_derived]
        impl #impl_generics ::core::cmp::Eq for #name #type_generics #where_clause {}
        #[automatically_derived]
        impl #impl_generics ::core::hash::Hash for #name #type_generics #where_clause {
            fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, state: &mut __UmpireHasher) {
                ::core::hash::Hash::hash(&::core::mem::discriminant(self), state);
            }
        }
        #[automatically_derived]
        impl #impl_generics ::core::fmt::Debug for #name #type_generics #where_clause {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match *self { #(Self::#names => out.write_str(#labels)),* }
            }
        }
        #[automatically_derived]
        impl #impl_generics ::core::fmt::Display for #name #type_generics #where_clause {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match *self { #(Self::#names => out.write_str(#spellings)),* }
            }
        }
        impl #impl_generics #name #type_generics #where_clause {
            /// Enumerates actions in their declaration order.
            pub fn actions() -> ::std::vec::Vec<Self> { ::std::vec![#(Self::#names),*] }
            /// Identifies environment-independent actions; this does not imply fairness.
            pub fn internal(&self) -> bool { #internal }
        }
    })
}
