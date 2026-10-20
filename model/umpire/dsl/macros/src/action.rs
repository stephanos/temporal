use darling::FromMeta;
use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Expr, Field, Generics};

struct Expansion {
    clones: Vec<TokenStream>,
    equals: Vec<TokenStream>,
    hashes: Vec<TokenStream>,
    debug: Vec<TokenStream>,
    display: Vec<TokenStream>,
    catalogs: Vec<TokenStream>,
    builders: Vec<TokenStream>,
    constructors: Vec<TokenStream>,
    lazy_catalogs: Vec<TokenStream>,
}

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let syn::Data::Enum(data) = &input.data else {
        manyhow::bail!(input, "Action requires an enum");
    };
    let name = &input.ident;
    let mut errors = manyhow::Emitter::new();
    let mut flags = [false; 2];
    for (flag, enabled) in ["internal", "fault"].into_iter().zip(&mut flags) {
        for attr in input.attrs.iter().filter(|attr| attr.path().is_ident(flag)) {
            if let Err(error) = darling::util::Flag::from_meta(&attr.meta) {
                errors.emit(error);
            }
            *enabled = true;
        }
    }
    let [internal, fault] = flags;
    let mut clones = Vec::new();
    let mut equals = Vec::new();
    let mut hashes = Vec::new();
    let mut debug = Vec::new();
    let mut display = Vec::new();
    let mut catalogs = Vec::new();
    let mut builders = Vec::new();
    let mut constructors = Vec::new();
    let mut lazy_catalogs = Vec::new();
    for variant in &data.variants {
        let ident = &variant.ident;
        for attr in &variant.attrs {
            for flag in ["internal", "fault"] {
                if attr.path().is_ident(flag) {
                    errors.emit(syn::Error::new_spanned(
                        attr,
                        format!("#[{flag}] belongs on the action enum, not a variant"),
                    ));
                }
            }
        }
        let fields: Vec<_> = variant.fields.iter().collect();
        let bindings: Vec<_> =
            (0..fields.len()).map(|index| format_ident!("__umpire_field_{index}")).collect();
        let others: Vec<_> =
            (0..fields.len()).map(|index| format_ident!("__umpire_other_{index}")).collect();
        let pattern = pattern_for(variant, &bindings);
        let other_pattern = pattern_for(variant, &others);
        let values: Vec<_> =
            bindings.iter().map(|binding| quote!(::core::clone::Clone::clone(#binding))).collect();
        let value = construct(variant, &values);
        clones.push(quote!(#pattern => #value));
        equals.push(quote!((&#pattern, &#other_pattern) => true #(&& #bindings == #others)*));
        hashes.push(quote!(#pattern => { #(::core::hash::Hash::hash(#bindings, state);)* }));
        debug.push(format_arm(variant, &pattern, &bindings, &ident.to_string()));
        display.push(format_arm(variant, &pattern, &bindings, &format!("{name}::{ident}")));
        let values: Vec<_> =
            bindings.iter().map(|binding| quote!(::core::clone::Clone::clone(&#binding))).collect();
        let value = construct(variant, &values);
        let mut catalog = quote!(__umpire_actions.push(#value););
        for (index, (field, binding)) in fields.iter().zip(&bindings).enumerate().rev() {
            match domain(field, &format!("{name}::{ident}"), index) {
                Ok(domain) => {
                    let ty = &field.ty;
                    catalog =
                        quote!(for #binding in #domain { let #binding: #ty = #binding; #catalog });
                }
                Err(error) => errors.emit(error),
            }
        }
        catalogs.push(catalog);
        if let Ok((builder, constructor, lazy_catalog)) = lazy_domain(input, variant) {
            builders.push(builder);
            constructors.push(constructor);
            lazy_catalogs.push(lazy_catalog);
        }
    }
    errors.into_result()?;
    Ok(implementations(
        input,
        &data.variants.iter().flat_map(|variant| variant.fields.iter()).collect::<Vec<_>>(),
        internal,
        fault,
        Expansion {
            clones,
            equals,
            hashes,
            debug,
            display,
            catalogs,
            builders,
            constructors,
            lazy_catalogs,
        },
    ))
}

fn implementations(
    input: &syn::DeriveInput,
    fields: &[&Field],
    internal: bool,
    fault: bool,
    expansion: Expansion,
) -> TokenStream {
    let Expansion {
        clones,
        equals,
        hashes,
        debug,
        display,
        catalogs,
        builders,
        constructors,
        lazy_catalogs,
    } = expansion;
    let name = &input.ident;
    let (_, types, _) = input.generics.split_for_impl();
    let clone = bounded(&input.generics, fields, &quote!(::core::clone::Clone));
    let (clone_impl, _, clone_where) = clone.split_for_impl();
    let eq = bounded(&input.generics, fields, &quote!(::core::cmp::PartialEq));
    let (eq_impl, _, eq_where) = eq.split_for_impl();
    let total_eq = bounded(&input.generics, fields, &quote!(::core::cmp::Eq));
    let (total_eq_impl, _, total_eq_where) = total_eq.split_for_impl();
    let hash = bounded(&input.generics, fields, &quote!(::core::hash::Hash));
    let (hash_impl, _, hash_where) = hash.split_for_impl();
    let fmt = bounded(&input.generics, fields, &quote!(::core::fmt::Debug));
    let (fmt_impl, _, fmt_where) = fmt.split_for_impl();
    let lazy = bounded(&clone, fields, &quote!('static));
    let (lazy_impl, _, lazy_where) = lazy.split_for_impl();
    let actor =
        bounded(&clone, fields, &quote!(::core::cmp::Eq + ::core::hash::Hash + ::core::fmt::Debug));
    let (actor_impl, _, actor_where) = actor.split_for_impl();
    quote! {
        #[automatically_derived]
        impl #clone_impl ::core::clone::Clone for #name #types #clone_where {
            fn clone(&self) -> Self { match *self { #(#clones),* } }
        }
        #[automatically_derived]
        impl #eq_impl ::core::cmp::PartialEq for #name #types #eq_where {
            fn eq(&self, other: &Self) -> bool {
                #[allow(unreachable_patterns)]
                match (self, other) { #(#equals,)* _ => false }
            }
        }
        #[automatically_derived]
        impl #total_eq_impl ::core::cmp::Eq for #name #types #total_eq_where {}
        #[automatically_derived]
        impl #hash_impl ::core::hash::Hash for #name #types #hash_where {
            fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, state: &mut __UmpireHasher) {
                ::core::hash::Hash::hash(&::core::mem::discriminant(self), state);
                match *self { #(#hashes),* }
            }
        }
        #[automatically_derived]
        impl #fmt_impl ::core::fmt::Debug for #name #types #fmt_where {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match *self { #(#debug),* }
            }
        }
        #[automatically_derived]
        impl #fmt_impl ::core::fmt::Display for #name #types #fmt_where {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match *self { #(#display),* }
            }
        }
        impl #clone_impl #name #types #clone_where {
            /// Enumerates actions in declaration and input-domain order.
            pub fn actions() -> ::std::vec::Vec<Self> {
                let mut __umpire_actions = ::std::vec::Vec::new();
                #(#catalogs)*
                __umpire_actions
            }
            /// Identifies environment-independent actions; this does not imply fairness.
            pub fn internal(&self) -> bool { #internal }
            /// Identifies actions representing faults, independently of internal status.
            pub fn fault(&self) -> bool { #fault }
        }
        impl #actor_impl ::umpire_dsl::Actor for #name #types #actor_where {
            fn actions() -> ::std::vec::Vec<Self> { Self::actions() }
            fn internal(&self) -> bool { self.internal() }
            fn fault(&self) -> bool { self.fault() }
        }
        impl #lazy_impl #name #types #lazy_where {
            #(#constructors)*
            #[doc(hidden)]
            pub fn __umpire_path_actions() -> ::umpire_dsl::ActionDomain<Self> {
                ::umpire_dsl::ActionDomain::concat(::std::vec![#(#lazy_catalogs),*])
            }
        }
        #(#builders)*
    }
}

fn pattern_for(variant: &syn::Variant, bindings: &[syn::Ident]) -> TokenStream {
    let ident = &variant.ident;
    match &variant.fields {
        syn::Fields::Named(fields) => {
            let names = fields.named.iter().map(|field| &field.ident);
            quote!(Self::#ident { #(#names: ref #bindings),* })
        }
        syn::Fields::Unnamed(_) => quote!(Self::#ident(#(ref #bindings),*)),
        syn::Fields::Unit => quote!(Self::#ident),
    }
}

fn construct(variant: &syn::Variant, values: &[TokenStream]) -> TokenStream {
    construct_for(variant, values, &quote!(Self))
}

fn construct_for(
    variant: &syn::Variant,
    values: &[TokenStream],
    actor: &TokenStream,
) -> TokenStream {
    let ident = &variant.ident;
    match &variant.fields {
        syn::Fields::Named(fields) => {
            let names = fields.named.iter().map(|field| &field.ident);
            quote!(#actor::#ident { #(#names: #values),* })
        }
        syn::Fields::Unnamed(_) => quote!(#actor::#ident(#(#values),*)),
        syn::Fields::Unit => quote!(#actor::#ident),
    }
}

fn lazy_domain(
    input: &syn::DeriveInput,
    variant: &syn::Variant,
) -> syn::Result<(TokenStream, TokenStream, TokenStream)> {
    let actor = &input.ident;
    let action = &variant.ident;
    let visibility = &input.vis;
    let builder = format_ident!("__UmpirePath{actor}{action}");
    let constructor = format_ident!("__umpire_path_{action}");
    let fields: Vec<_> = variant.fields.iter().collect();
    let bindings: Vec<_> =
        (0..fields.len()).map(|index| format_ident!("__umpire_field_{index}")).collect();
    let names: Vec<_> = fields
        .iter()
        .enumerate()
        .map(|(index, field)| field.ident.clone().unwrap_or_else(|| format_ident!("field_{index}")))
        .collect();
    let types: Vec<_> = fields.iter().map(|field| &field.ty).collect();
    let setters: Vec<_> = names
        .iter()
        .map(|name| format_ident!("__umpire_input_{}", name.to_string().trim_start_matches("r#")))
        .collect();
    let domains: Vec<_> = fields
        .iter()
        .enumerate()
        .map(|(index, field)| {
            let value = domain(field, &format!("{actor}::{action}"), index)?;
            Ok(match value {
                Expr::Range(_) => quote!(::umpire_dsl::InputDomain::range(#value)),
                Expr::Array(array) => {
                    let values = &array.elems;
                    quote!(::umpire_dsl::InputDomain::values(::std::vec![#values]))
                }
                _ => unreachable!(),
            })
        })
        .collect::<syn::Result<_>>()?;
    let clone = bounded(&input.generics, &fields, &quote!(::core::clone::Clone + 'static));
    let (impls, generics, constraints) = clone.split_for_impl();
    let indices: Vec<_> = (0..fields.len()).map(syn::Index::from).collect();
    let positions: Vec<_> = bindings
        .iter()
        .zip(&indices)
        .rev()
        .map(|(binding, index)| {
            quote! {
                let #binding = {
                    let __umpire_count = __umpire_counts[#index].as_ref().copied().unwrap_or(0);
                    let __umpire_position = __umpire_index % __umpire_count;
                    __umpire_index /= __umpire_count;
                    __umpire_position
                };
            }
        })
        .collect();
    let count_len = fields.len();
    let values: Vec<_> = bindings.iter().map(|binding| quote!(#binding)).collect();
    let value = construct_for(variant, &values, &quote!(#actor));
    let builder_code = quote! {
        #[doc(hidden)]
        #visibility struct #builder #clone #constraints {
            #(#names: ::umpire_dsl::InputDomain<#types>,)*
            __umpire_actor: ::core::marker::PhantomData<#actor #generics>,
        }
        impl #impls #builder #generics #constraints {
            #(
                #[doc(hidden)]
                pub fn #setters(mut self, domain: ::core::option::Option<::umpire_dsl::InputDomain<#types>>) -> Self {
                    if let ::core::option::Option::Some(domain) = domain {
                        self.#names = domain;
                    }
                    self
                }
            )*
            #[doc(hidden)]
            pub fn finish(self) -> ::umpire_dsl::ActionDomain<#actor #generics> {
                let __umpire_counts: [::core::result::Result<usize, ::umpire_dsl::PatternError>; #count_len] = [#(self.#names.count()),*];
                let __umpire_count = if __umpire_counts.iter().any(|count| matches!(count, Ok(0))) {
                    Ok(0)
                } else {
                    __umpire_counts.iter().try_fold(1usize, |product, count| {
                        product.checked_mul(count.as_ref().copied().map_err(::core::clone::Clone::clone)?).ok_or(::umpire_dsl::PatternError::ExpansionOverflow)
                    })
                };
                ::umpire_dsl::ActionDomain::new(__umpire_count, move |mut __umpire_index| {
                    #(#positions)*
                    #(
                        let #bindings = self.#names.get(#bindings);
                    )*
                    #value
                })
            }
        }
    };
    let constructor_code = quote! {
        #[doc(hidden)]
        #[allow(non_snake_case)]
        pub fn #constructor() -> #builder #generics {
            #builder { #(#names: #domains,)* __umpire_actor: ::core::marker::PhantomData }
        }
    };
    Ok((builder_code, constructor_code, quote!(Self::#constructor().finish())))
}

fn format_arm(
    variant: &syn::Variant,
    pattern: &TokenStream,
    bindings: &[syn::Ident],
    label: &str,
) -> TokenStream {
    match &variant.fields {
        syn::Fields::Named(fields) => {
            let names = fields.named.iter().map(|field| {
                field
                    .ident
                    .as_ref()
                    .map(ToString::to_string)
                    .unwrap_or_default()
                    .trim_start_matches("r#")
                    .to_owned()
            });
            quote!(#pattern => out.debug_struct(#label)#(.field(#names, #bindings))*.finish())
        }
        syn::Fields::Unnamed(_) => {
            quote!(#pattern => out.debug_tuple(#label)#(.field(#bindings))*.finish())
        }
        syn::Fields::Unit => quote!(#pattern => out.write_str(#label)),
    }
}

fn bounded(generics: &Generics, fields: &[&Field], bound: &TokenStream) -> Generics {
    let mut generics = generics.clone();
    for field in fields {
        let ty = &field.ty;
        generics.make_where_clause().predicates.push(syn::parse_quote!(#ty: #bound));
    }
    generics
}

fn domain(field: &Field, action: &str, index: usize) -> syn::Result<Expr> {
    let field_name = field.ident.as_ref().map_or_else(|| index.to_string(), ToString::to_string);
    let context = format!("{action} field `{field_name}`");
    let mut attrs = field.attrs.iter().filter(|attr| attr.path().is_ident("inputs"));
    let attr = attrs.next().ok_or_else(|| {
        syn::Error::new_spanned(
            field,
            format!("{context} requires #[inputs] with a bounded range or finite list"),
        )
    })?;
    if let Some(duplicate) = attrs.next() {
        return Err(syn::Error::new_spanned(
            duplicate,
            format!("{context} has duplicate #[inputs] domains"),
        ));
    }
    let value = attr.parse_args::<Expr>()?;
    match &value {
        Expr::Range(range) if range.start.is_some() && range.end.is_some() => Ok(value),
        Expr::Array(_) => Ok(value),
        _ => Err(syn::Error::new_spanned(
            attr,
            format!(
                "{context} #[inputs] requires a bounded range or finite list, such as 0..=3 or [false, true]"
            ),
        )),
    }
}
