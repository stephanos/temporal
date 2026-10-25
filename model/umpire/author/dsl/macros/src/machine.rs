use proc_macro2::TokenStream;
use quote::{quote, quote_spanned};
use syn::{ImplItem, Item, ItemMod};
use umpire_front::declare::{self, Kind, Machine, helper, mutable_reference};

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

fn postconditions(machine: &Machine) -> Vec<TokenStream> {
    let mut output = Vec::new();
    for registered in machine.of(Kind::Postcondition) {
        let (method, arity) = (&registered.ident, registered.arity);
        let args = match arity {
            1 => quote!(),
            2 => quote!(_before),
            _ => quote!(_before, outcome),
        };
        let has_world = registered.world;
        let args = if has_world {
            if arity == 1 { quote!(__umpire_world) } else { quote!(#args, __umpire_world) }
        } else {
            args
        };
        let world = if has_world { quote!(__umpire_world) } else { quote!(_) };
        let predicate = if arity == 3 {
            quote!(|#world, _before, after, _, outcome, _| after.#method(#args))
        } else {
            quote!(|#world, _before, after, _, _, _| after.#method(#args))
        };
        let label = method.to_string();
        if let Some(path) = &registered.after {
            output.push(
                quote!(::umpire_dsl::StepClaim::<Self>::after_checks(#label, #path.into(), #predicate)),
            );
        } else {
            output.push(quote!(::umpire_dsl::StepClaim::<Self>::with_checks(#label, #predicate)));
        }
    }
    output
}

fn progress_claims(machine: &Machine) -> Vec<TokenStream> {
    let mut output = Vec::new();
    for registered in machine.of(Kind::Progress) {
        let method = &registered.ident;
        let label = method.to_string();
        let has_world = registered.world;
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

/// Computed starts replace the State derive's field-order starts.
fn mark_computed_starts(items: &mut [Item], state: &syn::Ident) {
    for item in items {
        if let Item::Struct(input) = item
            && input.ident == *state
        {
            input.attrs.push(syn::parse_quote!(#[__umpire_computed_starts]));
        }
    }
}

/// Marks the declared World, the state and each actor so their derives export values to the
/// source lifter's binding and World-dependent domains receive the World.
fn bind_worlds(items: &mut [Item], state: &syn::Ident, actors: &[syn::Ident]) {
    for item in items {
        match item {
            Item::Enum(item) if actors.contains(&item.ident) => {
                item.attrs.push(syn::parse_quote!(#[__umpire_world]));
            }
            Item::Struct(item) if item.ident == "World" || item.ident == *state => {
                item.attrs.push(syn::parse_quote!(#[__umpire_world]));
            }
            _ => {}
        }
    }
}

fn value_exports(machine: &Machine, items: &mut [Item]) -> TokenStream {
    let state = &machine.state;
    let actors = machine.actor_idents();
    if machine.world {
        bind_worlds(items, state, &actors);
    } else {
        for item in items.iter_mut() {
            match item {
                Item::Enum(item) if actors.contains(&item.ident) => {
                    item.attrs.push(syn::parse_quote!(#[__umpire_values]));
                }
                Item::Struct(item) if item.ident == *state => {
                    item.attrs.push(syn::parse_quote!(#[__umpire_values]));
                }
                _ => {}
            }
        }
    }
    let enums = enum_values(machine);
    let actions = action_values(&actors);
    let failures = machine.starts().is_none().then(|| {
        quote! {
            #[cfg(not(target_arch = "wasm32"))]
            fn start_failure(&self, selected: &[(&str, i128)]) -> ::core::option::Option<::umpire_dsl::StartFailure> {
                #state::__umpire_start_failure(selected)
            }
        }
    });
    let field_types = items
        .iter()
        .filter_map(|item| match item {
            Item::Struct(item) if item.ident == *state => Some(item.fields.iter().map(|f| &f.ty)),
            _ => None,
        })
        .flatten();
    let bounds = super::state::deferred(&syn::Generics::default(), field_types);
    let (_, _, constraints) = bounds.split_for_impl();
    let world = (!machine.world).then(|| quote! {
            #[doc(hidden)]
            impl ::umpire_dsl::Values for World #constraints {
                fn values(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)> { ::std::vec![] }
                #failures
            }
        });
    quote!(#world #actions #(#enums)*)
}

fn nominal_values(items: &[Item]) -> Vec<TokenStream> {
    items.iter().filter_map(|item| {
        let Item::Struct(item) = item else { return None };
        let input = syn::DeriveInput {
            attrs: item.attrs.clone(), vis: item.vis.clone(), ident: item.ident.clone(),
            generics: item.generics.clone(),
            data: syn::Data::Struct(syn::DataStruct {
                struct_token: item.struct_token, fields: item.fields.clone(), semi_token: item.semi_token,
            }),
        };
        let derives_finite = item.attrs.iter().filter(|attr| attr.path().is_ident("derive"))
            .any(|attr| attr.parse_args_with(syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated)
                .is_ok_and(|paths| paths.iter().any(|path| {
                    let names = path.segments.iter().map(|part| part.ident.to_string()).collect::<Vec<_>>();
                    names == ["Finite"] || names == ["umpire_dsl", "Finite"]
                })));
        if !derives_finite || !input.generics.params.is_empty() { return None; }
        let declare::Finite::Newtype { field, .. } = declare::finite(&input).ok()? else { return None };
        let name = &item.ident;
        let label = name.to_string();
        let member = field.map_or_else(|| quote!(0), |field| quote!(#field));
        Some(quote! {
            #[doc(hidden)]
            impl ::umpire_dsl::IntoValue for #name {
                fn value(&self) -> ::umpire_dsl::Value {
                    ::umpire_dsl::Value::Bounded { name: #label, raw: <::core::primitive::i128 as ::core::convert::TryFrom<_>>::try_from(self.#member).unwrap_or_else(|_| ::core::panic!("nominal value exceeds the exported integer range")) }
                }
            }
        })
    }).collect()
}

/// A local fieldless enum, as the lifter admits it, exports its variant as a value unless its
/// `derive(Finite)` already does.
fn enum_values(machine: &Machine) -> Vec<TokenStream> {
    let mut values = Vec::new();
    for (name, variants) in &machine.enums {
        let labels = variants.iter().map(|variant| format!("{name}::{variant}"));
        values.push(quote! {
            #[doc(hidden)]
            impl ::umpire_dsl::IntoValue for #name {
                fn value(&self) -> ::umpire_dsl::Value {
                    match *self { #(Self::#variants => ::umpire_dsl::Value::Enum(#labels)),* }
                }
            }
        });
    }
    values
}

/// The carrier's values delegate to each actor's derived export.
fn action_values(actors: &[syn::Ident]) -> TokenStream {
    quote! {
        #[doc(hidden)]
        impl ::umpire_dsl::ActionValues<World> for Action
        where
            #(for<'__umpire> #actors: ::umpire_dsl::ActionValues<World>),*
        {
            fn action_value(&self) -> (&'static str, ::std::vec::Vec<(&'static str, ::umpire_dsl::Value)>) {
                match self { #(Self::#actors(value) => ::umpire_dsl::ActionValues::<World>::action_value(value)),* }
            }
            fn domains(world: &World) -> ::std::vec::Vec<(&'static str, &'static str, ::std::vec::Vec<::umpire_dsl::Value>)> {
                let mut domains = ::std::vec::Vec::new();
                #(domains.extend(<#actors as ::umpire_dsl::ActionValues<World>>::domains(world));)*
                domains
            }
        }
    }
}

/// Counts each declared variant's lazy domain before materializing, for machines with inputs.
/// Fieldless machines keep the default length check and their exact expansion.
fn catalog_admission(actors: &[declare::Actor]) -> Option<TokenStream> {
    let bearing = |actor: &declare::Actor| actor.variants.iter().any(|(_, fields)| *fields);
    if !actors.iter().any(bearing) {
        return None;
    }
    let entries = actors.iter().map(|actor| {
        let name = &actor.ident;
        if bearing(actor) {
            quote! {
                __umpire_variants.extend(
                    <#name as ::umpire_dsl::path_build::ActorCatalog<Self>>::catalog(self, limit)
                        .into_iter()
                        .map(|(name, domain)| (name, domain.map(Action::#name))),
                );
            }
        } else {
            let variants = actor.variants.iter().map(|(ident, _)| {
                let label = format!("{name}::{ident}");
                quote!((::std::string::String::from(#label), ::umpire_dsl::ActionDomain::new(Ok(1), |_| <#name>::#ident).map(Action::#name)))
            });
            quote!(__umpire_variants.extend([#(#variants),*]);)
        }
    });
    Some(quote! {
        fn checked_catalog(&self, limit: usize) -> ::core::result::Result<::umpire_dsl::ActionCatalog<Self::Action>, ::umpire_dsl::CatalogError> {
            let mut __umpire_variants = ::std::vec::Vec::new();
            #(#entries)*
            ::umpire_dsl::ActionCatalog::admit(limit, __umpire_variants)
        }
    })
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

fn fact_dispatch(
    actor: &syn::Ident,
    method: &syn::Ident,
    argument: Option<&TokenStream>,
    rejecting: bool,
    staying: bool,
) -> TokenStream {
    let evaluate = if rejecting && staying {
        quote!(let (outcome, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| { fact = next.#method(value #argument); });)
    } else if rejecting {
        quote!(let outcome = ::umpire_dsl::outcome::evaluate(|| { fact = next.#method(value #argument); }); let stayed = false;)
    } else if staying {
        quote!(let (_, stayed) = ::umpire_dsl::outcome::evaluate_staying(|| { fact = next.#method(value #argument); }); let outcome = ();)
    } else {
        quote!(let fact = next.#method(value #argument); let outcome = (); let stayed = false;)
    };
    let capture =
        (rejecting || staying).then(|| quote!(let mut fact = ::core::option::Option::None;));
    let rejected = if rejecting {
        quote!(matches!(outcome, ::umpire_dsl::Outcome::Rejected(_)))
    } else {
        quote!(false)
    };
    let mutation_check = rejecting.then(|| {
        quote! {
            if rejected && next != *state {
                return ::core::result::Result::Err(::umpire_dsl::StepError::RejectedAfterMutation);
            }
        }
    });
    let result = if rejecting { quote!(::core::result::Result::Ok(steps)) } else { quote!(steps) };
    quote!(Action::#actor(value) => {
        let mut next = state.clone();
        #capture
        #evaluate
        let rejected = #rejected;
        #mutation_check
        let mut steps = ::umpire_dsl::Steps::new();
        if next != *state || rejected || stayed {
            steps.push((outcome, fact.into_iter().collect(), next));
        }
        #result
    })
}

fn stepping(
    handlers: &[(syn::Ident, syn::Ident, bool, bool)],
    rejecting: bool,
    staying: bool,
) -> TokenStream {
    let dispatch = handlers.iter().map(|(actor, method, world, fact)| {
        let argument = world.then(|| quote!(, self));
        if *fact {
            return fact_dispatch(actor, method, argument.as_ref(), rejecting, staying);
        }
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
    fact: &TokenStream,
) -> TokenStream {
    if matches!(visibility, syn::Visibility::Public(_)) {
        quote!(World)
    } else {
        quote!(impl ::umpire_dsl::Machine<State = #state, Outcome = #outcome, Fact = #fact, Action: #(::core::convert::From<#actors> + ::umpire_dsl::ActorProjection<#actors>)+*>)
    }
}

fn invariant_claims(machine: &Machine) -> Vec<TokenStream> {
    machine
        .of(Kind::Invariant)
        .map(|registered| {
            let method = &registered.ident;
            let label = method.to_string();
            let has_world = registered.world;
            let world = if has_world { quote!(__umpire_world) } else { quote!(_) };
            let argument = has_world.then(|| quote!(__umpire_world));
            quote!(::umpire_dsl::Claim::<Self>::with_checks(
                #label,
                |#world, state| state.#method(#argument),
                |#world, state| ::umpire_dsl::claim::evaluate(|| state.#method(#argument)).verdict,
            ))
        })
        .collect()
}

fn assumption_claims(machine: &Machine) -> Vec<TokenStream> {
    let state = &machine.state;
    machine
        .of(Kind::Assumption)
        .map(|registered| {
            let method = &registered.ident;
            let label = method.to_string();
            let argument = registered.world.then(|| quote!(self));
            quote!((#label, #state::#method(#argument)))
        })
        .collect()
}

fn starts_call(machine: &Machine) -> TokenStream {
    let state = &machine.state;
    if let Some(registered) = machine.starts() {
        let method = &registered.ident;
        // A wrong return type is reported at the method, not at the state declaration.
        let state = syn::Ident::new(&state.to_string(), method.span());
        let argument = registered.world.then(|| quote_spanned!(method.span()=> self));
        quote_spanned!(method.span()=> #state::#method(#argument).into_iter().collect())
    } else {
        quote!(#state::starts())
    }
}

/// World-dependent actors enumerate from the declared World; static actors keep their
/// static catalog calls.
fn actor_lists(machine: &Machine) -> Vec<TokenStream> {
    machine
        .actors
        .iter()
        .map(|actor| {
            let dependent = machine.world && actor.world_dependent;
            let actor = &actor.ident;
            if dependent {
                quote!(<#actor as ::umpire_dsl::path_build::ActorCatalog<Self>>::actions(self))
            } else {
                quote!(#actor::actions())
            }
        })
        .collect()
}

/// A declared World is the machine and only associates the state; otherwise the module gets
/// the unit World, its constructor and the state-only registration.
fn association(
    declared: bool,
    visibility: &syn::Visibility,
    state: &syn::Ident,
    outcome: &TokenStream,
    actors: &[syn::Ident],
    fact: &TokenStream,
) -> (Option<TokenStream>, TokenStream, Option<TokenStream>) {
    if declared {
        return (
            None,
            quote!(impl ::umpire_dsl::Associated for #state { type World = World; }),
            None,
        );
    }
    let world_type = world_type(visibility, state, outcome, actors, fact);
    (
        Some(quote! {
            #[doc = "The module's registered state machine."]
            #visibility struct World;
        }),
        quote! {
            #[doc = "Builds the module's registered state machine."]
            pub fn world() -> #world_type {
                World
            }
        },
        Some(quote! {
            impl ::umpire_dsl::Registered for #state {
                type World = World;
                fn world() -> World { World }
            }
        }),
    )
}

/// The method edits generation needs: kind attributes are consumed, an outcome-reading
/// postcondition borrows the edge outcome, and a progress body converts into its claim.
fn edit_methods(items: &mut [Item], machine: &Machine) {
    for registered in &machine.methods {
        let Item::Impl(input) = &mut items[registered.item] else { continue };
        let ImplItem::Fn(method) = &mut input.items[registered.index] else { continue };
        if registered.kind == Kind::Postcondition && registered.arity == 3 {
            method.attrs.push(syn::parse_quote!(#[allow(
                clippy::trivially_copy_pass_by_ref,
                reason = "postconditions borrow the emitted edge outcome"
            )]));
        }
        if registered.kind == Kind::Progress {
            let body = &method.block;
            method.block = syn::parse_quote!({ ::core::convert::From::from(#body) });
        }
        method.attrs.retain(|attr| !registered.kind.marks(attr));
    }
}

#[allow(dead_code, reason = "expansion snapshots render the module without its embedded Model")]
pub(crate) fn expand(args: &TokenStream, input: &ItemMod) -> manyhow::Result<TokenStream> {
    let module = generate(&declare::machine(args, input)?, input)?;
    finish(input, &module)
}

/// The expanded module: the executable Model.
pub(crate) fn generate(machine: &Machine, input: &ItemMod) -> manyhow::Result<ItemMod> {
    let (state, visibility) = (&machine.state, &machine.visibility);
    let actors = machine.actor_idents();
    let outcome_type = if machine.rejecting { quote!(::umpire_dsl::Outcome) } else { quote!(()) };
    let fact_type = machine.fact.as_ref().map_or_else(|| quote!(()), |fact| quote!(#fact));
    let outcome_catalog = outcome_catalog(machine.rejecting);
    let admission = crate::roots::admission(input);
    let nominal =
        admission.content.as_ref().map_or_else(Vec::new, |(_, items)| nominal_values(items));
    let mut input = input.clone();
    input.attrs.insert(0, syn::parse_quote!(#[forbid(unsafe_code)]));
    let Some((_, items)) = &mut input.content else { unreachable!("declare reads inline modules") };
    admit_handler_stays(items)?;
    check_receivers(items, state)?;
    edit_methods(items, machine);
    if machine.starts().is_some() {
        mark_computed_starts(items, state);
    }
    let values = value_exports(machine, items);
    let values = quote!(#values #(#nominal)*);
    let post_tokens = postconditions(machine);
    let progress_tokens = progress_claims(machine);
    let invariant_tokens = invariant_claims(machine);
    let assumption_tokens = assumption_claims(machine);
    let start = starts_call(machine);
    let lists = actor_lists(machine);
    let handlers: Vec<_> = machine
        .of(Kind::On)
        .map(|registered| {
            (
                registered
                    .actor
                    .clone()
                    .unwrap_or_else(|| unreachable!("an #[on] method names its actor")),
                registered.ident.clone(),
                registered.world,
                registered.fact,
            )
        })
        .collect();
    let order = machine.claim_order();
    let stepping = stepping(&handlers, machine.rejecting, machine.staying);
    let assumption_helpers = assumption_helpers(
        machine.of(Kind::Assumption).next().is_some() || !progress_tokens.is_empty(),
    );
    let progress_helpers = progress_helpers(!progress_tokens.is_empty(), state);
    let actor_projections = actor_projections(&actors, state);
    let admission = catalog_admission(&machine.actors);
    let (world, association, registered) =
        association(machine.world, visibility, state, &outcome_type, &actors, &fact_type);
    let generated = quote! {
        #world

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

        #association

        impl ::umpire_dsl::Machine for World {
            type State = #state;
            type Action = Action;
            type Outcome = #outcome_type;
            type Fact = #fact_type;
            fn starts(&self) -> ::std::vec::Vec<Self::State> { #start }
            fn actions(&self) -> ::std::vec::Vec<Self::Action> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#lists.into_iter().map(Action::#actors));)*
                actions
            } #admission
            fn outcome_catalog(&self) -> ::core::option::Option<::std::vec::Vec<::umpire_dsl::OutcomeClass<Self::Outcome>>> {
                Some(#outcome_catalog)
            }
            fn fault_catalog(&self) -> ::core::option::Option<::std::vec::Vec<Self::Action>> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(#lists.into_iter().filter(#actors::fault).map(Action::#actors));)*
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

        #registered
        #values
    };
    items.extend(syn::parse2::<syn::File>(generated)?.items);
    Ok(input)
}

/// Resolves generated dependencies while retaining opaque variant declarations verbatim.
pub(crate) fn finish(original: &ItemMod, module: &ItemMod) -> manyhow::Result<TokenStream> {
    let mut module: ItemMod = syn::parse2(crate::roots::resolve(quote!(#module)))?;
    let Some((_, original_items)) = &original.content else { unreachable!("inline module") };
    let Some((_, items)) = &mut module.content else { unreachable!("inline module") };
    let mut actors = Vec::new();
    for original in original_items {
        let Item::Enum(original) = original else { continue };
        if !declare::derives(&original.attrs, "Action")? {
            continue;
        }
        actors.push(original);
        let Some(Item::Enum(generated)) = items
            .iter_mut()
            .find(|item| matches!(item, Item::Enum(item) if item.ident == original.ident))
        else {
            unreachable!("registered actor retained")
        };
        for (variant, source) in generated.variants.iter_mut().zip(&original.variants) {
            for (attr, source) in variant.attrs.iter_mut().zip(&source.attrs) {
                if source.path().is_ident("metadata") {
                    *attr = source.clone();
                }
            }
        }
    }
    let metadata =
        crate::action::metadata_bridge(&syn::parse_quote!(action_metadata), &quote!(#(#actors)*));
    items.extend(syn::parse2::<syn::File>(metadata)?.items);
    Ok(quote!(#module))
}
