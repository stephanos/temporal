//! Umpire library declarations that Models call, read from `umpire-dsl`'s own source, so their
//! lowered meaning is the authored one; nothing is recognized by its name alone.

const CLOSABLE: &str = include_str!("../../author/dsl/src/closable.rs");
const PHASED: &str = include_str!("../../author/dsl/src/phased.rs");

/// The public function `name` of `umpire_dsl::closable`.
pub(crate) fn law(name: &str) -> Option<syn::ItemFn> {
    parsed(CLOSABLE).into_iter().find_map(|item| match item {
        syn::Item::Fn(law)
            if law.sig.ident == name && matches!(law.vis, syn::Visibility::Public(_)) =>
        {
            Some(law)
        }
        _ => None,
    })
}

/// The method names of the `umpire_dsl` trait `name`, declared in `phased.rs`.
pub(crate) fn trait_methods(name: &str) -> Vec<String> {
    phased_trait(name).map_or_else(Vec::new, |found| {
        found
            .items
            .into_iter()
            .filter_map(|item| match item {
                syn::TraitItem::Fn(method) => Some(method.sig.ident.to_string()),
                _ => None,
            })
            .collect()
    })
}
fn phased_trait(name: &str) -> Option<syn::ItemTrait> {
    parsed(PHASED).into_iter().find_map(|item| match item {
        syn::Item::Trait(found) if found.ident == name => Some(found),
        _ => None,
    })
}

/// The default method `name` of `umpire_dsl::Phased`, as a method of the implementing State.
pub(crate) fn phased_default(name: &str) -> Option<syn::ImplItemFn> {
    phased_trait("Phased")?.items.into_iter().find_map(|item| match item {
        syn::TraitItem::Fn(method) if method.sig.ident == name => Some(syn::ImplItemFn {
            attrs: method.attrs,
            vis: syn::Visibility::Inherited,
            modifiers: syn::FnModifiers::default(),
            sig: method.sig,
            block: method.default?,
        }),
        _ => None,
    })
}

/// A library file's items; this crate's tests check that both files parse.
fn parsed(source: &str) -> Vec<syn::Item> {
    syn::parse_file(source).map(|file| file.items).unwrap_or_default()
}

#[cfg(test)]
mod tests {
    #[test]
    fn library_sources_parse_and_declare_the_laws_and_traits() {
        for source in [super::CLOSABLE, super::PHASED] {
            assert!(syn::parse_file(source).is_ok());
        }
        assert!(super::law("terminal_remains").is_some() && super::law("closed_rejects").is_some());
        assert_eq!(super::trait_methods("Phase"), ["closed"]);
        assert_eq!(super::trait_methods("Phased"), ["phase", "closed"]);
        assert!(super::phased_default("closed").is_some());
    }
}
