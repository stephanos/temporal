mod expr;
mod progress;
mod steps;
use expr::{
    constant_bool, constant_int, integer, pattern_names, result_type, same_scalar_type, scalar_type,
};

use std::collections::BTreeMap;

use quote::ToTokens;
use syn::{Expr as RustExpr, ImplItem, Item, Stmt, spanned::Spanned};

use crate::tree::{ActionDef, ClaimDef, Expr, FieldDef, PostDef, Start};
use crate::{Diagnostic, Lifted, Type};

pub(crate) fn parse(file: &str, ast: &syn::File) -> Result<Lifted, Diagnostic> {
    parse_selected(file, ast, selected_mode(ast))
}

#[allow(clippy::too_many_lines)]
fn parse_selected(file: &str, ast: &syn::File, mode: Option<String>) -> Result<Lifted, Diagnostic> {
    for item in &ast.items {
        if let Item::Impl(implementation) = item {
            for member in &implementation.items {
                if let ImplItem::Fn(method) = member
                    && (implementation.trait_.is_none()
                        || method.sig.ident == "clone"
                        || method.sig.ident == "into")
                {
                    return Err(refusal(
                        file,
                        method,
                        "authored inherent methods and clone/into overrides require unsupported manual helper resolution",
                    ));
                }
            }
        }
        if let Item::Macro(item) = item
            && item.mac.path.to_token_stream().to_string().replace(' ', "")
                != "umpire_testing::fixture"
        {
            return Err(refusal(file, item, "unsupported source macro may alter manual behavior"));
        }
        if let Item::Impl(implementation) = item
            && let Some((path, _)) = &implementation.trait_
            && path.segments.last().is_some_and(|segment| segment.ident == "Machine")
            && !canonical_machine(ast, path)
        {
            return Err(refusal(
                file,
                path,
                "Machine trait must resolve to umpire_dsl::Machine through a supported source import",
            ));
        }
    }
    for item in &ast.items {
        match item {
            Item::Impl(item)
                if item.trait_.as_ref().is_some_and(|(path, _)| canonical_machine(ast, path)) =>
            {
                attributes(file, &item.attrs)?;
                for member in &item.items {
                    if let ImplItem::Fn(method) = member {
                        attributes(file, &method.attrs)?;
                        if method.sig.asyncness.is_some()
                            || matches!(method.sig.safety, syn::Safety::Unsafe(_))
                            || !method.sig.generics.params.is_empty()
                        {
                            return Err(refusal(
                                file,
                                method,
                                "unsupported manual method signature",
                            ));
                        }
                    }
                }
            }
            Item::Struct(item) => {
                attributes(file, &item.attrs)?;
                for field in &item.fields {
                    attributes(file, &field.attrs)?;
                }
            }
            Item::Enum(item) => {
                attributes(file, &item.attrs)?;
                for variant in &item.variants {
                    attributes(file, &variant.attrs)?;
                }
            }
            Item::Const(item) => attributes(file, &item.attrs)?,
            _ => {}
        }
    }
    let implementation = ast.items.iter().find_map(|item| match item {
        Item::Impl(item)
            if item.trait_.as_ref().is_some_and(|(path, _)| canonical_machine(ast, path)) =>
        {
            Some(item)
        }
        _ => None,
    });
    let Some(implementation) = implementation else {
        return dependency(file, ast, mode);
    };
    if ast.items.iter().filter(|item| matches!(item, Item::Impl(item) if item.trait_.as_ref().is_some_and(|(path, _)| canonical_machine(ast, path)))).count() != 1 { return Err(refusal(file, implementation, "multiple manual Machine implementations require explicit source selection")); }
    let world = implementation.self_ty.to_token_stream().to_string();
    for item in &ast.items {
        if let Item::Macro(item) = item
            && item.mac.path.segments.last().is_some_and(|segment| segment.ident == "fixture")
        {
            let arguments = item
                .mac
                .parse_body_with(
                    syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
                )
                .map_err(|error| refusal(file, item, &error.to_string()))?;
            if arguments.len() != 2 {
                return Err(refusal(
                    file,
                    item,
                    "supplemental manual fixture claims are unsupported",
                ));
            }
            let selected = arguments.first().and_then(|expression| match expression {
                RustExpr::Call(call) => path_name(&call.func),
                expression => path_name(expression),
            });
            if selected.as_deref() != Some(world.as_str()) {
                return Err(refusal(
                    file,
                    item,
                    "fixture constructor does not select this Machine implementation",
                ));
            }
        }
    }
    if !implementation.generics.params.is_empty() {
        return Err(refusal(file, implementation, "generic Machine adapters are unsupported"));
    }
    let variants = ast
        .items
        .iter()
        .filter_map(|item| {
            if let Item::Enum(item) = item {
                Some(
                    item.variants
                        .iter()
                        .map(|variant| format!("{}::{}", item.ident, variant.ident))
                        .collect::<Vec<_>>(),
                )
            } else {
                None
            }
        })
        .flatten()
        .collect();
    let mut parser = Manual {
        file,
        mode,
        variants,
        state_name: String::new(),
        constants: BTreeMap::new(),
        fields: Vec::new(),
        actions: Vec::new(),
    };
    for item in &ast.items {
        if let Item::Const(item) = item {
            let ty = scalar_type(&item.ty).ok_or_else(|| {
                refusal(file, item, "constant type is outside the admitted scalar TLC range")
            })?;
            let mut value = parser.expr(&item.expr, &BTreeMap::new())?;
            if !same_scalar_type(&ty, &value.ty) {
                return Err(refusal(file, item, "constant value disagrees with its declared type"));
            }
            value.ty = ty;
            parser.constants.insert(item.ident.to_string(), value);
        }
    }
    let state_name = implementation
        .items
        .iter()
        .find_map(|item| match item {
            ImplItem::Type(item) if item.ident == "State" => {
                Some(item.ty.to_token_stream().to_string())
            }
            _ => None,
        })
        .ok_or_else(|| refusal(file, implementation, "Machine::State is required"))?;
    parser.state_name.clone_from(&state_name);
    let state = ast
        .items
        .iter()
        .find_map(|item| match item {
            Item::Struct(item) if item.ident == state_name => Some(item),
            _ => None,
        })
        .ok_or_else(|| refusal(file, implementation, "state must be a locally declared struct"))?;
    let structural_clone = state.attrs.iter().filter(|attribute| attribute.path().is_ident("derive")).try_fold(false, |found, attribute| {
        let derives = attribute.parse_args_with(syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated).map_err(|error| refusal(file, attribute, &error.to_string()))?;
        Ok::<_, Diagnostic>(found || derives.iter().any(|path| path.is_ident("Clone") && !ast.items.iter().any(|item| matches!(item, Item::Use(item) if use_binds(&item.tree, "Clone"))) || canonical_dsl_symbol(ast, path, "State")))
    })?;
    if !structural_clone {
        return Err(refusal(
            file,
            state,
            "manual state identity cloning requires a structural Clone or canonical State derive",
        ));
    }
    for (index, field) in state.fields.iter().enumerate() {
        parser.fields.push(FieldDef {
            name: field
                .ident
                .as_ref()
                .map_or_else(|| format!("field_{index}"), ToString::to_string),
            ty: scalar_type(&field.ty)
                .ok_or_else(|| refusal(file, field, "unsupported manual state type"))?,
            start: Start::Value(integer(0)),
        });
    }
    let method = |name: &str| {
        implementation.items.iter().find_map(|item| match item {
            ImplItem::Fn(item) if item.sig.ident == name => Some(item),
            _ => None,
        })
    };
    for unsupported in ["assumptions", "scopes", "claim_order"] {
        if let Some(method) = method(unsupported) {
            return Err(refusal(
                file,
                method,
                "manual named assumptions and custom claim-order registrations are unsupported",
            ));
        }
    }
    let starts = method("starts")
        .ok_or_else(|| refusal(file, implementation, "Machine::starts is required"))?;
    single_statement(file, &starts.block)?;
    let start_expr = tail(&starts.block).ok_or_else(|| refusal(file, starts, "empty starts"))?;
    let starts = parser.starts(parser.select(start_expr)?)?;
    if let Some(start) = starts.first() {
        for field in &mut parser.fields {
            field.start = Start::Value(start[&field.name].clone());
        }
    }
    let actions = method("actions")
        .ok_or_else(|| refusal(file, implementation, "Machine::actions is required"))?;
    single_statement(file, &actions.block)?;
    let action_values = parser
        .vector(tail(&actions.block).ok_or_else(|| refusal(file, actions, "empty actions"))?)?;
    for action in action_values {
        let name = path_name(&action)
            .ok_or_else(|| refusal(file, &action, "action catalog must contain enum variants"))?;
        parser.actions.push(ActionDef {
            operator: name.replace("::", "_"),
            name,
            internal: false,
            branches: Vec::new(),
        });
    }
    let step =
        method("step").ok_or_else(|| refusal(file, implementation, "Machine::step is required"))?;
    parser.steps(&step.block)?;
    if let Some(internal) = method("internal") {
        single_statement(file, &internal.block)?;
        for index in 0..parser.actions.len() {
            let aliases = BTreeMap::from([("action".into(), parser.actions[index].name.clone())]);
            let value = parser.expr(
                parser.select(
                    tail(&internal.block)
                        .ok_or_else(|| refusal(file, internal, "empty internal predicate"))?,
                )?,
                &aliases,
            )?;
            parser.actions[index].internal = constant_bool(&value).ok_or_else(|| {
                refusal(
                    file,
                    internal,
                    "internal predicate must depend only on action and world parameters",
                )
            })?;
        }
    }
    let world = implementation.self_ty.to_token_stream().to_string();
    let world_path =
        parser.mode.as_ref().map_or_else(|| world.clone(), |mode| format!("{world}({mode})"));
    let associated = |name: &str| {
        implementation
            .items
            .iter()
            .find_map(|item| match item {
                ImplItem::Type(item) if item.ident == name => Some(&item.ty),
                _ => None,
            })
            .ok_or_else(|| refusal(file, implementation, &format!("Machine::{name} is required")))
    };
    let outcome_type = result_type(file, associated("Outcome")?)?;
    let fact_type = result_type(file, associated("Fact")?)?;
    for branch in parser.actions.iter().flat_map(|action| &action.branches) {
        if branch.outcome.ty != outcome_type || branch.facts.iter().any(|fact| fact.ty != fact_type)
        {
            return Err(refusal(
                file,
                step,
                "step outcome or fact disagrees with the declared associated type",
            ));
        }
    }
    let mut result = Lifted {
        outcome_type,
        fact_type,
        inputs: vec![file.into()],
        starts: Some(starts),
        file: file.into(),
        name: state_name.clone(),
        state_name,
        state_tuple: matches!(state.fields, syn::Fields::Unnamed(_)),
        world_path,
        constants: parser.constants.clone(),
        fields: parser.fields.clone(),
        actions: parser.actions.clone(),
        invariants: Vec::new(),
        postconditions: Vec::new(),
        progress: Vec::new(),
        assumptions: Vec::new(),
        negative: Vec::new(),
    };
    for (name, post) in [("invariants", false), ("postconditions", true)] {
        if let Some(method) = method(name) {
            single_statement(file, &method.block)?;
            for item in parser.vector(
                tail(&method.block)
                    .ok_or_else(|| refusal(file, method, "empty claim registration"))?,
            )? {
                let RustExpr::Call(call) = item else {
                    return Err(refusal(
                        file,
                        &item,
                        "claim registration must be a direct constructor",
                    ));
                };
                let constructor = call.func.to_token_stream().to_string().replace(' ', "");
                let expected = if post { "StepClaim" } else { "Claim" };
                if !constructor.starts_with(&format!("{expected}::<"))
                    || !(constructor.ends_with("::new") || post && constructor.ends_with("::after"))
                {
                    return Err(refusal(file, &call, "unsupported claim constructor"));
                }
                let claim_name = text(
                    call.args
                        .first()
                        .ok_or_else(|| refusal(file, &call, "claim name is required"))?,
                )
                .ok_or_else(|| refusal(file, &call, "claim name must be a string"))?;
                let RustExpr::Closure(closure) = call
                    .args
                    .last()
                    .ok_or_else(|| refusal(file, &call, "claim predicate is required"))?
                else {
                    return Err(refusal(file, &call, "claim predicate must be a closure"));
                };
                let mut aliases = BTreeMap::new();
                for (index, parameter) in closure.inputs.iter().enumerate() {
                    if let syn::Pat::Ident(parameter) = parameter {
                        aliases.insert(
                            parameter.ident.to_string(),
                            if post && index == 2 {
                                "after"
                            } else if post && index == 4 {
                                "outcome"
                            } else if post && index == 5 {
                                "facts"
                            } else {
                                "state"
                            }
                            .to_string(),
                        );
                    }
                }
                let expr = parser.expr(&closure.body, &aliases)?;
                if expr.ty != Type::Bool {
                    return Err(refusal(file, closure, "claim predicate must be Boolean"));
                }
                if post {
                    let after = if call.args.len() == 3 { path_name(&call.args[1]) } else { None };
                    result.postconditions.push(PostDef { name: claim_name, expr, after });
                } else {
                    result.invariants.push(ClaimDef { name: claim_name, expr });
                }
            }
        }
    }
    if let Some(method) = method("progress") {
        parser.progress(method, &mut result)?;
    }
    Ok(result)
}

