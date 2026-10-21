use darling::FromMeta;
use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Expr, Field, Generics};

#[derive(Default)]
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
    world_catalogs: Vec<TokenStream>,
    retained_catalogs: Vec<TokenStream>,
    labels: Vec<String>,
    finite: Vec<syn::Type>,
}

#[derive(Clone)]
enum FieldDomain {
    Explicit(Expr),
    Finite,
    World(syn::Path),
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
    let world = world_context(input, &mut errors);
    let [internal, fault] = flags;
    let mut expansion = Expansion::default();
    let mut variant_domains = Vec::new();
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
            if attr.path().is_ident("domain") {
                errors.emit(syn::Error::new_spanned(
                    attr,
                    "#[domain] belongs on one variant field, not the variant",
                ));
            }
        }
        expansion.value_arms(name, variant);
        let fields: Vec<_> = variant.fields.iter().collect();
        let mut domains = Vec::new();
        for (index, field) in fields.iter().enumerate() {
            match domain(field, &format!("{name}::{ident}"), index, world.is_some()) {
                Ok(domain) => {
                    if matches!(domain, FieldDomain::Finite) {
                        expansion.finite.push(field.ty.clone());
                    }
                    domains.push(domain);
                }
                Err(error) => errors.emit(error),
            }
        }
        if domains.len() == fields.len() {
            variant_domains.push(domains);
        }
    }
    errors.into_result()?;
    let dependent = world.is_some()
        && variant_domains.iter().flatten().any(|domain| matches!(domain, FieldDomain::World(_)));
    let world = world.filter(|_| dependent);
    for (variant, domains) in data.variants.iter().zip(&variant_domains) {
        expansion.domains(input, variant, domains, world.as_ref());
    }
    let fields: Vec<_> = data.variants.iter().flat_map(|variant| variant.fields.iter()).collect();
    Ok(match world {
        Some(world) => world_implementations(input, &fields, internal, fault, &world, expansion),
        None => implementations(input, &fields, internal, fault, expansion),
    })
}

