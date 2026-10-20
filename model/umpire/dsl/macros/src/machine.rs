use proc_macro2::TokenStream;
use quote::{quote, quote_spanned};
use syn::{ImplItem, Item, ItemImpl, ItemMod, Path, Token, parse::Parser, punctuated::Punctuated};

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
        if let Some(rank) = KINDS.iter().position(|name| helper(attr, name)) {
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

fn check_impl(input: &ItemImpl, state: &syn::Ident) -> manyhow::Result<()> {
    if input.trait_.is_some() || !input.generics.params.is_empty() {
        manyhow::bail!(input, "machine methods require one nongeneric inherent impl {}", state);
    }
    let syn::Type::Path(ty) = input.self_ty.as_ref() else {
        manyhow::bail!(input.self_ty, "machine methods require the module's state type");
    };
    if ty.qself.is_some() || !ty.path.is_ident(state) {
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
) -> syn::Result<()> {
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
                    #[diagnostic::on_unimplemented(message = #message)]
                    trait __Allowed {}
                    impl __Allowed for () {}
                    fn __require(_: impl __Allowed) {}
                    trait __Guard #generics #context #constraints {
                        fn __check(&self, _: ::core::marker::PhantomData<#ty>);
                    }
                    impl #generics __Guard #arguments for #ty #constraints {
                        fn __check(&self, _: ::core::marker::PhantomData<#ty>) {
                            __require((&__Probe::<Self>(::core::marker::PhantomData)).__probe());
                        }
                    }
                };
            });
        }
        if guards.is_empty() {
            continue;
        }
        checks.push(syn::parse2(quote! {
            #(#attrs)*
            const _: () = {
                struct __Probe<T: ?Sized>(::core::marker::PhantomData<T>);
                struct __MutableModelMethod;
                trait __Readonly {
                    type __Result;
                    fn __probe(self) -> Self::__Result;
                }
                impl __Readonly for &__Probe<#state_path> {
                    type __Result = __MutableModelMethod;
                    fn __probe(self) -> Self::__Result { __MutableModelMethod }
                }
                impl<T: ?Sized> __Readonly for &&__Probe<T> {
                    type __Result = ();
                    fn __probe(self) {}
                }
                #(#guards)*
            };
        })?);
    }
    items.extend(checks);
    Ok(())
}

struct LocalReadonly<'a> {
    state: &'a syn::Ident,
    depth: usize,
    error: Option<syn::Error>,
}

impl syn::visit_mut::VisitMut for LocalReadonly<'_> {
    fn visit_item_mod_mut(&mut self, module: &mut ItemMod) {
        self.depth += 1;
        syn::visit_mut::visit_item_mod_mut(self, module);
        if let Some((_, items)) = &mut module.content
            && let Err(error) = readonly_checks(items, self.state, self.depth, false)
        {
            self.error = Some(error);
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
        if let Err(error) = readonly_checks(&mut items, self.state, self.depth, false) {
            self.error = Some(error);
            return;
        }
        let statements = std::mem::take(&mut block.stmts);
        block.stmts.extend(items.into_iter().skip(original_count).map(syn::Stmt::Item));
        block.stmts.extend(statements);
    }
}

fn check_receivers(items: &mut Vec<Item>, state: &syn::Ident) -> syn::Result<()> {
    let mut local = LocalReadonly { state, depth: 0, error: None };
    for item in &mut *items {
        syn::visit_mut::VisitMut::visit_item_mut(&mut local, item);
    }
    if let Some(error) = local.error {
        return Err(error);
    }
    readonly_checks(items, state, 0, true)
}

