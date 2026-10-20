use crate::{
    Diagnostic, Type,
    tree::{ActionDef, Branch, ClaimDef, Expr, ExprKind, FieldDef, Lifted, Op, PostDef, Start},
};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{BinOp, Expr as RustExpr, ImplItem, Item, Pat, Stmt};

mod expr;
mod progress;
mod vector;

type Result<T> = std::result::Result<T, Diagnostic>;
type Env = BTreeMap<String, Expr>;

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
    let mut parser = Parser {
        file,
        fields: Vec::new(),
        constants: Env::new(),
        enums: BTreeMap::new(),
        helpers: BTreeMap::new(),
        methods: BTreeMap::new(),
        vector_imported: admitted_imports.iter().any(|name| name == "Vector"),
    };
    for item in items {
        let attrs = match item {
            Item::Struct(v) => &v.attrs,
            Item::Enum(v) => &v.attrs,
            Item::Impl(v) => &v.attrs,
            Item::Const(v) => &v.attrs,
            Item::Use(v) => &v.attrs,
            _ => {
                return Err(Diagnostic::at(
                    file,
                    item,
                    "unsupported declaration in machine module",
                ));
            }
        };
        parser.attrs(attrs, &["doc", "derive", "internal"])?;
        if let Item::Enum(value) = item {
            if !value.generics.params.is_empty()
                || value
                    .variants
                    .iter()
                    .any(|v| !matches!(v.fields, syn::Fields::Unit) || v.discriminant.is_some())
            {
                return Err(Diagnostic::at(
                    file,
                    value,
                    "only fieldless enums without explicit discriminants are admitted",
                ));
            }
            let name = value.ident.to_string();
            let variants = value.variants.iter().map(|v| format!("{name}::{}", v.ident)).collect();
            parser.enums.insert(name.clone(), Type::Enum { name, variants });
        }
    }
    for item in items {
        if let Item::Const(value) = item {
            let ty = parser.ty(&value.ty)?;
            let expr = parser.expr(&value.expr, &Env::new(), &BTreeMap::new(), Some(&ty), 0)?;
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
                parser.attrs(
                    &method.attrs,
                    &[
                        "doc",
                        "on",
                        "effect",
                        "invariant",
                        "postcondition",
                        "assumption",
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
                if owner == "Model" {
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
        parser.attrs(&field.attrs, &["doc", "starts"])?;
        let name = field
            .ident
            .as_ref()
            .ok_or_else(|| Diagnostic::at(file, field, "unnamed state field"))?
            .to_string();
        if reserved(&name) {
            return Err(Diagnostic::at(file, field, format!("reserved observer name `{name}`")));
        }
        let ty = parser.ty(&field.ty)?;
        let start = if let Some(attr) = attr(&field.attrs, "starts") {
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
            Start::Value(parser.default_value(&ty, items)?)
        };
        parser.fields.push(FieldDef { name, ty, start });
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
                actions.push(ActionDef {
                    operator: operator(&name),
                    name,
                    internal: has(&actor.attrs, "internal"),
                    branches: Vec::new(),
                });
            }
        }
    }
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
                parser.handler(method, &mut actions)?;
            } else if has(&method.attrs, "invariant") {
                invariants.push(ClaimDef {
                    name: method.sig.ident.to_string(),
                    expr: parser.claim(&method.block, &initial, &Env::new(), 0)?,
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
                let mut env = Env::new();
                for input in &method.sig.inputs {
                    if let syn::FnArg::Typed(input) = input
                        && let Pat::Ident(id) = input.pat.as_ref()
                    {
                        for field in &parser.fields {
                            env.insert(
                                format!("{}.{}", id.ident, field.name),
                                Expr::var(&field.name, field.ty.clone(), false),
                            );
                        }
                    }
                }
                postconditions.push(PostDef {
                    name: method.sig.ident.to_string(),
                    after,
                    expr: parser.claim(&method.block, &parser.field_env(true), &env, 0)?,
                });
            } else if has(&method.attrs, "assumption") {
                let mut assumption = parser.assumption(tail(file, &method.block)?, &actions, 0)?;
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
        let args = hook
            .mac
            .mac
            .parse_body_with(
                syn::punctuated::Punctuated::<RustExpr, syn::Token![,]>::parse_terminated,
            )
            .map_err(|e| Diagnostic::span(hook.file, e.span(), e.to_string()))?;
        for expr in args.iter().skip(2) {
            if let RustExpr::Call(call) = expr
                && call.args.len() == 2
            {
                let Some(RustExpr::Lit(name)) = call.args.first() else { continue };
                let syn::Lit::Str(name) = &name.lit else { continue };
                let Some(RustExpr::Closure(closure)) = call.args.last() else {
                    continue;
                };
                let mut env = Env::new();
                if let Some(pat) = closure.inputs.iter().nth(1) {
                    let pat = if let Pat::Type(p) = pat { p.pat.as_ref() } else { pat };
                    if let Pat::Ident(id) = pat {
                        for f in &parser.fields {
                            env.insert(
                                format!("{}.{}", id.ident, f.name),
                                Expr::var(&f.name, f.ty.clone(), false),
                            );
                        }
                    }
                }
                negative.push(ClaimDef {
                    name: name.value(),
                    expr: parser.expr(&closure.body, &initial, &env, Some(&Type::Bool), 0)?,
                });
            }
        }
    }
    let name = pascal(&module.ident.to_string());
    Ok(Lifted {
        file: file.into(),
        name,
        state_name: state.ident.to_string(),
        outcome_type: Type::Unit,
        world_path: format!("{}::world()", module.ident),
        fields: parser.fields,
        constants: parser.constants,
        actions,
        invariants,
        postconditions,
        progress,
        assumptions,
        negative,
    })
}

struct Conditional<'a>(Option<&'a syn::Attribute>);
impl<'ast> syn::visit::Visit<'ast> for Conditional<'ast> {
    fn visit_attribute(&mut self, attribute: &'ast syn::Attribute) {
        if attribute.path().is_ident("cfg") || attribute.path().is_ident("cfg_attr") {
            self.0 = Some(attribute);
        }
    }
}

