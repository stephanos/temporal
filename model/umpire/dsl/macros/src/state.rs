use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Field, Generics};
use umpire_front::declare;

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    expand_struct(input, false)
}

pub(crate) fn expand_world(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    expand_struct(input, true)
}

fn expand_struct(input: &syn::DeriveInput, world: bool) -> manyhow::Result<TokenStream> {
    let fields = declare::named_fields(input, world)?;
    let ranges = declare::start_ranges(&fields)?;
    let computed =
        world || input.attrs.iter().any(|attr| attr.path().is_ident("__umpire_computed_starts"));

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
    for (field, range) in fields.iter().zip(&ranges).filter(|_| !computed) {
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
    let starts = (!computed).then(|| {
        quote! {
            impl #start_impl #name #type_generics #start_where {
                /// Enumerates initial states in field order with defaults for unannotated fields.
                pub fn starts() -> ::std::vec::Vec<Self> {
                    let mut __umpire_starts = ::std::vec::Vec::new();
                    #starts
                    __umpire_starts
                }
            }
        }
    });
    let constructor = world.then(|| constructor(input, &fields, &names));
    let exported = exported_values(input, &fields, &names);

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
        #starts
        #constructor
        #exported
    })
}

/// Inside a declared-World `#[machine]`, exports the named field values to the source lifter's
/// binding.
fn exported_values(
    input: &syn::DeriveInput,
    fields: &[&Field],
    names: &[&syn::Ident],
) -> Option<TokenStream> {
    let name = &input.ident;
    input.attrs.iter().any(|attr| attr.path().is_ident("__umpire_world")).then(|| {
        let exportable = deferred(&input.generics, fields.iter().map(|field| &field.ty));
        let (implementation, type_generics, constraints) = exportable.split_for_impl();
        let keys = names.iter().map(ToString::to_string);
        quote! {
            #[doc(hidden)]
            impl #implementation ::umpire_dsl::Values for #name #type_generics #constraints {
                fn values(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)> {
                    ::std::vec![#((#keys, ::umpire_dsl::IntoValue::value(&self.#names))),*]
                }
            }
        }
    })
}

fn constructor(input: &syn::DeriveInput, fields: &[&Field], names: &[&syn::Ident]) -> TokenStream {
    let name = &input.ident;
    let visibility = &input.vis;
    let types = fields.iter().map(|field| &field.ty);
    let (implementation, arguments, constraints) = input.generics.split_for_impl();
    quote! {
        impl #implementation #name #arguments #constraints {
            /// Constructs the fixed parameters in field declaration order.
            #visibility fn new(#(#names: #types),*) -> Self { Self { #(#names),* } }
        }
    }
}

/// Requires each field type to export a value, as a higher-ranked bound so that a field without
/// one leaves the Model compiling and refuses only where its values are requested.
pub(crate) fn deferred<'a>(
    generics: &Generics,
    types: impl IntoIterator<Item = &'a syn::Type>,
) -> Generics {
    let mut generics = generics.clone();
    for ty in types {
        generics
            .make_where_clause()
            .predicates
            .push(syn::parse_quote!(for<'__umpire> #ty: ::umpire_dsl::IntoValue));
    }
    generics
}

fn bounded(generics: &Generics, fields: &[&Field], bound: &TokenStream) -> Generics {
    let mut generics = generics.clone();
    for field in fields {
        let ty = &field.ty;
        generics.make_where_clause().predicates.push(syn::parse_quote!(#ty: #bound));
    }
    generics
}