#[derive(Clone)]
struct Manual<'a> {
    file: &'a str,
    mode: Option<String>,
    variants: Vec<String>,
    state_name: String,
    constants: BTreeMap<String, Expr>,
    fields: Vec<FieldDef>,
    actions: Vec<ActionDef>,
}

fn dependency(file: &str, ast: &syn::File, mode: Option<String>) -> Result<Lifted, Diagnostic> {
    fn fixture(tree: &syn::UseTree) -> Option<String> {
        match tree {
            syn::UseTree::Path(path) => path
                .ident
                .to_string()
                .strip_prefix("fixture_")
                .map(str::to_string)
                .or_else(|| fixture(&path.tree)),
            syn::UseTree::Group(group) => group.items.iter().find_map(fixture),
            _ => None,
        }
    }
    let import = ast
        .items
        .iter()
        .find_map(|item| match item {
            Item::Use(item) => fixture(&item.tree).map(|name| (item, name)),
            _ => None,
        })
        .ok_or_else(|| {
            refusal(file, ast, "no locally declared Machine implementation or fixture dependency")
        })?;
    let encoded = import.1.as_bytes();
    let mut name = String::new();
    let mut index = 0;
    while index < encoded.len() {
        if encoded[index] == b'_'
            && index + 2 < encoded.len()
            && let Ok(value) = u8::from_str_radix(&import.1[index + 1..index + 3], 16)
        {
            name.push(char::from(value));
            index += 3;
            continue;
        }
        name.push(char::from(encoded[index]));
        index += 1;
    }
    if name.contains('/') || name.contains('\\') || name == "." || name == ".." {
        return Err(refusal(file, import.0, "invalid fixture dependency identifier"));
    }
    let path = std::path::Path::new(file)
        .parent()
        .and_then(std::path::Path::parent)
        .ok_or_else(|| {
            refusal(
                file,
                import.0,
                "fixture dependency requires a source filename with its fixture directory",
            )
        })?
        .join(name)
        .join("model.rs");
    let input = path.to_string_lossy();
    let source = std::fs::read_to_string(&path).map_err(|error| {
        refusal(
            file,
            import.0,
            &format!("cannot read fixture dependency {}: {error}", path.display()),
        )
    })?;
    let ast = syn::parse_file(&source)
        .map_err(|error| Diagnostic::span(&input, error.span(), error.to_string()))?;
    if !ast.items.iter().any(|item| matches!(item, Item::Impl(item) if item.trait_.as_ref().is_some_and(|(path, _)| canonical_machine(&ast, path)))) { return Err(refusal(file, import.0, "fixture dependency must declare its Machine directly")); }
    let mut lifted = parse_selected(&input, &ast, mode)?;
    let qualifier = format!("crate::fixture_{}::", import.1);
    if let Some((world, rest)) = lifted.world_path.split_once('(') {
        lifted.world_path = format!("{qualifier}{world}({qualifier}{rest}");
    } else {
        lifted.world_path = format!("{qualifier}{}", lifted.world_path);
    }
    lifted.inputs.insert(0, file.into());
    crate::names::validate(&input, &ast, &lifted)?;
    Ok(lifted)
}

