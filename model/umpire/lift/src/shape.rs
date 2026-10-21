use crate::{
    Diagnostic, Type,
    body::{Env, Environment},
    syntax::{attr, has, path_name},
    tree::{ActionDef, ClaimDef, FieldDef, Lifted, PostDef, Shape, Start, WorldField},
};
use quote::ToTokens;
use std::collections::{BTreeMap, BTreeSet};
use syn::{ImplItem, Item};

type Result<T> = std::result::Result<T, Diagnostic>;

#[expect(clippy::too_many_lines, reason = "The admission pass preserves source declaration order")]
pub(crate) fn parse(file: &str, ast: &syn::File, hooks: &[crate::Hook<'_>]) -> Result<Lifted> {
    let modules: Vec<_> = ast
        .items
        .iter()
        .filter_map(|item| match item {
            Item::Mod(module) if has(&module.attrs, "machine") => Some(module),
            _ => None,
        })
        .collect();
    if modules.len() != 1 {
        return Err(Diagnostic::at(file, ast, "expected exactly one inline #[machine] module"));
    }
    let module = modules[0];
    canonical_machine(file, ast, module)?;
    let mut conditional = Conditional(None);
    syn::visit::Visit::visit_item_mod(&mut conditional, module);
    if let Some(attribute) = conditional.0 {
        return Err(Diagnostic::at(
            file,
            attribute,
            "unsupported conditional compilation inside machine source",
        ));
    }
    let (_, items) = module
        .content
        .as_ref()
        .ok_or_else(|| Diagnostic::at(file, module, "external machine modules are unsupported"))?;
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
                        file,
                        import,
                        format!("unsupported or aliased source dependency `{path}` as `{binding}`"),
                    ));
                }
                if admitted_imports.contains(&binding) {
                    return Err(Diagnostic::at(
                        file,
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
                file,
                module,
                format!("derive requires explicit canonical umpire_dsl::{required} import"),
            ));
        }
    }
    let domain_functions = domain_functions(items);
    let mut parser = Environment {
        file,
        fields: Vec::new(),
        declared_world: false,
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
                return Err(Diagnostic::at(
                    file,
                    item,
                    "unsupported declaration in machine module",
                ));
            }
        };
        if let Item::Impl(value) = item
            && let Some(attribute) = attr(&value.attrs, "starts")
        {
            return Err(Diagnostic::at(
                file,
                attribute,
                "#[starts] marks a method in impl Model: #[starts] fn starts(world: &World)",
            ));
        }
        attrs(file, attributes, &["doc", "derive", "internal"])?;
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
                    file,
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
    let worlds: Vec<_> = items
        .iter()
        .filter_map(|item| match item {
            Item::Struct(value) if derives(&value.attrs, "World") => Some(value),
            _ => None,
        })
        .collect();
    if worlds.len() > 1 {
        return Err(Diagnostic::at(file, worlds[1], "declare one World"));
    }
    let mut generated = BTreeSet::new();
    if let Some(world) = worlds.first() {
        parser.declared_world = true;
        if world.ident != "World" || !world.generics.params.is_empty() {
            return Err(Diagnostic::at(
                file,
                world,
                "source World must be a nongeneric named World",
            ));
        }
        if !admitted_imports.iter().any(|name| name == "World") {
            return Err(Diagnostic::at(
                file,
                world,
                "derive requires explicit canonical umpire_dsl::World import",
            ));
        }
        let syn::Fields::Named(fields) = &world.fields else {
            return Err(Diagnostic::at(file, world, "World must have named fields"));
        };
        for field in &fields.named {
            attrs(file, &field.attrs, &["doc"])?;
            let name = field
                .ident
                .as_ref()
                .ok_or_else(|| Diagnostic::at(file, field, "unnamed World field"))?
                .to_string();
            let ty = parser.ty(&field.ty)?;
            if !matches!(ty, Type::Bool | Type::Int { .. } | Type::Enum { .. }) {
                return Err(Diagnostic::at(
                    file,
                    field,
                    "World fields require admitted scalar types",
                ));
            }
            let symbol =
                crate::names::allocate(ast, &format!("__umpire_world_{name}"), &mut generated);
            parser.world_fields.push(WorldField {
                name,
                ty,
                symbol,
                location: Diagnostic::at(file, field, ""),
            });
        }
    }
    let mut computed = None;
    for item in items {
        if let Item::Const(value) = item {
            let ty = parser.ty(&value.ty)?;
            let expr = parser.expr(&value.expr, &Env::new(), &Env::new(), Some(&ty), 0)?;
            parser.constants.insert(value.ident.to_string(), expr);
        }
        if let Item::Impl(value) = item {
            if value.trait_.is_some() || !value.generics.params.is_empty() {
                return Err(Diagnostic::at(
                    file,
                    value,
                    "trait or generic impl in machine module is unsupported",
                ));
            }
            let owner = path_name(&value.self_ty.to_token_stream().to_string());
            for entry in &value.items {
                let ImplItem::Fn(method) = entry else {
                    return Err(Diagnostic::at(file, entry, "unsupported impl item"));
                };
                attrs(
                    file,
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
                        file,
                        method,
                        "async, unsafe or generic helper is unsupported",
                    ));
                }
                if has(&method.attrs, "starts") {
                    computed_starts(&parser, method, owner == "Model", &mut computed)?;
                } else if owner == "Model" {
                    parser.methods.insert(method.sig.ident.to_string(), method.clone());
                } else {
                    parser.helpers.insert(format!("{owner}::{}", method.sig.ident), method.clone());
                }
            }
        }
    }
    let states: Vec<_> = items
        .iter()
        .filter_map(|item| match item {
            Item::Struct(value) if derives(&value.attrs, "State") => Some(value),
            _ => None,
        })
        .collect();
    if states.len() != 1 || states[0].ident != "Model" {
        return Err(Diagnostic::at(file, module, "machine requires one named Model state"));
    }
    let state = states[0];
    if !state.generics.params.is_empty() {
        return Err(Diagnostic::at(file, state, "generic state is unsupported"));
    }
    let syn::Fields::Named(fields) = &state.fields else {
        return Err(Diagnostic::at(file, state, "state must have named fields"));
    };
    for field in &fields.named {
        attrs(file, &field.attrs, &["doc", "starts"])?;
        let name = field
            .ident
            .as_ref()
            .ok_or_else(|| Diagnostic::at(file, field, "unnamed state field"))?
            .to_string();
        if reserved(&name) {
            return Err(Diagnostic::at(file, field, format!("reserved observer name `{name}`")));
        }
        let ty = parser.ty(&field.ty)?;
        let start = if computed.is_some() {
            if let Some(attribute) = attr(&field.attrs, "starts") {
                return Err(Diagnostic::at(
                    file,
                    attribute,
                    "computed starts cannot be combined with field starts",
                ));
            }
            Start::Computed
        } else if let Some(attr) = attr(&field.attrs, "starts") {
            let range: syn::ExprRange =
                attr.parse_args().map_err(|e| Diagnostic::span(file, e.span(), e.to_string()))?;
            let (Some(lo), Some(hi)) = (range.start, range.end) else {
                return Err(Diagnostic::at(file, attr, "start range must have two finite bounds"));
            };
            let lo = parser.expr(&lo, &Env::new(), &Env::new(), Some(&ty), 0)?;
            let hi = parser.expr(&hi, &Env::new(), &Env::new(), Some(&ty), 0)?;
            if matches!(range.limits, syn::RangeLimits::HalfOpen(_)) {
                Start::ExclusiveRange(lo, hi)
            } else {
                Start::Range(lo, hi)
            }
        } else {
            Start::Value(parser.default_value(&ty)?)
        };
        parser.fields.push(FieldDef { name, ty, start, location: Diagnostic::at(file, field, "") });
    }
    let initial = parser.field_env(false);
    let mut actions = Vec::new();
    for item in items {
        if let Item::Enum(actor) = item {
            if !derives(&actor.attrs, "Action") {
                continue;
            }
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
                    attrs(file, &variant.attrs, &["doc"])?;
                    if !parser.declared_world {
                        return Err(Diagnostic::at(
                            file,
                            variant,
                            "action inputs require a declared World, whose binding supplies their typed catalog identity",
                        ));
                    }
                }
                // Chosen values are LET-bound inside one variant's relation, so each variant
                // allocates its names apart from the others but never like a module name.
                let mut scope = generated.clone();
                let mut inputs = Vec::new();
                for (index, field) in fields.into_iter().enumerate() {
                    let base =
                        field.ident.as_ref().map_or_else(|| index.to_string(), ToString::to_string);
                    let symbol =
                        crate::names::allocate(ast, &format!("__umpire_input_{base}"), &mut scope);
                    inputs.push(parser.input_field(field, index, &name, symbol)?);
                }
                actions.push(ActionDef {
                    operator: operator(&name),
                    name,
                    internal: has(&actor.attrs, "internal"),
                    branches: Vec::new(),
                    shape,
                    fields: inputs,
                    location: Diagnostic::at(file, variant, ""),
                });
            }
        }
    }
    parser.static_catalog(&actions, module)?;
    let mut invariants = Vec::new();
    let mut postconditions = Vec::new();
    let mut progress = Vec::new();
    let mut assumptions = Vec::new();
    for item in items {
        let Item::Impl(implementation) = item else { continue };
        if path_name(&implementation.self_ty.to_token_stream().to_string()) != "Model" {
            continue;
        }
        for method in &implementation.items {
            let ImplItem::Fn(method) = method else { continue };
            if has(&method.attrs, "on") {
                let actor = attr(&method.attrs, "on")
                    .ok_or_else(|| Diagnostic::at(file, method, "missing actor"))?
                    .parse_args::<syn::Path>()
                    .map_err(|e| Diagnostic::span(file, e.span(), e.to_string()))?
                    .to_token_stream()
                    .to_string();
                let matched = parser.handler(method)?;
                for action in
                    actions.iter_mut().filter(|a| a.name.starts_with(&format!("{actor}::")))
                {
                    let branches = parser.branches(matched, action, &parser.method_env(method)?)?;
                    action.branches.extend(branches);
                }
            } else if has(&method.attrs, "invariant") {
                invariants.push(ClaimDef {
                    name: method.sig.ident.to_string(),
                    expr: parser.invariant(method, &initial)?,
                });
            } else if has(&method.attrs, "postcondition") {
                let after = if let Some(attr) = attr(&method.attrs, "postcondition") {
                    match &attr.meta {
                        syn::Meta::List(list) => {
                            let arg: syn::MetaNameValue = syn::parse2(list.tokens.clone())
                                .map_err(|e| Diagnostic::span(file, e.span(), e.to_string()))?;
                            if !arg.path.is_ident("after") {
                                return Err(Diagnostic::at(
                                    file,
                                    &arg,
                                    "unsupported postcondition filter",
                                ));
                            }
                            Some(parser.action_name(&arg.value, &actions)?)
                        }
                        _ => None,
                    }
                } else {
                    None
                };
                postconditions.push(PostDef {
                    name: method.sig.ident.to_string(),
                    after,
                    expr: parser.postcondition(method)?,
                });
            } else if has(&method.attrs, "assumption") || has(&method.attrs, "scope") {
                let mut assumption = parser.assumption_body(method, &actions)?;
                assumption.name = method.sig.ident.to_string();
                assumptions.push(assumption);
            } else if has(&method.attrs, "progress") {
                progress.push(parser.progress(method, &actions)?);
            } else if !has(&method.attrs, "effect") {
                // Unmarked helpers are admitted only when their bodies are consumed through expression lowering.
                if method.sig.receiver().is_some_and(|r| r.mutability.is_some()) {
                    return Err(Diagnostic::at(
                        file,
                        method,
                        "unmarked mutable helper is unsupported",
                    ));
                }
            }
        }
    }
    for action in &actions {
        if !parser.methods.values().any(|m| {
            attr(&m.attrs, "on").is_some_and(|a| {
                a.parse_args::<syn::Path>()
                    .is_ok_and(|p| action.name.starts_with(&format!("{}::", p.to_token_stream())))
            })
        }) {
            return Err(Diagnostic::at(
                file,
                module,
                format!("action `{}` has no admitted handler", action.name),
            ));
        }
    }
    let mut negative = Vec::new();
    for hook in hooks {
        parser.file = hook.file;
        negative.extend(parser.supplemental(&hook.mac.mac, &initial)?);
    }
    let name = pascal(&module.ident.to_string());
    let index_symbol = crate::names::allocate(ast, "__umpire_i", &mut generated);
    let constant_symbols = if parser.declared_world {
        parser
            .constants
            .keys()
            .map(|name| {
                (
                    name.clone(),
                    crate::names::allocate(ast, &format!("__umpire_source_{name}"), &mut generated),
                )
            })
            .collect()
    } else {
        BTreeMap::new()
    };
    Ok(Lifted {
        file: file.into(),
        name,
        state_name: state.ident.to_string(),
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
        location: Diagnostic::at(file, module, ""),
        fields: parser.fields,
        constants: parser.constants.values,
        constant_symbols,
        actions,
        invariants,
        postconditions,
        progress,
        assumptions,
        negative,
        catalog: None,
        index_symbol,
    })
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

