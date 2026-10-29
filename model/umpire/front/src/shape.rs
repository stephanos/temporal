use crate::{
    Diagnostic, Locate, Type,
    body::{Env, Environment},
    declare::{self, Kind},
    syntax::{has, path_name},
    tree::{ActionDef, ClaimDef, FieldDef, Lifted, PostDef, Shape, Start, WorldField},
};
use heck::ToUpperCamelCase;
use quote::ToTokens;
use std::collections::{BTreeMap, BTreeSet};
use syn::{ImplItem, Item};

type Result<T> = std::result::Result<T, Diagnostic>;

/// Admit a `#[machine]` module, as the macro receives it.
#[expect(clippy::too_many_lines, reason = "The admission pass preserves source declaration order")]
pub(crate) fn parse(
    declared: &declare::Machine,
    module: &syn::ItemMod,
) -> Result<(Lifted, Vec<proc_macro2::TokenStream>)> {
    let mut conditional = Conditional(None);
    syn::visit::Visit::visit_item_mod(&mut conditional, module);
    if let Some(attribute) = conditional.0 {
        return Err(Diagnostic::at(
            attribute,
            "unsupported conditional compilation inside machine source",
        ));
    }
    let (_, items) = module
        .content
        .as_ref()
        .ok_or_else(|| Diagnostic::at(module, "external machine modules are unsupported"))?;
    let declarations = Declarations::read(items, declared)?;
    let domain_functions: BTreeSet<String> = declarations
        .actors
        .iter()
        .flat_map(|(_, inputs)| inputs.variants.iter().flatten())
        .filter_map(|domain| match domain {
            declare::FieldDomain::World(path) => path.get_ident().map(ToString::to_string),
            declare::FieldDomain::Explicit(_) | declare::FieldDomain::Finite => None,
        })
        .collect();
    let mut parser = Environment {
        state: declared.state.to_string(),
        fields: Vec::new(),
        declared_world: declared.world,
        world_fields: Vec::new(),
        constants: Env::new(),
        const_parameters: BTreeMap::new(),
        enums: BTreeMap::new(),
        defaults: BTreeMap::new(),
        bounded: BTreeMap::new(),
        bounded_defaults: Vec::new(),
        helpers: BTreeMap::new(),
        methods: BTreeMap::new(),
        vector_imported: admit_imports(items)?,
        outcome_type: if declared.rejecting { Type::outcome() } else { Type::Unit },
        fact_type: Type::Unit,
        fact_shadows: ["Option", "Some", "None", "core", "std"]
            .into_iter()
            .filter(|name| declare::shadows(items, name))
            .map(str::to_owned)
            .collect(),
        phased: false,
        checks: std::cell::RefCell::default(),
        dsl: imports(items)
            .into_iter()
            .filter_map(|(_, name, path)| Some((name, crate::syntax::dsl_path(&path)?.to_owned())))
            .collect(),
    };
    for item in items {
        let attributes = match item {
            Item::Struct(v) => &v.attrs,
            Item::Enum(v) => &v.attrs,
            Item::Impl(v) => &v.attrs,
            Item::Const(v) => &v.attrs,
            Item::Use(v) => &v.attrs,
            Item::Fn(v) if domain_functions.contains(&v.sig.ident.to_string()) => &v.attrs,
            _ => {
                return Err(Diagnostic::at(item, "unsupported declaration in machine module"));
            }
        };
        attrs(attributes, &["doc", "derive", "internal"])?;
        if let Item::Struct(value) = item
            && derives(&value.attrs, "Finite")
        {
            parser.admit_bounded(value)?;
        }
        if let Item::Enum(value) = item {
            let payload = derives(&value.attrs, "Action")
                && value.variants.iter().any(|v| !matches!(v.fields, syn::Fields::Unit));
            if !value.generics.params.is_empty()
                || value.variants.iter().any(|v| {
                    (!payload && !matches!(v.fields, syn::Fields::Unit)) || v.discriminant.is_some()
                })
            {
                return Err(Diagnostic::at(
                    value,
                    "only fieldless enums without explicit discriminants are admitted",
                ));
            }
            if payload {
                continue;
            }
            let name = value.ident.to_string();
            let variants = value.variants.iter().map(|v| format!("{name}::{}", v.ident)).collect();
            if let Some(variant) = value.variants.iter().find(|v| has(&v.attrs, "default")) {
                parser
                    .defaults
                    .entry(name.clone())
                    .or_insert_with(|| format!("{name}::{}", variant.ident));
            }
            parser.enums.insert(name.clone(), Type::Enum { name, variants });
        }
    }
    if let Some(fact) = &declared.fact {
        parser.fact_type = parser.enums[&fact.to_string()].clone();
    }
    for stock in [Type::outcome(), Type::rejection()] {
        if let Type::Enum { name, .. } = &stock {
            parser.enums.entry(name.clone()).or_insert(stock);
        }
    }
    if let Some((_, fields)) = &declarations.world {
        for (name, field) in fields {
            attrs(&field.attrs, &["doc"])?;
            let ty = parser.ty(&field.ty)?;
            if !matches!(
                ty,
                Type::Bool
                    | Type::Int { .. }
                    | Type::Bounded(_)
                    | Type::Optional(_)
                    | Type::Enum { .. }
            ) {
                return Err(Diagnostic::at(*field, "World fields require admitted scalar types"));
            }
            parser.world_fields.push(WorldField {
                name: name.clone(),
                ty,
                location: Diagnostic::at(*field, ""),
            });
        }
    }
    let state_name = declared.state.to_string();
    let local_enums: BTreeSet<String> = items
        .iter()
        .filter_map(|item| match item {
            Item::Enum(value) => Some(value.ident.to_string()),
            _ => None,
        })
        .collect();
    // Inherent methods, and the Umpire traits implemented, per owner.
    let mut inherent = Vec::new();
    let mut implemented = Vec::new();
    for item in items {
        if let Item::Const(value) = item {
            let ty = parser.ty(&value.ty)?;
            let expr = parser.expr(&value.expr, &Env::new(), &Env::new(), Some(&ty), 0)?;
            parser.constants.insert(value.ident.to_string(), expr);
        }
        if let Item::Impl(value) = item {
            let owner = path_name(&value.self_ty.to_token_stream().to_string());
            let stock = match &value.trait_ {
                _ if !value.generics.params.is_empty() => None,
                None => Some(None),
                _ if value.unsafety.is_some() || value.modifiers.polarity.is_some() => None,
                Some((path, _)) => match parser.dsl_path(path).as_deref() {
                    Some("Phase") if local_enums.contains(&owner) => Some(Some("Phase")),
                    Some("Phased") if owner == state_name => Some(Some("Phased")),
                    _ => None,
                },
            };
            let Some(stock) = stock else {
                return Err(Diagnostic::at(
                    value,
                    "trait or generic impl in machine module is unsupported",
                ));
            };
            parser.phased |= stock == Some("Phased");
            if let Some(stock) = stock {
                implemented.push((owner.clone(), stock));
            }
            for entry in &value.items {
                if let ImplItem::Type(phase) = entry
                    && stock == Some("Phased")
                    && phase.ident == "Phase"
                {
                    if !matches!(parser.ty(&phase.ty), Ok(Type::Enum { name, .. }) if local_enums.contains(&name))
                    {
                        return Err(Diagnostic::at(
                            &phase.ty,
                            "the Phased phase must be a local fieldless enum",
                        ));
                    }
                    continue;
                }
                let ImplItem::Fn(method) = entry else {
                    return Err(Diagnostic::at(entry, "unsupported impl item"));
                };
                attrs(
                    &method.attrs,
                    &[
                        "doc",
                        "starts",
                        "on",
                        "effect",
                        "invariant",
                        "postcondition",
                        "assumption",
                        "scope",
                        "progress",
                    ],
                )?;
                if method.sig.asyncness.is_some()
                    || matches!(method.sig.safety, syn::Safety::Unsafe(_))
                    || !method.sig.generics.params.is_empty()
                {
                    return Err(Diagnostic::at(
                        method,
                        "async, unsafe or generic helper is unsupported",
                    ));
                }
                if declared.starts().is_some_and(|starts| starts.ident == method.sig.ident) {
                    // Computed starts are native only; Init comes from the bound complete starts.
                } else {
                    let (table, name) = if owner == state_name {
                        (&mut parser.methods, method.sig.ident.to_string())
                    } else {
                        (&mut parser.helpers, format!("{owner}::{}", method.sig.ident))
                    };
                    // Rust resolves an inherent method before a trait method of the same name.
                    if stock.is_none() {
                        inherent.push((owner.clone(), method));
                        table.insert(name, method.clone());
                    } else {
                        table.entry(name).or_insert_with(|| method.clone());
                    }
                }
            }
        }
    }
    // Library bodies call the trait method, which the front end would read as the inherent one.
    for (owner, method) in &inherent {
        let name = method.sig.ident.to_string();
        for (_, stock) in implemented.iter().filter(|(implementor, _)| implementor == owner) {
            if crate::library::trait_methods(stock).contains(&name) {
                return Err(Diagnostic::at(
                    &method.sig.ident,
                    format!(
                        "inherent `{name}` shadows `umpire_dsl::{stock}::{name}`, which Umpire's library calls"
                    ),
                ));
            }
        }
    }
    for (name, field, range) in &declarations.state {
        attrs(&field.attrs, &["doc", "starts"])?;
        let ty = parser.ty(&field.ty)?;
        let start = if declared.starts().is_some() {
            Start::Computed
        } else if let Some(range) = range {
            let (Some(lo), Some(hi)) = (&range.start, &range.end) else {
                unreachable!("the State declaration admits only bounded start ranges")
            };
            let lo = parser.expr(lo, &Env::new(), &Env::new(), Some(&ty), 0)?;
            let hi = parser.expr(hi, &Env::new(), &Env::new(), Some(&ty), 0)?;
            if matches!(range.limits, syn::RangeLimits::HalfOpen(_)) {
                Start::ExclusiveRange(lo, hi)
            } else {
                Start::Range(lo, hi)
            }
        } else {
            Start::Value(parser.default_value(&ty)?)
        };
        parser.fields.push(FieldDef {
            name: name.clone(),
            ty,
            start,
            location: Diagnostic::at(*field, ""),
        });
    }
    let initial = parser.field_env(false);
    let mut actions = Vec::new();
    for (actor, inputs) in &declarations.actors {
        let actor_name = actor.ident.to_string();
        for variant in &actor.variants {
            let name = format!("{actor_name}::{}", variant.ident);
            let (shape, fields) = match &variant.fields {
                syn::Fields::Unit => (Shape::Unit, Vec::new()),
                syn::Fields::Unnamed(fields) => {
                    (Shape::Tuple, fields.unnamed.iter().collect::<Vec<_>>())
                }
                syn::Fields::Named(fields) => {
                    (Shape::Named, fields.named.iter().collect::<Vec<_>>())
                }
            };
            if shape != Shape::Unit {
                attrs(&variant.attrs, &["doc", "metadata"])?;
                if !parser.declared_world {
                    return Err(Diagnostic::at(
                        variant,
                        "action inputs require a declared World, whose binding supplies their typed catalog identity",
                    ));
                }
            }
            let mut fields_in = Vec::new();
            for (index, field) in fields.into_iter().enumerate() {
                fields_in.push(parser.input_field(field, index, &name)?);
            }
            actions.push(ActionDef {
                name,
                internal: inputs.internal,
                branches: Vec::new(),
                shape,
                fields: fields_in,
                location: Diagnostic::at(variant, ""),
            });
        }
    }
    let mut invariants = Vec::new();
    let mut postconditions = Vec::new();
    let mut progress = Vec::new();
    let mut assumptions = Vec::new();
    for registered in &declared.methods {
        let Item::Impl(implementation) = &items[registered.item] else {
            unreachable!("methods are registered in an impl")
        };
        let ImplItem::Fn(method) = &implementation.items[registered.index] else {
            unreachable!("registered methods are functions")
        };
        match registered.kind {
            Kind::On => {
                let actor = registered.actor.as_ref().map(ToString::to_string).unwrap_or_default();
                let matched = parser.handler(method)?;
                for action in
                    actions.iter_mut().filter(|a| a.name.starts_with(&format!("{actor}::")))
                {
                    let branches = parser.branches(
                        matched,
                        action,
                        &parser.method_env(method)?,
                        registered.fact,
                    )?;
                    action.branches.extend(branches);
                }
            }
            Kind::Invariant => invariants.push(ClaimDef {
                name: method.sig.ident.to_string(),
                expr: parser.invariant(method, &initial)?,
            }),
            Kind::Postcondition => postconditions.push(PostDef {
                name: method.sig.ident.to_string(),
                after: registered
                    .after
                    .as_ref()
                    .map(|after| path_name(&after.to_token_stream().to_string())),
                expr: parser.postcondition(method)?,
            }),
            Kind::Assumption => {
                let mut assumption = parser.assumption_body(method, &actions)?;
                assumption.name = method.sig.ident.to_string();
                assumptions.push(assumption);
            }
            Kind::Progress => progress.push(parser.progress(method, &actions)?),
            Kind::Starts | Kind::Effect => {}
        }
    }
    let name = module.ident.to_string().to_upper_camel_case();
    let lifted = Lifted {
        name,
        state_name,
        outcome_type: parser.outcome_type,
        fact_type: parser.fact_type,
        world_path: if parser.declared_world {
            format!("{}::World", module.ident)
        } else {
            format!("{}::world()", module.ident)
        },
        declared_world: parser.declared_world,
        world_fields: parser.world_fields,
        location: Diagnostic::at(module, ""),
        fields: parser.fields,
        constants: parser.constants.values,
        actions,
        invariants,
        postconditions,
        progress,
        assumptions,
        negative: Vec::new(),
    };
    Ok((lifted, parser.checks.into_inner()))
}