pub(crate) fn has(attrs: &[syn::Attribute], name: &str) -> bool {
    attr(attrs, name).is_some()
}
fn attr<'a>(attrs: &'a [syn::Attribute], name: &str) -> Option<&'a syn::Attribute> {
    attrs.iter().find(|a| a.path().is_ident(name))
}
fn derives(attrs: &[syn::Attribute], name: &str) -> bool {
    attrs.iter().filter(|a| a.path().is_ident("derive")).any(|a| {
        a.parse_args_with(
            syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated,
        )
        .is_ok_and(|paths| paths.iter().any(|p| p.is_ident(name)))
    })
}
pub(crate) fn operator(name: &str) -> String {
    name.replace("::", "_")
}
fn path_name(name: &str) -> String {
    name.replace(' ', "")
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
pub(crate) fn reserved(name: &str) -> bool {
    [
        "__umpire_edge_tick",
        "__umpire_pending",
        "__umpire_steps",
        "__umpire_action",
        "__umpire_choice",
        "__umpire_outcome",
        "__umpire_facts",
    ]
    .contains(&name)
}
fn tail<'a>(file: &str, block: &'a syn::Block) -> Result<&'a RustExpr> {
    match block.stmts.as_slice() {
        [Stmt::Expr(expr, None)] => Ok(expr),
        _ => Err(Diagnostic::at(file, block, "expected a single expression body")),
    }
}

