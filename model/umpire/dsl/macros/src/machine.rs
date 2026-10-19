use proc_macro2::TokenStream;
use quote::quote;
use syn::{ImplItem, Item, ItemImpl, ItemMod, Path, Token, parse::Parser, punctuated::Punctuated};

const GROUPS: [&str; 5] =
    ["transitions", "invariants", "postconditions", "assumptions", "progress"];

fn helper(attr: &syn::Attribute, name: &str) -> bool {
    attr.path().segments.last().is_some_and(|segment| segment.ident == name)
}

#[derive(Default)]
struct WithinAttribute<'ast>(Option<&'ast syn::Attribute>);

impl<'ast> syn::visit::Visit<'ast> for WithinAttribute<'ast> {
    fn visit_attribute(&mut self, attr: &'ast syn::Attribute) {
        if helper(attr, "within") && self.0.is_none() {
            self.0 = Some(attr);
        }
    }
}

fn derives(attrs: &[syn::Attribute], name: &str) -> syn::Result<bool> {
    for attr in attrs.iter().filter(|attr| attr.path().is_ident("derive")) {
        let paths = Punctuated::<Path, Token![,]>::parse_terminated
            .parse2(attr.meta.require_list()?.tokens.clone())?;
        if paths
            .iter()
            .any(|path| path.segments.last().is_some_and(|segment| segment.ident == name))
        {
            return Ok(true);
        }
    }
    Ok(false)
}

fn key(path: &Path) -> String {
    path.segments.iter().map(|segment| segment.ident.to_string()).collect::<Vec<_>>().join("::")
}

fn owner(path: &Path) -> Path {
    let mut owner = path.clone();
    owner.segments.pop();
    owner.segments.pop_punct();
    owner
}

fn group(input: &ItemImpl) -> manyhow::Result<Option<usize>> {
    let mut found = None;
    for attr in &input.attrs {
        if helper(attr, "scopes") {
            manyhow::bail!(attr, "use #[assumptions] instead of #[scopes]");
        }
        if let Some(rank) = GROUPS.iter().position(|name| helper(attr, name)) {
            if found.is_some() {
                manyhow::bail!(attr, "an impl requires exactly one machine group");
            }
            attr.meta.require_path_only()?;
            found = Some(rank);
        }
    }
    Ok(found)
}

fn check_impl(input: &ItemImpl, state: &syn::Ident) -> manyhow::Result<()> {
    if input.trait_.is_some() || !input.generics.params.is_empty() {
        manyhow::bail!(input, "machine groups require a nongeneric inherent impl");
    }
    let syn::Type::Path(ty) = input.self_ty.as_ref() else {
        manyhow::bail!(input.self_ty, "machine groups require the module's state type");
    };
    if ty.qself.is_some() || !ty.path.is_ident(state) {
        manyhow::bail!(input.self_ty, "machine groups require the module's state type");
    }
    Ok(())
}

fn qualifier(method: &syn::ImplItemFn, name: &str) -> manyhow::Result<Option<Path>> {
    let mut value = None;
    for attr in &method.attrs {
        if helper(attr, name) {
            if value.is_some() {
                manyhow::bail!(attr, "duplicate #[{}] qualifier", name);
            }
            value = Some(attr.parse_args()?);
        }
    }
    Ok(value)
}

#[derive(Default)]
struct Registrations {
    handlers: Vec<(Path, syn::Ident)>,
    invariants: Vec<syn::Ident>,
    posts: Vec<(syn::Ident, Option<Path>, bool)>,
    assumptions: Vec<syn::Ident>,
    progresses: Vec<syn::Ident>,
    order: Vec<String>,
}

fn check_claim(method: &syn::ImplItemFn) -> manyhow::Result<()> {
    if let Some(asyncness) = method.sig.asyncness {
        manyhow::bail!(asyncness, "invariants and postconditions must be synchronous");
    }
    if !matches!(method.sig.output, syn::ReturnType::Default) {
        manyhow::bail!(
            &method.sig.output,
            "invariants and postconditions return nothing; use check!"
        );
    }
    Ok(())
}