/// The State, World and actor declarations, read as their derives read them.
struct Declarations<'a> {
    /// State fields with their `#[starts]` ranges.
    state: Vec<(String, &'a syn::Field, Option<syn::ExprRange>)>,
    /// The declared World and its fields.
    world: Option<(&'a syn::ItemStruct, Vec<(String, &'a syn::Field)>)>,
    /// Actor enums with their markers and input domains.
    actors: Vec<(&'a syn::ItemEnum, declare::ActionEnum)>,
}
impl<'a> Declarations<'a> {
    fn read(items: &'a [Item], machine: &declare::Machine) -> Result<Self> {
        let named = |fields: Vec<&'a syn::Field>| {
            fields
                .into_iter()
                .filter_map(|field| field.ident.as_ref().map(|name| (name.to_string(), field)))
                .collect::<Vec<_>>()
        };
        let mut declarations = Self { state: Vec::new(), world: None, actors: Vec::new() };
        for item in items {
            match item {
                Item::Struct(state) if state.ident == machine.state => {
                    let fields = declare::struct_fields(state, Some(&state.fields), false)
                        .map_err(Diagnostic::syn)?;
                    let ranges = declare::start_ranges(&fields).map_err(Diagnostic::syn)?;
                    declarations.state = named(fields)
                        .into_iter()
                        .zip(ranges)
                        .map(|((name, field), range)| (name, field, range))
                        .collect();
                }
                Item::Struct(world) if machine.world && world.ident == "World" => {
                    let fields = declare::struct_fields(world, Some(&world.fields), true)
                        .map_err(Diagnostic::syn)?;
                    declarations.world = Some((world, named(fields)));
                }
                Item::Enum(actor)
                    if machine.actors.iter().any(|known| known.ident == actor.ident) =>
                {
                    let input = syn::DeriveInput::from(actor.clone());
                    let inputs =
                        declare::action_enum(&input, machine.world).map_err(Diagnostic::syn)?;
                    declarations.actors.push((actor, inputs));
                }
                _ => {}
            }
        }
        Ok(declarations)
    }
}