/// Admits the one `#[starts]` method: an associated fn of `impl Model` taking only an optional
/// trailing `&World`. Its body is native only; Init comes from the bound complete starts.
fn computed_starts<'a>(
    parser: &Environment<'_>,
    method: &'a syn::ImplItemFn,
    model: bool,
    computed: &mut Option<&'a syn::ImplItemFn>,
) -> Result<()> {
    let file = parser.file;
    let attribute = attr(&method.attrs, "starts").unwrap_or_else(|| unreachable!("marked"));
    if !parser.declared_world {
        return Err(Diagnostic::at(file, attribute, "computed starts require a declared World"));
    }
    if computed.is_some() {
        return Err(Diagnostic::at(file, attribute, "declare starts once"));
    }
    let parameters = parser.parameters(method)?;
    if !model
        || !matches!(attribute.meta, syn::Meta::Path(_))
        || method.sig.receiver().is_some()
        || !parameters.positional.is_empty()
    {
        return Err(Diagnostic::at(
            file,
            method,
            "#[starts] requires an associated fn of impl Model taking only an optional trailing &World",
        ));
    }
    *computed = Some(method);
    Ok(())
}

struct Conditional<'a>(Option<&'a syn::Attribute>);
impl<'ast> syn::visit::Visit<'ast> for Conditional<'ast> {
    fn visit_attribute(&mut self, attribute: &'ast syn::Attribute) {
        if attribute.path().is_ident("cfg") || attribute.path().is_ident("cfg_attr") {
            self.0 = Some(attribute);
        }
    }
}

