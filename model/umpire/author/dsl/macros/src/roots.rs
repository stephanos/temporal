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

    fn use_tree(&self, tree: &mut syn::UseTree) {
        match tree {
            syn::UseTree::Path(path) => {
                if let Some(facade) = self.root(&path.ident) {
                    path.ident = proc_macro2::Ident::new("umpire_dsl", path.ident.span());
                    if facade
                        && let syn::UseTree::Path(module) = &*path.tree
                        && module.ident == "dsl"
                    {
                        let inner = module.tree.clone();
                        path.tree = inner;
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
