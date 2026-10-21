use proc_macro2::TokenStream;
use quote::{quote, quote_spanned};
use syn::{ImplItem, Item, ItemImpl, ItemMod, Path, Token, parse::Parser, punctuated::Punctuated};

#[path = "machine/declared.rs"]
mod declared;
#[path = "machine/readonly.rs"]
mod readonly;
#[path = "machine/registration.rs"]
mod registration;

use registration::registrations;

const KINDS: [&str; 6] = ["on", "effect", "invariant", "postcondition", "assumption", "progress"];

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

fn invokes(tokens: TokenStream, macro_name: &str) -> bool {
    let tokens: Vec<_> = tokens.into_iter().collect();
    tokens.windows(3).any(|pair| {
        matches!(pair,
            [proc_macro2::TokenTree::Ident(name), proc_macro2::TokenTree::Punct(mark), proc_macro2::TokenTree::Group(group)]
                if name == macro_name && mark.as_char() == '!'
                    && group.delimiter() != proc_macro2::Delimiter::None
        )
    }) || tokens.into_iter().any(|token| match token {
        proc_macro2::TokenTree::Group(group) => invokes(group.stream(), macro_name),
        _ => false,
    })
}

/// Admits `stay!()` in a handler body; the macro's plain arm is a compile error everywhere else.
fn admit_stay(tokens: TokenStream) -> TokenStream {
    let mut tokens: Vec<_> = tokens.into_iter().collect();
    for index in 0..tokens.len() {
        let admitted = match &tokens[index..] {
            [
                proc_macro2::TokenTree::Ident(name),
                proc_macro2::TokenTree::Punct(mark),
                proc_macro2::TokenTree::Group(group),
                ..,
            ] => name == "stay" && mark.as_char() == '!' && group.stream().is_empty(),
            _ => false,
        };
        if let proc_macro2::TokenTree::Group(group) = &tokens[index] {
            let mut nested = proc_macro2::Group::new(group.delimiter(), admit_stay(group.stream()));
            nested.set_span(group.span());
            tokens[index] = nested.into();
        }
        if admitted && let proc_macro2::TokenTree::Group(group) = &tokens[index + 2] {
            let mut marked =
                proc_macro2::Group::new(group.delimiter(), quote!(__umpire_on_handler));
            marked.set_span(group.span());
            tokens[index + 2] = marked.into();
        }
    }
    tokens.into_iter().collect()
}

