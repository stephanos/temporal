use darling::FromMeta;
use proc_macro2::TokenStream;
use quote::quote;

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let syn::Data::Enum(data) = &input.data else {
        manyhow::bail!(input, "Action requires an enum");
    };
    let variants = &data.variants;
    let mut errors = manyhow::Emitter::new();
    let mut internals = Vec::new();
    for variant in variants {
        let attrs: Vec<_> =
            variant.attrs.iter().filter(|attr| attr.path().is_ident("internal")).collect();
        for attr in &attrs {
            if let Err(error) = darling::util::Flag::from_meta(&attr.meta) {
                errors.emit(error);
            }
        }
        if !matches!(variant.fields, syn::Fields::Unit) {
            let message = if attrs.is_empty() {
                "Action currently requires unit variants"
            } else {
                "#[internal] requires a unit action variant"
            };
            errors.emit(syn::Error::new_spanned(&variant.ident, message));
        }
        if !attrs.is_empty() {
            internals.push(&variant.ident);
        }
    }
    errors.into_result()?;
    let name = &input.ident;
    let names: Vec<_> = variants.iter().map(|variant| &variant.ident).collect();
    let spellings = names.iter().map(|variant| format!("{name}::{variant}"));
    let (impl_generics, type_generics, where_clause) = input.generics.split_for_impl();
    let internal = if internals.is_empty() {
        quote!(false)
    } else {
        quote!(::core::matches!(self, #(Self::#internals)|*))
    };
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
        impl #impl_generics ::umpire_dsl::Spell for #name #type_generics #where_clause {
            fn spell(&self, out: &mut ::std::string::String) {
                match *self { #(Self::#names => out.push_str(#spellings)),* }
            }
        }
        impl #impl_generics #name #type_generics #where_clause {
            /// Enumerates actions in their declaration order.
            pub fn actions() -> ::std::vec::Vec<Self> { ::std::vec![#(Self::#names),*] }
            /// Identifies actions marked internal for weak fairness.
            pub fn internal(&self) -> bool { #internal }
        }
    })
}
