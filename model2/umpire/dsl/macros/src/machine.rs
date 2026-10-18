use proc_macro2::TokenStream;
use quote::{format_ident, quote};
use syn::{Expr, ImplItem, ItemImpl, Meta, Path, Token, parse::Parser, punctuated::Punctuated};

fn helper(attr: &syn::Attribute, name: &str) -> bool {
    attr.path().segments.last().is_some_and(|segment| segment.ident == name)
}

fn option(attr: &syn::Attribute, name: &str) -> syn::Result<Option<Expr>> {
    if matches!(attr.meta, Meta::Path(_)) {
        return Ok(None);
    }
    let value: syn::MetaNameValue = attr.parse_args()?;
    if !value.path.is_ident(name) {
        return Err(syn::Error::new_spanned(value.path, format!("expected {name} = ...")));
    }
    Ok(Some(value.value))
}

fn key(path: &Path) -> String {
    path.segments.iter().map(|segment| segment.ident.to_string()).collect::<Vec<_>>().join("::")
}

fn state_name(input: &ItemImpl) -> manyhow::Result<&syn::Ident> {
    if input.trait_.is_some() || !input.generics.params.is_empty() {
        manyhow::bail!(input, "#[machine] requires a nongeneric inherent impl");
    }
    let syn::Type::Path(state) = input.self_ty.as_ref() else {
        manyhow::bail!(input.self_ty, "#[machine] requires a named state type");
    };
    if state.qself.is_some()
        || state.path.segments.len() != 1
        || !matches!(state.path.segments[0].arguments, syn::PathArguments::None)
    {
        manyhow::bail!(state, "#[machine] requires an unqualified state name");
    }
    Ok(&state.path.segments[0].ident)
}

fn actors(args: TokenStream, input: &ItemImpl) -> manyhow::Result<Vec<Path>> {
    let options = Punctuated::<Meta, Token![,]>::parse_terminated.parse2(args)?;
    if options.len() != 1 {
        manyhow::bail!(input.self_ty, "expected actions = (Actor, ...)");
    }
    let Meta::NameValue(value) = &options[0] else {
        manyhow::bail!(options[0], "expected actions = (Actor, ...)");
    };
    let Expr::Tuple(tuple) = &value.value else {
        manyhow::bail!(value, "expected actions = (Actor, ...)");
    };
    if !value.path.is_ident("actions") || tuple.elems.is_empty() {
        manyhow::bail!(value, "expected a nonempty actions = (Actor, ...) catalog");
    }
    Ok(tuple
        .elems
        .iter()
        .map(|expr| {
            if let Expr::Path(path) = expr {
                Ok(path.path.clone())
            } else {
                Err(syn::Error::new_spanned(expr, "expected an action enum path"))
            }
        })
        .collect::<Result<_, _>>()?)
}

fn owner(path: &Path) -> Path {
    let mut owner = path.clone();
    owner.segments.pop();
    owner.segments.pop_punct();
    owner
}

