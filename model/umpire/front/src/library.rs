//! Umpire library declarations that Models call, read from `umpire-dsl`'s own source, so their
//! lowered meaning is the authored one; nothing is recognized by its name alone.

const CLOSABLE: &str = include_str!("../../dsl/src/closable.rs");
const PHASED: &str = include_str!("../../dsl/src/phased.rs");

/// The public function `name` of `umpire_dsl::closable`.
pub(crate) fn law(name: &str) -> Option<syn::ItemFn> {
    syn::parse_file(CLOSABLE).ok()?.items.into_iter().find_map(|item| match item {
        syn::Item::Fn(law)
            if law.sig.ident == name && matches!(law.vis, syn::Visibility::Public(_)) =>
        {
            Some(law)
        }
        _ => None,
    })
}

/// The default method `name` of `umpire_dsl::Phased`, as a method of the implementing State.
pub(crate) fn phased_default(name: &str) -> Option<syn::ImplItemFn> {
    syn::parse_file(PHASED).ok()?.items.into_iter().find_map(|item| match item {
        syn::Item::Trait(phased) if phased.ident == "Phased" => {
            phased.items.into_iter().find_map(|item| match item {
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
        _ => None,
    })
}
