use proc_macro2::TokenStream;
use quote::quote;
use syn::{Item, ItemMod};

use super::{Registrations, derives, helper};

pub(super) fn present(items: &[Item]) -> syn::Result<bool> {
    items.iter().try_fold(false, |found, item| {
        Ok(found || matches!(item, Item::Struct(item) if derives(&item.attrs, "World")?))
    })
}

pub(super) fn world_parameter(
    method: &syn::ImplItemFn,
    world: &syn::Ident,
) -> manyhow::Result<bool> {
    let mut parameters = Vec::new();
    for (index, input) in method.sig.inputs.iter().enumerate() {
        let syn::FnArg::Typed(input) = input else { continue };
        let ty = match input.ty.as_ref() {
            syn::Type::Reference(reference) => reference.elem.as_ref(),
            ty => ty,
        };
        if !matches!(ty, syn::Type::Path(ty) if ty.qself.is_none() && ty.path.segments.last().is_some_and(|segment| segment.ident == *world))
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

fn merge_bounds(target: &mut syn::Generics, source: &syn::Generics) {
    for parameter in &source.params {
        if !target.params.iter().any(|known| parameter_name(known) == parameter_name(parameter)) {
            let mut parameter = parameter.clone();
            match &mut parameter {
                syn::GenericParam::Type(parameter) => parameter.bounds.clear(),
                syn::GenericParam::Lifetime(parameter) => parameter.bounds.clear(),
                syn::GenericParam::Const(_) => {}
            }
            target.params.push(parameter);
        }
    }
    if let Some(clause) = &source.where_clause {
        target.make_where_clause().predicates.extend(clause.predicates.clone());
    }
    for parameter in &source.params {
        match parameter {
            syn::GenericParam::Type(parameter) if !parameter.bounds.is_empty() => {
                let name = &parameter.ident;
                let bounds = &parameter.bounds;
                target.make_where_clause().predicates.push(syn::parse_quote!(#name: #bounds));
            }
            syn::GenericParam::Lifetime(parameter) if !parameter.bounds.is_empty() => {
                let name = &parameter.lifetime;
                let bounds = &parameter.bounds;
                target.make_where_clause().predicates.push(syn::parse_quote!(#name: #bounds));
            }
            _ => {}
        }
    }
}

fn parameter_name(parameter: &syn::GenericParam) -> String {
    match parameter {
        syn::GenericParam::Type(parameter) => parameter.ident.to_string(),
        syn::GenericParam::Const(parameter) => parameter.ident.to_string(),
        syn::GenericParam::Lifetime(parameter) => parameter.lifetime.to_string(),
    }
}

fn typed(name: &syn::Ident, generics: &syn::Generics) -> syn::Type {
    let (_, arguments, _) = generics.split_for_impl();
    syn::parse_quote!(#name #arguments)
}

fn starts(items: &mut [Item], state: &syn::Ident) -> manyhow::Result<bool> {
    let mut computed = false;
    for item in &mut *items {
        let Item::Impl(input) = item else { continue };
        let Some(attribute) = input.attrs.iter().find(|attribute| helper(attribute, "starts"))
        else {
            continue;
        };
        attribute.meta.require_path_only()?;
        if computed {
            manyhow::bail!(attribute, "declare starts once");
        }
        super::check_impl(input, state, true)?;
        let [syn::ImplItem::Fn(method)] = input.items.as_slice() else {
            manyhow::bail!(input, "#[starts] requires one fn starts(world: &World)");
        };
        if method.sig.ident != "starts"
            || method.sig.receiver().is_some()
            || method.sig.inputs.len() != 1
            || !world_parameter(method, &syn::parse_quote!(World))?
            || method.sig.asyncness.is_some()
        {
            manyhow::bail!(&method.sig, "#[starts] requires fn starts(world: &World)");
        }
        computed = true;
        input.attrs.retain(|attribute| !helper(attribute, "starts"));
    }
    if computed {
        for item in items {
            if let Item::Struct(input) = item
                && input.ident == *state
            {
                if let Some(attribute) = input
                    .fields
                    .iter()
                    .flat_map(|field| &field.attrs)
                    .find(|attribute| helper(attribute, "starts"))
                {
                    manyhow::bail!(attribute, "declare starts once");
                }
                input.attrs.push(syn::parse_quote!(#[__umpire_computed_starts]));
            }
        }
    }
    Ok(computed)
}

fn declarations(
    items: &[Item],
    module: &syn::Ident,
    state: &syn::Ident,
) -> manyhow::Result<(syn::ItemStruct, syn::ItemStruct)> {
    let worlds: Vec<_> = items
        .iter()
        .filter_map(|item| match item {
            Item::Struct(item) if item.ident == "World" => Some(item.clone()),
            _ => None,
        })
        .collect();
    let [world] = worlds.as_slice() else {
        manyhow::bail!(module, "declare World once as #[derive(World)] struct World");
    };
    if !derives(&world.attrs, "World")? {
        manyhow::bail!(world, "declared World requires #[derive(World)]");
    }
    for item in items {
        if let Item::Struct(item) = item
            && item.ident != "World"
            && derives(&item.attrs, "World")?
        {
            manyhow::bail!(item, "declare World once");
        }
    }
    let state_item = items
        .iter()
        .find_map(|item| match item {
            Item::Struct(item) if item.ident == *state => Some(item.clone()),
            _ => None,
        })
        .unwrap_or_else(|| unreachable!("catalog found state"));
    if state_item.generics.params.iter().map(parameter_name).collect::<Vec<_>>()
        != world.generics.params.iter().map(parameter_name).collect::<Vec<_>>()
    {
        manyhow::bail!(&world.generics, "Model and World must declare the same generic parameters");
    }
    Ok((state_item, world.clone()))
}

fn machine_generics(
    state: &syn::ItemStruct,
    world: &syn::Generics,
    actor_items: &[&syn::ItemEnum],
    actor_types: &[syn::Type],
    computed: bool,
) -> (syn::Generics, syn::Generics) {
    let mut action_generics = syn::Generics::default();
    for actor in actor_items {
        merge_bounds(&mut action_generics, &actor.generics);
    }
    let mut generics = syn::Generics::default();
    merge_bounds(&mut generics, &state.generics);
    merge_bounds(&mut generics, world);
    merge_bounds(&mut generics, &action_generics);
    for field in &state.fields {
        let ty = &field.ty;
        generics.make_where_clause().predicates.push(syn::parse_quote!(#ty: ::core::clone::Clone + ::core::cmp::Eq + ::core::hash::Hash + ::core::fmt::Debug));
        if !computed && !field.attrs.iter().any(|attribute| helper(attribute, "starts")) {
            generics
                .make_where_clause()
                .predicates
                .push(syn::parse_quote!(#ty: ::core::default::Default));
        }
    }
    for actor in actor_types {
        generics
            .make_where_clause()
            .predicates
            .push(syn::parse_quote!(#actor: ::umpire_dsl::Actor));
        action_generics
            .make_where_clause()
            .predicates
            .push(syn::parse_quote!(#actor: ::umpire_dsl::Actor));
    }
    (generics, action_generics)
}

pub(super) fn expand(
    mut input: ItemMod,
    rejecting: bool,
    staying: bool,
) -> manyhow::Result<TokenStream> {
    let (_, items) =
        input.content.as_mut().unwrap_or_else(|| unreachable!("inline module checked"));
    let (state, _, actors, actions) = super::catalog(items, &input.ident)?;
    let (state_item, world) = declarations(items, &input.ident, &state)?;
    let computed = starts(items, &state)?;
    super::admit_handler_stays(items)?;
    super::check_receivers_with_generics(items, &state, Some(&state_item.generics))?;
    let registration = super::registrations(items, &state, &actors, Some(&world.ident))?;
    let posts = super::postconditions(&registration, &actions)?;
    let progresses = super::progress_claims(&registration);
    let step =
        super::stepping(&registration.handlers, rejecting, staying, &registration.world_methods);
    let outcome = if rejecting { quote!(::umpire_dsl::Outcome) } else { quote!(()) };
    let outcome_catalog = super::outcome_catalog(rejecting);
    let state_type = typed(&state, &state_item.generics);
    let world_type = typed(&world.ident, &world.generics);
    let actor_items: Vec<_> = items
        .iter()
        .filter_map(|item| match item {
            Item::Enum(item) if actors.contains(&item.ident) => Some(item),
            _ => None,
        })
        .collect();
    let actor_types: Vec<_> =
        actor_items.iter().map(|item| typed(&item.ident, &item.generics)).collect();
    let (generics, action_generics) =
        machine_generics(&state_item, &world.generics, &actor_items, &actor_types, computed);
    let (implementation, _, constraints) = generics.split_for_impl();
    let (_, action_arguments, _) = action_generics.split_for_impl();
    let action_type = quote!(Action #action_arguments);
    let (projection_implementation, _, projection_constraints) = action_generics.split_for_impl();
    let visibility = &state_item.vis;
    let carrier = action_carrier(visibility, &action_generics, &actors, &actor_types);
    let projections = actors.iter().zip(&actor_types).map(|(actor, ty)| {
        let other = (actors.len() > 1).then(|| quote!(_ => None,));
        quote! {
            impl #implementation ::umpire_dsl::RegisteredActor<#state_type> for #ty #constraints {}
            impl #projection_implementation ::umpire_dsl::ActorProjection<#ty> for #action_type #projection_constraints {
                fn project(&self) -> Option<&#ty> { match self { Self::#actor(value) => Some(value), #other } }
            }
        }
    });
    let invariants = invariants(&registration);
    let assumptions = registration.assumptions.iter().map(|method| {
        let label = method.to_string();
        let argument = registration.world_methods.contains(&label).then(|| quote!(self));
        quote!((#label, <#state_type>::#method(#argument)))
    });
    let helpers = helpers(
        &state_type,
        &state_item.generics,
        &action_generics,
        &action_type,
        !registration.assumptions.is_empty() || !progresses.is_empty(),
        !progresses.is_empty(),
    )?;
    let start = if computed {
        quote!(<#state_type>::starts(self).into_iter().collect())
    } else {
        quote!(<#state_type>::starts())
    };
    let Registrations { order, .. } = &registration;
    let generated = quote! {
        #carrier
        #(#projections)*
        #helpers
        impl #implementation ::umpire_dsl::Machine for #world_type #constraints {
            type State = #state_type;
            type Action = #action_type;
            type Outcome = #outcome;
            type Fact = ();
            fn starts(&self) -> ::std::vec::Vec<Self::State> { #start }
            fn actions(&self) -> ::std::vec::Vec<Self::Action> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(<#actor_types as ::umpire_dsl::Actor>::actions().into_iter().map(Action::#actors));)*
                actions
            }
            fn outcome_catalog(&self) -> Option<::std::vec::Vec<::umpire_dsl::OutcomeClass<Self::Outcome>>> { Some(#outcome_catalog) }
            fn fault_catalog(&self) -> Option<::std::vec::Vec<Self::Action>> {
                let mut actions = ::std::vec::Vec::new();
                #(actions.extend(<#actor_types as ::umpire_dsl::Actor>::actions().into_iter().filter(::umpire_dsl::Actor::fault).map(Action::#actors));)*
                Some(actions)
            }
            fn internal(&self, action: &Self::Action) -> bool {
                match action { #(Action::#actors(value) => ::umpire_dsl::Actor::internal(value)),* }
            }
            #step
            fn invariants(&self) -> ::std::vec::Vec<::umpire_dsl::Claim<Self>> { ::std::vec![#(#invariants),*] }
            fn postconditions(&self) -> ::std::vec::Vec<::umpire_dsl::StepClaim<Self>> { ::std::vec![#(#posts),*] }
            fn progress(&self) -> ::std::vec::Vec<::umpire_dsl::Progress<Self>> { ::std::vec![#(#progresses),*] }
            fn assumptions(&self) -> ::std::vec::Vec<(&'static str, ::umpire_dsl::Assumption<Self::Action>)> { ::std::vec![#(#assumptions),*] }
            fn claim_order(&self) -> ::std::vec::Vec<&'static str> { ::std::vec![#(#order),*] }
        }
        impl #implementation ::umpire_dsl::Associated for #state_type #constraints {
            type World = #world_type;
        }
    };
    items.extend(syn::parse2::<syn::File>(generated)?.items);
    Ok(quote!(#input))
}

fn invariants(registration: &Registrations) -> Vec<TokenStream> {
    registration
        .invariants
        .iter()
        .map(|method| {
            let label = method.to_string();
            let has_world = registration.world_methods.contains(&label);
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

fn action_carrier(
    visibility: &syn::Visibility,
    generics: &syn::Generics,
    actors: &[syn::Ident],
    types: &[syn::Type],
) -> TokenStream {
    let (implementation, arguments, constraints) = generics.split_for_impl();
    quote! {
        #[doc = "The registered actors' action carrier."]
        #visibility enum Action #generics #constraints { #(#[doc = stringify!(#actors)] #actors(#types)),* }
        impl #implementation ::core::clone::Clone for Action #arguments #constraints {
            fn clone(&self) -> Self { match self { #(Self::#actors(value) => Self::#actors(value.clone())),* } }
        }
        impl #implementation ::core::cmp::PartialEq for Action #arguments #constraints {
            fn eq(&self, other: &Self) -> bool {
                #[allow(unreachable_patterns)]
                match (self, other) { #((Self::#actors(value), Self::#actors(other)) => value == other,)* _ => false }
            }
        }
        impl #implementation ::core::cmp::Eq for Action #arguments #constraints {}
        impl #implementation ::core::hash::Hash for Action #arguments #constraints {
            fn hash<__UmpireHasher: ::core::hash::Hasher>(&self, hasher: &mut __UmpireHasher) {
                ::core::hash::Hash::hash(&::core::mem::discriminant(self), hasher);
                match self { #(Self::#actors(value) => ::core::hash::Hash::hash(value, hasher)),* }
            }
        }
        impl #implementation ::core::fmt::Debug for Action #arguments #constraints {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match self { #(Self::#actors(value) => out.debug_tuple(stringify!(#actors)).field(value).finish()),* }
            }
        }
        impl #implementation ::core::fmt::Display for Action #arguments #constraints {
            fn fmt(&self, out: &mut ::core::fmt::Formatter<'_>) -> ::core::fmt::Result {
                match self { #(Self::#actors(value) => ::core::fmt::Display::fmt(value, out)),* }
            }
        }
        #(impl #implementation ::core::convert::From<#types> for Action #arguments #constraints {
            fn from(value: #types) -> Self { Self::#actors(value) }
        })*
    }
}

struct GenericHelpers<'a> {
    state: &'a syn::Type,
    generics: &'a syn::Generics,
    action_generics: &'a syn::Generics,
}
impl syn::visit_mut::VisitMut for GenericHelpers<'_> {
    fn visit_type_mut(&mut self, ty: &mut syn::Type) {
        if let syn::Type::Path(path) = ty
            && path.qself.is_none()
        {
            if path.path.is_ident("__UmpireState") {
                *ty = self.state.clone();
                return;
            }
            let parameters = if path.path.is_ident("LeadsTo") {
                Some(self.generics)
            } else if path.path.is_ident("Assumption") {
                Some(self.action_generics)
            } else {
                None
            };
            if let Some(parameters) = parameters {
                let name = path.path.segments[0].ident.clone();
                *ty = typed(&name, parameters);
                return;
            }
        }
        syn::visit_mut::visit_type_mut(self, ty);
    }
    fn visit_item_struct_mut(&mut self, item: &mut syn::ItemStruct) {
        item.generics = self.generics.clone();
        syn::visit_mut::visit_item_struct_mut(self, item);
    }
    fn visit_item_trait_mut(&mut self, item: &mut syn::ItemTrait) {
        item.generics = self.generics.clone();
        syn::visit_mut::visit_item_trait_mut(self, item);
    }
    fn visit_item_impl_mut(&mut self, item: &mut syn::ItemImpl) {
        item.generics = self.generics.clone();
        if let Some((path, _)) = &mut item.trait_
            && path.is_ident("ClaimedPath")
        {
            let (_, arguments, _) = self.generics.split_for_impl();
            *path = syn::parse_quote!(ClaimedPath #arguments);
        }
        syn::visit_mut::visit_item_impl_mut(self, item);
    }
    fn visit_item_fn_mut(&mut self, item: &mut syn::ItemFn) {
        item.sig.generics = self.generics.clone();
        syn::visit_mut::visit_item_fn_mut(self, item);
    }
}

fn helpers(
    state: &syn::Type,
    state_generics: &syn::Generics,
    action_generics: &syn::Generics,
    action: &TokenStream,
    assumptions: bool,
    progress: bool,
) -> syn::Result<TokenStream> {
    let mut output = TokenStream::new();
    if assumptions {
        let (_, arguments, constraints) = action_generics.split_for_impl();
        let mut function_generics = action_generics.clone();
        function_generics
            .params
            .push(syn::parse_quote!(__UmpireAction: ::core::convert::Into<#action>));
        function_generics.params.push(syn::parse_quote!(const __UMPIRE_N: usize));
        let (function_parameters, _, function_constraints) = function_generics.split_for_impl();
        let constructors = ["without", "only", "fair"].map(|name| {
            let name = syn::Ident::new(name, proc_macro2::Span::call_site());
            quote! {
                fn #name #function_parameters (actions: [__UmpireAction; __UMPIRE_N]) -> Assumption #arguments #function_constraints {
                    ::umpire_dsl::Assumption::#name(&actions.into_iter().map(::core::convert::Into::into).collect::<::std::vec::Vec<#action>>())
                }
            }
        });
        output.extend(quote! {
            use ::umpire_dsl::AssumptionExt as _;
            #[allow(type_alias_bounds, reason = "the action carrier enforces its own authored bounds")]
            type Assumption #action_generics #constraints = ::umpire_dsl::Assumption<#action>;
            #(#constructors)*
        });
    }
    if progress {
        let name = syn::Ident::new("__UmpireState", proc_macro2::Span::call_site());
        let original = super::progress_helpers(true, &name).unwrap_or_default();
        let mut file = syn::parse2::<syn::File>(original)?;
        syn::visit_mut::VisitMut::visit_file_mut(
            &mut GenericHelpers { state, generics: state_generics, action_generics },
            &mut file,
        );
        output.extend(quote!(#file));
    }
    Ok(output)
}