#[derive(Default)]
struct Registrations<'a> {
    handlers: Vec<(Path, &'a syn::Ident)>,
    invariants: Vec<&'a syn::Ident>,
    posts: Vec<(&'a syn::Ident, Option<Expr>)>,
    scopes: Vec<&'a syn::Ident>,
    progresses: Vec<(&'a syn::Ident, Option<Expr>)>,
    order: Vec<String>,
}

fn registrations<'a>(input: &'a ItemImpl, actors: &[Path]) -> manyhow::Result<Registrations<'a>> {
    let mut errors = manyhow::Emitter::new();
    let actor_names: Vec<_> =
        actors.iter().map(|path| &path.segments[path.segments.len() - 1].ident).collect();
    for (i, actor_name) in actor_names.iter().enumerate() {
        if actor_names[..i].contains(actor_name) {
            errors.emit(syn::Error::new_spanned(actor_name, "duplicate actor name in actions"));
        }
    }
    let mut registration = Registrations::default();
    for item in &input.items {
        let attrs = match item {
            ImplItem::Fn(method) => &method.attrs,
            ImplItem::Const(item) => &item.attrs,
            _ => continue,
        };
        let registrations: Vec<_> = attrs
            .iter()
            .filter(|attr| {
                ["on", "invariant", "postcondition", "scope", "progress"]
                    .iter()
                    .any(|name| helper(attr, name))
            })
            .collect();
        if registrations.is_empty() {
            continue;
        }
        let ImplItem::Fn(method) = item else {
            errors.emit(syn::Error::new_spanned(item, "machine registrations require ordinary functions; use a scope function instead of a const"));
            continue;
        };
        if registrations.len() != 1 && registrations.iter().any(|attr| !helper(attr, "on")) {
            errors.emit(syn::Error::new_spanned(
                &method.sig.ident,
                "a method requires exactly one registration attribute",
            ));
            continue;
        }
        let method_name = &method.sig.ident;
        for attr in registrations {
            if helper(attr, "on") {
                let path: Path = attr.parse_args()?;
                if registration.handlers.iter().any(|(known, _)| key(known) == key(&path)) {
                    errors.emit(syn::Error::new_spanned(
                        &path,
                        format!("duplicate #[on] for action {}", key(&path)),
                    ));
                }
                let actor = owner(&path);
                if !actors.iter().any(|known| key(known) == key(&actor)) {
                    errors.emit(syn::Error::new_spanned(
                        &path,
                        format!("unknown action {} in #[on]", key(&path)),
                    ));
                }
                registration.handlers.push((path, method_name));
            } else if helper(attr, "scope") {
                if !matches!(attr.meta, Meta::Path(_)) {
                    errors.emit(syn::Error::new_spanned(attr, "#[scope] takes no arguments"));
                }
                registration.scopes.push(method_name);
            } else {
                registration.order.push(method_name.to_string());
                if helper(attr, "invariant") {
                    if !matches!(attr.meta, Meta::Path(_)) {
                        errors
                            .emit(syn::Error::new_spanned(attr, "#[invariant] takes no arguments"));
                    }
                    registration.invariants.push(method_name);
                } else if helper(attr, "postcondition") {
                    registration.posts.push((method_name, option(attr, "after")?));
                } else {
                    registration.progresses.push((method_name, option(attr, "within")?));
                }
            }
        }
    }
    errors.into_result()?;
    Ok(registration)
}