fn transition_actor<'a>(
    method: &syn::ImplItemFn,
    actors: &'a [syn::Ident],
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
    let actor = (method.sig.inputs.len() == 2)
        .then(|| method.sig.inputs.last())
        .flatten()
        .and_then(|argument| match argument {
            syn::FnArg::Typed(argument) => match argument.ty.as_ref() {
                syn::Type::Path(ty) if ty.qself.is_none() => {
                    actors.iter().find(|actor| ty.path.is_ident(*actor))
                }
                _ => None,
            },
            syn::FnArg::Receiver(_) => None,
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

/// Refuses a claim returning a literal `path!` chain, possibly under `.assuming(..)`; a `Path`
/// built elsewhere is refused by `Path::claim` when evaluated.
fn check_progress(method: &syn::ImplItemFn) -> manyhow::Result<()> {
    let Some(syn::Stmt::Expr(tail, None)) = method.block.stmts.last() else { return Ok(()) };
    let mut tail = tail;
    while let syn::Expr::MethodCall(call) = tail {
        tail = &call.receiver;
    }
    let syn::Expr::Macro(call) = tail else { return Ok(()) };
    if call.mac.path.segments.last().is_none_or(|segment| segment.ident != "path") {
        return Ok(());
    }
    let Ok(parts) =
        Punctuated::<syn::Expr, Token![,]>::parse_terminated.parse2(call.mac.tokens.clone())
    else {
        return Ok(());
    };
    let gap = |part: &syn::Expr| matches!(part, syn::Expr::Range(range) if range.start.is_none());
    if parts.iter().filter(|part| !gap(part)).count() > 2 {
        manyhow::bail!(
            call,
            "a progress claim has one trigger and one target; ask a longer progression with find or Run::along"
        );
    }
    Ok(())
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

fn registrations(
    items: &mut [Item],
    state: &syn::Ident,
    actors: &[syn::Ident],
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
                .find(|attr| helper(attr, KINDS[rank]))
                .ok_or_else(|| syn::Error::new_spanned(&method.sig, "missing machine kind"))?;
            check_order(attr, rank, previous)?;
            previous = Some(rank);
            if matches!(rank, 2 | 3) {
                check_claim(method)?;
            }
            let name = method.sig.ident.clone();
            match rank {
                0 => {
                    let named: Path = attr.parse_args()?;
                    if !actors.iter().any(|actor| named.is_ident(actor)) {
                        manyhow::bail!(attr, "#[on] names an actor enum declared in this module");
                    }
                    let actor = transition_actor(method, actors)?;
                    if !named.is_ident(actor) {
                        manyhow::bail!(
                            &method.sig,
                            "#[on({})] requires a {} parameter",
                            key(&named),
                            key(&named)
                        );
                    }
                    if registration.handlers.iter().any(|(known, _)| known == actor) {
                        manyhow::bail!(
                            &method.sig.ident,
                            "actor {} already has an #[on] method",
                            actor
                        );
                    }
                    registration.handlers.push((actor.clone(), name));
                }
                1 => {}
                2 => registration.invariants.push(name),
                3 => {
                    let after = after_action(attr)?;
                    let arity = method.sig.inputs.len();
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
                4 => registration.assumptions.push(name),
                5 => {
                    check_progress(method)?;
                    let body = &method.block;
                    method.block = syn::parse_quote!({ ::core::convert::From::from(#body) });
                    registration.progresses.push(name);
                }
                _ => unreachable!(),
            }
            if matches!(rank, 2 | 3 | 5) {
                registration.order.push(method.sig.ident.to_string());
            }
            method.attrs.retain(|attr| !helper(attr, KINDS[rank]));
        }
    }
    for actor in actors {
        if !registration.handlers.iter().any(|(known, _)| known == actor) {
            manyhow::bail!(actor, "actor {} has no #[on] method", actor);
        }
    }
    Ok(registration)
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
        let predicate = if *arity == 3 {
            quote!(|_, _before, after, _, outcome, _| after.#method(#args))
        } else {
            quote!(|_, _before, after, _, _, _| after.#method(#args))
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
        output.push(quote!(::umpire_dsl::Progress::<Self>::evaluated(#label, |_, state| {
            let value = state.#method();
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

fn stepping(handlers: &[(syn::Ident, syn::Ident)], rejecting: bool, staying: bool) -> TokenStream {
    let dispatch = handlers.iter().map(|(actor, method)| {
        if rejecting && staying {
            return quote!(Action::#actor(value) => {
                let mut next = state.clone();
                let (outcome, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| next.#method(value));
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
                let outcome = ::umpire_dsl::outcome::evaluate(|| next.#method(value));
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
                let (_, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| next.#method(value));
                let mut steps = ::umpire_dsl::Steps::new();
                if next != *state || stayed { steps.push(((), ::std::vec::Vec::new(), next)); }
                steps
            });
        }
        quote!(Action::#actor(value) => {
            let mut next = state.clone();
            next.#method(value);
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

pub(crate) fn expand(args: &TokenStream, input: &ItemMod) -> manyhow::Result<TokenStream> {
    if !args.is_empty() {
        manyhow::bail!(args, "#[machine] takes no arguments");
    }
    let mut within = WithinAttribute::default();
    syn::visit::Visit::visit_item_mod(&mut within, input);
    if let Some(attr) = within.0 {
        manyhow::bail!(attr, "a progress claim names its assumption with .assuming(..)");
    }
    let rejecting = invokes(quote!(#input), "reject");
    let staying = invokes(quote!(#input), "stay");
    let outcome_type = if rejecting { quote!(::umpire_dsl::Outcome) } else { quote!(()) };
    let mut input = input.clone();
    input.attrs.insert(0, syn::parse_quote!(#[forbid(unsafe_code)]));
    let Some((_, items)) = &mut input.content else {
        manyhow::bail!(input, "#[machine] requires an inline module");
    };
    let (state, visibility, actors, actions) = catalog(items, &input.ident)?;
    admit_handler_stays(items)?;
    check_receivers(items, &state)?;
    let registration = registrations(items, &state, &actors)?;
    let post_tokens = postconditions(&registration, &actions)?;
    let progress_tokens = progress_claims(&registration);
    let Registrations { handlers, invariants, assumptions, order, .. } = registration;
    let stepping = stepping(&handlers, rejecting, staying);
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
    let world_type = if matches!(visibility, syn::Visibility::Public(_)) {
        quote!(World)
    } else {
        quote!(impl ::umpire_dsl::Machine<State = #state, Outcome = #outcome_type, Fact = (), Action: #(::core::convert::From<#actors>)+*>)
    };
    let generated = quote! {
        #[doc = "The module's registered state machine."]
        #visibility struct World;

        #[derive(Clone, Debug, PartialEq, Eq, Hash)]
        #[doc = "The registered actors' action carrier."]
        #visibility enum Action { #(#[doc = stringify!(#actors)] #actors(#actors)),* }

        #(impl ::core::convert::From<#actors> for Action {
            fn from(value: #actors) -> Self { Self::#actors(value) }
        })*

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