fn registrations(
    items: &mut [Item],
    state: &syn::Ident,
    actors: &[syn::Ident],
) -> manyhow::Result<Registrations> {
    let mut registration = Registrations::default();
    let mut seen = [false; GROUPS.len()];
    let mut previous = None;
    for item in items {
        let Item::Impl(input) = item else { continue };
        let Some(rank) = group(input)? else { continue };
        let attr = input
            .attrs
            .iter()
            .find(|attr| helper(attr, GROUPS[rank]))
            .ok_or_else(|| syn::Error::new_spanned(&*input, "missing machine group"))?;
        if seen[rank] {
            manyhow::bail!(
                attr,
                "duplicate #[{}] group; each machine group may appear only once",
                GROUPS[rank]
            );
        }
        if let Some(previous) = previous
            && rank < previous
        {
            manyhow::bail!(
                attr,
                "out-of-order #[{}] group; must precede #[{}]",
                GROUPS[rank],
                GROUPS[previous]
            );
        }
        seen[rank] = true;
        previous = Some(rank);
        check_impl(input, state)?;
        input.attrs.retain(|attr| !helper(attr, GROUPS[rank]));
        for item in &mut input.items {
            let ImplItem::Fn(method) = item else { continue };
            let on = qualifier(method, "on")?;
            let after = qualifier(method, "after")?;
            for (name, present, expected) in
                [("on", on.is_some(), 0), ("after", after.is_some(), 2)]
            {
                if present && rank != expected {
                    manyhow::bail!(
                        &method.sig.ident,
                        "#[{}] requires the #[{}] group",
                        name,
                        GROUPS[expected]
                    );
                }
            }
            if matches!(rank, 1 | 2) {
                check_claim(method)?;
            }
            let name = method.sig.ident.clone();
            match rank {
                0 => {
                    if let Some(path) = on {
                        if let Some(asyncness) = method.sig.asyncness {
                            manyhow::bail!(asyncness, "#[on] handlers must be synchronous");
                        }
                        if !matches!(method.sig.output, syn::ReturnType::Default) {
                            manyhow::bail!(
                                &method.sig.output,
                                "#[on] handlers return nothing; enabledness is a changed state"
                            );
                        }
                        if registration.handlers.iter().any(|(known, _)| key(known) == key(&path)) {
                            manyhow::bail!(&path, "duplicate #[on] for action {}", key(&path));
                        }
                        if !actors.iter().any(|actor| owner(&path).is_ident(actor)) {
                            manyhow::bail!(&path, "unknown action {} in #[on]", key(&path));
                        }
                        registration.handlers.push((path, name));
                    }
                }
                1 => registration.invariants.push(name),
                2 => {
                    let arity = method.sig.inputs.len();
                    if !(1..=2).contains(&arity) || method.sig.receiver().is_none() {
                        manyhow::bail!(
                            &method.sig,
                            "postconditions require &self and an optional before: &Self"
                        );
                    }
                    registration.posts.push((name, after, arity == 2));
                }
                3 => registration.assumptions.push(name),
                4 => registration.progresses.push(name),
                _ => unreachable!(),
            }
            if matches!(rank, 1 | 2 | 4) {
                registration.order.push(method.sig.ident.to_string());
            }
            method.attrs.retain(|attr| !["on", "after"].iter().any(|name| helper(attr, name)));
        }
    }
    Ok(registration)
}