struct Conditional<'a>(Option<&'a syn::Attribute>);
impl<'ast> syn::visit::Visit<'ast> for Conditional<'ast> {
    fn visit_attribute(&mut self, attribute: &'ast syn::Attribute) {
        if attribute.path().is_ident("cfg") || attribute.path().is_ident("cfg_attr") {
            self.0 = Some(attribute);
        }
    }
}

/// The module's declarations already read, so a malformed derive refuses there first.
fn derives(attrs: &[syn::Attribute], name: &str) -> bool {
    declare::derives(attrs, name).unwrap_or(false)
}
fn attrs(attrs: &[syn::Attribute], allowed: &[&str]) -> Result<()> {
    for attr in attrs {
        let name = attr.path().to_token_stream().to_string();
        if !allowed.contains(&name.as_str()) {
            return Err(Diagnostic::at(
                attr,
                format!(
                    "unsupported attribute `{name}` (cfg and external resolution are not inferred)"
                ),
            ));
        }
    }
    Ok(())
}
/// Each `use` binding of the module: its bound name and the path it imports.
fn imports(items: &[Item]) -> Vec<(&Item, String, String)> {
    fn walk(tree: &syn::UseTree, prefix: &str, out: &mut Vec<(String, String)>) {
        match tree {
            syn::UseTree::Path(path) => walk(&path.tree, &format!("{prefix}{}::", path.ident), out),
            syn::UseTree::Name(name) => {
                out.push((name.ident.to_string(), format!("{prefix}{}", name.ident)));
            }
            syn::UseTree::Rename(rename) => {
                out.push((rename.rename.to_string(), format!("{prefix}{}", rename.ident)));
            }
            syn::UseTree::Group(group) => {
                group.items.iter().for_each(|item| walk(item, prefix, out));
            }
            syn::UseTree::Glob(_) => out.push(("*".into(), format!("{prefix}*"))),
        }
    }
    let mut out = Vec::new();
    for item in items {
        if let Item::Use(import) = item {
            let mut bindings = Vec::new();
            walk(&import.tree, "", &mut bindings);
            out.extend(bindings.into_iter().map(|(name, path)| (item, name, path)));
        }
    }
    out
}