fn refusal(file: &str, node: &impl Spanned, message: &str) -> Diagnostic {
    let start = node.span().start();
    Diagnostic {
        file: file.into(),
        line: start.line,
        column: start.column + 1,
        message: format!("manual source: {message}"),
    }
}

fn canonical_machine(ast: &syn::File, path: &syn::Path) -> bool {
    canonical_dsl_symbol(ast, path, "Machine")
}

fn use_binds(tree: &syn::UseTree, name: &str) -> bool {
    match tree {
        syn::UseTree::Path(path) => use_binds(&path.tree, name),
        syn::UseTree::Name(leaf) => leaf.ident == name,
        syn::UseTree::Rename(leaf) => leaf.rename == name,
        syn::UseTree::Group(group) => group.items.iter().any(|tree| use_binds(tree, name)),
        syn::UseTree::Glob(_) => true,
    }
}

fn canonical_dsl_symbol(ast: &syn::File, path: &syn::Path, symbol: &str) -> bool {
    fn foreign_import(tree: &syn::UseTree, prefix: &str, name: &str, canonical: &str) -> bool {
        match tree {
            syn::UseTree::Path(path) => {
                foreign_import(&path.tree, &format!("{prefix}{}::", path.ident), name, canonical)
            }
            syn::UseTree::Name(leaf) => {
                leaf.ident == name && format!("{prefix}{}", leaf.ident) != canonical
            }
            syn::UseTree::Rename(leaf) => {
                leaf.rename == name && format!("{prefix}{}", leaf.ident) != canonical
            }
            syn::UseTree::Group(group) => {
                group.items.iter().any(|tree| foreign_import(tree, prefix, name, canonical))
            }
            syn::UseTree::Glob(_) => false,
        }
    }
    fn imported(tree: &syn::UseTree, prefix: &str, name: &str, canonical: &str) -> bool {
        match tree {
            syn::UseTree::Path(path) => {
                imported(&path.tree, &format!("{prefix}{}::", path.ident), name, canonical)
            }
            syn::UseTree::Name(leaf) => {
                leaf.ident == name && format!("{prefix}{}", leaf.ident) == canonical
            }
            syn::UseTree::Rename(leaf) => {
                leaf.rename == name && format!("{prefix}{}", leaf.ident) == canonical
            }
            syn::UseTree::Group(group) => {
                group.items.iter().any(|tree| imported(tree, prefix, name, canonical))
            }
            syn::UseTree::Glob(_) => false,
        }
    }
    if ast.items.iter().any(|item| matches!(item, Item::Mod(item) if item.ident == "umpire_dsl") || matches!(item, Item::ExternCrate(item) if item.rename.as_ref().is_some_and(|(_, name)| name == "umpire_dsl") && item.ident != "umpire_dsl")) { return false; }
    let path = path.to_token_stream().to_string().replace(' ', "");
    let canonical = format!("umpire_dsl::{symbol}");
    if ast.items.iter().any(|item| matches!(item, Item::Use(item) if foreign_import(&item.tree, "", "umpire_dsl", "umpire_dsl"))) { return false; }
    if !path.contains("::") && ast.items.iter().any(|item| {
        matches!(item, Item::Trait(item) if item.ident == path)
            || matches!(item, Item::Type(item) if item.ident == path)
            || matches!(item, Item::Use(item) if foreign_import(&item.tree, "", &path, &canonical))
    }) {
        return false;
    }
    path == canonical
        || path == format!("::{canonical}")
        || !path.contains("::")
            && ast
                .items
                .iter()
                .any(|item| matches!(item, Item::Use(item) if imported(&item.tree, "", &path, &canonical)))
}