fn admit_handler_stays(items: &mut [Item]) -> syn::Result<()> {
    for item in items {
        let Item::Impl(input) = item else { continue };
        for item in &mut input.items {
            if let ImplItem::Fn(method) = item
                && method.attrs.iter().any(|attr| helper(attr, "on"))
            {
                let block = &method.block;
                method.block = syn::parse2(admit_stay(quote!(#block)))?;
            }
        }
    }
    Ok(())
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

fn kind(method: &syn::ImplItemFn) -> manyhow::Result<Option<usize>> {
    let mut found = None;
    for attr in &method.attrs {
        if let Some(rank) = KINDS
            .iter()
            .position(|name| helper(attr, name))
            .or_else(|| helper(attr, "scope").then_some(4))
        {
            if found.is_some() {
                manyhow::bail!(attr, "a method requires exactly one machine kind attribute");
            }
            if rank != 0 && rank != 3 {
                attr.meta.require_path_only()?;
            }
            found = Some(rank);
        }
    }
    Ok(found)
}

fn check_impl(input: &ItemImpl, state: &syn::Ident, parameterized: bool) -> manyhow::Result<()> {
    if input.trait_.is_some() || (!parameterized && !input.generics.params.is_empty()) {
        manyhow::bail!(input, "machine methods require one nongeneric inherent impl {}", state);
    }
    let syn::Type::Path(ty) = input.self_ty.as_ref() else {
        manyhow::bail!(input.self_ty, "machine methods require the module's state type");
    };
    if ty.qself.is_some()
        || !(ty.path.is_ident(state)
            || (parameterized
                && ty.path.segments.len() == 1
                && ty.path.segments[0].ident == *state))
    {
        manyhow::bail!(input.self_ty, "machine methods require the module's state type");
    }
    Ok(())
}

#[derive(Default)]
struct Registrations {
    handlers: Vec<(syn::Ident, syn::Ident)>,
    invariants: Vec<syn::Ident>,
    posts: Vec<(syn::Ident, Option<Path>, usize)>,
    assumptions: Vec<syn::Ident>,
    progresses: Vec<syn::Ident>,
    order: Vec<String>,
    world_methods: std::collections::HashSet<String>,
}

fn mutable_reference(receiver: &syn::Receiver) -> bool {
    match &receiver.kind {
        syn::ReceiverKind::Reference(_, _, mutability) => mutability.is_some(),
        syn::ReceiverKind::Typed(_, ty) => {
            matches!(ty.as_ref(), syn::Type::Reference(ty) if ty.mutability.is_some())
        }
        _ => false,
    }
}

fn check_readonly(input: &ItemImpl, state: &syn::Ident) -> manyhow::Result<()> {
    let syn::Type::Path(ty) = input.self_ty.as_ref() else { return Ok(()) };
    let local_state = ty.qself.is_none()
        && (ty.path.is_ident(state)
            || (ty.path.segments.len() == 2
                && ty.path.segments[0].ident == "self"
                && ty.path.segments[1].ident == *state));
    if !local_state {
        return Ok(());
    }
    for item in &input.items {
        let ImplItem::Fn(method) = item else { continue };
        if matches!(kind(method)?, Some(0 | 1)) {
            continue;
        }
        if let Some(receiver) = method.sig.receiver()
            && (receiver.mutability.is_some() || mutable_reference(receiver))
        {
            manyhow::bail!(
                &method.sig.ident,
                "{} takes &mut self; only #[on] and #[effect] methods change the state",
                method.sig.ident
            );
        }
    }
    Ok(())
}

fn readonly_checks(
    items: &mut Vec<Item>,
    state: &syn::Ident,
    depth: usize,
    registered: bool,
    state_generics: Option<&syn::Generics>,
) -> syn::Result<bool> {
    let probe_root = readonly::root(depth);
    let probe = if state_generics.is_some() {
        quote!(#probe_root::__UmpireReadonlyProbe)
    } else {
        quote!(__Probe)
    };
    let import = state_generics.map(|_| quote!(use #probe_root::__UmpireReadonly as _;));
    let mut state_path = quote!(#state);
    for _ in 0..depth {
        state_path = quote!(super::#state_path);
    }
    if depth == 0 {
        state_path = quote!(self::#state);
    }
    let mut checks = Vec::new();
    for item in &mut *items {
        let Item::Impl(input) = item else { continue };
        let ty = &input.self_ty;
        let attrs = input
            .attrs
            .iter()
            .filter(|attr| attr.path().is_ident("cfg") || attr.path().is_ident("cfg_attr"));
        let (generics, arguments, constraints) = input.generics.split_for_impl();
        let context = input.trait_.as_ref().map(|(path, _)| quote!(: #path));
        let methods = input.items.iter().filter_map(|item| match item {
            ImplItem::Fn(method)
                if !(registered
                    && method
                        .attrs
                        .iter()
                        .any(|attr| helper(attr, "on") || helper(attr, "effect")))
                    && method.sig.receiver().is_some_and(|receiver| {
                        receiver.mutability.is_some() || mutable_reference(receiver)
                    }) =>
            {
                Some(method)
            }
            _ => None,
        });
        let mut guards = Vec::new();
        for method in methods {
            let message = format!(
                "{} takes &mut self; only #[on] and #[effect] methods change the state",
                method.sig.ident
            );
            guards.push(quote_spanned! {method.sig.ident.span()=>
                const _: () = {
                    #import
                    #[diagnostic::on_unimplemented(message = #message)]
                    trait __Allowed {}
                    impl __Allowed for () {}
                    fn __require(_: impl __Allowed) {}
                    trait __Guard #generics #context #constraints {
                        fn __check(&self, _: ::core::marker::PhantomData<#ty>);
                    }
                    impl #generics __Guard #arguments for #ty #constraints {
                        fn __check(&self, _: ::core::marker::PhantomData<#ty>) {
                            __require((&#probe::<Self>(::core::marker::PhantomData)).__probe());
                        }
                    }
                };
            });
        }
        if guards.is_empty() {
            continue;
        }
        let local_probe = state_generics.is_none().then(|| readonly::legacy_probe(&state_path));
        checks.push(syn::parse2(quote! {
            #(#attrs)*
            const _: () = {
                #local_probe
                #(#guards)*
            };
        })?);
    }
    let generated = !checks.is_empty();
    items.extend(checks);
    Ok(generated)
}

struct LocalReadonly<'a> {
    state: &'a syn::Ident,
    generics: Option<&'a syn::Generics>,
    depth: usize,
    error: Option<syn::Error>,
    checks: bool,
}

impl syn::visit_mut::VisitMut for LocalReadonly<'_> {
    fn visit_item_mod_mut(&mut self, module: &mut ItemMod) {
        self.depth += 1;
        syn::visit_mut::visit_item_mod_mut(self, module);
        if let Some((_, items)) = &mut module.content {
            match readonly_checks(items, self.state, self.depth, false, self.generics) {
                Ok(generated) => self.checks |= generated,
                Err(error) => self.error = Some(error),
            }
        }
        self.depth -= 1;
    }

    fn visit_block_mut(&mut self, block: &mut syn::Block) {
        syn::visit_mut::visit_block_mut(self, block);
        let mut items: Vec<_> = block
            .stmts
            .iter()
            .filter_map(|statement| match statement {
                syn::Stmt::Item(Item::Impl(input)) => Some(Item::Impl(input.clone())),
                _ => None,
            })
            .collect();
        let original_count = items.len();
        match readonly_checks(&mut items, self.state, self.depth, false, self.generics) {
            Ok(generated) => self.checks |= generated,
            Err(error) => {
                self.error = Some(error);
                return;
            }
        }
        let statements = std::mem::take(&mut block.stmts);
        block.stmts.extend(items.into_iter().skip(original_count).map(syn::Stmt::Item));
        block.stmts.extend(statements);
    }
}

fn check_receivers(items: &mut Vec<Item>, state: &syn::Ident) -> syn::Result<()> {
    check_receivers_with_generics(items, state, None)
}

fn check_receivers_with_generics(
    items: &mut Vec<Item>,
    state: &syn::Ident,
    generics: Option<&syn::Generics>,
) -> syn::Result<()> {
    let mut local = LocalReadonly { state, generics, depth: 0, error: None, checks: false };
    for item in &mut *items {
        syn::visit_mut::VisitMut::visit_item_mut(&mut local, item);
    }
    if let Some(error) = local.error {
        return Err(error);
    }
    let generated = readonly_checks(items, state, 0, true, generics)?;
    if let Some(generics) = generics
        && (local.checks || generated)
    {
        items.extend(syn::parse2::<syn::File>(readonly::probe(state, generics))?.items);
    }
    Ok(())
}

fn transition_actor<'a>(
    method: &syn::ImplItemFn,
    actors: &'a [syn::Ident],
    arity: usize,
    parameterized: bool,
) -> manyhow::Result<&'a syn::Ident> {
    if let Some(asyncness) = method.sig.asyncness {
        manyhow::bail!(asyncness, "#[on] methods must be synchronous");
    }
    if !matches!(method.sig.output, syn::ReturnType::Default) {
        manyhow::bail!(
            &method.sig.output,
            "#[on] methods return nothing; enabledness is a changed state"
        );
    }
    let actor =
        (arity == 2).then(|| method.sig.inputs.iter().nth(1)).flatten().and_then(|argument| {
            match argument {
                syn::FnArg::Typed(argument) => match argument.ty.as_ref() {
                    syn::Type::Path(ty) if ty.qself.is_none() => actors.iter().find(|actor| {
                        ty.path.is_ident(*actor)
                            || (parameterized
                                && ty.path.segments.len() == 1
                                && ty.path.segments[0].ident == **actor)
                    }),
                    _ => None,
                },
                syn::FnArg::Receiver(_) => None,
            }
        });
    let mutable_receiver = method.sig.receiver().is_some_and(mutable_reference);
    match actor {
        Some(actor) if mutable_receiver => Ok(actor),
        _ => manyhow::bail!(
            &method.sig,
            "an #[on] method takes its actor enum: fn user(&mut self, action: User)"
        ),
    }
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

fn check_progress(method: &syn::ImplItemFn, actors: &[syn::Ident]) -> manyhow::Result<()> {
    struct ClaimPaths<'a> {
        actors: &'a [syn::Ident],
        error: Option<syn::Error>,
    }
    impl syn::visit::Visit<'_> for ClaimPaths<'_> {
        fn visit_macro(&mut self, call: &syn::Macro) {
            if call.path.segments.last().is_some_and(|segment| segment.ident == "path")
                && let Err(error) = crate::path::validate_claim(call.tokens.clone(), self.actors)
            {
                let error = syn::Error::new_spanned(call, error.to_string());
                if let Some(previous) = &mut self.error {
                    previous.combine(error);
                } else {
                    self.error = Some(error);
                }
            }
            syn::visit::visit_macro(self, call);
        }
    }
    let mut paths = ClaimPaths { actors, error: None };
    syn::visit::Visit::visit_impl_item_fn(&mut paths, method);
    paths.error.map_or(Ok(()), |error| Err(error.into()))
}

fn check_legacy(input: &ItemImpl, state: &syn::Ident) -> manyhow::Result<()> {
    for attr in &input.attrs {
        if let Some(name) = [
            "transitions",
            "effects",
            "invariants",
            "postconditions",
            "assumptions",
            "scopes",
            "progress",
        ]
        .iter()
        .find(|name| helper(attr, name))
        {
            manyhow::bail!(
                attr,
                "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[{}]",
                name
            );
        }
    }
    if let Some((path, _)) = &input.trait_
        && path.segments.last().is_some_and(|part| {
            [
                "Transitions",
                "Effects",
                "Invariants",
                "Postconditions",
                "Assumptions",
                "Scopes",
                "Progress",
            ]
            .iter()
            .any(|name| part.ident == name)
        })
        && matches!(input.self_ty.as_ref(), syn::Type::Path(ty) if ty.path.is_ident(state))
    {
        manyhow::bail!(
            path,
            "use per-item machine attributes on an inherent impl {} instead of marker traits",
            state
        );
    }
    Ok(())
}

fn after_action(attr: &syn::Attribute) -> syn::Result<Option<Path>> {
    let mut after = None;
    if matches!(attr.meta, syn::Meta::List(_)) {
        attr.parse_nested_meta(|meta| {
            if !meta.path.is_ident("after") {
                return Err(meta.error("expected after = Actor::Variant"));
            }
            if after.is_some() {
                return Err(meta.error("duplicate after qualifier"));
            }
            after = Some(meta.value()?.parse::<Path>()?);
            Ok(())
        })?;
    } else {
        attr.meta.require_path_only()?;
    }
    Ok(after)
}

fn check_order(attr: &syn::Attribute, rank: usize, previous: Option<usize>) -> manyhow::Result<()> {
    if previous.is_some_and(|previous| rank < previous) {
        let before = rank.checked_sub(1).map_or_else(
            || "the start of the impl".to_owned(),
            |rank| format!("#[{}]", KINDS[rank]),
        );
        let after = KINDS
            .get(rank + 1)
            .map_or_else(|| "the end of the impl".to_owned(), |name| format!("#[{name}]"));
        manyhow::bail!(
            attr,
            "#[{}] methods must follow {} and precede {}",
            KINDS[rank],
            before,
            after
        );
    }
    Ok(())
}

fn postconditions(
    registration: &Registrations,
    actions: &[Path],
) -> manyhow::Result<Vec<TokenStream>> {
    let mut output = Vec::new();
    for (method, after, arity) in &registration.posts {
        let args = match arity {
            1 => quote!(),
            2 => quote!(_before),
            _ => quote!(_before, outcome),
        };
        let has_world = registration.world_methods.contains(&method.to_string());
        let args = if has_world {
            if *arity == 1 { quote!(__umpire_world) } else { quote!(#args, __umpire_world) }
        } else {
            args
        };
        let world = if has_world { quote!(__umpire_world) } else { quote!(_) };
        let predicate = if *arity == 3 {
            quote!(|#world, _before, after, _, outcome, _| after.#method(#args))
        } else {
            quote!(|#world, _before, after, _, _, _| after.#method(#args))
        };
        let label = method.to_string();
        if let Some(path) = after {
            if !actions.iter().any(|known| key(known) == key(path)) {
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
        let has_world = registration.world_methods.contains(&label);
        let argument = has_world.then(|| quote!(__umpire_world));
        let world = if has_world { quote!(__umpire_world) } else { quote!(_) };
        output.push(quote!(::umpire_dsl::Progress::<Self>::evaluated(#label, |#world, state| {
            let value = state.#method(#argument);
            ::umpire_dsl::ProgressEvaluation {
                trigger: value.trigger.judge(state).into(),
                target: value.target.judge(state).into(),
                assumption: value.assumption,
                bound: value.bound,
            }
        })));
    }
    output
}

fn catalog(
    items: &[Item],
    module: &syn::Ident,
) -> manyhow::Result<(syn::Ident, syn::Visibility, Vec<syn::Ident>, Vec<Path>)> {
    let mut state = None;
    let mut actors = Vec::new();
    let mut actions = Vec::new();
    for item in items {
        match item {
            Item::Struct(item) if derives(&item.attrs, "State")? => {
                if state.is_some() {
                    manyhow::bail!(item, "#[machine] requires exactly one #[derive(State)] struct");
                }
                state = Some((item.ident.clone(), item.vis.clone()));
            }
            Item::Enum(item) if derives(&item.attrs, "Action")? => {
                let actor = &item.ident;
                actors.push(actor.clone());
                actions.extend(item.variants.iter().map(|variant| {
                    let variant = &variant.ident;
                    syn::parse_quote!(#actor::#variant)
                }));
            }
            _ => {}
        }
    }
    let (state, visibility) = state.ok_or_else(|| {
        syn::Error::new_spanned(module, "#[machine] requires exactly one #[derive(State)] struct")
    })?;
    Ok((state, visibility, actors, actions))
}

fn assumption_helpers(enabled: bool) -> Option<TokenStream> {
    enabled.then(|| {
        let constructors = ["without", "only", "fair"].map(|name| {
            let name = syn::Ident::new(name, proc_macro2::Span::call_site());
            quote! {
                fn #name<A: ::core::convert::Into<Action>, const N: usize>(actions: [A; N]) -> Assumption {
                    Assumption::#name(&actions.into_iter().map(::core::convert::Into::into).collect::<::std::vec::Vec<Action>>())
                }
            }
        });
        quote! {
            use ::umpire_dsl::AssumptionExt as _;
            type Assumption = ::umpire_dsl::Assumption<Action>;
            #(#constructors)*
        }
    })
}

fn progress_helpers(enabled: bool, state: &syn::Ident) -> Option<TokenStream> {
    enabled.then(|| {
        quote! {
            struct LeadsTo {
                trigger: ::umpire_dsl::Waypoint<#state>,
                target: ::umpire_dsl::Waypoint<#state>,
                assumption: Assumption,
                bound: Option<u32>,
            }
            #[allow(dead_code, reason = "a claim may be written as path! instead")]
            fn whenever(trigger: bool) -> LeadsTo {
                ::umpire_dsl::path![trigger, .., true].into()
            }
            #[allow(dead_code, reason = "a claim may be written as path! instead")]
            impl LeadsTo {
                fn eventually(mut self, target: bool) -> Self {
                    self.target = ::umpire_dsl::Waypoint::Holds(target);
                    self
                }
                fn within(mut self, bound: u32) -> Self { self.bound = Some(bound); self }
                fn assuming(mut self, assumption: Assumption) -> Self {
                    self.assumption = assumption;
                    self
                }
            }
            impl ::core::convert::From<::umpire_dsl::Path<#state>> for LeadsTo {
                fn from(path: ::umpire_dsl::Path<#state>) -> Self {
                    let (trigger, gap, target) = path.claim();
                    LeadsTo { trigger, target, assumption: Assumption::default(), bound: gap.bound() }
                }
            }
            #[allow(dead_code, reason = "a path! claim may carry no assumption")]
            trait ClaimedPath {
                fn assuming(self, assumption: Assumption) -> LeadsTo;
            }
            impl ClaimedPath for ::umpire_dsl::Path<#state> {
                fn assuming(self, assumption: Assumption) -> LeadsTo {
                    LeadsTo::from(self).assuming(assumption)
                }
            }
        }
    })
}

fn stepping(
    handlers: &[(syn::Ident, syn::Ident)],
    rejecting: bool,
    staying: bool,
    world_methods: &std::collections::HashSet<String>,
) -> TokenStream {
    let dispatch = handlers.iter().map(|(actor, method)| {
        let argument = world_methods.contains(&method.to_string()).then(|| quote!(, self));
        if rejecting && staying {
            return quote!(Action::#actor(value) => {
                let mut next = state.clone();
                let (outcome, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| next.#method(value #argument));
                let rejected = matches!(outcome, ::umpire_dsl::Outcome::Rejected(_));
                if rejected && next != *state {
                    return ::core::result::Result::Err(::umpire_dsl::StepError::RejectedAfterMutation);
                }
                let mut steps = ::umpire_dsl::Steps::new();
                if next != *state || rejected || stayed {
                    steps.push((outcome, ::std::vec::Vec::new(), next));
                }
                ::core::result::Result::Ok(steps)
            });
        }
        if rejecting {
            return quote!(Action::#actor(value) => {
                let mut next = state.clone();
                let outcome = ::umpire_dsl::outcome::evaluate(|| next.#method(value #argument));
                if matches!(outcome, ::umpire_dsl::Outcome::Rejected(_)) && next != *state {
                    return ::core::result::Result::Err(::umpire_dsl::StepError::RejectedAfterMutation);
                }
                let mut steps = ::umpire_dsl::Steps::new();
                if next != *state || matches!(outcome, ::umpire_dsl::Outcome::Rejected(_)) {
                    steps.push((outcome, ::std::vec::Vec::new(), next));
                }
                ::core::result::Result::Ok(steps)
            });
        }
        if staying {
            return quote!(Action::#actor(value) => {
                let mut next = state.clone();
                let (_, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| next.#method(value #argument));
                let mut steps = ::umpire_dsl::Steps::new();
                if next != *state || stayed { steps.push(((), ::std::vec::Vec::new(), next)); }
                steps
            });
        }
        quote!(Action::#actor(value) => {
            let mut next = state.clone();
            next.#method(value #argument);
            let mut steps = ::umpire_dsl::Steps::new();
            if next != *state { steps.push(((), ::std::vec::Vec::new(), next)); }
            steps
        })
    });
    if rejecting {
        quote! {
            fn step(&self, state: &Self::State, action: Self::Action)
                -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>
            {
                match <Self as ::umpire_dsl::Machine>::checked_step(self, state, action) {
                    ::core::result::Result::Ok(steps) => steps,
                    ::core::result::Result::Err(error) => panic!("unchecked Model step: {}", error),
                }
            }
            fn checked_step(&self, state: &Self::State, action: Self::Action)
                -> ::core::result::Result<::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>, ::umpire_dsl::StepError>
            {
                match action { #(#dispatch),* }
            }
        }
    } else {
        quote! {
            fn step(&self, state: &Self::State, action: Self::Action)
                -> ::umpire_dsl::Steps<Self::State, Self::Outcome, Self::Fact>
            {
                match action { #(#dispatch),* }
            }
        }
    }
}

fn outcome_catalog(rejecting: bool) -> TokenStream {
    if rejecting {
        quote!(::umpire_dsl::Outcome::catalog())
    } else {
        quote!(::std::vec![::umpire_dsl::OutcomeClass::new("()", |_| true)])
    }
}

fn actor_projections(actors: &[syn::Ident], state: &syn::Ident) -> Vec<TokenStream> {
    actors
        .iter()
        .map(|actor| {
            let others = (actors.len() > 1).then(|| quote!(_ => None,));
            quote! {
            impl ::umpire_dsl::RegisteredActor<#state> for #actor {}
                impl ::umpire_dsl::ActorProjection<#actor> for Action {
                    fn project(&self) -> ::core::option::Option<&#actor> {
                        match self { Self::#actor(value) => Some(value), #others }
                    }
                }
            }
        })
        .collect()
}

fn world_type(
    visibility: &syn::Visibility,
    state: &syn::Ident,
    outcome: &TokenStream,
    actors: &[syn::Ident],
) -> TokenStream {
    if matches!(visibility, syn::Visibility::Public(_)) {
        quote!(World)
    } else {
        quote!(impl ::umpire_dsl::Machine<State = #state, Outcome = #outcome, Fact = (), Action: #(::core::convert::From<#actors> + ::umpire_dsl::ActorProjection<#actors>)+*>)
    }
}

fn check_within(input: &ItemMod) -> manyhow::Result<()> {
    let mut within = WithinAttribute::default();
    syn::visit::Visit::visit_item_mod(&mut within, input);
    if let Some(attr) = within.0 {
        manyhow::bail!(attr, "a progress claim names its assumption with .assuming(..)");
    }
    Ok(())
}

pub(crate) fn expand(args: &TokenStream, input: &ItemMod) -> manyhow::Result<TokenStream> {
    if !args.is_empty() {
        manyhow::bail!(args, "#[machine] takes no arguments");
    }
    check_within(input)?;
    let rejecting = invokes(quote!(#input), "reject");
    let staying = invokes(quote!(#input), "stay");
    let outcome_type = if rejecting { quote!(::umpire_dsl::Outcome) } else { quote!(()) };
    let outcome_catalog = outcome_catalog(rejecting);
    let mut input = input.clone();
    input.attrs.insert(0, syn::parse_quote!(#[forbid(unsafe_code)]));
    let Some((_, items)) = &mut input.content else {
        manyhow::bail!(input, "#[machine] requires an inline module");
    };
    if declared::present(items)? {
        return declared::expand(input, rejecting, staying);
    }
    let (state, visibility, actors, actions) = catalog(items, &input.ident)?;
    admit_handler_stays(items)?;
    check_receivers(items, &state)?;
    let registration = registrations(items, &state, &actors, None)?;
    let post_tokens = postconditions(&registration, &actions)?;
    let progress_tokens = progress_claims(&registration);
    let Registrations { handlers, invariants, assumptions, order, world_methods, .. } =
        registration;
    let stepping = stepping(&handlers, rejecting, staying, &world_methods);
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
    let assumption_helpers =
        assumption_helpers(!assumptions.is_empty() || !progress_tokens.is_empty());
    let progress_helpers = progress_helpers(!progress_tokens.is_empty(), &state);
    let actor_projections = actor_projections(&actors, &state);
    let world_type = world_type(&visibility, &state, &outcome_type, &actors);
    let generated = quote! {
        #[doc = "The module's registered state machine."]
        #visibility struct World;

        #[derive(Clone, Debug, PartialEq, Eq, Hash)]
        #[doc = "The registered actors' action carrier."]
        #visibility enum Action { #(#[doc = stringify!(#actors)] #actors(#actors)),* }

        #(impl ::core::convert::From<#actors> for Action {
            fn from(value: #actors) -> Self { Self::#actors(value) }
        })*

        #(#actor_projections)*

        impl ::core::fmt::Display for Action {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match *self { #(Self::#actors(ref value) => ::core::fmt::Display::fmt(value, out)),* }
            }
        }

        #assumption_helpers
        #progress_helpers

        #[doc = "Builds the module's registered state machine."]
        pub fn world() -> #world_type {
            World
        }

        impl ::umpire_dsl::Machine for World {
            type State = #state;
            type Action = Action;
            type Outcome = #outcome_type;
            type Fact = ();
            fn starts(&self) -> ::std::vec::Vec<Self::State> { #state::starts() }
            fn actions(&self) -> ::std::vec::Vec<Self::Action> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#actors::actions().into_iter().map(Action::#actors));)*
                actions
            }
            fn outcome_catalog(&self) -> ::core::option::Option<::std::vec::Vec<::umpire_dsl::OutcomeClass<Self::Outcome>>> {
                Some(#outcome_catalog)
            }
            fn fault_catalog(&self) -> ::core::option::Option<::std::vec::Vec<Self::Action>> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#actors::actions().into_iter().filter(#actors::fault).map(Action::#actors));)*
                Some(actions)
            }
            fn internal(&self, action: &Self::Action) -> bool {
                match action { #(Action::#actors(value) => value.internal()),* }
            }
            #stepping
            fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> { ::std::vec![#(#invariant_tokens),*] }
            fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> { ::std::vec![#(#post_tokens),*] }
            fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> { ::std::vec![#(#progress_tokens),*] }
            fn assumptions(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Assumption<Self::Action>)> { ::std::vec![#(#assumption_tokens),*] }
            fn claim_order(&self) -> ::std::vec::Vec<&'static str> { ::std::vec![#(#order),*] }
        }

        impl ::umpire_dsl::Registered for #state {
            type World = World;
            fn world() -> World { World }
        }
    };
    items.extend(syn::parse2::<syn::File>(generated)?.items);
    Ok(quote!(#input))
}
