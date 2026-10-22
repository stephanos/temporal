use crate::{
    Diagnostic, Type,
    body::{Env, Environment},
    declare::{self, Kind},
    syntax::{attr, has, path_name},
    tree::{ActionDef, ClaimDef, FieldDef, Lifted, PostDef, Shape, Start, WorldField},
};
use quote::ToTokens;
use std::collections::{BTreeMap, BTreeSet};
use syn::{ImplItem, Item};

type Result<T> = std::result::Result<T, Diagnostic>;

/// Admit the one `#[machine]` module; a refusal inside a hook names it in `failed_hook`.
#[expect(clippy::too_many_lines, reason = "The admission pass preserves source declaration order")]
pub(crate) fn parse(
    ast: &syn::File,
    hooks: &[&syn::ItemMacro],
    failed_hook: &mut Option<usize>,
) -> Result<Lifted> {
    let modules: Vec<_> = ast
        .items
        .iter()
        .filter_map(|item| match item {
            Item::Mod(module) if has(&module.attrs, "machine") => Some(module),
            _ => None,
        })
        .collect();
    if modules.len() != 1 {
        return Err(Diagnostic::at(ast, "expected exactly one inline #[machine] module"));
    }
    let module = modules[0];
    let marker = canonical_machine(ast, module)?;
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
    let mut admitted_imports = Vec::new();
    for item in items {
        if let Item::Use(import) = item {
            let mut bindings = Vec::new();
            imports(&import.tree, "", &mut bindings);
            for (binding, path) in bindings {
                if binding == "*"
                    || (path != format!("umpire_dsl::{binding}")
                        && !(binding == "Vector" && path == "imbl::Vector"))
                {
                    return Err(Diagnostic::at(
                        import,
                        format!("unsupported or aliased source dependency `{path}` as `{binding}`"),
                    ));
                }
                if admitted_imports.contains(&binding) {
                    return Err(Diagnostic::at(
                        import,
                        format!("ambiguous source import `{binding}`"),
                    ));
                }
                admitted_imports.push(binding);
            }
        }
    }
    for required in ["State", "Action"] {
        if !admitted_imports.iter().any(|name| name == required) {
            return Err(Diagnostic::at(
                module,
                format!("derive requires explicit canonical umpire_dsl::{required} import"),
            ));
        }
    }
    let arguments = match &marker.meta {
        syn::Meta::List(list) => list.tokens.clone(),
        _ => proc_macro2::TokenStream::new(),
    };
    let declared = declare::machine(&arguments, module).map_err(Diagnostic::from)?;
    let declarations = Declarations::read(items, &declared)?;
    let domain_functions = domain_functions(items);
    let mut parser = Environment {
        state: declared.state.to_string(),
        fields: Vec::new(),
        declared_world: declared.world,
        world_fields: Vec::new(),
        constants: Env::new(),
        enums: BTreeMap::new(),
        defaults: BTreeMap::new(),
        helpers: BTreeMap::new(),
        methods: BTreeMap::new(),
        vector_imported: admitted_imports.iter().any(|name| name == "Vector"),
        finite: BTreeSet::new(),
        domain_functions: BTreeMap::new(),
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
        if let Item::Fn(function) = item {
            parser.domain_functions.insert(function.sig.ident.to_string(), function.clone());
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
            if derives(&value.attrs, "Finite") {
                parser.finite.insert(name.clone());
            }
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
    if let Some((world, fields)) = &declarations.world {
        if !admitted_imports.iter().any(|name| name == "World") {
            return Err(Diagnostic::at(
                *world,
                "derive requires explicit canonical umpire_dsl::World import",
            ));
        }
        for (name, field) in fields {
            attrs(&field.attrs, &["doc"])?;
            let ty = parser.ty(&field.ty)?;
            if !matches!(ty, Type::Bool | Type::Int { .. } | Type::Enum { .. }) {
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
    for item in items {
        if let Item::Const(value) = item {
            let ty = parser.ty(&value.ty)?;
            let expr = parser.expr(&value.expr, &Env::new(), &Env::new(), Some(&ty), 0)?;
            parser.constants.insert(value.ident.to_string(), expr);
        }
        if let Item::Impl(value) = item {
            if value.trait_.is_some() || !value.generics.params.is_empty() {
                return Err(Diagnostic::at(
                    value,
                    "trait or generic impl in machine module is unsupported",
                ));
            }
            let owner = path_name(&value.self_ty.to_token_stream().to_string());
            for entry in &value.items {
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
                if has(&method.attrs, "starts") {
                    // Computed starts are native only; Init comes from the bound complete starts.
                } else if owner == state_name {
                    parser.methods.insert(method.sig.ident.to_string(), method.clone());
                } else {
                    parser.helpers.insert(format!("{owner}::{}", method.sig.ident), method.clone());
                }
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
        for (variant, domains) in actor.variants.iter().zip(&inputs.variants) {
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
                attrs(&variant.attrs, &["doc"])?;
                if !parser.declared_world {
                    return Err(Diagnostic::at(
                        variant,
                        "action inputs require a declared World, whose binding supplies their typed catalog identity",
                    ));
                }
            }
            let mut fields_in = Vec::new();
            for (index, (field, domain)) in fields.into_iter().zip(domains).enumerate() {
                fields_in.push(parser.input_field(field, index, &name, domain)?);
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
    parser.static_catalog(&actions, module)?;
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
                    let branches = parser.branches(matched, action, &parser.method_env(method)?)?;
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
                    .map(|after| parser.action_name(after, &actions))
                    .transpose()?,
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
    let mut negative = Vec::new();
    for (index, hook) in hooks.iter().enumerate() {
        negative.extend(
            parser.supplemental(&hook.mac, &initial).inspect_err(|_| *failed_hook = Some(index))?,
        );
    }
    let name = pascal(&module.ident.to_string());
    Ok(Lifted {
        name,
        state_name,
        outcome_type: Type::Unit,
        world_path: if parser.declared_world {
            format!("{}::World", module.ident)
        } else {
            format!("{}::world()", module.ident)
        },
        declared_world: parser.declared_world,
        world_fields: parser.world_fields,
        world_values: None,
        complete_starts: None,
        location: Diagnostic::at(module, ""),
        fields: parser.fields,
        constants: parser.constants.values,
        actions,
        invariants,
        postconditions,
        progress,
        assumptions,
        negative,
        catalog: None,
    })
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
                        .map_err(Diagnostic::from)?;
                    let ranges = declare::start_ranges(&fields).map_err(Diagnostic::from)?;
                    declarations.state = named(fields)
                        .into_iter()
                        .zip(ranges)
                        .map(|((name, field), range)| (name, field, range))
                        .collect();
                }
                Item::Struct(world) if machine.world && world.ident == "World" => {
                    let fields = declare::struct_fields(world, Some(&world.fields), true)
                        .map_err(Diagnostic::from)?;
                    declarations.world = Some((world, named(fields)));
                }
                Item::Enum(actor)
                    if machine.actors.iter().any(|known| known.ident == actor.ident) =>
                {
                    let input = syn::DeriveInput::from(actor.clone());
                    let inputs =
                        declare::action_enum(&input, machine.world).map_err(Diagnostic::from)?;
                    declarations.actors.push((actor, inputs));
                }
                _ => {}
            }
        }
        Ok(declarations)
    }
}

/// Free functions named by a variant field's `#[domain(..)]`, admitted only as domain sources.
fn domain_functions(items: &[Item]) -> BTreeSet<String> {
    items
        .iter()
        .filter_map(|item| match item {
            Item::Enum(actor) if derives(&actor.attrs, "Action") => Some(actor),
            _ => None,
        })
        .flat_map(|actor| actor.variants.iter().flat_map(|variant| &variant.fields))
        .filter_map(|field| attr(&field.attrs, "domain"))
        .filter_map(|domain| domain.parse_args::<syn::Ident>().ok())
        .map(|name| name.to_string())
        .collect()
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
fn pascal(name: &str) -> String {
    name.split('_')
        .filter(|s| !s.is_empty())
        .map(|part| {
            let mut chars = part.chars();
            chars
                .next()
                .map_or(String::new(), |c| c.to_uppercase().collect::<String>() + chars.as_str())
        })
        .collect()
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
fn canonical_machine<'a>(ast: &syn::File, module: &'a syn::ItemMod) -> Result<&'a syn::Attribute> {
    let marker = module
        .attrs
        .iter()
        .find(|attribute| attribute.path().is_ident("machine"))
        .ok_or_else(|| Diagnostic::at(module, "unsupported machine marker"))?;
    let mut bindings = Vec::new();
    for item in &ast.items {
        if let Item::Use(import) = item {
            imports(&import.tree, "", &mut bindings);
        }
    }
    let markers: Vec<_> = bindings.iter().filter(|(binding, _)| binding == "machine").collect();
    if markers.len() != 1
        || markers[0].1 != "umpire_dsl::machine"
        || bindings.iter().any(|(binding, _)| binding == "*")
    {
        return Err(Diagnostic::at(
            marker,
            "machine marker requires one explicit canonical umpire_dsl::machine import",
        ));
    }
    Ok(marker)
}
fn imports(tree: &syn::UseTree, prefix: &str, out: &mut Vec<(String, String)>) {
    match tree {
        syn::UseTree::Path(path) => imports(&path.tree, &format!("{prefix}{}::", path.ident), out),
        syn::UseTree::Name(name) => {
            out.push((name.ident.to_string(), format!("{prefix}{}", name.ident)));
        }
        syn::UseTree::Rename(rename) => {
            out.push((rename.rename.to_string(), format!("{prefix}{}", rename.ident)));
        }
        syn::UseTree::Group(group) => {
            for item in &group.items {
                imports(item, prefix, out);
            }
        }
        syn::UseTree::Glob(_) => out.push(("*".into(), format!("{prefix}*"))),
    }
}