fn path_name(expr: &RustExpr) -> Option<String> {
    match expr {
        RustExpr::Path(path) => Some(
            path.path.segments.iter().map(|s| s.ident.to_string()).collect::<Vec<_>>().join("::"),
        ),
        _ => None,
    }
}

fn text(expr: &RustExpr) -> Option<String> {
    if let RustExpr::Lit(value) = expr
        && let syn::Lit::Str(value) = &value.lit
    {
        return Some(value.value());
    }
    None
}

fn tail(block: &syn::Block) -> Option<&RustExpr> {
    match block.stmts.last()? {
        Stmt::Expr(expr, _) => Some(expr),
        _ => None,
    }
}

fn single_statement(file: &str, block: &syn::Block) -> Result<(), Diagnostic> {
    if block.stmts.len() == 1 {
        Ok(())
    } else {
        Err(refusal(file, block, "unsupported statements before a pure returned expression"))
    }
}

fn attributes(file: &str, attributes: &[syn::Attribute]) -> Result<(), Diagnostic> {
    for attribute in attributes {
        if attribute.path().is_ident("doc") || attribute.path().is_ident("starts") {
            continue;
        }
        if attribute.path().is_ident("derive") {
            let paths = attribute
                .parse_args_with(
                    syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated,
                )
                .map_err(|error| refusal(file, attribute, &error.to_string()))?;
            if paths.iter().all(|path| {
                path.segments.last().is_some_and(|segment| {
                    matches!(
                        segment.ident.to_string().as_str(),
                        "Clone"
                            | "Copy"
                            | "Debug"
                            | "PartialEq"
                            | "Eq"
                            | "Hash"
                            | "State"
                            | "Action"
                    )
                })
            }) {
                continue;
            }
        }
        return Err(refusal(
            file,
            attribute,
            "unsupported attribute on a manual behavior declaration",
        ));
    }
    Ok(())
}

