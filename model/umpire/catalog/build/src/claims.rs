//! Descriptor-checked field claim metadata from executable catalog declarations.

use std::{collections::BTreeMap, fs, io};

use quote::{format_ident, quote};
use umpire_catalog_syntax::{catalogs, proto::Graph};

use crate::{System, coverage};

/// Generate semantic field claims without inferring constraints from physical carriers.
pub fn generate(graph: &Graph, system: &System<'_>) -> io::Result<String> {
    let mut claims = BTreeMap::new();
    for path in coverage::inputs(&system.catalog_root)?.files {
        let mut source = fs::read_to_string(&path)?;
        if path.file_name().is_some_and(|name| name == "claims.rs") {
            source = crate::mirrors::expand(&source, graph)?;
        }
        for catalog in catalogs(&source).map_err(io::Error::other)? {
            for message in catalog.messages {
                let owner = message.proto.value();
                let descriptor = graph.message(&owner).map_err(io::Error::other)?;
                for field in message.fields {
                    let names = if field.variants.is_empty() {
                        vec![field.name.to_string().trim_start_matches("r#").to_owned()]
                    } else {
                        field.variants.iter().map(|variant| variant.proto.value()).collect()
                    };
                    let kind = field.kind.to_string();
                    let ignored = field
                        .tags
                        .iter()
                        .find(|tag| tag.name == "Ignored")
                        .and_then(|tag| tag.argument.as_ref())
                        .and_then(|argument| match argument {
                            syn::Lit::Str(reason) => Some(reason.value()),
                            _ => None,
                        });
                    for name in names {
                        if !descriptor.fields.iter().any(|candidate| candidate.name == name) {
                            return Err(io::Error::other(format!("unknown claim: {owner}.{name}")));
                        }
                        let declaration = format!("{owner}.{name}");
                        let reason = ignored.as_ref().map(|reason| {
                            if reason.contains(&declaration) {
                                reason.clone()
                            } else {
                                format!("{reason} [descriptor field {declaration}]")
                            }
                        });
                        if claims.insert(declaration.clone(), (kind.clone(), reason)).is_some() {
                            return Err(io::Error::other(format!(
                                "duplicate claim: {declaration}"
                            )));
                        }
                    }
                }
            }
        }
    }
    let function = format_ident!("{}_field_claim", system.prefix);
    let arms = claims.iter().map(|(declaration, (kind, ignored))| {
        let ignored = ignored.as_ref().map_or_else(|| quote!(None), |reason| quote!(Some(#reason)));
        quote! { #declaration => Some(::umpire_catalog::FieldClaim { declaration: #declaration, kind: #kind, ignored: #ignored }) }
    });
    let file = syn::parse2(quote! {
        /// Resolve an exact descriptor field against its explicit catalog claim.
        pub fn #function(declaration: &str) -> Option<::umpire_catalog::FieldClaim> {
            match declaration { #(#arms,)* _ => None }
        }
    })
    .map_err(io::Error::other)?;
    Ok(prettyplease::unparse(&file))
}
