//! The declarations of a `#[machine]` module: the attribute grammar, registration and claim
//! order, the State/World/actor schema, starts, input domains and `&World` parameters.
//!
//! Native generation and the lifted reading both read declarations here, so an unsupported
//! declaration has one located error. These refusals are structural: without the declaration
//! the Model has no executable meaning, so the macros report them as compile errors.
use proc_macro2::TokenStream;
use quote::ToTokens;
use syn::{ImplItem, Item, ItemImpl, ItemMod, Path, Token, parse::Parser, punctuated::Punctuated};

/// Refuse at `node` with a formatted message.
macro_rules! bail {
    ($node:expr, $($message:tt)+) => {
        return Err(syn::Error::new_spanned($node, format!($($message)+)))
    };
}

mod action;
mod fields;
pub use action::{ActionEnum, FieldDomain, action_enum};
pub(crate) use fields::struct_fields;
pub use fields::{Finite, finite, integer, named_fields, start_ranges};

type Result<T> = syn::Result<T>;

/// A registered method kind, in the order one impl declares them.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord)]
pub enum Kind {
    /// `#[starts]`: computed initial states.
    Starts,
    /// `#[on(Actor)]`: the actor's handler.
    On,
    /// `#[effect]`: a state change a handler calls.
    Effect,
    /// `#[invariant]`.
    Invariant,
    /// `#[postcondition]`.
    Postcondition,
    /// `#[assumption]`, or its legacy spelling `#[scope]`.
    Assumption,
    /// `#[progress]`.
    Progress,
}

const KINDS: [Kind; 7] = [
    Kind::Starts,
    Kind::On,
    Kind::Effect,
    Kind::Invariant,
    Kind::Postcondition,
    Kind::Assumption,
    Kind::Progress,
];

impl Kind {
    /// The attribute name.
    #[must_use]
    pub const fn name(self) -> &'static str {
        match self {
            Self::Starts => "starts",
            Self::On => "on",
            Self::Effect => "effect",
            Self::Invariant => "invariant",
            Self::Postcondition => "postcondition",
            Self::Assumption => "assumption",
            Self::Progress => "progress",
        }
    }
    /// Whether `attr` marks a method of this kind.
    #[must_use]
    pub fn marks(self, attr: &syn::Attribute) -> bool {
        helper(attr, self.name()) || (self == Self::Assumption && helper(attr, "scope"))
    }
}

/// A registered method of the State's machine impl.
#[derive(Clone, Debug)]
pub struct Method {
    /// Method name.
    pub ident: syn::Ident,
    /// Registered kind.
    pub kind: Kind,
    /// Whether the method takes the declared World as a trailing `&World`.
    pub world: bool,
    /// Parameter count without the World, the receiver included.
    pub arity: usize,
    /// The actor an `#[on]` method handles.
    pub actor: Option<syn::Ident>,
    /// Whether the handler returns an optional fact.
    pub fact: bool,
    /// A postcondition's `after = Actor::Variant` filter.
    pub after: Option<Path>,
    /// The impl's position among the module's items.
    pub item: usize,
    /// The method's position among the impl's items.
    pub index: usize,
}

/// An actor enum of the module.
#[derive(Clone, Debug)]
pub struct Actor {
    /// Enum name.
    pub ident: syn::Ident,
    /// Variants in declaration order, with whether each carries input fields.
    pub variants: Vec<(syn::Ident, bool)>,
    /// Whether a field draws its domain from the World through `#[domain]`.
    pub world_dependent: bool,
}

/// A `#[machine]` module's declarations.
#[derive(Clone, Debug)]
pub struct Machine {
    /// The `#[derive(State)]` struct.
    pub state: syn::Ident,
    /// Its visibility.
    pub visibility: syn::Visibility,
    /// `#[derive(Action)]` enums in declaration order.
    pub actors: Vec<Actor>,
    /// Every `Actor::Variant` in declaration order.
    pub actions: Vec<Path>,
    /// Whether the module declares `#[derive(World)] struct World`.
    pub world: bool,
    /// Whether any body uses `reject!`.
    pub rejecting: bool,
    /// Whether any body uses `stay!`.
    pub staying: bool,
    /// The optional local fact carrier.
    pub fact: Option<syn::Ident>,
    /// Registered methods in declaration order.
    pub methods: Vec<Method>,
    /// Local fieldless enums outside the actors and `derive(Finite)`, with their variants.
    pub enums: Vec<(syn::Ident, Vec<syn::Ident>)>,
}

