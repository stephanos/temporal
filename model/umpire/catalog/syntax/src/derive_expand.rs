//! Expansion of admitted mirror structs into typed operations.

use proc_macro2::TokenStream;
use quote::{format_ident, quote, quote_spanned};
use syn::{Expr, Ident, Type, ext::IdentExt, spanned::Spanned};

use crate::{ApiAttributes, ApiExpectation, DeriveCatalog, DeriveField, DeriveRule, marker};

/// Generates a catalog factory and exhaustive, recursive transport inspection.
pub fn expand_derive(catalog: &DeriveCatalog) -> syn::Result<TokenStream> {
    let context = &catalog.context;
    let name = &catalog.name;
    let ty = &catalog.ty;
    let proto = &catalog.proto;
    let description = &catalog.description;
    let policy = format_ident!("{}", if catalog.request { "request" } else { "response" });
    let names = catalog.fields.iter().map(|f| &f.name);
    let oneofs = catalog.fields.iter().filter(|field| !field.attributes.variants.is_empty()).map(|field| {
        let name = &field.name;
        let types = generic_types(&field.ty)?;
        let ty = types.first().ok_or_else(|| syn::Error::new_spanned(&field.ty, "OneOf requires its concrete enum type"))?;
        let variants = field.attributes.variants.iter().map(|(name, _)| name);
        Ok(quote!(if let Some(value) = &message.#name { match value { #(#ty::#variants(_) => ()),* } }))
    }).collect::<syn::Result<Vec<_>>>()?;
    let pattern = (!catalog.supplemental).then(|| {
        quote_spanned!(ty.span()=>
            #[allow(deprecated)]
        let _: fn(&#ty) = |message| { let #ty { #(#names: _),* } = message; #(#oneofs)* };
        )
    });
    let fields = catalog
        .fields
        .iter()
        .map(|field| field_expansion(catalog, field))
        .collect::<syn::Result<Vec<_>>>()?;
    let rules = catalog.rules.iter().map(rule_expansion).collect::<Vec<_>>();
    let mut supplemental_factories = Vec::new();
    let mut supplemental_names = Vec::new();
    let mut map_factories = Vec::new();
    for field in &catalog.fields {
        for (index, additional) in field.attributes.also.iter().enumerate() {
            let factory = format_ident!("__{}_also_{}", field.name.unraw(), index);
            let extra = DeriveCatalog::supplemental_field(field, additional)?;
            let expanded = field_expansion(catalog, &extra)?;
            supplemental_factories.push(quote! {
                fn #factory(limits: &#context) -> Result<::umpire_catalog::Catalog<#ty>, ::umpire_catalog::AdmissionErrors> {
                    let _ = limits;
                    ::umpire_catalog::Catalog::<#ty>::#policy(#proto) #expanded .build()
                }
            });
            supplemental_names.push(factory);
        }
        if marker(&field.ty)?.ident == "Map" {
            map_factories.push(map_factory(catalog, field)?);
        }
    }
    let inspect = if catalog.supplemental { quote!() } else { inspect_expansion(catalog)? };
    Ok(quote! {
        #[allow(deprecated)]
        impl #name {
            /// Admits the typed catalog with the supplied dynamic configuration.
            pub fn catalog(limits: &#context) -> ::core::result::Result<
                ::umpire_catalog::Catalog<#ty>, ::umpire_catalog::AdmissionErrors
            > {
                #pattern
                let _ = limits;
                ::umpire_catalog::Catalog::<#ty>::#policy(#proto)
                    .description(#description)
                    #(#fields)*
                    #(#rules)*
                    .build()
            }
            /// All primary and supplemental field intents for this mirror.
            pub fn catalogs(limits: &#context) -> Result<Vec<::umpire_catalog::Catalog<#ty>>, ::umpire_catalog::AdmissionErrors> {
                Ok(vec![Self::catalog(limits)?, #(Self::#supplemental_names(limits)?),*])
            }
            #(#supplemental_factories)*
            #(#map_factories)*
        }
        #inspect
    })
}

fn generic_types(ty: &Type) -> syn::Result<Vec<&Type>> {
    let segment = marker(ty)?;
    if let syn::PathArguments::AngleBracketed(args) = &segment.arguments {
        return args
            .args
            .iter()
            .map(|arg| {
                if let syn::GenericArgument::Type(ty) = arg {
                    Ok(ty)
                } else {
                    Err(syn::Error::new_spanned(arg, "expected kind type argument"))
                }
            })
            .collect();
    }
    Ok(Vec::new())
}

fn constraint_value(name: &Ident, value: &Expr, numeric: bool) -> TokenStream {
    if let Expr::Path(_) = value
        && !matches!(
            name.to_string().as_str(),
            "prefix" | "suffix" | "contains" | "unique" | "defined_only"
        )
    {
        let cast = if numeric { quote!(i128) } else { quote!(usize) };
        return quote!(limits.get(&#value) as #cast);
    }
    if matches!(name.to_string().as_str(), "prefix" | "suffix" | "contains") {
        quote!((#value).to_owned())
    } else {
        quote!(#value)
    }
}

fn kind_expansion(ty: &Type, attrs: &ApiAttributes) -> syn::Result<TokenStream> {
    let ty = crate::derive::semantic_type(ty)?;
    if crate::derive::secret_map(ty)? {
        return Ok(quote!(::umpire_catalog::Kind::Secret));
    }
    let kind = &marker(ty)?.ident;
    let kind_name = kind.to_string();
    if kind_name == "Enum" {
        let args = generic_types(ty)?;
        let enum_type = args.first().ok_or_else(|| {
            syn::Error::new_spanned(ty, "Enum requires a concrete protobuf enum type")
        })?;
        let defined_only = attrs
            .constraints
            .iter()
            .find(|(name, _)| name == "defined_only")
            .map(|(_, value)| value);
        return Ok(if let Some(defined_only) = defined_only {
            quote!({
                if #defined_only {
                    ::umpire_catalog::defined_enum::<#enum_type>()
                } else {
                    ::umpire_catalog::Kind::Enum(::umpire_catalog::EnumRules {
                        defined_only: false,
                        values: ::std::vec::Vec::new(),
                    })
                }
            })
        } else {
            quote!(::umpire_catalog::defined_enum::<#enum_type>())
        });
    }
    let (runtime, rules, numeric) = match kind_name.as_str() {
        "Identifier" | "Text" => (kind.clone(), Some(format_ident!("StringRules")), false),
        "Timestamp" | "Duration" | "Counter" | "Number" => {
            (kind.clone(), Some(format_ident!("NumberRules")), true)
        }
        "RawDuration" => (format_ident!("Duration"), Some(format_ident!("NumberRules")), true),
        "Bytes" | "Message" | "OneOf" | "Map" | "Vec" | "Ignored" => {
            (format_ident!("Bytes"), Some(format_ident!("BytesRules")), false)
        }
        _ => (kind.clone(), None, false),
    };
    if let Some(rules) = rules {
        let values = attrs
            .constraints
            .iter()
            .filter(|(name, _)| {
                !matches!(name.to_string().as_str(), "min_items" | "max_items" | "unique")
            })
            .map(|(name, value)| {
                let value = constraint_value(name, value, numeric);
                quote!(#name: Some(#value))
            });
        Ok(
            quote!(::umpire_catalog::Kind::#runtime(::umpire_catalog::#rules { #(#values,)* ..::core::default::Default::default() })),
        )
    } else {
        Ok(quote!(::umpire_catalog::Kind::#runtime))
    }
}

fn expected_expansion(expected: &ApiExpectation) -> TokenStream {
    let value = match expected {
        ApiExpectation::Accepts => quote!(::umpire_catalog::Expectation::Accepts),
        ApiExpectation::Unobservable => quote!(::umpire_catalog::Expectation::Unobservable),
        ApiExpectation::AnyStatus => quote!(::umpire_catalog::Expectation::RejectsAnyStatus),
        ApiExpectation::Rejects(code, Some(pattern)) => {
            quote!(::umpire_catalog::Expectation::Rejects { code: ::umpire_catalog::Code::#code as i32, message_pattern: #pattern.to_owned(), details_type: None })
        }
        ApiExpectation::Rejects(code, None) => {
            quote!(::umpire_catalog::Expectation::RejectsCode(::umpire_catalog::Code::#code))
        }
    };
    quote!(::umpire_catalog::Expected { expectation: #value })
}

fn field_expansion(catalog: &DeriveCatalog, field: &DeriveField) -> syn::Result<TokenStream> {
    if marker(&field.ty)?.ident == "OneOf" {
        let args = generic_types(&field.ty)?;
        let oneof = args.first().ok_or_else(|| {
            syn::Error::new_spanned(&field.ty, "OneOf requires its protobuf enum type")
        })?;
        let source = &field.name;
        let ty = &catalog.ty;
        let mut expansions = Vec::new();
        for (variant, proto) in &field.attributes.variants {
            let path = format!("{}.{}", catalog.proto.value(), proto.value());
            let get = quote!(|message: &#ty| match message.#source.as_ref() {
                Some(#oneof::#variant(value)) => Some(value),
                _ => None,
            });
            let set = quote!(|message: &mut #ty, value| {
                match value {
                    Some(value) => message.#source = Some(#oneof::#variant(value)),
                    None if matches!(message.#source, Some(#oneof::#variant(_))) => message.#source = None,
                    None => (),
                }
            });
            let string_rules = payload_rules(
                field,
                &["min_len", "max_len", "min_bytes", "max_bytes", "prefix", "suffix", "contains"],
                false,
            );
            let bytes_rules = payload_rules(field, &["min_len", "max_len"], false);
            let projection = quote!({
                use ::umpire_catalog::views::MessageOneof as _;
                ::umpire_catalog::views::OneofProjection::new(#get, #set).field(
                    #path,
                    ::umpire_catalog::StringRules { #(#string_rules,)* ..Default::default() },
                    ::umpire_catalog::BytesRules { #(#bytes_rules,)* ..Default::default() },
                )
            });
            expansions.push(field_projection(catalog, field, Some((path, projection)))?);
        }
        return Ok(quote!(#(#expansions)*));
    }
    field_projection(catalog, field, None)
}

fn payload_rules(field: &DeriveField, names: &[&str], numeric: bool) -> Vec<TokenStream> {
    field
        .attributes
        .constraints
        .iter()
        .filter(|(name, _)| names.contains(&name.to_string().as_str()))
        .map(|(name, value)| {
            let value = constraint_value(name, value, numeric);
            quote!(#name: Some(#value))
        })
        .collect()
}

fn field_projection(
    catalog: &DeriveCatalog,
    field: &DeriveField,
    projection: Option<(String, TokenStream)>,
) -> syn::Result<TokenStream> {
    let attrs = &field.attributes;
    let ty = &catalog.ty;
    let name = &field.name;
    let path = projection.as_ref().map_or_else(
        || format!("{}.{}", catalog.proto.value(), name.unraw()),
        |(path, _)| path.clone(),
    );
    let kind = kind_expansion(&field.ty, attrs)?;
    let marker_ty = &field.ty;
    if let Some(reason) =
        attrs.tags.iter().find(|tag| tag.name == "Ignored").and_then(|tag| tag.argument.as_ref())
    {
        return Ok(quote! {
            .ignored(::umpire_catalog::IgnoredField::new(
                #path, #kind, |message: &#ty| &message.#name,
                |message: &mut #ty| &mut message.#name, #reason,
            ))
        });
    }
    let (get, set) = if let Some(via) = &attrs.via {
        (quote!(#via::view(message)), quote!(#via::set(message, value)))
    } else if marker(&field.ty)?.ident == "Map"
        && generic_types(&field.ty)?
            .get(1)
            .is_some_and(|value| marker(value).is_ok_and(|kind| kind.ident == "Message"))
    {
        let args = generic_types(&field.ty)?;
        let payloads = generic_types(
            args.get(1)
                .ok_or_else(|| syn::Error::new_spanned(&field.ty, "Map requires a value type"))?,
        )?;
        let payload = payloads.first().ok_or_else(|| {
            syn::Error::new_spanned(&field.ty, "Message requires its actual payload type")
        })?;
        (
            quote!(::umpire_catalog::views::message_entries::<#payload>(&message.#name).iter().map(::prost::Message::encode_to_vec).collect::<Vec<_>>()),
            quote!({ message.#name = ::umpire_catalog::views::decode_message_entries::<#payload>(value)?; Ok(()) }),
        )
    } else {
        (
            quote!(<_ as ::umpire_catalog::views::View<#marker_ty>>::view(&message.#name)),
            quote!(<_ as ::umpire_catalog::views::View<#marker_ty>>::set(&mut message.#name, value)),
        )
    };
    let description = &field.description;
    let tags = attrs.tags.iter().map(|tag| {
        let name = &tag.name;
        let value = match name.to_string().as_str() {
            "Cached" => { let value = &tag.argument; quote!(::umpire_catalog::Consequence::Cached { ttl: ::core::time::Duration::from_secs(#value) }) },
            "EventuallyConsistent" => { let value = &tag.argument; quote!(::umpire_catalog::Consequence::EventuallyConsistent { polls: #value }) },
            _ => quote!(::umpire_catalog::Consequence::#name),
        };
        quote!(.tag(&#value))
    });
    let expected = attrs.expected.as_ref().map(|expectation| {
        let expected = expected_expansion(expectation);
        quote!(.expects(&#expected))
    });
    let repeated: Vec<_> = attrs
        .constraints
        .iter()
        .filter(|(name, _)| {
            matches!(name.to_string().as_str(), "min_items" | "max_items" | "unique")
        })
        .map(|(name, value)| {
            let value = constraint_value(name, value, false);
            if name == "unique" { quote!(#name: #value) } else { quote!(#name: Some(#value)) }
        })
        .collect();
    let repeated = (!repeated.is_empty()).then(|| quote!(.repeated(::umpire_catalog::RepeatedRules { #(#repeated,)* ..::core::default::Default::default() })));
    let repeated_check =
        repeated.as_ref().map(|_| quote!(::umpire_catalog::types::check_repeated(&value);));
    let projected_field = if let Some((_, projection)) = projection {
        projection
    } else {
        quote!(::umpire_catalog::Field::try_projected(
            #path, #kind, |message: &#ty| { let value = #get; #repeated_check value }, |message: &mut #ty, value| { #set },
        ))
    };
    Ok(quote! {
        .field(#projected_field.description(#description) #(#tags)* #repeated #expected)
    })
}

fn presence(rule: &DeriveRule) -> TokenStream {
    let operator = format_ident!(
        "{}",
        match rule.operator.to_string().as_str() {
            "required" => "Required",
            "any_of" => "AnyOf",
            "one_of" => "OneOf",
            "requires" => "Requires",
            _ => "Excludes",
        }
    );
    let fields = rule.fields.iter().map(|field| field.unraw().to_string());
    if rule.operator == "requires" || rule.operator == "excludes" {
        quote!(::umpire_catalog::Presence::#operator(#(#fields.to_owned()),*))
    } else {
        quote!(::umpire_catalog::Presence::#operator(vec![#(#fields.to_owned()),*]))
    }
}

fn rule_expansion(rule: &DeriveRule) -> TokenStream {
    let presence = presence(rule);
    let expected = rule.expected.as_ref().map(expected_expansion);
    let expected = expected.map_or_else(|| quote!(None), |expected| quote!(Some(#expected)));
    quote!(.rule(::umpire_catalog::Rule { presence: #presence, expected: #expected }))
}

fn map_factory(catalog: &DeriveCatalog, field: &DeriveField) -> syn::Result<TokenStream> {
    let context = &catalog.context;
    let args = generic_types(&field.ty)?;
    let [key, value] = args.as_slice() else {
        return Err(syn::Error::new_spanned(&field.ty, "Map requires key and value kind types"));
    };
    let factory = format_ident!("__{}_entry_catalog", field.name.unraw());
    let entry =
        if matches!(marker(value)?.ident.to_string().as_str(), "Text" | "Identifier" | "Uuid") {
            quote!(::umpire_catalog::views::TextEntry)
        } else {
            quote!(::umpire_catalog::views::MetadataEntry)
        };
    let proto = crate::derive::map_entry_proto(&catalog.proto, &field.name);
    let key_path = format!("{proto}.key");
    let value_path = format!("{proto}.value");
    let key_kind = kind_expansion(key, &ApiAttributes::default())?;
    let value_kind = kind_expansion(value, &ApiAttributes::default())?;
    let (value_get, value_set) = if marker(value)?.ident == "Message" {
        (
            quote!(entry.value.clone()),
            quote!({
                entry.value = value;
                Ok(())
            }),
        )
    } else {
        (
            quote!(<_ as ::umpire_catalog::views::View<#value>>::view(&entry.value)),
            quote!(<_ as ::umpire_catalog::views::View<#value>>::set(&mut entry.value, value)),
        )
    };
    Ok(quote! {
        /// Admits the generated protobuf map-entry catalog.
        pub fn #factory(limits: &#context) -> Result<::umpire_catalog::Catalog<#entry>, ::umpire_catalog::AdmissionErrors> {
            let _ = limits;
            ::umpire_catalog::Catalog::response(#proto)
                .field(::umpire_catalog::Field::try_projected(
                    #key_path, #key_kind,
                    |entry: &#entry| <_ as ::umpire_catalog::views::View<#key>>::view(&entry.key),
                    |entry: &mut #entry, value| <_ as ::umpire_catalog::views::View<#key>>::set(&mut entry.key, value),
                ).tag(&::umpire_catalog::Consequence::Decided))
                .field(::umpire_catalog::Field::try_projected(
                    #value_path, #value_kind,
                    |entry: &#entry| #value_get,
                    |entry: &mut #entry, value| #value_set,
                ).tag(&::umpire_catalog::Consequence::Decided))
                .build()
        }
    })
}

fn inspect_expansion(catalog: &DeriveCatalog) -> syn::Result<TokenStream> {
    let context = &catalog.context;
    let ty = &catalog.ty;
    let mirror = &catalog.name;
    let response_checks = catalog.response_checks.as_ref().map(|checks| {
        quote! {
            if response {
                output.extend(::umpire_catalog::validate(self, &#checks::catalog(context)?));
            }
        }
    });
    let mut nested = Vec::new();
    let mut fill_nested = Vec::new();
    let mut fill_ids = Vec::new();
    for field in &catalog.fields {
        let name = &field.name;
        let kind = marker(&field.ty)?.ident.to_string();
        if field.attributes.tags.iter().any(|tag| tag.name == "Ignored") {
            continue;
        }
        let semantic = marker(crate::derive::semantic_type(&field.ty)?)?.ident.to_string();
        if kind == "Message" || (kind == "Vec" && semantic == "Message") {
            nested.push(quote!(::umpire_catalog::Inspect::inspect(&self.#name, context, response, output)?;));
            fill_nested.push(quote!(changed |= ::umpire_catalog::Inspect::<#context>::fill_ids(&mut self.#name, seed)?;));
        }
        if kind == "Map" {
            let (inspect, fill) = map_inspection(catalog, field)?;
            nested.push(inspect);
            fill_nested.push(fill);
        }
        if kind == "OneOf" {
            let args = generic_types(&field.ty)?;
            let oneof = args.first().ok_or_else(|| {
                syn::Error::new_spanned(&field.ty, "OneOf requires its protobuf enum type")
            })?;
            if field.attributes.variants.is_empty() {
                return Err(syn::Error::new(
                    name.span(),
                    "OneOf requires exhaustive #[mirror(oneof(...))] alternatives",
                ));
            }
            let variants =
                field.attributes.variants.iter().map(|(variant, _)| variant).collect::<Vec<_>>();
            nested.push(quote! {
                if let Some(value) = self.#name.as_ref() { match value { #(#oneof::#variants(value) => ::umpire_catalog::Inspect::inspect(value, context, response, output)?),* } }
            });
            fill_nested.push(quote! {
                if let Some(value) = self.#name.as_mut() { match value { #(#oneof::#variants(value) => changed |= ::umpire_catalog::Inspect::<#context>::fill_ids(value, seed)?),* } }
            });
        }
        if field.attributes.tags.iter().any(|tag| tag.name == "IdempotencyKey") {
            fill_ids.push(quote! {
                if self.#name.is_empty() {
                    let mut rng = ::umpire_catalog::Rng::new(seed);
                    self.#name = <String as ::umpire_catalog::FieldValue>::generate(&::umpire_catalog::Kind::Uuid, &mut rng)?;
                    changed = true;
                }
            });
        }
    }
    Ok(quote! {
        impl ::umpire_catalog::Inspect<#context> for #ty {
            fn inspect(&self, context: &#context, response: bool, output: &mut Vec<::umpire_catalog::Violation>) -> Result<(), ::umpire_catalog::AdmissionErrors> {
                for catalog in #mirror::catalogs(context)? {
                    output.extend(::umpire_catalog::validate(self, &catalog));
                }
                #response_checks
                #(#nested)*
                let _ = response;
                Ok(())
            }
            fn fill_ids(&mut self, seed: u64) -> Result<bool, ::umpire_catalog::GenerationError> {
                let mut changed = false;
                let _ = seed;
                #(#fill_ids)*
                #(#fill_nested)*
                Ok(changed)
            }
        }
    })
}

fn map_inspection(
    catalog: &DeriveCatalog,
    field: &DeriveField,
) -> syn::Result<(TokenStream, TokenStream)> {
    let name = &field.name;
    let context = &catalog.context;
    let mirror = &catalog.name;
    let factory = format_ident!("__{}_entry_catalog", name.unraw());
    let args = generic_types(&field.ty)?;
    let value = args
        .get(1)
        .ok_or_else(|| syn::Error::new_spanned(&field.ty, "Map requires a value type"))?;
    let kind = marker(value)?.ident.to_string();
    let message = kind == "Message";
    let entries = format_ident!(
        "{}",
        if matches!(kind.as_str(), "Text" | "Identifier" | "Uuid") {
            "text_entries"
        } else if message {
            "message_entries"
        } else {
            "metadata_entries"
        }
    );
    let nested = message.then(|| {
        quote! { for value in self.#name.values() {
            ::umpire_catalog::Inspect::inspect(value, context, response, output)?;
        } }
    });
    let fill = message.then(|| {
        quote! { for value in self.#name.values_mut() {
            changed |= ::umpire_catalog::Inspect::<#context>::fill_ids(value, seed)?;
        } }
    });
    Ok((
        quote! {
            for entry in ::umpire_catalog::views::#entries(&self.#name) {
                output.extend(::umpire_catalog::validate(&entry, &#mirror::#factory(context)?));
            }
            #nested
        },
        quote!(#fill),
    ))
}
