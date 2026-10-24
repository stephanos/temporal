//! Typed Rust expansion of checked catalog declarations.

use proc_macro2::TokenStream;
use quote::{format_ident, quote, quote_spanned};
use syn::ext::IdentExt;
use syn::spanned::Spanned;

use crate::tree::{Actions, CustomTag, Enum, Expected, Field, Message, Rule, Tag};

/// Expands declarations into typed catalogs and exhaustive conversion functions.
pub fn expand(catalog: &crate::Catalog) -> syn::Result<TokenStream> {
    let vis = &catalog.vis;
    let name = &catalog.name;
    let docs = matches!(vis, syn::Visibility::Public(_)).then(|| {
        quote! { #[doc = concat!("Exhaustive typed annotation catalog `", stringify!(#name), "`.")] }
    });
    let messages = catalog.messages.iter().map(message);
    let probe_factories = catalog.messages.iter().map(|message| &message.factory);
    let enums = catalog.enums.iter().map(enumeration);
    let actions = catalog.actions.iter().map(action);
    let tags = catalog.tags.iter().map(custom_tag);
    let services = catalog.services.iter().flat_map(|service| {
        service.methods.iter().map(|(method, binding)| {
            let rpc = format!("{}.{}", service.proto.value(), method);
            let disposition = binding.kind.to_string();
            let target = &binding.target;
            quote! { (#rpc, #disposition, #target) }
        })
    });
    Ok(quote! {
        #docs
        #vis mod #name {
            #[allow(unused_imports)]
            use super::*;

            #(#tags)*
            #(#enums)*
            #(#messages)*
            #(#actions)*

            /// Every explicitly declared expectation, independent of RPC probe registration.
            pub fn expected_probe_cases() -> ::core::result::Result<
                ::std::vec::Vec<::umpire_catalog::ProbeCase>, ::umpire_catalog::AdmissionErrors
            > {
                let mut cases: ::std::vec::Vec<::umpire_catalog::ProbeCase> = ::std::vec::Vec::new();
                #(cases.extend(#probe_factories()?.probe_cases());)*
                cases.sort_by(|a, b| a.id.cmp(&b.id));
                Ok(cases)
            }

            /// Explicit protobuf RPC classifications.
            pub const RPC_BINDINGS: &[(&str, &str, &str)] = &[#(#services),*];
        }
    })
}

fn message(message: &Message) -> TokenStream {
    let factory = &message.factory;
    let ty = &message.ty;
    let proto = &message.proto;
    let policy = if message.request { format_ident!("request") } else { format_ident!("response") };
    let names = message.fields.iter().map(|field| &field.name);
    let pattern = quote_spanned! {ty.span()=> let #ty { #(#names: _),* } = message; };
    let checks = message.fields.iter().map(field_check);
    let fields = message.fields.iter().map(|field| field_factory(message, field));
    let rules = message.rules.iter().map(rule);
    quote! {
        #[doc = concat!("Admits the exhaustive catalog for `", #proto, "`.")]
        pub fn #factory() -> ::core::result::Result<
            ::umpire_catalog::Catalog<#ty>, ::umpire_catalog::AdmissionErrors
        > {
            let _: fn(&#ty) = |message| {
                #pattern
                #(#checks)*
            };
            ::umpire_catalog::Catalog::<#ty>::#policy(#proto)
                #(#fields)*
                #(.rule(#rules))*
                .build()
        }
    }
}

fn field_check(field: &Field) -> TokenStream {
    let name = &field.name;
    let variants = field.variants.iter().map(|variant| {
        let path = &variant.path;
        quote! { #path(_) => () }
    });
    let exhaustive = if field.variants.is_empty() {
        quote! {}
    } else {
        quote! {
            match message.#name.as_ref() {
                Some(value) => match value { #(#variants),* },
                None => (),
            }
        }
    };
    let witness = if field.opaque {
        quote! {}
    } else {
        let marker = &field.kind;
        let value = if let Some(projection) = &field.projection {
            let get = &projection.get;
            quote! { &(#get)(message) }
        } else {
            quote! { &message.#name }
        };
        let repeated = field.constraints.iter().any(|(name, _)| is_repeated(name)).then(|| {
            quote! {
                ::umpire_catalog::types::check_repeated(#value);
            }
        });
        quote! {
            ::umpire_catalog::types::check::<
                ::umpire_catalog::types::#marker, _
            >(#value);
            #repeated
        }
    };
    quote! { #exhaustive #witness }
}

fn field_factory(message: &Message, field: &Field) -> TokenStream {
    let ty = &message.ty;
    let proto = &message.proto;
    let name = &field.name;
    let path = format!("{}.{}", proto.value(), name.unraw());
    let kind = kind(field);
    if let Some(reason) = ignored_reason(field).filter(|_| field.opaque) {
        if !field.variants.is_empty() {
            let fields = field.variants.iter().map(|variant| {
                let variant_path = &variant.path;
                let path = format!("{}.{}", proto.value(), variant.proto.value());
                quote! {
                    .ignored(::umpire_catalog::IgnoredField::projected(
                        #path, #kind,
                        |message: &#ty| match message.#name.as_ref() {
                            Some(#variant_path(value)) => Some(value.clone()),
                            _ => None,
                        },
                        |message: &mut #ty, value| message.#name = value.map(#variant_path),
                        #reason,
                    ))
                }
            });
            return quote! { #(#fields)* };
        }
        return quote! {
            .ignored(::umpire_catalog::IgnoredField::new(
                #path, #kind,
                |message: &#ty| &message.#name,
                |message: &mut #ty| &mut message.#name,
                #reason,
            ))
        };
    }
    if let Some(projection) = field.projection.as_ref().filter(|_| !field.variants.is_empty()) {
        let projected = &projection.ty;
        let get = &projection.get;
        let set = &projection.set;
        let fields = field.variants.iter().map(|variant| {
            let variant_path = &variant.path;
            let path = format!("{}.{}", proto.value(), variant.proto.value());
            annotated_field(
                field,
                &quote! {
                    ::umpire_catalog::Field::<#ty, Option<#projected>>::projected(
                        #path, #kind,
                        |message: &#ty| {
                            if matches!(message.#name.as_ref(), Some(#variant_path(_))) {
                                Some((#get)(message))
                            } else {
                                None
                            }
                        },
                        |message: &mut #ty, value: Option<#projected>| {
                            if matches!(message.#name.as_ref(), Some(#variant_path(_))) {
                                if let Some(value) = value {
                                    (#set)(message, value);
                                } else {
                                    message.#name = None;
                                }
                            }
                        },
                    )
                },
            )
        });
        return quote! { #(#fields)* };
    }
    let field_constructor = if let Some(projection) = &field.projection {
        let projected = &projection.ty;
        let get = &projection.get;
        let set = &projection.set;
        quote! {
            ::umpire_catalog::Field::<#ty, #projected>::projected(
                #path, #kind, #get, #set,
            )
        }
    } else {
        quote! {
            ::umpire_catalog::Field::new(
                #path, #kind,
                |message: &#ty| &message.#name,
                |message: &mut #ty| &mut message.#name,
            )
        }
    };
    annotated_field(field, &field_constructor)
}

fn annotated_field(field: &Field, field_constructor: &TokenStream) -> TokenStream {
    let tags = field.tags.iter().map(|tag| {
        let value = consequence(tag);
        quote! { .tag(&#value) }
    });
    let repeated = repeated(field);
    let expected = field.expected.as_ref().map(|value| {
        let expected = expectation(value);
        quote! { .expects(&#expected) }
    });
    quote! { .field(#field_constructor #(#tags)* #repeated #expected) }
}

fn ignored_reason(field: &Field) -> Option<&syn::Lit> {
    field.tags.iter().find(|tag| tag.name == "Ignored").and_then(|tag| tag.argument.as_ref())
}

fn kind(field: &Field) -> TokenStream {
    let name = &field.kind;
    let rules_name = match name.to_string().as_str() {
        "Identifier" | "Text" => Some(format_ident!("StringRules")),
        "Timestamp" | "Duration" | "Counter" => Some(format_ident!("NumberRules")),
        "Random" | "Bytes" => Some(format_ident!("BytesRules")),
        _ => None,
    };
    if let Some(rules_name) = rules_name {
        let fields =
            field.constraints.iter().filter(|(name, _)| !is_repeated(name)).map(|(name, value)| {
                let value = if let syn::Lit::Str(_) = value {
                    quote! { #value.to_owned() }
                } else {
                    quote! { #value }
                };
                quote! { #name: Some(#value) }
            });
        quote! {
            ::umpire_catalog::Kind::#name(::umpire_catalog::#rules_name {
                #(#fields,)* ..::core::default::Default::default()
            })
        }
    } else if name == "Enum" {
        let values = field.enumeration.as_ref().map_or_else(
            || quote! { ::std::vec::Vec::new() },
            |factory| {
                let values = format_ident!("{}_values", factory);
                quote! { #values() }
            },
        );
        let defined_only = field
            .constraints
            .iter()
            .find(|(name, _)| name == "defined_only")
            .map_or_else(|| quote! { true }, |(_, value)| quote! { #value });
        quote! {
            ::umpire_catalog::Kind::Enum(::umpire_catalog::EnumRules {
                defined_only: #defined_only, values: #values,
            })
        }
    } else {
        quote! { ::umpire_catalog::Kind::#name }
    }
}

fn is_repeated(name: &syn::Ident) -> bool {
    matches!(name.to_string().as_str(), "min_items" | "max_items" | "unique")
}

fn repeated(field: &Field) -> TokenStream {
    let constraints: Vec<_> =
        field.constraints.iter().filter(|(name, _)| is_repeated(name)).collect();
    if constraints.is_empty() {
        return quote! {};
    }
    let fields = constraints.into_iter().map(|(name, value)| {
        if name == "unique" {
            quote! { #name: #value }
        } else {
            quote! { #name: Some(#value) }
        }
    });
    quote! {
        .repeated(::umpire_catalog::RepeatedRules {
            #(#fields,)* ..::core::default::Default::default()
        })
    }
}

fn expectation(expected: &Expected) -> TokenStream {
    let outcome = &expected.outcome;
    let expectation = if let Some((code, message, details)) = &expected.reject {
        let details = details
            .as_ref()
            .map_or_else(|| quote! { None }, |value| quote! { Some(#value.to_owned()) });
        quote! {
            ::umpire_catalog::Expectation::Rejects {
                code: #code, message_pattern: #message.to_owned(), details_type: #details,
            }
        }
    } else {
        quote! { ::umpire_catalog::Expectation::#outcome }
    };
    quote! { ::umpire_catalog::Expected { expectation: #expectation } }
}

fn rule(rule: &Rule) -> TokenStream {
    let operator = match rule.operator.to_string().as_str() {
        "required" => format_ident!("Required"),
        "any_of" => format_ident!("AnyOf"),
        "one_of" => format_ident!("OneOf"),
        "requires" => format_ident!("Requires"),
        "excludes" => format_ident!("Excludes"),
        _ => unreachable!("validated presence operator"),
    };
    let fields: Vec<_> = rule.fields.iter().map(|field| field.unraw().to_string()).collect();
    let args = if matches!(rule.operator.to_string().as_str(), "requires" | "excludes") {
        quote! { #(#fields.to_owned()),* }
    } else {
        quote! { ::std::vec![#(#fields.to_owned()),*] }
    };
    let expected = rule.expected.as_ref().map_or_else(
        || quote! { None },
        |value| {
            let expected = expectation(value);
            quote! { Some(#expected) }
        },
    );
    quote! {
        ::umpire_catalog::Rule {
            presence: ::umpire_catalog::Presence::#operator(#args), expected: #expected,
        }
    }
}

fn enumeration(enumeration: &Enum) -> TokenStream {
    let factory = &enumeration.factory;
    let values_factory = format_ident!("{}_values", factory);
    let ty = &enumeration.ty;
    let proto = &enumeration.proto;
    let paths: Vec<_> = enumeration.variants.iter().map(|variant| &variant.path).collect();
    quote! {
        #[doc = concat!("Converts a known `", #proto, "` discriminant.")]
        pub fn #factory(value: i32) -> ::core::result::Result<#ty, ::umpire_catalog::UnknownEnum> {
            let variant = <#ty as ::core::convert::TryFrom<i32>>::try_from(value)
                .map_err(|_| ::umpire_catalog::UnknownEnum { value, enum_name: #proto })?;
            match variant { #(#paths => ()),* }
            Ok(variant)
        }

        #[doc = concat!("Lists declared `", #proto, "` discriminants.")]
        pub fn #values_factory() -> ::std::vec::Vec<i32> {
            ::std::vec![#(#paths as i32),*]
        }
    }
}

fn action(actions: &Actions) -> TokenStream {
    let factory = &actions.factory;
    let ty = &actions.ty;
    let actors = actions.actors.iter().map(|(actor, bindings)| {
        let arms = bindings.iter().map(|(pattern, binding)| {
            let kind = &binding.kind;
            let target = &binding.target;
            quote! { #pattern => ::umpire_catalog::Binding::#kind(#target) }
        });
        quote! { #actor(actor) => match actor { #(#arms),* } }
    });
    quote! {
        /// Classifies every declared model action.
        pub fn #factory(action: &#ty) -> ::umpire_catalog::Binding {
            match action { #(#actors),* }
        }
    }
}

fn consequence(tag: &Tag) -> TokenStream {
    let name = &tag.name;
    match (name.to_string().as_str(), &tag.argument) {
        ("Cached", Some(ttl)) => {
            quote! { ::umpire_catalog::Consequence::Cached { ttl: ::core::time::Duration::from_secs(#ttl) } }
        }
        ("EventuallyConsistent", Some(polls)) => {
            quote! { ::umpire_catalog::Consequence::EventuallyConsistent { polls: #polls } }
        }
        ("Ignored", Some(reason)) => {
            quote! { ::umpire_catalog::Consequence::Ignored(#reason.to_owned()) }
        }
        ("Monotone" | "Immutable" | "IdempotencyKey" | "Decided" | "Observed", _) => {
            quote! { ::umpire_catalog::Consequence::#name }
        }
        _ => quote! { #name },
    }
}

fn custom_tag(tag: &CustomTag) -> TokenStream {
    let name = &tag.name;
    let consequence = consequence(&tag.consequence);
    let kinds = tag.kinds.iter().map(|kind| {
        if matches!(kind.to_string().as_str(), "Uuid" | "Secret") {
            quote! { ::umpire_catalog::Kind::#kind }
        } else {
            quote! { ::umpire_catalog::Kind::#kind(_) }
        }
    });
    quote! {
        #[doc = concat!("Declarative `", stringify!(#name), "` annotation tag.")]
        pub struct #name;

        impl ::umpire_catalog::Tag for #name {
            fn name(&self) -> &str { stringify!(#name) }
            fn consequence(&self) -> ::umpire_catalog::Consequence { #consequence }
            fn applies_to(&self, kind: &::umpire_catalog::Kind) -> bool {
                matches!(kind, #(#kinds)|*)
            }
        }
    }
}
