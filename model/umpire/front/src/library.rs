//! Umpire library declarations that Models call, read from `umpire-dsl`'s own source, so their
//! lowered meaning is the authored one; nothing is recognized by its name alone.

const CLOSABLE: &str = include_str!("../../author/dsl/src/closable.rs");
const COUNTDOWN: &str = include_str!("../../author/dsl/src/countdown.rs");
const PHASED: &str = include_str!("../../author/dsl/src/phased.rs");

/// A public function or associated method of a registered authored library.
pub(crate) fn function(module: &str, name: &str) -> Option<syn::ItemFn> {
    let items = parsed(source(module)?);
    if let Some((owner, method)) = name.split_once("::") {
        return items.into_iter().find_map(|item| {
            let syn::Item::Impl(item) = item else { return None };
            if item.trait_.is_some() {
                return None;
            }
            let syn::Type::Path(path) = item.self_ty.as_ref() else { return None };
            if path.path.segments.last()?.ident != owner {
                return None;
            }
            item.items.into_iter().find_map(|entry| {
                let syn::ImplItem::Fn(mut function) = entry else { return None };
                if function.sig.ident != method
                    || !matches!(function.vis, syn::Visibility::Public(_))
                {
                    return None;
                }
                function.sig.generics = item.generics.clone();
                Some(syn::ItemFn {
                    modifiers: function.modifiers,
                    attrs: function.attrs,
                    vis: function.vis,
                    sig: function.sig,
                    block: Box::new(function.block),
                })
            })
        });
    }
    items.into_iter().find_map(|item| match item {
        syn::Item::Fn(law)
            if law.sig.ident == name && matches!(law.vis, syn::Visibility::Public(_)) =>
        {
            Some(law)
        }
        _ => None,
    })
}

pub(crate) fn source(module: &str) -> Option<&'static str> {
    match module {
        "closable" => Some(CLOSABLE),
        "countdown" => Some(COUNTDOWN),
        _ => None,
    }
}

/// A library newtype's authored finite range and single backing field.
pub(crate) fn bounded(module: &str, name: &str) -> Option<(syn::ItemStruct, syn::ExprRange)> {
    bounded_in(source(module)?, name)
}

fn bounded_in(source: &str, name: &str) -> Option<(syn::ItemStruct, syn::ExprRange)> {
    let items = parsed(source);
    let structure = items.iter().find_map(|item| match item {
        syn::Item::Struct(value) if value.ident == name => Some(value.clone()),
        _ => None,
    })?;
    let domain = items.iter().find_map(|item| {
        let syn::Item::Impl(value) = item else { return None };
        let (trait_path, _) = value.trait_.as_ref()?;
        if !trait_path.is_ident("Finite") {
            return None;
        }
        let syn::Type::Path(owner) = value.self_ty.as_ref() else { return None };
        if owner.path.segments.last()?.ident != name {
            return None;
        }
        value.items.iter().find_map(|item| match item {
            syn::ImplItem::Fn(method) if method.sig.ident == "domain" => Some(method),
            _ => None,
        })
    })?;
    let [syn::Stmt::Expr(syn::Expr::MethodCall(map), None)] = domain.block.stmts.as_slice() else {
        return None;
    };
    if map.method != "map" || map.args.len() != 1 {
        return None;
    }
    let syn::Expr::Paren(range) = map.receiver.as_ref() else { return None };
    let syn::Expr::Range(range) = range.expr.as_ref() else { return None };
    let identity = match (&structure.fields, &map.args[0]) {
        (syn::Fields::Unnamed(fields), syn::Expr::Path(path)) => {
            fields.unnamed.len() == 1 && path.path.is_ident("Self")
        }
        (syn::Fields::Named(fields), syn::Expr::Closure(closure)) if fields.named.len() == 1 => {
            matches!((closure.inputs.first(), closure.body.as_ref()),
                (Some(syn::Pat::Ident(input)), syn::Expr::Struct(body))
                if closure.inputs.len() == 1 && body.path.is_ident("Self") && body.rest.is_none()
                && body.fields.len() == 1 && body.fields[0].member == syn::Member::Named(fields.named[0].ident.clone()?)
                && matches!(&body.fields[0].expr, syn::Expr::Path(path) if path.path.is_ident(&input.ident)))
        }
        _ => false,
    };
    identity.then(|| (structure, range.clone()))
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

/// A library file's items; tests check that every registered source parses.
fn parsed(source: &str) -> Vec<syn::Item> {
    syn::parse_file(source).map(|file| file.items).unwrap_or_default()
}

#[cfg(test)]
mod tests {
    #[test]
    fn library_sources_parse_and_declare_the_laws_and_traits() {
        for source in [super::CLOSABLE, super::PHASED, super::COUNTDOWN] {
            assert!(syn::parse_file(source).is_ok());
        }
        assert!(
            super::function("closable", "terminal_remains").is_some()
                && super::function("closable", "closed_rejects").is_some()
        );
        assert_eq!(super::trait_methods("Phase"), ["closed"]);
        assert_eq!(super::trait_methods("Phased"), ["phase", "closed"]);
        assert!(super::phased_default("closed").is_some());
    }
    #[test]
    fn finite_ranges_follow_source_and_nonidentity_maps_refuse() {
        let changed = super::COUNTDOWN.replace("(0..=N)", "(2..=N)");
        let (_, range) =
            super::bounded_in(&changed, "Remaining").unwrap_or_else(|| panic!("range"));
        assert!(
            matches!(range.start.as_deref(), Some(syn::Expr::Lit(value)) if matches!(&value.lit, syn::Lit::Int(value) if value.base10_digits() == "2"))
        );
        assert!(
            super::bounded_in(
                &super::COUNTDOWN.replace("Self { ticks }", "Self { ticks: ticks + 1 }"),
                "Remaining"
            )
            .is_none()
        );
    }
}
