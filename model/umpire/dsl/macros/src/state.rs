use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Attribute, Expr, ExprRange, Field, Generics};

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let syn::Data::Struct(data) = &input.data else {
        manyhow::bail!(input, "State requires a named-field struct");
    };
    let syn::Fields::Named(fields) = &data.fields else {
        manyhow::bail!(input, "State requires a named-field struct");
    };
    let fields: Vec<_> = fields.named.iter().collect();
    let ranges = start_ranges(&fields)?;

    let names: Vec<_> = fields
        .iter()
        .map(|field| {
            field
                .ident
                .as_ref()
                .ok_or_else(|| syn::Error::new_spanned(input, "State requires named fields"))
        })
        .collect::<Result<_, _>>()?;
    let bindings: Vec<_> =
        (0..fields.len()).map(|index| format_ident!("__umpire_field_{index}")).collect();
    let values = bindings.iter().map(|binding| quote!(::core::clone::Clone::clone(&#binding)));
    let mut starts = quote! { __umpire_starts.push(Self { #(#names: #values),* }); };
    for ((field, range), binding) in fields.iter().zip(&ranges).zip(&bindings).rev() {
        let ty = &field.ty;
        starts = if let Some(range) = range {
            quote! { for #binding in #range { #starts } }
        } else {
            quote! { let #binding: #ty = ::core::default::Default::default(); #starts }
        };
    }
    let name = &input.ident;
    let (_, type_generics, _) = input.generics.split_for_impl();
    let clone = bounded(&input.generics, &fields, &quote!(::core::clone::Clone));
    let (clone_impl, _, clone_where) = clone.split_for_impl();
    let eq = bounded(&input.generics, &fields, &quote!(::core::cmp::PartialEq));
    let (eq_impl, _, eq_where) = eq.split_for_impl();
    let total_eq = bounded(&input.generics, &fields, &quote!(::core::cmp::Eq));
    let (total_eq_impl, _, total_eq_where) = total_eq.split_for_impl();
    let hash = bounded(&input.generics, &fields, &quote!(::core::hash::Hash));
    let (hash_impl, _, hash_where) = hash.split_for_impl();
    let debug = bounded(&input.generics, &fields, &quote!(::core::fmt::Debug));
    let (debug_impl, _, debug_where) = debug.split_for_impl();
    let mut start_bounds = clone.clone();
    for (field, range) in fields.iter().zip(&ranges) {
        if range.is_none() {
            let ty = &field.ty;
            start_bounds
                .make_where_clause()
                .predicates
                .push(syn::parse_quote!(#ty: ::core::default::Default));
        }
    }
    let (start_impl, _, start_where) = start_bounds.split_for_impl();
    let label = name.to_string();
    let labels: Vec<_> =
        names.iter().map(|field| field.to_string().trim_start_matches("r#").to_owned()).collect();

    Ok(quote! {
        #[automatically_derived]
        impl #clone_impl ::core::clone::Clone for #name #type_generics #clone_where {
            fn clone(&self) -> Self {
                Self { #(#names: ::core::clone::Clone::clone(&self.#names)),* }
            }
        }
        #[automatically_derived]
        impl #eq_impl ::core::cmp::PartialEq for #name #type_generics #eq_where {
            fn eq(&self, other: &Self) -> bool { true #(&& self.#names == other.#names)* }
        }
        #[automatically_derived]
        impl #total_eq_impl ::core::cmp::Eq for #name #type_generics #total_eq_where {}
        #[automatically_derived]
        impl #hash_impl ::core::hash::Hash for #name #type_generics #hash_where {
            fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, state: &mut __UmpireHasher) {
                #(::core::hash::Hash::hash(&self.#names, state);)*
            }
        }
        #[automatically_derived]
        impl #debug_impl ::core::fmt::Debug for #name #type_generics #debug_where {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                out.debug_struct(#label)#(.field(#labels, &self.#names))*.finish()
            }
        }
        #[automatically_derived]
        impl #debug_impl ::core::fmt::Display for #name #type_generics #debug_where {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                ::core::fmt::Debug::fmt(self, out)
            }
        }
        impl #start_impl #name #type_generics #start_where {
            /// Enumerates initial states in field order with defaults for unannotated fields.
            pub fn starts() -> ::std::vec::Vec<Self> {
                let mut __umpire_starts = ::std::vec::Vec::new();
                #starts
                __umpire_starts
            }
        }
    })
}

fn bounded(generics: &Generics, fields: &[&Field], bound: &TokenStream) -> Generics {
    let mut generics = generics.clone();
    for field in fields {
        let ty = &field.ty;
        generics.make_where_clause().predicates.push(syn::parse_quote!(#ty: #bound));
    }
    generics
}

fn start_ranges(fields: &[&Field]) -> manyhow::Result<Vec<Option<ExprRange>>> {
    let mut errors = manyhow::Emitter::new();
    let mut ranges = Vec::new();
    for field in fields {
        let mut range = None;
        for attr in field.attrs.iter().filter(|attr| attr.path().is_ident("starts")) {
            match start_range(attr) {
                Ok(value) if range.is_none() => range = Some(value),
                Ok(_) => errors.emit(syn::Error::new_spanned(attr, "duplicate #[starts] domain")),
                Err(error) => errors.emit(error),
            }
        }
        ranges.push(range);
    }
    errors.into_result()?;
    Ok(ranges)
}

fn start_range(attr: &Attribute) -> syn::Result<ExprRange> {
    match attr.parse_args::<Expr>()? {
        Expr::Range(range) if range.start.is_some() && range.end.is_some() => Ok(range),
        value => Err(syn::Error::new_spanned(
            value,
            "#[starts] requires a bounded range, such as 0..=40",
        )),
    }
}
