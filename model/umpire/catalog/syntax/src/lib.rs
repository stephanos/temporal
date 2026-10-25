//! API-independent declaration grammar shared by expansion and build-time coverage.

#![forbid(unsafe_code)]

mod derive;
mod derive_expand;
pub mod descriptor;
mod expand;
pub mod naming;
mod parse;
pub mod proto;
pub mod services;
mod tree;
pub use derive::*;
pub use derive_expand::expand_derive;
pub use expand::expand;
pub use tree::*;

use syn::visit::Visit;

/// Parse every annotation invocation in a Rust catalog input, including nested modules.
pub fn catalogs(source: &str) -> syn::Result<Vec<Catalog>> {
    struct Catalogs {
        result: syn::Result<Vec<Catalog>>,
    }
    impl<'ast> Visit<'ast> for Catalogs {
        fn visit_item_struct(&mut self, node: &'ast syn::ItemStruct) {
            if node
                .attrs
                .iter()
                .any(|attr| attr.path().is_ident("mirror") || attr.path().is_ident("rule"))
            {
                let parsed = syn::parse2::<syn::DeriveInput>(quote::quote!(#node))
                    .and_then(|input| DeriveCatalog::parse(&input))
                    .and_then(|mirror| {
                        if mirror.supplemental {
                            Ok(None)
                        } else {
                            mirror.admission_catalog().map(|mut catalog| {
                                catalog.enums.clear();
                                Some(catalog)
                            })
                        }
                    });
                match (&mut self.result, parsed) {
                    (Ok(catalogs), Ok(Some(catalog))) => catalogs.push(catalog),
                    (result @ Ok(_), Err(error)) => *result = Err(error),
                    _ => {}
                }
            }
            syn::visit::visit_item_struct(self, node);
        }
        fn visit_macro(&mut self, node: &'ast syn::Macro) {
            if node.path.segments.last().is_some_and(|segment| segment.ident == "annotate") {
                let parsed = syn::parse2(node.tokens.clone());
                match (&mut self.result, parsed) {
                    (Ok(catalogs), Ok(catalog)) => catalogs.push(catalog),
                    (result @ Ok(_), Err(error)) => *result = Err(error),
                    _ => {}
                }
            }
            syn::visit::visit_macro(self, node);
        }
    }
    let file = syn::parse_file(source)?;
    let mut visitor = Catalogs { result: Ok(Vec::new()) };
    visitor.visit_file(&file);
    visitor.result
}

impl Catalog {
    /// Project descriptor obligations from the same declarations that generate typed operations.
    pub fn coverage(&self) -> descriptor::Coverage {
        use descriptor::{
            Coverage, Disposition, EnumCoverage, MessageCoverage, MethodCoverage, ServiceCoverage,
        };
        Coverage {
            messages: self
                .messages
                .iter()
                .map(|message| MessageCoverage {
                    name: message.proto.value(),
                    opaque_fields: message
                        .fields
                        .iter()
                        .filter(|field| field.opaque)
                        .flat_map(|field| {
                            if field.variants.is_empty() {
                                vec![field.name.to_string().trim_start_matches("r#").to_owned()]
                            } else {
                                field.variants.iter().map(|variant| variant.proto.value()).collect()
                            }
                        })
                        .collect(),
                    fields: message
                        .fields
                        .iter()
                        .flat_map(|field| {
                            if field.variants.is_empty() {
                                vec![field.name.to_string().trim_start_matches("r#").to_owned()]
                            } else {
                                field.variants.iter().map(|variant| variant.proto.value()).collect()
                            }
                        })
                        .collect(),
                })
                .collect(),
            enums: self
                .enums
                .iter()
                .map(|enumeration| EnumCoverage {
                    name: enumeration.proto.value(),
                    values: enumeration
                        .variants
                        .iter()
                        .map(|variant| variant.proto.value())
                        .collect(),
                })
                .collect(),
            services: self
                .services
                .iter()
                .map(|service| ServiceCoverage {
                    name: service.proto.value(),
                    methods: service
                        .methods
                        .iter()
                        .map(|(name, binding)| MethodCoverage {
                            name: name.to_string(),
                            disposition: match binding.kind.to_string().as_str() {
                                "Action" => Disposition::Action(binding.target.value()),
                                "Observed" => Disposition::Observed,
                                _ => Disposition::OutOfScope(binding.target.value()),
                            },
                        })
                        .collect(),
                })
                .collect(),
        }
    }
}

#[cfg(test)]
mod tests {
    #[test]
    fn context_and_api_parameters_require_the_catalog_item() -> syn::Result<()> {
        for attribute in [
            "context = crate::Other",
            "api = crate::other",
            "also(text, context = crate::Other)",
            "also(text, api = crate::other)",
        ] {
            let source = format!(
                "#[mirror(response = \"example.Wire\", type = Wire)] struct Mirror {{ #[mirror({attribute})] value: Text }}"
            );
            let input = syn::parse_str(&source)?;
            assert!(
                super::DeriveCatalog::parse(&input).is_err(),
                "accepted field parameter: {attribute}"
            );
        }
        Ok(())
    }
}