impl Machine {
    /// The registered methods of one kind, in declaration order.
    pub fn of(&self, kind: Kind) -> impl Iterator<Item = &Method> {
        self.methods.iter().filter(move |method| method.kind == kind)
    }
    /// The `#[starts]` method.
    #[must_use]
    pub fn starts(&self) -> Option<&Method> {
        self.of(Kind::Starts).next()
    }
    /// Invariant, postcondition and progress names, in declaration order.
    #[must_use]
    pub fn claim_order(&self) -> Vec<String> {
        self.methods
            .iter()
            .filter(|method| {
                matches!(method.kind, Kind::Invariant | Kind::Postcondition | Kind::Progress)
            })
            .map(|method| method.ident.to_string())
            .collect()
    }
    /// The actor idents.
    #[must_use]
    pub fn actor_idents(&self) -> Vec<syn::Ident> {
        self.actors.iter().map(|actor| actor.ident.clone()).collect()
    }
}

/// Whether an attribute's last path segment is `name`.
#[must_use]
pub fn helper(attr: &syn::Attribute, name: &str) -> bool {
    attr.path().segments.last().is_some_and(|segment| segment.ident == name)
}

/// Whether an item derives `name`, by any path ending in it.
pub fn derives(attrs: &[syn::Attribute], name: &str) -> Result<bool> {
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

/// Whether the tokens invoke `name!` with a delimited body, at any depth.
#[must_use]
pub fn invokes(tokens: TokenStream, name: &str) -> bool {
    let tokens: Vec<_> = tokens.into_iter().collect();
    tokens.windows(3).any(|pair| {
        matches!(pair,
            [proc_macro2::TokenTree::Ident(ident), proc_macro2::TokenTree::Punct(mark), proc_macro2::TokenTree::Group(group)]
                if ident == name && mark.as_char() == '!'
                    && group.delimiter() != proc_macro2::Delimiter::None
        )
    }) || tokens.into_iter().any(|token| match token {
        proc_macro2::TokenTree::Group(group) => invokes(group.stream(), name),
        _ => false,
    })
}

/// A method's machine kind and the attribute that marks it.
pub fn kind(method: &syn::ImplItemFn) -> Result<Option<(Kind, &syn::Attribute)>> {
    let mut found = None;
    for attr in &method.attrs {
        if let Some(kind) = KINDS.iter().copied().find(|kind| kind.marks(attr)) {
            if found.is_some() {
                bail!(attr, "a method requires exactly one machine kind attribute");
            }
            if !matches!(kind, Kind::On | Kind::Postcondition) {
                attr.meta.require_path_only()?;
            }
            found = Some((kind, attr));
        }
    }
    Ok(found)
}

/// Whether a receiver borrows the state mutably.
#[must_use]
pub fn mutable_reference(receiver: &syn::Receiver) -> bool {
    match &receiver.kind {
        syn::ReceiverKind::Reference(_, _, mutability) => mutability.is_some(),
        syn::ReceiverKind::Typed(_, ty) => {
            matches!(ty.as_ref(), syn::Type::Reference(ty) if ty.mutability.is_some())
        }
        _ => false,
    }
}

/// The position of a signature's World parameter, which is declared once, last and as an
/// immutable `&World`.
pub fn world_parameter(signature: &syn::Signature) -> Result<Option<usize>> {
    let mut parameters = Vec::new();
    for (index, input) in signature.inputs.iter().enumerate() {
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
        bail!(parameters[1].1, "declare the World parameter once");
    }
    if let Some((index, input)) = parameters.first() {
        if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if reference.mutability.is_none())
        {
            bail!(&input.ty, "the World parameter must be an immutable &World");
        }
        if *index + 1 != signature.inputs.len() {
            bail!(input, "the World parameter must be last");
        }
    }
    Ok(parameters.first().map(|(index, _)| *index))
}