fn postconditions(registration: &Registrations) -> manyhow::Result<Vec<TokenStream>> {
    let mut output = Vec::new();
    for (method, after, before) in &registration.posts {
        let args = before.then(|| quote!(_before));
        let predicate = quote!(|_, _before, after, _, _, _| after.#method(#args));
        let label = method.to_string();
        if let Some(path) = after {
            if !registration.handlers.iter().any(|(known, _)| key(known) == key(path)) {
                manyhow::bail!(path, "unknown after action {}", key(path));
            }
            output.push(
                quote!(::umpire_dsl::StepClaim::<Self>::after_checks(#label, #path.into(), #predicate)),
            );
        } else {
            output.push(quote!(::umpire_dsl::StepClaim::<Self>::with_checks(#label, #predicate)));
        }
    }
    Ok(output)
}

fn progress_claims(registration: &Registrations) -> Vec<TokenStream> {
    let mut output = Vec::new();
    for method in &registration.progresses {
        let label = method.to_string();
        output.push(quote!(::umpire_dsl::Progress::<Self>::evaluated(#label, |_, state| {
            let value = state.#method();
            ::umpire_dsl::ProgressEvaluation {
                trigger: value.trigger.into(),
                target: value.target.into(),
                assumption: value.assumption,
            }
        })));
    }
    output
}

fn catalog(items: &[Item], module: &syn::Ident) -> manyhow::Result<(syn::Ident, Vec<syn::Ident>)> {
    let mut state = None;
    let mut actors = Vec::new();
    for item in items {
        match item {
            Item::Struct(item) if derives(&item.attrs, "State")? => {
                if state.is_some() {
                    manyhow::bail!(item, "#[machine] requires exactly one #[derive(State)] struct");
                }
                state = Some(item.ident.clone());
            }
            Item::Enum(item) if derives(&item.attrs, "Action")? => actors.push(item.ident.clone()),
            _ => {}
        }
    }
    let state = state.ok_or_else(|| {
        syn::Error::new_spanned(module, "#[machine] requires exactly one #[derive(State)] struct")
    })?;
    Ok((state, actors))
}

fn progress_helpers(enabled: bool) -> Option<TokenStream> {
    enabled.then(|| {
        quote! {
            struct Progress { trigger: bool, target: bool, assumption: Assumption }
            fn whenever(trigger: bool) -> Progress {
                Progress { trigger, target: true, assumption: Assumption::default() }
            }
            impl Progress {
                fn eventually(mut self, target: bool) -> Self { self.target = target; self }
                fn assuming(mut self, assumption: Assumption) -> Self {
                    self.assumption = assumption;
                    self
                }
            }
        }
    })
}

pub(crate) fn expand(args: &TokenStream, input: &ItemMod) -> manyhow::Result<TokenStream> {
    if !args.is_empty() {
        manyhow::bail!(args, "#[machine] takes no arguments");
    }
    let mut within = WithinAttribute::default();
    syn::visit::Visit::visit_item_mod(&mut within, input);
    if let Some(attr) = within.0 {
        manyhow::bail!(attr, "a progress claim names its assumption with .assuming(..)");
    }
    let mut input = input.clone();
    let Some((_, items)) = &mut input.content else {
        manyhow::bail!(input, "#[machine] requires an inline module");
    };
    let (state, actors) = catalog(items, &input.ident)?;
    let registration = registrations(items, &state, &actors)?;
    let post_tokens = postconditions(&registration)?;
    let progress_tokens = progress_claims(&registration);
    let Registrations { handlers, invariants, assumptions, order, .. } = registration;
    let dispatch = actors.iter().map(|actor| {
        let arms = handlers.iter().filter(|(path, _)| owner(path).is_ident(actor)).map(
            |(path, method)| {
                quote!(#path => {
                    let mut next = state.clone();
                    next.#method();
                    let mut steps = ::umpire_dsl::Steps::new();
                    if next != *state { steps.push(((), ::std::vec::Vec::new(), next)); }
                    steps
                })
            },
        );
        quote!(Action::#actor(value) => match value { #(#arms),* })
    });
    let invariant_tokens = invariants.iter().map(|method| {
        let label = method.to_string();
        quote!(::umpire_dsl::Claim::<Self>::with_checks(
            #label,
            |_, state| state.#method(),
            |_, state| ::umpire_dsl::claim::evaluate(|| state.#method()).verdict,
        ))
    });
    let assumption_tokens = assumptions.iter().map(|method| {
        let label = method.to_string();
        quote!((#label, #state::#method()))
    });
    let assumption_helpers = (!assumptions.is_empty() || !progress_tokens.is_empty()).then(|| {
        quote! {
            type Assumption = ::umpire_dsl::Assumption<Action>;
            fn without<A: ::core::convert::Into<Action>, const N: usize>(actions: [A; N]) -> Assumption {
                Assumption { excluded: actions.into_iter().map(::core::convert::Into::into).collect() }
            }
        }
    });
    let progress_helpers = progress_helpers(!progress_tokens.is_empty());
    let generated = quote! {
        struct World;

        #[derive(Clone, PartialEq, Eq, Hash)]
        enum Action { #(#actors(#actors)),* }

        #(impl ::core::convert::From<#actors> for Action {
            fn from(value: #actors) -> Self { Self::#actors(value) }
        })*

        impl ::umpire_dsl::Spell for Action {
            fn spell(&self, out: &mut ::std::string::String) {
                match self { #(Self::#actors(value) => ::umpire_dsl::Spell::spell(value, out)),* }
            }
        }

        #assumption_helpers
        #progress_helpers

        #[doc = "Builds the module's registered state machine."]
        pub fn world() -> impl ::umpire_dsl::Machine<State = #state, Outcome = (), Fact = (), Action: #(::core::convert::From<#actors>)+*> {
            World
        }

        impl ::umpire_dsl::Machine for World {
            type State = #state;
            type Action = Action;
            type Outcome = ();
            type Fact = ();
            fn starts(&self) -> ::std::vec::Vec<Self::State> { #state::starts() }
            fn actions(&self) -> ::std::vec::Vec<Self::Action> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#actors::actions().into_iter().map(Action::#actors));)*
                actions
            }
            fn internal(&self, action: &Self::Action) -> bool {
                match action { #(Action::#actors(value) => value.internal()),* }
            }
            fn step(&self, state: &Self::State, action: Self::Action)
                -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>
            {
                match action { #(#dispatch),* }
            }
            fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> { ::std::vec![#(#invariant_tokens),*] }
            fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> { ::std::vec![#(#post_tokens),*] }
            fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> { ::std::vec![#(#progress_tokens),*] }
            fn assumptions(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Assumption<Self::Action>)> { ::std::vec![#(#assumption_tokens),*] }
            fn claim_order(&self) -> ::std::vec::Vec<&'static str> { ::std::vec![#(#order),*] }
        }
    };
    items.extend(syn::parse2::<syn::File>(generated)?.items);
    Ok(quote!(#input))
}