fn postconditions(
    registration: &Registrations<'_>,
    action: &syn::Ident,
) -> manyhow::Result<Vec<TokenStream>> {
    let mut errors = manyhow::Emitter::new();
    let mut post_tokens = Vec::new();
    for (method, after) in &registration.posts {
        let predicate = quote!(|_, before, after, _, _, _| after.#method(before).into());
        let label = method.to_string();
        if let Some(after) = after {
            let Expr::Path(path) = &after else {
                manyhow::bail!(after, "after requires an action path");
            };
            if !registration.handlers.iter().any(|(known, _)| key(known) == key(&path.path)) {
                errors.emit(syn::Error::new_spanned(
                    after,
                    format!("unknown after action {}", key(&path.path)),
                ));
                continue;
            }
            let actor = &path.path.segments[path.path.segments.len() - 2].ident;
            post_tokens.push(quote!(::umpire_dsl::StepClaim::<Self>::after(#label, #action::#actor(#after), #predicate)));
        } else {
            post_tokens.push(quote!(::umpire_dsl::StepClaim::<Self>::new(#label, #predicate)));
        }
    }
    errors.into_result()?;
    Ok(post_tokens)
}

fn progress_claims(
    registration: &Registrations<'_>,
    state: &syn::Ident,
) -> manyhow::Result<Vec<TokenStream>> {
    let mut output = Vec::new();
    for (method, within) in &registration.progresses {
        let label = method.to_string();
        let within = if let Some(expr) = within {
            let Expr::Call(call) = expr else {
                manyhow::bail!(
                    expr,
                    "within requires a registered scope function call: Self::scope()"
                );
            };
            let Expr::Path(path) = call.func.as_ref() else {
                manyhow::bail!(
                    expr,
                    "within requires a registered scope function call: Self::scope()"
                );
            };
            if !call.args.is_empty()
                || path.path.segments.len() != 2
                || (path.path.segments[0].ident != "Self" && path.path.segments[0].ident != *state)
            {
                manyhow::bail!(
                    expr,
                    "within requires a registered scope function call: Self::scope()"
                );
            }
            let scope = &path.path.segments[1].ident;
            if !registration.scopes.contains(&scope) {
                manyhow::bail!(expr, "unknown scope {} in within", scope);
            }
            quote!(claim = claim.within(#state::#scope());)
        } else {
            TokenStream::new()
        };
        output.push(
            quote!({ let mut claim = #state::#method(); claim.name = #label; #within claim }),
        );
    }
    Ok(output)
}

pub(crate) fn expand(args: TokenStream, input: &ItemImpl) -> manyhow::Result<TokenStream> {
    let state = state_name(input)?;
    let world = format_ident!("{state}World");
    let action = format_ident!("{state}Action");
    let actors = actors(args, input)?;
    let registration = registrations(input, &actors)?;
    let post_tokens = postconditions(&registration, &action)?;
    let progress_tokens = progress_claims(&registration, state)?;
    let Registrations { handlers, invariants, scopes, order, .. } = registration;
    let actor_names: Vec<_> =
        actors.iter().map(|path| &path.segments[path.segments.len() - 1].ident).collect();
    let dispatch = actors.iter().zip(&actor_names).map(|(actor, variant)| {
        let arms = handlers.iter().filter(|(path, _)| key(&owner(path)) == key(actor)).map(
            |(path, method)| {
                quote!(#path => {
                    let mut next = state.clone();
                    let enabled: ::core::option::Option<()> = next.#method();
                    let mut steps = ::umpire_dsl::Steps::new();
                    if enabled.is_some() { steps.push(((), ::std::vec::Vec::new(), next)); }
                    steps
                })
            },
        );
        quote!(#action::#variant(value) => match value { #(#arms),* })
    });
    let invariant_tokens = invariants.iter().map(|method| {
        let label = method.to_string();
        quote!(::umpire_dsl::Claim::<Self>::new(#label, |_, state| state.#method().into()))
    });
    let scope_tokens = scopes.iter().map(|method| {
        let label = method.to_string();
        quote!((#label, #state::#method()))
    });
    Ok(quote! {
        #input

        #[doc = "The registered state machine's world."]
        struct #world;

        #[doc = "Actions in actor declaration order, spelled with their original Rust names."]
        #[derive(Clone, PartialEq, Eq, Hash)]
        enum #action { #(#actor_names(#actors)),* }

        impl ::umpire_dsl::Spell for #action {
            fn spell(&self, out: &mut ::std::string::String) {
                match self { #(Self::#actor_names(value) => ::umpire_dsl::Spell::spell(value, out)),* }
            }
        }

        impl ::umpire_dsl::Machine for #world {
            type State = #state;
            type Action = #action;
            type Outcome = ();
            type Fact = ();
            fn starts(&self) -> ::std::vec::Vec<Self::State> { #state::starts() }
            fn actions(&self) -> ::std::vec::Vec<Self::Action> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#actors::actions().into_iter().map(#action::#actor_names));)*
                actions
            }
            fn internal(&self, action: &Self::Action) -> bool {
                match action { #(#action::#actor_names(value) => value.internal()),* }
            }
            fn step(&self, state: &Self::State, action: Self::Action)
                -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>
            {
                match action { #(#dispatch),* }
            }
            fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> {
                ::std::vec![#(#invariant_tokens),*]
            }
            fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> {
                ::std::vec![#(#post_tokens),*]
            }
            fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> {
                ::std::vec![#(#progress_tokens),*]
            }
            fn scopes(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Scope<Self::Action>)> {
                ::std::vec![#(#scope_tokens),*]
            }
            fn claim_order(&self) -> ::std::vec::Vec<&'static str> { ::std::vec![#(#order),*] }
        }
    })
}