/// Read a `#[machine]` module and its attribute arguments.
pub fn machine(args: &TokenStream, input: &ItemMod) -> Result<Machine> {
    if !args.is_empty() {
        bail!(args, "#[machine] takes no arguments");
    }
    check_within(input)?;
    let Some((_, items)) = &input.content else {
        bail!(input, "#[machine] requires an inline module");
    };
    refuse_generics(items)?;
    let (state, visibility, actors, actions) = catalog(items, &input.ident)?;
    let world = declared_world(items, &input.ident)?;
    let idents: Vec<_> = actors.iter().map(|actor| actor.ident.clone()).collect();
    let methods = registrations(items, &state, &idents, world)?;
    if methods.iter().any(|method| method.kind == Kind::Starts) {
        computed_starts(items, &state)?;
    }
    for method in &methods {
        if let Some(path) = &method.after {
            if !actions.iter().any(|known| key(known) == key(path)) {
                bail!(path, "unknown after action {}", key(path));
            }
            if actors.iter().any(|actor| {
                actor.variants.iter().any(|(variant, inputs)| {
                    *inputs && key(path) == format!("{}::{variant}", actor.ident)
                })
            }) {
                bail!(path, "`{}` carries inputs; it is not one catalog action", key(path));
            }
        }
    }
    let enums = enums(items, &idents)?;
    let tokens = input.to_token_stream();
    Ok(Machine {
        state,
        visibility,
        actors,
        actions,
        world,
        rejecting: invokes(tokens.clone(), "reject"),
        staying: invokes(tokens, "stay"),
        methods,
        fact: None,
        enums,
    })
}

fn check_within(input: &ItemMod) -> Result<()> {
    #[derive(Default)]
    struct Within<'ast>(Option<&'ast syn::Attribute>);
    impl<'ast> syn::visit::Visit<'ast> for Within<'ast> {
        fn visit_attribute(&mut self, attr: &'ast syn::Attribute) {
            if helper(attr, "within") && self.0.is_none() {
                self.0 = Some(attr);
            }
        }
    }
    let mut within = Within::default();
    syn::visit::Visit::visit_item_mod(&mut within, input);
    match within.0 {
        Some(attr) => Err(syn::Error::new_spanned(
            attr,
            "a progress claim names its assumption with .assuming(..)",
        )),
        None => Ok(()),
    }
}

/// The lifted reading admits one concrete Model, so no declaration is generic.
fn refuse_generics(items: &[Item]) -> Result<()> {
    for item in items {
        let generics = match item {
            Item::Struct(item)
                if derives(&item.attrs, "State")? || derives(&item.attrs, "World")? =>
            {
                &item.generics
            }
            Item::Enum(item) if derives(&item.attrs, "Action")? => &item.generics,
            _ => continue,
        };
        if !generics.params.is_empty() {
            bail!(generics, "#[machine] does not admit generic Model, World or actor declarations");
        }
    }
    Ok(())
}

