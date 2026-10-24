use proc_macro_crate::{FoundCrate, crate_name};

#[path = "../../../dependency_roots.rs"]
mod dependency_roots;

pub(crate) fn resolve(tokens: proc_macro2::TokenStream) -> proc_macro2::TokenStream {
    dependency_roots::resolve(tokens, ROOTS)
}

const ROOTS: &[(&str, &str)] = &[("umpire-dsl", "dsl")];

pub(crate) fn admission(input: &syn::ItemMod) -> syn::ItemMod {
    use syn::visit_mut::VisitMut;

    let dependency = |package| match crate_name(package) {
        Ok(FoundCrate::Name(name)) => Some(name),
        _ => None,
    };
    let mut input = input.clone();
    Admission { dsl: dependency("umpire-dsl"), facade: dependency("umpire") }
        .visit_item_mod_mut(&mut input);
    input
}

struct Admission {
    dsl: Option<String>,
    facade: Option<String>,
}

impl Admission {
    fn root(&self, name: &proc_macro2::Ident) -> Option<bool> {
        let name = name.to_string();
        if self.facade.as_deref() == Some(&name) {
            Some(true)
        } else if self.dsl.as_deref() == Some(&name) {
            Some(false)
        } else {
            None
        }
    }

    fn facade_use_tree(tree: &mut syn::UseTree) {
        match tree {
            syn::UseTree::Path(module) if module.ident == "dsl" => {
                *tree = (*module.tree).clone();
            }
            syn::UseTree::Group(group) => {
                for tree in &mut group.items {
                    Self::facade_use_tree(tree);
                }
            }
            _ => {}
        }
    }

    fn use_tree(&self, tree: &mut syn::UseTree) {
        match tree {
            syn::UseTree::Path(path) => {
                if let Some(facade) = self.root(&path.ident) {
                    path.ident = proc_macro2::Ident::new("umpire_dsl", path.ident.span());
                    if facade {
                        Self::facade_use_tree(&mut path.tree);
                    }
                }
            }
            syn::UseTree::Group(group) => {
                for tree in &mut group.items {
                    self.use_tree(tree);
                }
            }
            _ => {}
        }
    }
}

impl syn::visit_mut::VisitMut for Admission {
    fn visit_path_mut(&mut self, path: &mut syn::Path) {
        if let Some(first) = path.segments.first_mut()
            && let Some(facade) = self.root(&first.ident)
        {
            first.ident = proc_macro2::Ident::new("umpire_dsl", first.ident.span());
            if facade && path.segments.iter().nth(1).is_some_and(|segment| segment.ident == "dsl") {
                path.segments = path
                    .segments
                    .iter()
                    .enumerate()
                    .filter(|(index, _)| *index != 1)
                    .map(|(_, segment)| segment.clone())
                    .collect();
            }
        }
        syn::visit_mut::visit_path_mut(self, path);
    }

    fn visit_item_use_mut(&mut self, item: &mut syn::ItemUse) {
        self.use_tree(&mut item.tree);
    }
}

#[cfg(test)]
mod tests {
    use quote::ToTokens;

    use super::Admission;

    #[test]
    fn normalizes_facade_imports_through_groups_without_stripping_other_prefixes() -> syn::Result<()>
    {
        let admission =
            Admission { dsl: Some("direct_dsl".into()), facade: Some("author_kit".into()) };
        for (source, expected) in [
            ("author_kit::dsl::{Action, check}", "umpire_dsl::{Action, check}"),
            ("author_kit::{dsl::{Action, check}, State}", "umpire_dsl::{{Action, check}, State}"),
            ("author_kit::{{dsl::Action}, State}", "umpire_dsl::{{Action}, State}"),
            ("author_kit::{other::dsl::Action}", "umpire_dsl::{other::dsl::Action}"),
            ("direct_dsl::{dsl::Action}", "umpire_dsl::{dsl::Action}"),
        ] {
            let mut actual: syn::UseTree = syn::parse_str(source)?;
            admission.use_tree(&mut actual);
            let expected: syn::UseTree = syn::parse_str(expected)?;
            assert_eq!(
                actual.to_token_stream().to_string(),
                expected.to_token_stream().to_string()
            );
        }
        Ok(())
    }
}