/// The subset reads these macros by name, as Umpire's and the prelude's.
const READ_BY_NAME: [&str; 6] = ["check", "stay", "reject", "unknown", "path", "matches"];

/// Refuse an import that could rebind a macro the subset reads by name; whether
/// `imbl::Vector` is imported under its own name.
fn admit_imports(items: &[Item]) -> Result<bool> {
    let mut vector = false;
    for (item, name, path) in imports(items) {
        let dsl = crate::syntax::dsl_path(&path);
        let umpire = dsl == Some(name.as_str()) && name != "matches";
        if let Some(path) = dsl
            && path
                .split("::")
                .next()
                .is_some_and(|module| crate::library::source(module).is_some())
            && (name == "*" || path.rsplit("::").next() != Some(name.as_str()))
        {
            return Err(Diagnostic::at(
                item,
                "library imports require explicit unaliased bindings",
            ));
        }
        if (name == "*" && dsl != Some("*")) || (READ_BY_NAME.contains(&name.as_str()) && !umpire) {
            return Err(Diagnostic::at(
                item,
                format!("`use {path}` can rebind a macro the subset reads as Umpire's"),
            ));
        }
        vector |= name == "Vector" && path == "imbl::Vector";
    }
    Ok(vector)
}

#[cfg(test)]
mod tests {
    #[test]
    fn facade_roots_preserve_admission_and_library_paths() -> Result<(), Box<dyn std::error::Error>>
    {
        let source = r"
#[machine]
mod counter {
    use ROOT::{Action, State, check};
    #[derive(State)]
    struct Model { value: u8 }
    #[derive(Action, Copy)]
    enum User { Step }
    impl Model {
        #[on(User)]
        fn user(&mut self, action: User) {
            match action {
                User::Step if self.value < 2 => self.value += 1,
                _ => {}
            }
        }
        #[invariant]
        fn bounded(&self) { check!(self.value <= 2); }
    }
}
";
        let mut canonical = None;
        for root in ["umpire_dsl", "umpire::dsl", "umpire"] {
            let ast = syn::parse_file(&source.replace("ROOT", root))?;
            let (model, _, _) = crate::read(&ast, &[]).map_err(|error| format!("{error:?}"))?;
            let printed = format!("{model:?}");
            if let Some(expected) = &canonical {
                assert_eq!(&printed, expected);
            } else {
                canonical = Some(printed);
            }
            let environment = crate::body::Environment::of(&model);
            let path = syn::parse_str(&format!("{root}::closable::terminal_remains"))?;
            assert_eq!(environment.dsl_path(&path).as_deref(), Some("closable::terminal_remains"));
        }
        Ok(())
    }
}