fn catalog(
    items: &[Item],
    module: &syn::Ident,
) -> Result<(syn::Ident, syn::Visibility, Vec<Actor>, Vec<Path>)> {
    let mut state = None;
    let mut actors = Vec::new();
    let mut actions = Vec::new();
    for item in items {
        match item {
            Item::Struct(item) if derives(&item.attrs, "State")? => {
                if state.is_some() {
                    bail!(item, "#[machine] requires exactly one #[derive(State)] struct");
                }
                state = Some((item.ident.clone(), item.vis.clone()));
            }
            Item::Enum(item) if derives(&item.attrs, "Action")? => {
                let actor = &item.ident;
                actors.push(Actor {
                    ident: actor.clone(),
                    variants: item
                        .variants
                        .iter()
                        .map(|variant| (variant.ident.clone(), !variant.fields.is_empty()))
                        .collect(),
                    world_dependent: item
                        .variants
                        .iter()
                        .flat_map(|variant| &variant.fields)
                        .any(|field| field.attrs.iter().any(|attr| attr.path().is_ident("domain"))),
                });
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

/// Whether the module declares its World; a declared World is the one `struct World`.
fn declared_world(items: &[Item], module: &syn::Ident) -> Result<bool> {
    let mut derived = false;
    for item in items {
        if let Item::Struct(item) = item {
            derived |= derives(&item.attrs, "World")?;
        }
    }
    if !derived {
        return Ok(false);
    }
    let worlds: Vec<_> = items
        .iter()
        .filter_map(|item| match item {
            Item::Struct(item) if item.ident == "World" => Some(item),
            _ => None,
        })
        .collect();
    let [world] = worlds.as_slice() else {
        bail!(module, "declare World once as #[derive(World)] struct World");
    };
    if !derives(&world.attrs, "World")? {
        bail!(world, "declared World requires #[derive(World)]");
    }
    for item in items {
        if let Item::Struct(item) = item
            && item.ident != "World"
            && derives(&item.attrs, "World")?
        {
            bail!(item, "declare World once");
        }
    }
    Ok(true)
}

/// Computed starts replace the State derive's field-order starts, so field `#[starts]` ranges
/// would be a second declaration.
fn computed_starts(items: &[Item], state: &syn::Ident) -> Result<()> {
    for item in items {
        if let Item::Struct(input) = item
            && input.ident == *state
            && let Some(attribute) = input
                .fields
                .iter()
                .flat_map(|field| &field.attrs)
                .find(|attribute| helper(attribute, "starts"))
        {
            bail!(attribute, "declare starts once");
        }
    }
    Ok(())
}

/// A local fieldless enum, outside the actors and `derive(Finite)`, as values export it.
fn enums(items: &[Item], actors: &[syn::Ident]) -> Result<Vec<(syn::Ident, Vec<syn::Ident>)>> {
    let mut enums = Vec::new();
    for item in items {
        let Item::Enum(item) = item else { continue };
        if actors.contains(&item.ident)
            || !item.generics.params.is_empty()
            || item.variants.iter().any(|variant| !variant.fields.is_empty())
            || derives(&item.attrs, "Finite")?
        {
            continue;
        }
        enums.push((
            item.ident.clone(),
            item.variants.iter().map(|variant| variant.ident.clone()).collect(),
        ));
    }
    Ok(enums)
}

fn check_impl(input: &ItemImpl, state: &syn::Ident) -> Result<()> {
    if input.trait_.is_some() || !input.generics.params.is_empty() {
        bail!(input, "machine methods require one nongeneric inherent impl {state}");
    }
    let syn::Type::Path(ty) = input.self_ty.as_ref() else {
        bail!(&input.self_ty, "machine methods require the module's state type");
    };
    if ty.qself.is_some() || !ty.path.is_ident(state) {
        bail!(&input.self_ty, "machine methods require the module's state type");
    }
    Ok(())
}

fn check_readonly(input: &ItemImpl, state: &syn::Ident) -> Result<()> {
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
        if matches!(kind(method)?, Some((Kind::On | Kind::Effect, _))) {
            continue;
        }
        if let Some(receiver) = method.sig.receiver()
            && (receiver.mutability.is_some() || mutable_reference(receiver))
        {
            bail!(
                &method.sig.ident,
                "{} takes &mut self; only #[on] and #[effect] methods change the state",
                method.sig.ident
            );
        }
    }
    Ok(())
}

fn transition_actor<'a>(
    method: &syn::ImplItemFn,
    actors: &'a [syn::Ident],
    arity: usize,
) -> Result<&'a syn::Ident> {
    if let Some(asyncness) = method.sig.asyncness {
        bail!(asyncness, "#[on] methods must be synchronous");
    }
    if !matches!(method.sig.output, syn::ReturnType::Default) {
        bail!(&method.sig.output, "#[on] methods return nothing; enabledness is a changed state");
    }
    let actor =
        (arity == 2).then(|| method.sig.inputs.iter().nth(1)).flatten().and_then(|argument| {
            match argument {
                syn::FnArg::Typed(argument) => match argument.ty.as_ref() {
                    syn::Type::Path(ty) if ty.qself.is_none() => {
                        actors.iter().find(|actor| ty.path.is_ident(*actor))
                    }
                    _ => None,
                },
                syn::FnArg::Receiver(_) => None,
            }
        });
    let mutable_receiver = method.sig.receiver().is_some_and(mutable_reference);
    match actor {
        Some(actor) if mutable_receiver => Ok(actor),
        _ => Err(syn::Error::new_spanned(
            &method.sig,
            "an #[on] method takes its actor enum: fn user(&mut self, action: User)",
        )),
    }
}

fn check_claim(method: &syn::ImplItemFn) -> Result<()> {
    if let Some(asyncness) = method.sig.asyncness {
        bail!(asyncness, "invariants and postconditions must be synchronous");
    }
    if !matches!(method.sig.output, syn::ReturnType::Default) {
        bail!(&method.sig.output, "invariants and postconditions return nothing; use check!");
    }
    Ok(())
}

fn check_progress(method: &syn::ImplItemFn, actors: &[syn::Ident]) -> Result<()> {
    struct ClaimPaths<'a> {
        actors: &'a [syn::Ident],
        error: Option<syn::Error>,
    }
    impl ClaimPaths<'_> {
        fn refuse(&mut self, error: syn::Error) {
            if let Some(previous) = &mut self.error {
                previous.combine(error);
            } else {
                self.error = Some(error);
            }
        }
    }
    impl syn::visit::Visit<'_> for ClaimPaths<'_> {
        fn visit_expr_method_call(&mut self, call: &syn::ExprMethodCall) {
            let mut receiver = call.receiver.as_ref();
            while let syn::Expr::MethodCall(inner) = receiver {
                receiver = inner.receiver.as_ref();
            }
            if call.method != "assuming"
                && let syn::Expr::Macro(path) = receiver
                && path.mac.path.segments.last().is_some_and(|segment| segment.ident == "path")
            {
                // The outermost builder refuses once for the whole chain.
                return self.refuse(syn::Error::new_spanned(
                    path,
                    "a path! claim takes only .assuming(..)",
                ));
            }
            syn::visit::visit_expr_method_call(self, call);
        }
        fn visit_macro(&mut self, call: &syn::Macro) {
            if call.path.segments.last().is_some_and(|segment| segment.ident == "path")
                && let Err(error) = crate::path::validate_claim(call, self.actors)
            {
                self.refuse(error);
            }
            syn::visit::visit_macro(self, call);
        }
    }
    let mut paths = ClaimPaths { actors, error: None };
    syn::visit::Visit::visit_impl_item_fn(&mut paths, method);
    paths.error.map_or(Ok(()), Err)
}