impl Expansion {
    fn value_arms(&mut self, name: &syn::Ident, variant: &syn::Variant) {
        let ident = &variant.ident;
        let count = variant.fields.len();
        let bindings: Vec<_> =
            (0..count).map(|index| format_ident!("__umpire_field_{index}")).collect();
        let others: Vec<_> =
            (0..count).map(|index| format_ident!("__umpire_other_{index}")).collect();
        let pattern = pattern_for(variant, &bindings);
        let other_pattern = pattern_for(variant, &others);
        let values: Vec<_> =
            bindings.iter().map(|binding| quote!(::core::clone::Clone::clone(#binding))).collect();
        let value = construct(variant, &values);
        self.clones.push(quote!(#pattern => #value));
        self.equals.push(quote!((&#pattern, &#other_pattern) => true #(&& #bindings == #others)*));
        self.hashes.push(quote!(#pattern => { #(::core::hash::Hash::hash(#bindings, state);)* }));
        self.debug.push(format_arm(variant, &pattern, &bindings, &ident.to_string()));
        self.display.push(format_arm(variant, &pattern, &bindings, &format!("{name}::{ident}")));
    }

    fn domains(
        &mut self,
        input: &syn::DeriveInput,
        variant: &syn::Variant,
        domains: &[FieldDomain],
        world: Option<&syn::Type>,
    ) {
        let fields: Vec<_> = variant.fields.iter().collect();
        let bindings: Vec<_> =
            (0..fields.len()).map(|index| format_ident!("__umpire_field_{index}")).collect();
        let values: Vec<_> =
            bindings.iter().map(|binding| quote!(::core::clone::Clone::clone(&#binding))).collect();
        let value = construct(variant, &values);
        let mut catalog = quote!(__umpire_actions.push(#value););
        for ((field, binding), domain) in fields.iter().zip(&bindings).zip(domains).rev() {
            let ty = &field.ty;
            let domain = match domain {
                FieldDomain::Explicit(domain) => quote!(#domain),
                FieldDomain::Finite => quote!(<#ty as ::umpire_dsl::Finite>::domain()),
                FieldDomain::World(path) => {
                    quote::quote_spanned!(path_span(path)=> #path(__umpire_world))
                }
            };
            catalog = quote!(for #binding in #domain { let #binding: #ty = #binding; #catalog });
        }
        if world.is_some() {
            self.world_catalogs.push(catalog);
        } else {
            self.catalogs.push(catalog);
        }
        let (builder, constructor, lazy_catalog, retained_catalog) =
            lazy_domain(input, variant, domains, world);
        self.builders.push(builder);
        self.constructors.push(constructor);
        self.lazy_catalogs.push(lazy_catalog);
        self.retained_catalogs.push(retained_catalog);
        self.labels.push(format!("{}::{}", input.ident, variant.ident));
    }
}

fn path_span(path: &syn::Path) -> proc_macro2::Span {
    syn::spanned::Spanned::span(path)
}

fn world_context(input: &syn::DeriveInput, errors: &mut manyhow::Emitter) -> Option<syn::Type> {
    for attr in input.attrs.iter().filter(|attr| attr.path().is_ident("domain")) {
        errors.emit(syn::Error::new_spanned(
            attr,
            "#[domain] belongs on one variant field, not the action enum",
        ));
    }
    let attr = input.attrs.iter().find(|attr| attr.path().is_ident("__umpire_world"))?;
    match attr.parse_args::<syn::Type>() {
        Ok(world) => Some(world),
        Err(error) => {
            errors.emit(error);
            None
        }
    }
}

struct Bounds {
    inherent: Generics,
    actor: Generics,
    lazy: Generics,
}

fn value_traits(
    input: &syn::DeriveInput,
    fields: &[&Field],
    expansion: &Expansion,
) -> (TokenStream, Bounds) {
    let Expansion { clones, equals, hashes, debug, display, finite, .. } = expansion;
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
    let mut inherent = clone.clone();
    for ty in finite {
        inherent.make_where_clause().predicates.push(
            syn::parse_quote_spanned!(syn::spanned::Spanned::span(ty)=> #ty: ::umpire_dsl::Finite),
        );
    }
    let lazy = bounded(&inherent, fields, &quote!('static));
    let actor = bounded(
        &inherent,
        fields,
        &quote!(::core::cmp::Eq + ::core::hash::Hash + ::core::fmt::Debug),
    );
    let output = quote! {
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
    };
    (output, Bounds { inherent, actor, lazy })
}

fn implementations(
    input: &syn::DeriveInput,
    fields: &[&Field],
    internal: bool,
    fault: bool,
    expansion: Expansion,
) -> TokenStream {
    let (values, bounds) = value_traits(input, fields, &expansion);
    let admission = (!fields.is_empty()).then(|| {
        catalog_impl(
            input,
            &bounds.lazy,
            None,
            &expansion.labels,
            &expansion.lazy_catalogs,
            &quote!(Self::actions()),
        )
    });
    let Expansion { catalogs, builders, constructors, lazy_catalogs, .. } = expansion;
    let name = &input.ident;
    let (_, types, _) = input.generics.split_for_impl();
    let (clone_impl, _, clone_where) = bounds.inherent.split_for_impl();
    let (lazy_impl, _, lazy_where) = bounds.lazy.split_for_impl();
    let (actor_impl, _, actor_where) = bounds.actor.split_for_impl();
    quote! {
        #values
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
        #admission
    }
}

fn catalog_impl(
    input: &syn::DeriveInput,
    lazy: &Generics,
    world: Option<&syn::Type>,
    labels: &[String],
    lazy_catalogs: &[TokenStream],
    actions: &TokenStream,
) -> TokenStream {
    let name = &input.ident;
    let (_, types, _) = input.generics.split_for_impl();
    let mut generics = lazy.clone();
    let world = world.map_or_else(
        || {
            generics.params.insert(0, syn::parse_quote!(__UmpireWorld));
            quote!(__UmpireWorld)
        },
        |world| quote!(#world),
    );
    let (implementation, _, constraints) = generics.split_for_impl();
    quote! {
        impl #implementation ::umpire_dsl::path_build::ActorCatalog<#world> for #name #types #constraints {
            #[allow(unused_variables)]
            fn catalog(__umpire_world: &#world, __umpire_retain: usize) -> ::std::vec::Vec<(::std::string::String, ::umpire_dsl::ActionDomain<Self>)> {
                ::std::vec![#((::std::string::String::from(#labels), #lazy_catalogs)),*]
            }
            #[allow(unused_variables)]
            fn actions(__umpire_world: &#world) -> ::std::vec::Vec<Self> {
                #actions
            }
        }
    }
}

/// A World-dependent actor has no static catalog: it classifies values and enumerates its
/// domain only from an explicitly supplied World.
fn world_implementations(
    input: &syn::DeriveInput,
    fields: &[&Field],
    internal: bool,
    fault: bool,
    world: &syn::Type,
    expansion: Expansion,
) -> TokenStream {
    let (values, bounds) = value_traits(input, fields, &expansion);
    let world_catalogs = &expansion.world_catalogs;
    let admission = catalog_impl(
        input,
        &bounds.lazy,
        Some(world),
        &expansion.labels,
        &expansion.retained_catalogs,
        &quote! {
            let mut __umpire_actions = ::std::vec::Vec::new();
            #(#world_catalogs)*
            __umpire_actions
        },
    );
    let Expansion { builders, constructors, lazy_catalogs, .. } = expansion;
    let name = &input.ident;
    let (_, types, _) = input.generics.split_for_impl();
    let (clone_impl, _, clone_where) = bounds.inherent.split_for_impl();
    let (lazy_impl, _, lazy_where) = bounds.lazy.split_for_impl();
    let (actor_impl, _, actor_where) = bounds.actor.split_for_impl();
    quote! {
        #values
        impl #clone_impl #name #types #clone_where {
            /// Identifies environment-independent actions; this does not imply fairness.
            pub fn internal(&self) -> bool { #internal }
            /// Identifies actions representing faults, independently of internal status.
            pub fn fault(&self) -> bool { #fault }
        }
        impl #actor_impl ::umpire_dsl::ActorValue for #name #types #actor_where {
            fn internal(&self) -> bool { self.internal() }
            fn fault(&self) -> bool { self.fault() }
        }
        impl #lazy_impl #name #types #lazy_where {
            #(#constructors)*
            /// Enumerates actions lazily in declaration and input-domain order, using the
            /// selected World's declared domains.
            pub fn actions_in(__umpire_world: &#world) -> ::umpire_dsl::ActionDomain<Self> {
                ::umpire_dsl::ActionDomain::concat(::std::vec![#(#lazy_catalogs),*])
            }
            #[doc(hidden)]
            pub fn __umpire_path_actions() -> ::umpire_dsl::path_build::Unbound<Self> {
                ::umpire_dsl::path_build::Unbound::new()
            }
        }
        #(#builders)*
        #admission
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
    field_domains: &[FieldDomain],
    world: Option<&syn::Type>,
) -> (TokenStream, TokenStream, TokenStream, TokenStream) {
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
    let dependent = field_domains.iter().any(|domain| matches!(domain, FieldDomain::World(_)));
    let domains = input_domains(field_domains, &types);
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
    let Some(world) = world.filter(|_| dependent) else {
        let constructor_code = quote! {
            #[doc(hidden)]
            #[allow(non_snake_case)]
            pub fn #constructor() -> #builder #generics {
                #builder { #(#names: #domains,)* __umpire_actor: ::core::marker::PhantomData }
            }
        };
        let lazy = quote!(Self::#constructor().finish());
        return (builder_code, constructor_code, lazy.clone(), lazy);
    };
    world_domain(
        &BuilderParts { actor, action, visibility, builder: &builder, constructor: &constructor },
        &clone,
        (&names, &setters, &types, &domains),
        world,
        &builder_code,
    )
}

struct BuilderParts<'a> {
    actor: &'a syn::Ident,
    action: &'a syn::Ident,
    visibility: &'a syn::Visibility,
    builder: &'a syn::Ident,
    constructor: &'a syn::Ident,
}

/// A World-dependent variant's zero-argument builder only refuses; its domain is built from an
/// explicitly supplied World.
fn world_domain(
    parts: &BuilderParts<'_>,
    clone: &Generics,
    (names, setters, types, domains): (&[syn::Ident], &[syn::Ident], &[&syn::Type], &[TokenStream]),
    world: &syn::Type,
    builder_code: &TokenStream,
) -> (TokenStream, TokenStream, TokenStream, TokenStream) {
    let BuilderParts { actor, action, visibility, builder, constructor } = parts;
    let (impls, generics, constraints) = clone.split_for_impl();
    let bound = format_ident!("__umpire_path_{action}_in");
    let unbound = format_ident!("__UmpireUnbound{actor}{action}");
    let builder_code = quote! {
        #builder_code
        #[doc(hidden)]
        #visibility struct #unbound #clone #constraints {
            __umpire_actor: ::core::marker::PhantomData<#actor #generics>,
        }
        impl #impls #unbound #generics #constraints {
            #(
                #[doc(hidden)]
                pub fn #setters(self, _: ::core::option::Option<::umpire_dsl::InputDomain<#types>>) -> Self {
                    self
                }
            )*
            #[doc(hidden)]
            pub fn finish(self) -> ::umpire_dsl::path_build::Unbound<#actor #generics> {
                ::umpire_dsl::path_build::Unbound::new()
            }
        }
    };
    let constructor_code = quote! {
        #[doc(hidden)]
        #[allow(non_snake_case)]
        pub fn #constructor() -> #unbound #generics {
            #unbound { __umpire_actor: ::core::marker::PhantomData }
        }
        #[doc(hidden)]
        #[allow(non_snake_case)]
        pub fn #bound(__umpire_world: &#world, __umpire_retain: usize) -> #builder #generics {
            #builder { #(#names: #domains,)* __umpire_actor: ::core::marker::PhantomData }
        }
    };
    (
        builder_code,
        constructor_code,
        quote!(Self::#bound(__umpire_world, usize::MAX).finish()),
        quote!(Self::#bound(__umpire_world, __umpire_retain).finish()),
    )
}

fn input_domains(field_domains: &[FieldDomain], types: &[&syn::Type]) -> Vec<TokenStream> {
    field_domains
        .iter()
        .zip(types)
        .map(|(domain, ty)| match domain {
            FieldDomain::Explicit(value @ Expr::Range(_)) => {
                quote!(::umpire_dsl::InputDomain::range(#value))
            }
            FieldDomain::Explicit(Expr::Array(array)) => {
                let values = &array.elems;
                quote!(::umpire_dsl::InputDomain::values(::std::vec![#values]))
            }
            FieldDomain::Explicit(_) => unreachable!(),
            FieldDomain::Finite => quote!(<#ty as ::umpire_dsl::Finite>::__umpire_input_domain()),
            FieldDomain::World(path) => quote::quote_spanned!(path_span(path)=>
                ::umpire_dsl::InputDomain::iterated(#path(__umpire_world), __umpire_retain)
            ),
        })
        .collect()
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

fn domain(field: &Field, action: &str, index: usize, world: bool) -> syn::Result<FieldDomain> {
    let field_name = field.ident.as_ref().map_or_else(|| index.to_string(), ToString::to_string);
    let context = format!("{action} field `{field_name}`");
    let mut domains = field.attrs.iter().filter(|attr| attr.path().is_ident("domain"));
    if let Some(domain) = domains.next() {
        if let Some(duplicate) = domains.next() {
            return Err(syn::Error::new_spanned(
                duplicate,
                format!("{context} has duplicate #[domain] declarations"),
            ));
        }
        if let Some(inputs) = field.attrs.iter().find(|attr| attr.path().is_ident("inputs")) {
            return Err(syn::Error::new_spanned(
                inputs,
                format!("{context} declares both #[inputs] and #[domain]; declare one domain"),
            ));
        }
        let path = domain.parse_args::<syn::Path>().map_err(|_| {
            syn::Error::new_spanned(
                domain,
                format!("{context} #[domain] names a function, such as #[domain(sizes)]"),
            )
        })?;
        if !world {
            return Err(syn::Error::new_spanned(
                domain,
                format!(
                    "{context} #[domain] requires the World of a #[machine] module that declares #[derive(World)]"
                ),
            ));
        }
        return Ok(FieldDomain::World(path));
    }
    let mut attrs = field.attrs.iter().filter(|attr| attr.path().is_ident("inputs"));
    let Some(attr) = attrs.next() else {
        if crate::finite::integer(&field.ty) {
            return Err(syn::Error::new_spanned(
                field,
                format!("{context} requires #[inputs] with a bounded range or finite list"),
            ));
        }
        return Ok(FieldDomain::Finite);
    };
    if let Some(duplicate) = attrs.next() {
        return Err(syn::Error::new_spanned(
            duplicate,
            format!("{context} has duplicate #[inputs] domains"),
        ));
    }
    let value = attr.parse_args::<Expr>()?;
    match &value {
        Expr::Range(range) if range.start.is_some() && range.end.is_some() => {
            Ok(FieldDomain::Explicit(value))
        }
        Expr::Array(_) => Ok(FieldDomain::Explicit(value)),
        _ => Err(syn::Error::new_spanned(
            attr,
            format!(
                "{context} #[inputs] requires a bounded range or finite list, such as 0..=3 or [false, true]"
            ),
        )),
    }
}
