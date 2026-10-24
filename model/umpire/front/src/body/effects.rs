use super::{
    Env, Environment, Result,
    outcome::{Flow, rejects},
};
use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind, Op},
    syntax::has,
    tree::{ActionDef, Branch},
};
use quote::ToTokens;
use syn::{BinOp, Expr as RustExpr, Pat, Stmt};

impl Environment {
    pub(crate) fn handler<'a>(&self, method: &'a syn::ImplItemFn) -> Result<&'a syn::ExprMatch> {
        let parameters = self.parameters(method)?;
        let [Stmt::Expr(RustExpr::Match(matched), _)] = method.block.stmts.as_slice() else {
            return self.err(method, "action handler must match its action parameter");
        };
        let parameter = parameters
            .positional
            .first()
            .and_then(|input| match input.pat.as_ref() {
                Pat::Ident(id) => Some(id.ident.to_string()),
                _ => None,
            })
            .ok_or_else(|| Diagnostic::at(method, "missing action parameter"))?;
        if !matches!(matched.expr.as_ref(),RustExpr::Path(p)if p.path.is_ident(&parameter)) {
            return self.err(&matched.expr, "handler must match the declared action parameter");
        }
        Ok(matched)
    }
    pub(crate) fn branches(
        &self,
        matched: &syn::ExprMatch,
        action: &ActionDef,
        env: &Env,
        fact: bool,
    ) -> Result<Vec<Branch>> {
        let mut branches = Vec::new();
        let mut previous = Expr::bool(false);
        for arm in &matched.arms {
            let (pattern, guard) = if let Pat::Guard(pattern) = &arm.pat {
                (&*pattern.pat, Some(&*pattern.guard))
            } else {
                (&arm.pat, None)
            };
            let Some(matched) = self.action_pattern(pattern, action)? else {
                continue;
            };
            // Payload names shadow the World parameter and source constants lexically, as in Rust.
            let mut locals = env.clone();
            for (name, value) in matched.bindings {
                locals.insert(name, value);
            }
            let guard = if let Some(guard) = guard {
                matched.condition.and(self.expr(
                    guard,
                    &self.field_env(false),
                    &locals,
                    Some(&Type::Bool),
                    0,
                )?)
            } else {
                matched.condition
            };
            let effective = previous.clone().not().and(guard.clone());
            previous = previous.or(guard);
            let mut state = self.field_env(false);
            let mut checks = Vec::new();
            let (body, stays) = without_stay(&arm.body);
            let mut flow = Flow::new(&self.outcome_type);
            let mut definitions = Vec::new();
            let facts = self.effects(
                &body,
                &mut state,
                &mut locals,
                &mut checks,
                &mut flow,
                &mut definitions,
                0,
                fact,
            )?;
            self.bounded_vector_growth(&body, &state, &effective, &definitions)?;
            // A rejection is an enabled self-loop, as natively; it is never a dropped branch.
            let outcome = flow.outcome();
            let stays = stays || rejects(&outcome);
            branches.push(Branch {
                definitions,
                guard: effective,
                updates: state.values,
                checks,
                outcome,
                facts: facts
                    .unwrap_or_else(|| default_expr(&Type::Seq(Box::new(self.fact_type.clone())))),
                preserve_self_loop: stays,
            });
        }
        Ok(branches)
    }
    #[expect(
        clippy::too_many_lines,
        reason = "SSA lowering follows the admitted statement variants"
    )]
    #[expect(clippy::too_many_arguments, reason = "SSA lowering threads its frame by reference")]
    fn effects(
        &self,
        node: &RustExpr,
        state: &mut Env,
        locals: &mut Env,
        checks: &mut Vec<Expr>,
        flow: &mut Flow,
        definitions: &mut Vec<(String, Expr)>,
        depth: usize,
        fact: bool,
    ) -> Result<Option<Expr>> {
        if depth > 32 {
            return self.err(node, "recursive effects are unsupported");
        }
        let mut result = None;
        match node {
            RustExpr::Block(block) => {
                let mut scoped = locals.clone();
                for (index, statement) in block.block.stmts.iter().enumerate() {
                    if flow.returned() {
                        break;
                    }
                    match statement {
                        Stmt::Local(local) => {
                            let (pattern, expected) = match &local.pat {
                                Pat::Type(value) => (value.pat.as_ref(), Some(self.ty(&value.ty)?)),
                                pattern => (pattern, None),
                            };
                            let Pat::Ident(id) = pattern else {
                                return self.err(local, "only simple local bindings are admitted");
                            };
                            let Some(init) = &local.init else {
                                return self.err(local, "uninitialized local is unsupported");
                            };
                            if init.diverge.is_some() {
                                return self.err(local, "let-else is unsupported");
                            }
                            let value =
                                self.expr(&init.expr, state, &scoped, expected.as_ref(), depth)?;
                            checks.push(flow.guard(value.clone(), default_expr(&value.ty)));
                            scoped.insert(id.ident.to_string(), define(value, definitions));
                        }
                        Stmt::Expr(expr, semi) => {
                            let tail =
                                fact && index + 1 == block.block.stmts.len() && semi.is_none();
                            result = self.effects(
                                expr,
                                state,
                                &mut scoped,
                                checks,
                                flow,
                                definitions,
                                depth,
                                tail,
                            )?;
                        }
                        Stmt::Macro(mac) if Self::is_reject(&mac.mac) => {
                            self.reject(&mac.mac, state, &scoped, flow, depth)?;
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
                    .ok_or_else(|| Diagnostic::at(node, "unknown assignment field"))?
                    .ty;
                let value = self.expr(&assign.right, state, locals, Some(ty), depth)?;
                // After a conditional return, an update applies only where the step still runs.
                checks.push(flow.guard(value.clone(), default_expr(&value.ty)));
                let before = state[&field].clone();
                state.insert(field, flow.guard(value, before));
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
                    .ok_or_else(|| Diagnostic::at(node, "unknown assignment field"))?
                    .clone();
                if !matches!(left.ty, Type::Int { .. }) {
                    return self.err(node, "compound assignment requires an integer");
                }
                let right = self.expr(&binary.right, state, locals, Some(&left.ty), depth)?;
                let value = Expr::binary(op, left.clone(), right);
                checks.push(flow.guard(value.clone(), default_expr(&value.ty)));
                state.insert(field, flow.guard(value, left));
            }
            RustExpr::If(condition) => {
                let cond = self.expr(&condition.cond, state, locals, Some(&Type::Bool), depth)?;
                // After a conditional return, the condition is evaluated only where the step
                // still runs, as natively; `/\` stops at a returned path.
                let cond = if flow.diverged() { flow.running.clone().and(cond) } else { cond };
                checks.push(cond.clone());
                let cond = define(cond, definitions);
                let mut yes = state.clone();
                let mut no = state.clone();
                let mut yc = Vec::new();
                let mut nc = Vec::new();
                let mut yf = flow.clone();
                let mut nf = flow.clone();
                let yes_expr = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: condition.then_branch.clone(),
                });
                let yes_fact = self.effects(
                    &yes_expr,
                    &mut yes,
                    &mut locals.clone(),
                    &mut yc,
                    &mut yf,
                    definitions,
                    depth,
                    fact,
                )?;
                let mut no_fact = None;
                if let Some((_, other)) = &condition.else_branch {
                    no_fact = self.effects(
                        other,
                        &mut no,
                        &mut locals.clone(),
                        &mut nc,
                        &mut nf,
                        definitions,
                        depth,
                        fact,
                    )?;
                }
                if fact {
                    let Some(no_fact) = no_fact else {
                        return self.err(node, "fact tail requires an exhaustive if/else");
                    };
                    let Some(yes_fact) = yes_fact else {
                        return self.err(node, "fact tail requires a result");
                    };
                    result = Some(Expr::select(cond.clone(), yes_fact, no_fact));
                }
                flow.join(&cond, yf, nf);
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
                self.vector_push(call, state, locals, checks, flow, depth)?;
            }
            RustExpr::Macro(mac) if Self::is_reject(&mac.mac) => {
                self.reject(&mac.mac, state, locals, flow, depth)?;
            }
            RustExpr::MethodCall(call) if matches!(call.receiver.as_ref(),RustExpr::Path(p)if p.path.is_ident("self")) =>
            {
                let method = self
                    .methods
                    .get(&call.method.to_string())
                    .ok_or_else(|| Diagnostic::at(node, "unknown effect helper"))?;
                if !has(&method.attrs, "effect") || method.sig.receiver().is_none() {
                    return self.err(
                        node,
                        "only #[effect] calls with an optional trailing World are admitted",
                    );
                }
                if !matches!(method.sig.output, syn::ReturnType::Default)
                    && !matches!(&method.sig.output, syn::ReturnType::Type(_, ty) if matches!(ty.as_ref(), syn::Type::Tuple(tuple) if tuple.elems.is_empty()))
                {
                    return self.err(&method.sig.output, "effect helpers must return unit");
                }
                let body = RustExpr::Block(syn::ExprBlock {
                    attrs: Vec::new(),
                    label: None,
                    block: method.block.clone(),
                });
                // A rejection returns from the helper only; its caller runs on.
                let running = flow.running.clone();
                self.effects(
                    &body,
                    state,
                    &mut self.helper_env(method, &call.args, locals)?,
                    checks,
                    flow,
                    definitions,
                    depth + 1,
                    false,
                )?;
                flow.running = running;
            }
            _ if fact => result = Some(self.fact_result(node, locals)?),
            _ => return self.err(node, format!("unsupported effect `{}`", node.to_token_stream())),
        }
        for value in state.values_mut() {
            *value = define(value.clone(), definitions);
        }
        flow.running = define(flow.running.clone(), definitions);
        if fact && result.is_none() {
            return self.err(node, "optional fact handler requires a tail None, Some(F::Variant), or exhaustive if/else");
        }
        Ok(result)
    }
    fn fact_result(&self, node: &RustExpr, locals: &Env) -> Result<Expr> {
        let empty = || default_expr(&Type::Seq(Box::new(self.fact_type.clone())));
        let (path, argument) = match node {
            RustExpr::Path(p) if p.qself.is_none() => (&p.path, None),
            RustExpr::Call(c) => {
                let RustExpr::Path(p) = c.func.as_ref() else {
                    return self.err(node, "unsupported fact constructor");
                };
                if p.qself.is_some() || c.args.len() != 1 {
                    return self.err(node, "Some requires one fact variant");
                }
                (&p.path, c.args.first())
            }
            _ => return self.err(node, "unsupported optional fact tail"),
        };
        let names: Vec<_> = path.segments.iter().map(|s| s.ident.to_string()).collect();
        let name = names.join("::");
        let bare = names.len() == 1;
        let canonical = matches!(
            name.as_str(),
            "None"
                | "Some"
                | "Option::None"
                | "Option::Some"
                | "core::option::Option::None"
                | "core::option::Option::Some"
                | "std::option::Option::None"
                | "std::option::Option::Some"
        );
        let root = &names[0];
        if !canonical
            || (names.len() <= 2 && path.leading_colon.is_some())
            || path.segments.iter().any(|s| !matches!(s.arguments, syn::PathArguments::None))
            || ((bare || path.leading_colon.is_none())
                && (locals.contains_key(root) || self.fact_shadows.contains(root)))
        {
            return self
                .err(node, "fact result requires an unshadowed canonical Option constructor");
        }
        if names.last().is_some_and(|name| name == "None") && argument.is_none() {
            return Ok(empty());
        }
        if names.last().is_none_or(|name| name != "Some") {
            return self.err(node, "None takes no argument");
        }
        let Some(RustExpr::Path(value)) = argument else {
            return self.err(node, "Some requires F::Variant");
        };
        let Type::Enum { name, variants } = &self.fact_type else { unreachable!() };
        if value.qself.is_some()
            || value.path.leading_colon.is_some()
            || value.path.segments.len() != 2
            || value.path.segments.iter().any(|s| !matches!(s.arguments, syn::PathArguments::None))
            || value.path.segments[0].ident != name.as_str()
            || locals.contains_key(name)
        {
            return self.err(value, "Some requires a variant of the declared fact enum");
        }
        let variant = format!("{}::{}", name, value.path.segments[1].ident);
        if !variants.contains(&variant) {
            return self.err(value, "unknown fact variant");
        }
        Ok(Expr {
            ty: empty().ty,
            kind: ExprKind::Sequence(vec![Expr {
                ty: self.fact_type.clone(),
                kind: ExprKind::Enum(variant),
            }]),
        })
    }
    pub(super) fn assigned_field(&self, node: &RustExpr) -> Result<String> {
        if let RustExpr::Field(field) = node
            && matches!(field.base.as_ref(),RustExpr::Path(p)if p.path.is_ident("self"))
        {
            return Ok(field.member.to_token_stream().to_string());
        }
        self.err(node, "assignment must target a declared self field")
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
pub(super) fn default_expr(ty: &Type) -> Expr {
    match ty {
        Type::Bool => Expr::bool(false),
        Type::Int { .. } | Type::Optional(_) => Expr::int(0, ty.clone()),
        Type::Bounded(value) => Expr {
            kind: ExprKind::Bounded(Box::new(Expr::int(value.low, value.backing_type()))),
            ty: ty.clone(),
        },
        Type::Seq(_) => Expr { kind: ExprKind::Sequence(Vec::new()), ty: ty.clone() },
        Type::Enum { variants, .. } => {
            Expr { kind: ExprKind::Enum(variants[0].clone()), ty: ty.clone() }
        }
        _ => Expr::unit(),
    }
}

/// Keep values referenced by later statements as leaves rather than copying their trees.
fn define(value: Expr, definitions: &mut Vec<(String, Expr)>) -> Expr {
    if matches!(
        value.kind,
        ExprKind::Bounded(..)
            | ExprKind::Payload(..)
            | ExprKind::Some(..)
            | ExprKind::OptionalOr(..)
            | ExprKind::Unary(..)
            | ExprKind::Binary(..)
            | ExprKind::If(..)
            | ExprKind::Tuple(..)
            | ExprKind::Sequence(..)
            | ExprKind::Length(..)
            | ExprKind::Append(..)
            | ExprKind::GetOr(..)
            | ExprKind::Checks(..)
    ) {
        let name = definitions.len().to_string();
        let ty = value.ty.clone();
        definitions.push((name.clone(), value));
        Expr { kind: ExprKind::Local(name), ty }
    } else {
        value
    }
}

#[cfg(test)]
mod optional_fact_tests {
    fn source(carrier: &str, result: &str, body: &str) -> syn::File {
        syn::parse_file(&format!(
            r"
            #[machine] mod sample {{
                use umpire_dsl::{{Action, State}};
                #[derive(State)] struct Model {{ active: bool }}
                #[derive(Action)] enum User {{ Step }}
                {carrier}
                impl Model {{
                    #[on(User)] fn user(&mut self, action: User) {result} {{
                        match action {{ User::Step => {{ {body} }} }}
                    }}
                }}
            }}
        "
        ))
        .unwrap_or_else(|error| panic!("{error:?}"))
    }

    #[test]
    fn optional_fact_declarations_and_tails_are_bounded() {
        let carrier = "#[derive(Debug)] enum Fact { A, B }";
        for result in ["Option<Fact>", "core::option::Option<Fact>", "::std::option::Option<Fact>"]
        {
            let ast = source(carrier, &format!("-> {result}"), "None");
            let (model, _, _) = crate::read(&ast, &[]).unwrap_or_else(|error| panic!("{error:?}"));
            assert!(matches!(model.fact_type, crate::Type::Enum { .. }));
            assert_eq!(crate::body::Environment::of(&model).enums["Fact"], model.fact_type);
        }
        for result in [
            "()",
            "Vec<Fact>",
            "Result<Fact, ()>",
            "Option<Option<Fact>>",
            "Option<&Fact>",
            "Option<self::Fact>",
            "other::Option<Fact>",
            "Option<Fact, Fact>",
        ] {
            assert!(
                crate::read(&source(carrier, &format!("-> {result}"), "None"), &[]).is_err(),
                "{result}"
            );
        }
        for carrier in [
            "enum Fact { A }",
            "#[derive(Debug)] enum Fact {}",
            "#[derive(Debug)] enum Fact { A(bool) }",
            "#[derive(Debug)] enum Fact { A = 1 }",
            "#[derive(Debug)] enum Fact<T> { A }",
            "type Fact = bool;",
            "#[derive(Debug)] enum Fact { A } struct Option;",
        ] {
            assert!(
                crate::read(&source(carrier, "-> Option<Fact>", "None"), &[]).is_err(),
                "{carrier}"
            );
        }
        for body in [
            "Some(Fact::A);",
            "if self.active { None }",
            "let saved = None; saved",
            "return Some(Fact::A)",
            "Other::Some(Fact::A)",
            "Some(Fact::Missing)",
            "None(Fact::A)",
            "Some(Fact::A, Fact::B)",
            "let Some = true; Some(Fact::A)",
            "let Option = true; Option::None",
            "let Fact = true; Some(Fact::A)",
        ] {
            assert!(crate::read(&source(carrier, "-> Option<Fact>", body), &[]).is_err(), "{body}");
        }
        for body in [
            "None",
            "Option::None",
            "::core::option::Option::None",
            "Some(Fact::A)",
            "Option::Some(Fact::A)",
            "::std::option::Option::Some(Fact::A)",
            "if self.active { None } else { Some(Fact::B) }",
        ] {
            assert!(crate::read(&source(carrier, "-> Option<Fact>", body), &[]).is_ok(), "{body}");
        }
    }

    #[test]
    fn optional_tail_preserves_prefix_state() {
        let ast = syn::parse_file(
            r"
            #[machine] mod sample {
                use umpire_dsl::{Action, State};
                #[derive(State)] struct Model { active: bool }
                #[derive(Action)] enum User { Step }
                #[derive(Debug)] enum Fact { A, B }
                impl Model {
                    #[on(User)] fn user(&mut self, action: User) -> Option<Fact> {
                        match action { User::Step => {
                            let before = self.active;
                            self.active = !before;
                            if self.active { Some(Fact::A) } else { None }
                        } }
                    }
                }
            }
        ",
        )
        .unwrap_or_else(|error| panic!("{error:?}"));
        let (model, _, _) = crate::read(&ast, &[]).unwrap_or_else(|error| panic!("{error:?}"));
        assert!(matches!(model.fact_type, crate::Type::Enum { .. }));
        assert_eq!(
            model.actions[0].branches[0].facts.ty,
            crate::Type::Seq(Box::new(model.fact_type.clone()))
        );
    }
}