fn check_legacy(input: &ItemImpl, state: &syn::Ident) -> Result<()> {
    for attr in &input.attrs {
        if helper(attr, "starts") {
            bail!(
                attr,
                "#[starts] marks a method in impl {state}: #[starts] fn starts(world: &World) -> impl IntoIterator<Item = Self>"
            );
        }
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
            bail!(
                attr,
                "use per-item #[on(Actor)], #[effect], #[invariant], #[postcondition], #[assumption], and #[progress] instead of #[{name}]"
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
        bail!(
            path,
            "use per-item machine attributes on an inherent impl {state} instead of marker traits"
        );
    }
    Ok(())
}

fn after_action(attr: &syn::Attribute) -> Result<Option<Path>> {
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

fn check_order(attr: &syn::Attribute, kind: Kind, previous: Option<Kind>) -> Result<()> {
    if previous.is_some_and(|previous| kind < previous) {
        let rank = kind as usize;
        let before = rank.checked_sub(1).map_or_else(
            || "the start of the impl".to_owned(),
            |rank| format!("#[{}]", KINDS[rank].name()),
        );
        let after = KINDS
            .get(rank + 1)
            .map_or_else(|| "the end of the impl".to_owned(), |kind| format!("#[{}]", kind.name()));
        bail!(attr, "#[{}] methods must follow {before} and precede {after}", kind.name());
    }
    Ok(())
}

fn register_handler<'a>(
    methods: &[Method],
    method: &syn::ImplItemFn,
    attr: &syn::Attribute,
    actors: &'a [syn::Ident],
    arity: usize,
) -> Result<&'a syn::Ident> {
    let named: Path = attr.parse_args()?;
    if !actors.iter().any(|actor| named.is_ident(actor)) {
        bail!(attr, "#[on] names an actor enum declared in this module");
    }
    let actor = transition_actor(method, actors, arity)?;
    if !named.is_ident(actor) {
        bail!(&method.sig, "#[on({})] requires a {} parameter", key(&named), key(&named));
    }
    if methods.iter().any(|known| known.actor.as_ref() == Some(actor)) {
        bail!(&method.sig.ident, "actor {actor} already has an #[on] method");
    }
    Ok(actor)
}