fn selected_mode(ast: &syn::File) -> Option<String> {
    ast.items.iter().find_map(|item| match item {
        Item::Macro(item)
            if item.mac.path.segments.last().is_some_and(|s| s.ident == "fixture") =>
        {
            let expressions = item
                .mac
                .parse_body_with(
                    syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
                )
                .ok()?;
            let RustExpr::Call(call) = expressions.first()? else {
                return None;
            };
            path_name(call.args.first()?)
        }
        _ => None,
    })
}

impl Manual<'_> {
    fn starts(&self, expression: &RustExpr) -> Result<Vec<BTreeMap<String, Expr>>, Diagnostic> {
        if let RustExpr::MethodCall(collect) = expression {
            if collect.method != "collect" {
                return Err(refusal(self.file, expression, "unsupported starts iterator"));
            }
            let RustExpr::MethodCall(map) = collect.receiver.as_ref() else {
                return Err(refusal(self.file, collect, "starts collect must follow map"));
            };
            if map.method != "map" {
                return Err(refusal(self.file, map, "starts collect must follow map"));
            }
            let range = if let RustExpr::Paren(expression) = map.receiver.as_ref() {
                expression.expr.as_ref()
            } else {
                map.receiver.as_ref()
            };
            let RustExpr::Range(range) = range else {
                return Err(refusal(
                    self.file,
                    range,
                    "starts map requires a finite integer range",
                ));
            };
            let Some(RustExpr::Closure(closure)) = map.args.first() else {
                return Err(refusal(self.file, map, "starts map requires a closure"));
            };
            let Some(syn::Pat::Ident(binding)) = closure.inputs.first() else {
                return Err(refusal(self.file, closure, "starts range closure needs one binding"));
            };
            let binding_type = self.start_binding_type(&closure.body, &binding.ident)?;
            let first = self.expr(
                range
                    .start
                    .as_deref()
                    .ok_or_else(|| refusal(self.file, range, "unbounded starts range"))?,
                &BTreeMap::new(),
            )?;
            let last = self.expr(
                range
                    .end
                    .as_deref()
                    .ok_or_else(|| refusal(self.file, range, "unbounded starts range"))?,
                &BTreeMap::new(),
            )?;
            let (Some(first), Some(last)) = (constant_int(&first), constant_int(&last)) else {
                return Err(refusal(self.file, range, "starts range bounds must be constants"));
            };
            let last =
                if matches!(range.limits, syn::RangeLimits::Closed(_)) { last } else { last - 1 };
            if let Type::Int { min, max } = &binding_type
                && (first < *min || last > *max)
            {
                return Err(refusal(
                    self.file,
                    range,
                    "starts range exceeds its inferred Rust scalar type",
                ));
            }
            if last - first > 10000 {
                return Err(refusal(
                    self.file,
                    range,
                    "starts range exceeds 10001 authored values",
                ));
            }
            return (first..=last)
                .map(|value| {
                    let mut parser = self.clone();
                    parser
                        .constants
                        .insert(binding.ident.to_string(), Expr::int(value, binding_type.clone()));
                    parser.state(&closure.body, &BTreeMap::new())
                })
                .collect();
        }
        self.vector(expression)?.iter().map(|value| self.state(value, &BTreeMap::new())).collect()
    }
    fn select<'a>(&self, expression: &'a RustExpr) -> Result<&'a RustExpr, Diagnostic> {
        if let RustExpr::Match(expression) = expression {
            if !matches!(expression.expr.as_ref(), RustExpr::Field(field) if matches!(&field.member, syn::Member::Unnamed(index) if index.index == 0) && matches!(path_name(&field.base).as_deref(), Some("self" | "world")))
            {
                return Err(refusal(
                    self.file,
                    expression,
                    "only a selected world-parameter match is supported here",
                ));
            }
            let mode = self.mode.as_ref().ok_or_else(|| {
                refusal(self.file, expression, "a world parameter requires a fixture constructor")
            })?;
            for arm in &expression.arms {
                if pattern_names(&arm.pat).iter().any(|name| name == mode || name == "_") {
                    return Ok(&arm.body);
                }
            }
            return Err(refusal(
                self.file,
                expression,
                "world parameter match has no selected arm",
            ));
        }
        Ok(expression)
    }

    fn vector(&self, expression: &RustExpr) -> Result<Vec<RustExpr>, Diagnostic> {
        match self.select(expression)? {
            RustExpr::Macro(expression) if expression.mac.path.is_ident("vec") => expression
                .mac
                .parse_body_with(
                    syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
                )
                .map(|items| items.into_iter().collect())
                .map_err(|error| refusal(self.file, expression, &error.to_string())),
            RustExpr::Array(expression) => Ok(expression.elems.iter().cloned().collect()),
            RustExpr::Reference(expression) => self.vector(&expression.expr),
            expression => Err(refusal(self.file, expression, "expected an authored finite vector")),
        }
    }
}
