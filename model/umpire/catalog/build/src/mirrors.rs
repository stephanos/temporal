//! Structural mirrors generated from descriptors and separately authored semantic claims.

use std::io;

use quote::{format_ident, quote};
use syn::ext::IdentExt;
use umpire_catalog_syntax::{DeriveCatalog, proto::Graph};

/// Complete structural fields from the descriptor without inventing semantic claims.
pub fn expand(source: &str, graph: &Graph) -> io::Result<String> {
    let mut file = syn::parse_file(source).map_err(io::Error::other)?;
    for item in &mut file.items {
        let syn::Item::Struct(item) = item else { continue };
        if !item.attrs.iter().any(|attr| attr.path().is_ident("mirror")) {
            continue;
        }
        let input = syn::parse2(quote!(#item)).map_err(io::Error::other)?;
        let claims = DeriveCatalog::parse(&input).map_err(io::Error::other)?;
        if claims.supplemental {
            continue;
        }
        let proto = claims.proto.value();
        let descriptor = graph.message(&proto).map_err(io::Error::other)?;
        let syn::Fields::Named(fields) = &mut item.fields else {
            return Err(io::Error::other("catalog claims require named fields"));
        };
        for field in &descriptor.fields {
            if claims.fields.iter().any(|claim| {
                claim.name.unraw() == field.name
                    || claim.attributes.variants.iter().any(|(_, name)| name.value() == field.name)
            }) {
                continue;
            }
            if let Some(oneof) = &field.oneof {
                if claims.fields.iter().any(|claim| claim.name.unraw() == oneof.member) {
                    return Err(io::Error::other(format!(
                        "unclaimed variant of claimed field: {proto}.{}",
                        field.name
                    )));
                }
                if fields.named.iter().any(|candidate| {
                    candidate.ident.as_ref().is_some_and(|name| name.unraw() == oneof.member)
                }) {
                    continue;
                }
                fields.named.push(ignored_oneof(descriptor, oneof)?);
            } else {
                let name: syn::Ident = syn::parse_str(&field.member).map_err(io::Error::other)?;
                let reason = format!(
                    "Unclaimed descriptor field {proto}.{}; no semantic read or fill is declared",
                    field.name
                );
                fields.named.push(syn::parse_quote! {
                    #[mirror(ignored = #reason)]
                    #name: ::umpire_catalog::kinds::Ignored
                });
            }
        }
        let mut ordered: Vec<_> = fields.named.iter().cloned().collect();
        ordered.sort_by_key(|claim| {
            descriptor
                .fields
                .iter()
                .position(|field| {
                    claim.ident.as_ref().is_some_and(|name| {
                        name.unraw() == field.name
                            || field
                                .oneof
                                .as_ref()
                                .is_some_and(|oneof| name.unraw() == oneof.member)
                    })
                })
                .unwrap_or(usize::MAX)
        });
        fields.named = ordered.into_iter().collect();
    }
    Ok(prettyplease::unparse(&file))
}

/// Export generated structural mirrors for expansion in the semantic claim owner's context.
pub fn generate(source: &str, graph: &Graph, name: &str) -> io::Result<String> {
    let expanded = expand(source, graph)?;
    let file = syn::parse_file(&expanded).map_err(io::Error::other)?;
    let name = format_ident!("{name}");
    let file = syn::parse2(quote! {
        /// Descriptor-generated structural mirrors with explicit semantic claims.
        #[macro_export]
        macro_rules! #name { () => { #file }; }
    })
    .map_err(io::Error::other)?;
    Ok(prettyplease::unparse(&file))
}

fn ignored_oneof(
    descriptor: &umpire_catalog_syntax::proto::Message,
    oneof: &umpire_catalog_syntax::proto::Oneof,
) -> io::Result<syn::Field> {
    let proto = &descriptor.name;
    let name: syn::Ident = syn::parse_str(&oneof.member).map_err(io::Error::other)?;
    let variants = descriptor
        .fields
        .iter()
        .filter(|candidate| {
            candidate.oneof.as_ref().is_some_and(|candidate| candidate.member == oneof.member)
        })
        .map(|candidate| {
            let variant = candidate
                .oneof
                .as_ref()
                .ok_or_else(|| io::Error::other("missing oneof identity"))?;
            let variant = format_ident!(
                "{}",
                variant
                    .variant
                    .rsplit("::")
                    .next()
                    .ok_or_else(|| io::Error::other("missing variant name"))?
            );
            let proto = &candidate.name;
            Ok(quote!(#variant = #proto))
        })
        .collect::<io::Result<Vec<_>>>()?;
    let ty: syn::Type = syn::parse_str(
        oneof.variant.rsplit_once("::").ok_or_else(|| io::Error::other("missing oneof type"))?.0,
    )
    .map_err(io::Error::other)?;
    let reason = format!(
        "Unclaimed descriptor field {proto}.{}; no semantic read or fill is declared",
        oneof.member
    );
    Ok(syn::parse_quote! {
        #[mirror(ignored = #reason, oneof(#(#variants),*))]
        #name: ::umpire_catalog::kinds::OneOf<#ty>
    })
}
