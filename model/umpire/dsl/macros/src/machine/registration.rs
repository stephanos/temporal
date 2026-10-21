use syn::{ImplItem, Item, Path};

use super::{
    KINDS, Registrations, after_action, check_claim, check_impl, check_legacy, check_order,
    check_progress, check_readonly, helper, key, kind, transition_actor,
};

fn register_handler(
    registration: &mut Registrations,
    method: &syn::ImplItemFn,
    attr: &syn::Attribute,
    actors: &[syn::Ident],
    arity: usize,
) -> manyhow::Result<()> {
    let named: Path = attr.parse_args()?;
    if !actors.iter().any(|actor| named.is_ident(actor)) {
        manyhow::bail!(attr, "#[on] names an actor enum declared in this module");
    }
    let actor = transition_actor(method, actors, arity)?;
    if !named.is_ident(actor) {
        manyhow::bail!(&method.sig, "#[on({})] requires a {} parameter", key(&named), key(&named));
    }
    if registration.handlers.iter().any(|(known, _)| known == actor) {
        manyhow::bail!(&method.sig.ident, "actor {} already has an #[on] method", actor);
    }
    registration.handlers.push((actor.clone(), method.sig.ident.clone()));
    Ok(())
}

fn register_starts(
    registration: &mut Registrations,
    method: &syn::ImplItemFn,
    attr: &syn::Attribute,
    declared: bool,
    has_world: bool,
) -> manyhow::Result<()> {
    if !declared {
        manyhow::bail!(attr, "computed starts require a declared World");
    }
    if registration.starts.is_some() {
        manyhow::bail!(attr, "declare starts once");
    }
    if method.sig.receiver().is_some()
        || method.sig.inputs.len() != usize::from(has_world)
        || method.sig.asyncness.is_some()
        || !method.sig.generics.params.is_empty()
    {
        manyhow::bail!(
            &method.sig,
            "#[starts] requires an associated fn taking only an optional trailing &World: fn starts(world: &World) -> impl IntoIterator<Item = Self>"
        );
    }
    registration.starts = Some((method.sig.ident.clone(), has_world));
    Ok(())
}

pub(super) fn world_parameter(method: &syn::ImplItemFn) -> manyhow::Result<bool> {
    let mut parameters = Vec::new();
    for (index, input) in method.sig.inputs.iter().enumerate() {
        let syn::FnArg::Typed(input) = input else { continue };
        let ty = match input.ty.as_ref() {
            syn::Type::Reference(reference) => reference.elem.as_ref(),
            ty => ty,
        };
        if !matches!(ty, syn::Type::Path(ty) if ty.qself.is_none() && ty.path.segments.last().is_some_and(|segment| segment.ident == "World"))
        {
            continue;
        }
        parameters.push((index, input));
    }
    if parameters.len() > 1 {
        manyhow::bail!(parameters[1].1, "declare the World parameter once");
    }
    if let Some((index, input)) = parameters.first() {
        if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if reference.mutability.is_none())
        {
            manyhow::bail!(&input.ty, "the World parameter must be an immutable &World");
        }
        if *index + 1 != method.sig.inputs.len() {
            manyhow::bail!(input, "the World parameter must be last");
        }
    }
    Ok(!parameters.is_empty())
}

pub(super) fn registrations(
    items: &mut [Item],
    state: &syn::Ident,
    actors: &[syn::Ident],
    declared: bool,
) -> manyhow::Result<Registrations> {
    let mut registration = Registrations::default();
    let mut registered_impl = false;
    let mut previous = None;
    for item in items {
        let Item::Impl(input) = item else { continue };
        check_legacy(input, state)?;
        check_readonly(input, state)?;
        let annotated =
            input.items.iter().try_fold(false, |found, item| -> manyhow::Result<bool> {
                Ok(found || matches!(item, ImplItem::Fn(method) if kind(method)?.is_some()))
            })?;
        if !annotated {
            continue;
        }
        if registered_impl {
            manyhow::bail!(input, "machine methods belong in one impl {}", state);
        }
        registered_impl = true;
        check_impl(input, state)?;
        for item in &mut input.items {
            let ImplItem::Fn(method) = item else { continue };
            let Some(rank) = kind(method)? else { continue };
            let attr = method
                .attrs
                .iter()
                .find(|attr| helper(attr, KINDS[rank]) || (rank == 5 && helper(attr, "scope")))
                .ok_or_else(|| syn::Error::new_spanned(&method.sig, "missing machine kind"))?;
            check_order(attr, rank, previous)?;
            previous = Some(rank);
            if matches!(rank, 3 | 4) {
                check_claim(method)?;
            }
            let name = method.sig.ident.clone();
            let has_world = declared && world_parameter(method)?;
            let arity = method.sig.inputs.len() - usize::from(has_world);
            if has_world {
                registration.world_methods.insert(name.to_string());
            }
            match rank {
                0 => register_starts(&mut registration, method, attr, declared, has_world)?,
                1 => register_handler(&mut registration, method, attr, actors, arity)?,
                2 => {}
                3 => registration.invariants.push(name),
                4 => {
                    let after = after_action(attr)?;
                    if !(1..=3).contains(&arity) || method.sig.receiver().is_none() {
                        manyhow::bail!(
                            &method.sig,
                            "postconditions require &self, optional before: &Self, then optional outcome: &Outcome"
                        );
                    }
                    registration.posts.push((name, after, arity));
                    if arity == 3 {
                        method.attrs.push(syn::parse_quote!(#[allow(
                            clippy::trivially_copy_pass_by_ref,
                            reason = "postconditions borrow the emitted edge outcome"
                        )]));
                    }
                }
                5 => registration.assumptions.push(name),
                6 => {
                    check_progress(method, actors)?;
                    let body = &method.block;
                    method.block = syn::parse_quote!({ ::core::convert::From::from(#body) });
                    registration.progresses.push(name);
                }
                _ => unreachable!(),
            }
            if matches!(rank, 3 | 4 | 6) {
                registration.order.push(method.sig.ident.to_string());
            }
            method
                .attrs
                .retain(|attr| !(helper(attr, KINDS[rank]) || rank == 5 && helper(attr, "scope")));
        }
    }
    for actor in actors {
        if !registration.handlers.iter().any(|(known, _)| known == actor) {
            manyhow::bail!(actor, "actor {} has no #[on] method", actor);
        }
    }
    Ok(registration)
}