struct Parser<'a> {
    file: &'a str,
    fields: Vec<FieldDef>,
    constants: Env,
    enums: BTreeMap<String, Type>,
    helpers: BTreeMap<String, syn::ImplItemFn>,
    methods: BTreeMap<String, syn::ImplItemFn>,
    vector_imported: bool,
}
impl Parser<'_> {
    fn err<T>(&self, node: &impl ToTokens, message: impl Into<String>) -> Result<T> {
        Err(Diagnostic::at(self.file, node, message))
    }
    fn attrs(&self, attrs: &[syn::Attribute], allowed: &[&str]) -> Result<()> {
        for attr in attrs {
            let name = attr.path().to_token_stream().to_string();
            if !allowed.contains(&name.as_str()) {
                return self.err(attr,format!("unsupported attribute `{name}` (cfg and external resolution are not inferred)"));
            }
        }
        Ok(())
    }
    fn handler(&self, method: &syn::ImplItemFn, actions: &mut [ActionDef]) -> Result<()> {
        let actor = attr(&method.attrs, "on")
            .ok_or_else(|| Diagnostic::at(self.file, method, "missing actor"))?
            .parse_args::<syn::Path>()
            .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?
            .to_token_stream()
            .to_string();
        let [Stmt::Expr(RustExpr::Match(matched), _)] = method.block.stmts.as_slice() else {
            return self.err(method, "action handler must match its action parameter");
        };
        let parameter = method
            .sig
            .inputs
            .iter()
            .find_map(|v| {
                if let syn::FnArg::Typed(v) = v {
                    if let Pat::Ident(id) = v.pat.as_ref() {
                        Some(id.ident.to_string())
                    } else {
                        None
                    }
                } else {
                    None
                }
            })
            .ok_or_else(|| Diagnostic::at(self.file, method, "missing action parameter"))?;
        if !matches!(matched.expr.as_ref(),RustExpr::Path(p)if p.path.is_ident(&parameter)) {
            return self.err(&matched.expr, "handler must match the declared action parameter");
        }
        for action in actions.iter_mut().filter(|a| a.name.starts_with(&format!("{actor}::"))) {
            let mut previous = Expr::bool(false);
            for arm in &matched.arms {
                let (pattern, guard) = if let Pat::Guard(pattern) = &arm.pat {
                    (&*pattern.pat, Some(&*pattern.guard))
                } else {
                    (&arm.pat, None)
                };
                if !self.action_pattern(pattern, &action.name)? {
                    continue;
                }
                let guard = if let Some(guard) = guard {
                    self.expr(guard, &self.field_env(false), &Env::new(), Some(&Type::Bool), 0)?
                } else {
                    Expr::bool(true)
                };
                let effective = previous.clone().not().and(guard.clone());
                previous = previous.or(guard);
                let mut state = self.field_env(false);
                let mut locals = Env::new();
                let mut checks = Vec::new();
                let (body, stays) = without_stay(&arm.body);
                self.effects(&body, &mut state, &mut locals, &mut checks, 0)?;
                self.bounded_vector_growth(&body, &state, &effective)?;
                action.branches.push(Branch {
                    guard: effective,
                    updates: state,
                    checks,
                    outcome: Expr::unit(),
                    preserve_self_loop: stays,
                });
            }
        }
        Ok(())
    }
    fn action_pattern(&self, pat: &Pat, name: &str) -> Result<bool> {
        match pat {
            Pat::Wild(_) => Ok(true),
            Pat::Path(path) => Ok(path_name(&path.path.to_token_stream().to_string()) == name),
            Pat::Or(or) => {
                for case in &or.cases {
                    if self.action_pattern(case, name)? {
                        return Ok(true);
                    }
                }
                Ok(false)
            }
            _ => self.err(pat, "unsupported action match pattern"),
        }
    }
    #[expect(
        clippy::too_many_lines,
        reason = "SSA lowering follows the admitted statement variants"
    )]
    fn effects(
        &self,
        node: &RustExpr,
        state: &mut Env,
        locals: &mut Env,
        checks: &mut Vec<Expr>,
        depth: usize,
    ) -> Result<()> {
        if depth > 32 {
            return self.err(node, "recursive effects are unsupported");
        }
        match node {
            RustExpr::Block(block) => {
                let mut scoped = locals.clone();
                for statement in &block.block.stmts {
                    match statement {
                        Stmt::Local(local) => {
                            let Pat::Ident(id) = &local.pat else {
                                return self.err(local, "only simple local bindings are admitted");
                            };
                            let Some(init) = &local.init else {
                                return self.err(local, "uninitialized local is unsupported");
                            };
                            if init.diverge.is_some() {
                                return self.err(local, "let-else is unsupported");
                            }
                            let value = self.expr(&init.expr, state, &scoped, None, depth)?;
                            checks.push(value.clone());
                            scoped.insert(id.ident.to_string(), value);
                        }
                        Stmt::Expr(expr, _) => {
                            self.effects(expr, state, &mut scoped, checks, depth)?;
                        }
                        _ => return self.err(statement, "unsupported effect statement"),
                    }
                }
            }
            RustExpr::Assign(assign) => {
                let field = self.assigned_field(&assign.left)?;
                let ty = &self
                    .fields
                    .iter()
                    .find(|f| f.name == field)
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown assignment field"))?
                    .ty;
                let value = self.expr(&assign.right, state, locals, Some(ty), depth)?;
                checks.push(value.clone());
                state.insert(field, value);
            }
            RustExpr::Binary(binary) => {
                let op = match binary.op {
                    BinOp::AddAssign(_) => Op::Add,
                    BinOp::SubAssign(_) => Op::Sub,
                    BinOp::MulAssign(_) => Op::Mul,
                    _ => return self.err(node, "unsupported effect binary operator"),
                };
                let field = self.assigned_field(&binary.left)?;
                let left = state
                    .get(&field)
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown assignment field"))?
                    .clone();
                if !matches!(left.ty, Type::Int { .. }) {
                    return self.err(node, "compound assignment requires an integer");
                }
                let right = self.expr(&binary.right, state, locals, Some(&left.ty), depth)?;
                let value = Expr::binary(op, left, right);
                checks.push(value.clone());
                state.insert(field, value);
            }
            RustExpr::If(condition) => {
                let cond = self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                let mut yes = state.clone();
                let mut no = state.clone();
                let mut yc = Vec::new();
                let mut nc = Vec::new();
                let yes_expr = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: condition.then_branch.clone(),
                });
                self.effects(&yes_expr, &mut yes, &mut locals.clone(), &mut yc, depth)?;
                if let Some((_, other)) = &condition.else_branch {
                    self.effects(other, &mut no, &mut locals.clone(), &mut nc, depth)?;
                }
                for (name, value) in state.iter_mut() {
                    *value = Expr::select(cond.clone(), yes[name].clone(), no[name].clone());
                }
                for value in yc {
                    checks.push(Expr::select(cond.clone(), value.clone(), default_expr(&value.ty)));
                }
                for value in nc {
                    checks.push(Expr::select(cond.clone(), default_expr(&value.ty), value));
                }
            }
            RustExpr::MethodCall(call) if call.method == "push_back" => {
                self.vector_push(call, state, locals, checks, depth)?;
            }
            RustExpr::MethodCall(call) if matches!(call.receiver.as_ref(),RustExpr::Path(p)if p.path.is_ident("self")) =>
            {
                let method = self
                    .methods
                    .get(&call.method.to_string())
                    .ok_or_else(|| Diagnostic::at(self.file, node, "unknown effect helper"))?;
                if !has(&method.attrs, "effect")
                    || !call.args.is_empty()
                    || method.sig.inputs.len() != 1
                {
                    return self.err(node, "only parameterless #[effect] calls are admitted");
                }
                let body = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: method.block.clone(),
                });
                self.effects(&body, state, &mut Env::new(), checks, depth + 1)?;
            }
            _ => return self.err(node, format!("unsupported effect `{}`", node.to_token_stream())),
        }
        Ok(())
    }
    fn assigned_field(&self, node: &RustExpr) -> Result<String> {
        if let RustExpr::Field(field) = node
            && matches!(field.base.as_ref(),RustExpr::Path(p)if p.path.is_ident("self"))
        {
            return Ok(field.member.to_token_stream().to_string());
        }
        self.err(node, "assignment must target a declared self field")
    }
    fn claim(&self, block: &syn::Block, state: &Env, locals: &Env, depth: usize) -> Result<Expr> {
        let mut checks = Vec::new();
        for statement in &block.stmts {
            let value = match statement {
                Stmt::Macro(mac) if mac.mac.path.is_ident("check") => {
                    let expr: RustExpr = syn::parse2(mac.mac.tokens.clone())
                        .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
                    self.expr(&expr, state, locals, Some(&Type::Bool), depth)?
                }
                Stmt::Expr(RustExpr::Macro(mac), _) if mac.mac.path.is_ident("check") => {
                    let expr: RustExpr = syn::parse2(mac.mac.tokens.clone())
                        .map_err(|e| Diagnostic::span(self.file, e.span(), e.to_string()))?;
                    self.expr(&expr, state, locals, Some(&Type::Bool), depth)?
                }
                Stmt::Expr(RustExpr::If(condition), _) => {
                    let cond =
                        self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                    let yes = cond.clone().implies(self.claim(
                        &condition.then_branch,
                        state,
                        locals,
                        depth,
                    )?);
                    if let Some((_, other)) = &condition.else_branch {
                        let RustExpr::Block(other) = other.as_ref() else {
                            return self.err(other, "claim else-if is unsupported");
                        };
                        yes.and(cond.not().implies(self.claim(
                            &other.block,
                            state,
                            locals,
                            depth,
                        )?))
                    } else {
                        yes
                    }
                }
                _ => {
                    return self
                        .err(statement, "claim body supports check! and conditional checks only");
                }
            };
            checks.push(value);
        }
        Ok(Expr { kind: ExprKind::Checks(checks), ty: Type::Bool })
    }
}
fn is_stay(mac: &syn::Macro) -> bool {
    mac.path.is_ident("stay") && mac.tokens.is_empty()
}
/// Strips `stay!()` from an arm's top level; one nested in a condition stays and is refused.
fn without_stay(body: &RustExpr) -> (RustExpr, bool) {
    match body {
        RustExpr::Macro(mac) if is_stay(&mac.mac) => (syn::parse_quote!({}), true),
        RustExpr::Block(block) => {
            let mut block = block.clone();
            let before = block.block.stmts.len();
            block.block.stmts.retain(|statement| match statement {
                Stmt::Macro(mac) => !is_stay(&mac.mac),
                Stmt::Expr(RustExpr::Macro(mac), _) => !is_stay(&mac.mac),
                _ => true,
            });
            let stays = block.block.stmts.len() != before;
            (RustExpr::Block(block), stays)
        }
        _ => (body.clone(), false),
    }
}
fn default_expr(ty: &Type) -> Expr {
    match ty {
        Type::Bool => Expr::bool(false),
        Type::Int { .. } => Expr::int(0, ty.clone()),
        Type::Seq(_) => Expr { kind: ExprKind::Sequence(Vec::new()), ty: ty.clone() },
        Type::Enum { variants, .. } => {
            Expr { kind: ExprKind::Enum(variants[0].clone()), ty: ty.clone() }
        }
        _ => Expr::unit(),
    }
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