fn register_starts(
    methods: &[Method],
    method: &syn::ImplItemFn,
    attr: &syn::Attribute,
    declared: bool,
    has_world: bool,
) -> Result<()> {
    if !declared {
        bail!(attr, "computed starts require a declared World");
    }
    if methods.iter().any(|known| known.kind == Kind::Starts) {
        bail!(attr, "declare starts once");
    }
    if method.sig.receiver().is_some()
        || method.sig.inputs.len() != usize::from(has_world)
        || method.sig.asyncness.is_some()
        || !method.sig.generics.params.is_empty()
    {
        bail!(
            &method.sig,
            "#[starts] requires an associated fn taking only an optional trailing &World: fn starts(world: &World) -> impl IntoIterator<Item = Self>"
        );
    }
    Ok(())
}

fn registrations(
    items: &[Item],
    state: &syn::Ident,
    actors: &[syn::Ident],
    declared: bool,
) -> Result<Vec<Method>> {
    let mut methods: Vec<Method> = Vec::new();
    let mut registered_impl = false;
    let mut previous = None;
    for (item, input) in items.iter().enumerate() {
        let Item::Impl(input) = input else { continue };
        check_legacy(input, state)?;
        check_readonly(input, state)?;
        let annotated = input.items.iter().try_fold(false, |found, item| -> Result<bool> {
            Ok(found || matches!(item, ImplItem::Fn(method) if kind(method)?.is_some()))
        })?;
        if !annotated {
            continue;
        }
        if registered_impl {
            bail!(input, "machine methods belong in one impl {state}");
        }
        registered_impl = true;
        check_impl(input, state)?;
        for (index, entry) in input.items.iter().enumerate() {
            let ImplItem::Fn(method) = entry else { continue };
            let Some((kind, attr)) = kind(method)? else { continue };
            check_order(attr, kind, previous)?;
            previous = Some(kind);
            if matches!(kind, Kind::Invariant | Kind::Postcondition) {
                check_claim(method)?;
            }
            let world = declared && world_parameter(&method.sig)?.is_some();
            let arity = method.sig.inputs.len() - usize::from(world);
            let mut actor = None;
            let mut after = None;
            let positional = method
                .sig
                .inputs
                .iter()
                .filter(|input| matches!(input, syn::FnArg::Typed(_)))
                .count()
                - usize::from(world);
            match kind {
                Kind::Invariant if positional > 0 => {
                    bail!(method, "an invariant takes only &self and an optional trailing World");
                }
                Kind::Assumption if positional > 0 => {
                    bail!(method, "assumptions take only an optional trailing World");
                }
                Kind::Progress if positional > 0 => {
                    bail!(
                        method,
                        "progress methods take only &self and an optional trailing World"
                    );
                }
                Kind::Starts => register_starts(&methods, method, attr, declared, world)?,
                Kind::On => {
                    actor = Some(register_handler(&methods, method, attr, actors, arity)?.clone());
                }
                Kind::Postcondition => {
                    after = after_action(attr)?;
                    if !(1..=3).contains(&arity) || method.sig.receiver().is_none() {
                        bail!(
                            &method.sig,
                            "postconditions require &self, optional before: &Self, then optional outcome: &Outcome"
                        );
                    }
                }
                Kind::Progress => check_progress(method, actors)?,
                Kind::Effect | Kind::Invariant | Kind::Assumption => {}
            }
            methods.push(Method {
                ident: method.sig.ident.clone(),
                kind,
                world,
                arity,
                actor,
                fact: false,
                after,
                item,
                index,
            });
        }
    }
    for actor in actors {
        if !methods.iter().any(|method| method.actor.as_ref() == Some(actor)) {
            bail!(actor, "actor {actor} has no #[on] method");
        }
    }
    Ok(methods)
}