fn derives(attrs: &[syn::Attribute], name: &str) -> bool {
    attrs.iter().filter(|a| a.path().is_ident("derive")).any(|a| {
        a.parse_args_with(
            syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated,
        )
        .is_ok_and(|paths| paths.iter().any(|p| p.is_ident(name)))
    })
}
fn operator(name: &str) -> String {
    name.replace("::", "_")
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
fn reserved(name: &str) -> bool {
    [
        "__umpire_edge_tick",
        "__umpire_pending",
        "__umpire_steps",
        "__umpire_action",
        "__umpire_choice",
        "__umpire_outcome",
        "__umpire_facts",
        "__umpire_input",
    ]
    .contains(&name)
}
fn attrs(file: &str, attrs: &[syn::Attribute], allowed: &[&str]) -> Result<()> {
    for attr in attrs {
        let name = attr.path().to_token_stream().to_string();
        if !allowed.contains(&name.as_str()) {
            return Err(Diagnostic::at(
                file,
                attr,
                format!(
                    "unsupported attribute `{name}` (cfg and external resolution are not inferred)"
                ),
            ));
        }
    }
    Ok(())
}
fn canonical_machine(file: &str, ast: &syn::File, module: &syn::ItemMod) -> Result<()> {
    let marker = module
        .attrs
        .iter()
        .find(|attribute| attribute.path().is_ident("machine"))
        .ok_or_else(|| Diagnostic::at(file, module, "unsupported machine marker"))?;
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
            file,
            marker,
            "machine marker requires one explicit canonical umpire_dsl::machine import",
        ));
    }
    Ok(())
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
