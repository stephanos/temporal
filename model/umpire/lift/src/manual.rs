mod expr;
mod progress;
use expr::{
    constant_bool, constant_int, integer, pattern_names, result_type, same_scalar_type, scalar_type,
};

use std::collections::BTreeMap;

use quote::ToTokens;
use syn::{Expr as RustExpr, ImplItem, Item, Stmt, spanned::Spanned};

use crate::tree::{ActionDef, Branch, ClaimDef, Expr, FieldDef, Op, PostDef, Start};
use crate::{Diagnostic, Lifted, Type};

pub(crate) fn parse(file: &str, ast: &syn::File) -> Result<Lifted, Diagnostic> {
    parse_selected(file, ast, selected_mode(ast))
}

#[allow(clippy::too_many_lines)]
fn parse_selected(file: &str, ast: &syn::File, mode: Option<String>) -> Result<Lifted, Diagnostic> {
    for item in &ast.items {
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
    fn imported(tree: &syn::UseTree, prefix: &str, name: &str) -> bool {
        match tree {
            syn::UseTree::Path(path) => {
                imported(&path.tree, &format!("{prefix}{}::", path.ident), name)
            }
            syn::UseTree::Name(leaf) => {
                leaf.ident == name && format!("{prefix}{}", leaf.ident) == "umpire_dsl::Machine"
            }
            syn::UseTree::Rename(leaf) => {
                leaf.rename == name && format!("{prefix}{}", leaf.ident) == "umpire_dsl::Machine"
            }
            syn::UseTree::Group(group) => {
                group.items.iter().any(|tree| imported(tree, prefix, name))
            }
            syn::UseTree::Glob(_) => false,
        }
    }
    if ast.items.iter().any(|item| matches!(item, Item::Mod(item) if item.ident == "umpire_dsl") || matches!(item, Item::ExternCrate(item) if item.rename.as_ref().is_some_and(|(_, name)| name == "umpire_dsl") && item.ident != "umpire_dsl")) { return false; }
    let path = path.to_token_stream().to_string().replace(' ', "");
    if ast.items.iter().any(|item| matches!(item, Item::Use(item) if foreign_import(&item.tree, "", "umpire_dsl", "umpire_dsl"))) { return false; }
    if !path.contains("::") && ast.items.iter().any(|item| matches!(item, Item::Trait(item) if item.ident == path) || matches!(item, Item::Type(item) if item.ident == path) || matches!(item, Item::Use(item) if foreign_import(&item.tree, "", &path, "umpire_dsl::Machine"))) { return false; }
    path == "umpire_dsl::Machine"
        || path == "::umpire_dsl::Machine"
        || !path.contains("::")
            && ast
                .items
                .iter()
                .any(|item| matches!(item, Item::Use(item) if imported(&item.tree, "", &path)))
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
                    parser.constants.insert(binding.ident.to_string(), integer(value));
                    parser.state(&closure.body, &BTreeMap::new())
                })
                .collect();
        }
        self.vector(expression)?.iter().map(|value| self.state(value, &BTreeMap::new())).collect()
    }
    fn steps(&mut self, block: &syn::Block) -> Result<(), Diagnostic> {
        if block.stmts.iter().any(|statement| matches!(statement, Stmt::Local(local) if local.init.as_ref().is_some_and(|init| matches!(init.expr.as_ref(), RustExpr::Match(matching) if matches!(matching.expr.as_ref(), RustExpr::Tuple(_)))))) { return self.optional_match(block); }
        if block
            .stmts
            .iter()
            .any(|statement| matches!(statement, Stmt::Expr(RustExpr::Match(_), _)))
        {
            return self.mutating_match(block);
        }
        if let Some(edges) = block.stmts.iter().find_map(|statement| match statement { Stmt::Local(local) if matches!(&local.pat, syn::Pat::Type(pattern) if matches!(pattern.pat.as_ref(), syn::Pat::Ident(name) if name.ident == "edges")) => local.init.as_ref().map(|init| &*init.expr), _ => None }) {
            let edges = self.vector(self.select(edges)?)?;
            for edge in edges {
                let RustExpr::Tuple(edge) = edge else { return Err(refusal(self.file, &edge, "edge table entries must be tuples")); };
                if edge.elems.len() != 3 || self.fields.len() != 1 { return Err(refusal(self.file, &edge, "edge table requires a scalar state and (from, action, to) tuples")); }
                let action = path_name(&edge.elems[1]).ok_or_else(|| refusal(self.file, &edge, "edge action must be an enum variant"))?;
                let index = self.actions.iter().position(|candidate| candidate.name.rsplit("::").next() == action.rsplit("::").next()).ok_or_else(|| refusal(self.file, &edge, "edge action absent from catalog"))?;
                let guard = Expr::binary(Op::Eq, Expr::var(&self.fields[0].name, self.fields[0].ty.clone(), false), self.expr(&edge.elems[0], &BTreeMap::new())?);
                let updates = BTreeMap::from([(self.fields[0].name.clone(), self.expr(&edge.elems[2], &BTreeMap::new())?)]);
                let choice = self.actions[index].branches.iter().filter(|branch| branch.guard == guard).count();
                self.actions[index].branches.push(Branch { guard, updates, choice, checks: Vec::new(), outcome: Expr::unit(), facts: Vec::new(), preserve_self_loop: true });
            }
            self.validate_edge_loop(block)?;
            return Ok(());
        }
        if block.stmts.len() < 2
            || block.stmts.first().is_none_or(|statement| {
                statement.to_token_stream().to_string().replace(' ', "")
                    != "letmutsteps=Steps::new();"
            })
            || block
                .stmts
                .last()
                .is_none_or(|statement| statement.to_token_stream().to_string() != "steps")
        {
            return Err(refusal(
                self.file,
                block,
                "manual pushes must append to one initially empty returned steps vector",
            ));
        }
        let mut body = block.clone();
        body.stmts = body.stmts[1..body.stmts.len() - 1].to_vec();
        for index in 0..self.actions.len() {
            let aliases = BTreeMap::from([("action".into(), self.actions[index].name.clone())]);
            let mut branches = Vec::new();
            self.walk_steps(&body, &Expr::bool(true), &aliases, &mut branches)?;
            self.actions[index].branches = branches;
        }
        Ok(())
    }

    fn optional_match(&mut self, block: &syn::Block) -> Result<(), Diagnostic> {
        let matching = block
            .stmts
            .iter()
            .find_map(|statement| match statement {
                Stmt::Local(local) => {
                    local.init.as_ref().and_then(|init| match init.expr.as_ref() {
                        RustExpr::Match(matching) => Some(matching),
                        _ => None,
                    })
                }
                _ => None,
            })
            .ok_or_else(|| refusal(self.file, block, "expected optional next-state match"))?;
        let RustExpr::Tuple(scrutinee) = matching.expr.as_ref() else {
            return Err(refusal(
                self.file,
                matching,
                "optional state match must match state fields and action",
            ));
        };
        for action_index in 0..self.actions.len() {
            let aliases =
                BTreeMap::from([("action".into(), self.actions[action_index].name.clone())]);
            for arm in &matching.arms {
                if matches!(arm.pat, syn::Pat::Wild(_)) {
                    if path_name(&arm.body).as_deref() != Some("None") {
                        return Err(refusal(
                            self.file,
                            arm,
                            "unmatched optional state must be None",
                        ));
                    }
                    continue;
                }
                let syn::Pat::Tuple(pattern) = &arm.pat else {
                    return Err(refusal(self.file, arm, "optional state requires tuple patterns"));
                };
                if pattern.elems.len() != scrutinee.elems.len() {
                    return Err(refusal(self.file, arm, "tuple pattern has wrong arity"));
                }
                let mut guard = Expr::bool(true);
                for (pattern, value) in pattern.elems.iter().zip(&scrutinee.elems) {
                    guard = guard.and(self.pattern(pattern, self.expr(value, &aliases)?)?);
                }
                let RustExpr::Call(some) = arm.body.as_ref() else {
                    return Err(refusal(self.file, arm, "optional state arm must construct Some"));
                };
                if path_name(&some.func).as_deref() != Some("Some") || some.args.len() != 1 {
                    return Err(refusal(self.file, arm, "optional state arm must construct Some"));
                }
                let updates = self.state(&some.args[0], &aliases)?;
                let previous = self.actions[action_index]
                    .branches
                    .iter()
                    .fold(Expr::bool(false), |previous, branch| previous.or(branch.guard.clone()));
                self.actions[action_index].branches.push(Branch {
                    guard: guard.and(previous.not()),
                    updates,
                    checks: Vec::new(),
                    choice: 0,
                    outcome: Expr::bool(true),
                    facts: Vec::new(),
                    preserve_self_loop: true,
                });
            }
        }
        self.validate_optional_return(block)
    }

    fn validate_optional_return(&self, block: &syn::Block) -> Result<(), Diagnostic> {
        let suffix = block
            .stmts
            .iter()
            .filter_map(|statement| match statement {
                Stmt::Expr(RustExpr::If(expression), _) => Some(expression),
                _ => None,
            })
            .collect::<Vec<_>>();
        if suffix.len() != 1
            || suffix[0].to_token_stream().to_string().replace(' ', "")
                != "ifletSome(next)=next{steps.push((true,vec![],next));}"
        {
            return Err(refusal(
                self.file,
                block,
                "optional state match must push its selected state once with a true outcome and no facts",
            ));
        }
        if block.stmts.len() != 4 {
            return Err(refusal(
                self.file,
                block,
                "unsupported statements around optional state match",
            ));
        }
        if block.stmts.first().is_none_or(|statement| {
            statement.to_token_stream().to_string().replace(' ', "") != "letmutsteps=Steps::new();"
        }) || !matches!(block.stmts.get(1), Some(Stmt::Local(local)) if matches!(&local.pat, syn::Pat::Ident(binding) if binding.ident == "next"))
            || block
                .stmts
                .last()
                .is_none_or(|statement| statement.to_token_stream().to_string() != "steps")
        {
            return Err(refusal(
                self.file,
                block,
                "optional state match requires an initially empty returned steps vector",
            ));
        }
        Ok(())
    }

    #[allow(clippy::too_many_lines)]
    fn mutating_match(&mut self, block: &syn::Block) -> Result<(), Diagnostic> {
        let Some(Stmt::Local(local)) = block.stmts.first() else {
            return Err(refusal(self.file, block, "expected cloned next state"));
        };
        let syn::Pat::Ident(next) = &local.pat else {
            return Err(refusal(self.file, local, "expected next-state binding"));
        };
        if !local.init.as_ref().is_some_and(|init| matches!(init.expr.as_ref(), RustExpr::MethodCall(call) if call.method == "clone" && call.args.is_empty() && path_name(&call.receiver).as_deref() == Some("state"))) { return Err(refusal(self.file, local, "next state must clone state")); }
        let Some(Stmt::Expr(RustExpr::Match(matching), _)) = block.stmts.get(1) else {
            return Err(refusal(self.file, block, "expected action match"));
        };
        if path_name(&matching.expr).as_deref() != Some("action") {
            return Err(refusal(self.file, matching, "manual step must match the action"));
        }
        for arm in &matching.arms {
            if matches!(arm.pat, syn::Pat::Wild(_)) {
                if !matches!(arm.body.as_ref(), RustExpr::Return(expression) if expression.expr.as_ref().is_some_and(|expression| expression.to_token_stream().to_string().replace(' ', "") == "Steps::new()"))
                {
                    return Err(refusal(self.file, arm, "fallback must return empty steps"));
                }
                continue;
            }
            let names = pattern_names(&arm.pat);
            let RustExpr::Block(body) = arm.body.as_ref() else {
                return Err(refusal(self.file, arm, "action arm requires an assignment block"));
            };
            let guard = if let syn::Pat::Guard(pattern) = &arm.pat {
                self.expr(&pattern.guard, &BTreeMap::new())?
            } else {
                Expr::bool(true)
            };
            let mut updates = BTreeMap::new();
            for statement in &body.block.stmts {
                let Stmt::Expr(expression, _) = statement else {
                    return Err(refusal(
                        self.file,
                        statement,
                        "only next-state assignments are admitted",
                    ));
                };
                let (left, right, add) = match expression {
                    RustExpr::Assign(expression) => (&*expression.left, &*expression.right, false),
                    RustExpr::Binary(expression)
                        if matches!(expression.op, syn::BinOp::AddAssign(_)) =>
                    {
                        (&*expression.left, &*expression.right, true)
                    }
                    _ => return Err(refusal(self.file, expression, "unsupported state mutation")),
                };
                let RustExpr::Field(field) = left else {
                    return Err(refusal(self.file, left, "assignment must target a state field"));
                };
                if path_name(&field.base).as_deref() != Some(next.ident.to_string().as_str()) {
                    return Err(refusal(
                        self.file,
                        left,
                        "assignment must target the cloned next state",
                    ));
                }
                let name = field.member.to_token_stream().to_string();
                let definition = self
                    .fields
                    .iter()
                    .find(|field| field.name == name)
                    .ok_or_else(|| refusal(self.file, field, "unknown next-state field"))?;
                let mut value = self.expr(right, &BTreeMap::new())?;
                if !same_scalar_type(&definition.ty, &value.ty) {
                    return Err(refusal(
                        self.file,
                        right,
                        "assignment value disagrees with the state field type",
                    ));
                }
                if add {
                    let definition = self
                        .fields
                        .iter()
                        .find(|field| field.name == name)
                        .ok_or_else(|| refusal(self.file, field, "unknown field"))?;
                    value = Expr::binary(
                        Op::Add,
                        updates
                            .get(&name)
                            .cloned()
                            .unwrap_or_else(|| Expr::var(&name, definition.ty.clone(), false)),
                        value,
                    );
                }
                updates.insert(name, value);
            }
            for action in &mut self.actions {
                if names.contains(&action.name) {
                    let previous =
                        action.branches.iter().fold(Expr::bool(false), |previous, branch| {
                            previous.or(branch.guard.clone())
                        });
                    action.branches.push(Branch {
                        guard: guard.clone().and(previous.not()),
                        updates: updates.clone(),
                        choice: 0,
                        checks: Vec::new(),
                        outcome: Expr::unit(),
                        facts: Vec::new(),
                        preserve_self_loop: true,
                    });
                }
            }
        }
        if block.stmts.len() != 5 {
            return Err(refusal(
                self.file,
                block,
                "unsupported statements around the action match",
            ));
        }
        if block.stmts.get(2).is_none_or(|statement| {
            statement.to_token_stream().to_string().replace(' ', "") != "letmutsteps=Steps::new();"
        }) || block
            .stmts
            .last()
            .is_none_or(|statement| statement.to_token_stream().to_string() != "steps")
        {
            return Err(refusal(
                self.file,
                block,
                "action match must return an initially empty steps vector",
            ));
        }
        let Some(Stmt::Expr(RustExpr::MethodCall(push), _)) = block.stmts.get(3) else {
            return Err(refusal(self.file, block, "expected final next-state push"));
        };
        if push.method != "push"
            || path_name(&push.receiver).as_deref() != Some("steps")
            || push.args.first().is_none_or(|arg| {
                arg.to_token_stream().to_string().replace(' ', "")
                    != format!("((),Vec::new(),{})", next.ident)
            })
        {
            return Err(refusal(
                self.file,
                push,
                "expected unit outcome, no facts and the cloned next state",
            ));
        }
        Ok(())
    }

    fn validate_edge_loop(&self, block: &syn::Block) -> Result<(), Diagnostic> {
        let statements = block
            .stmts
            .iter()
            .filter(|statement| !matches!(statement, Stmt::Item(Item::Use(_))))
            .collect::<Vec<_>>();
        if statements.len() != 4
            || !matches!(statements.get(1), Some(Stmt::Local(local)) if local.init.as_ref().is_some_and(|init| init.expr.to_token_stream().to_string().replace(' ', "") == "Steps::new()"))
            || !matches!(statements.last(), Some(Stmt::Expr(RustExpr::Path(path), None)) if path.path.is_ident("steps"))
        {
            return Err(refusal(self.file, block, "unsupported statements around edge table"));
        }
        let loops = block
            .stmts
            .iter()
            .filter_map(|statement| match statement {
                Stmt::Expr(RustExpr::ForLoop(loop_), _) => Some(loop_),
                _ => None,
            })
            .collect::<Vec<_>>();
        if loops.len() != 1 {
            return Err(refusal(self.file, block, "edge table must have exactly one loop"));
        }
        let iteration = loops[0];
        let state_name: syn::Path = syn::parse_str(&self.state_name)
            .map_err(|error| refusal(self.file, iteration, &error.to_string()))?;
        let expected: syn::ExprForLoop = syn::parse_quote! { for &(from, edge_action, to) in edges { if state.0 == from && action == edge_action { steps.push(((), vec![], #state_name(to))); } } };
        let actual = iteration.to_token_stream().to_string();
        let expected = expected.to_token_stream().to_string();
        if actual != expected {
            return Err(refusal(
                self.file,
                iteration,
                "unsupported edge-table loop; expected exact source guard and tuple projection",
            ));
        }
        Ok(())
    }

    fn walk_steps(
        &self,
        block: &syn::Block,
        guard: &Expr,
        aliases: &BTreeMap<String, String>,
        branches: &mut Vec<Branch>,
    ) -> Result<(), Diagnostic> {
        let conditions = block
            .stmts
            .iter()
            .filter(|statement| matches!(statement, Stmt::Expr(RustExpr::If(_), _)))
            .count();
        if conditions > 1 || conditions == 1 && block.stmts.len() != 1 {
            return Err(refusal(
                self.file,
                block,
                "overlapping conditional push groups require dynamic choice numbering and are unsupported",
            ));
        }
        let mut choice = 0;
        for statement in &block.stmts {
            match statement {
                Stmt::Expr(RustExpr::If(expression), _) => {
                    let condition = self.expr(&expression.cond, aliases)?;
                    if condition.ty != Type::Bool {
                        return Err(refusal(
                            self.file,
                            expression,
                            "step condition must be Boolean",
                        ));
                    }
                    self.walk_steps(
                        &expression.then_branch,
                        &guard.clone().and(condition.clone()),
                        aliases,
                        branches,
                    )?;
                    if let Some((_, otherwise)) = &expression.else_branch {
                        let RustExpr::Block(otherwise) = otherwise.as_ref() else {
                            return Err(refusal(
                                self.file,
                                expression,
                                "else-if manual step is unsupported",
                            ));
                        };
                        self.walk_steps(
                            &otherwise.block,
                            &guard.clone().and(condition.not()),
                            aliases,
                            branches,
                        )?;
                    }
                }
                Stmt::Expr(RustExpr::MethodCall(expression), _)
                    if expression.method == "push"
                        && path_name(&expression.receiver).as_deref() == Some("steps")
                        && expression.args.len() == 1 =>
                {
                    let Some(RustExpr::Tuple(value)) = expression.args.first() else {
                        return Err(refusal(
                            self.file,
                            expression,
                            "step must push an outcome/facts/state tuple",
                        ));
                    };
                    if value.elems.len() != 3 {
                        return Err(refusal(
                            self.file,
                            value,
                            "step tuple must have exactly three elements",
                        ));
                    }
                    let outcome = self.expr(&value.elems[0], aliases)?;
                    let facts = self
                        .vector(&value.elems[1])?
                        .iter()
                        .map(|value| self.expr(value, aliases))
                        .collect::<Result<Vec<_>, _>>()?;
                    let updates = self.state(&value.elems[2], aliases)?;
                    branches.push(Branch {
                        guard: guard.clone(),
                        updates,
                        outcome,
                        facts,
                        choice,
                        checks: Vec::new(),
                        preserve_self_loop: true,
                    });
                    choice += 1;
                }
                _ => {
                    return Err(refusal(self.file, statement, "unsupported manual step statement"));
                }
            }
        }
        Ok(())
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
