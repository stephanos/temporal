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
            self.effects(
                &body,
                &mut state,
                &mut locals,
                &mut checks,
                &mut flow,
                &mut definitions,
                0,
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
                facts: default_expr(&Type::Seq(Box::new(self.fact_type.clone()))),
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
    ) -> Result<()> {
        if depth > 32 {
            return self.err(node, "recursive effects are unsupported");
        }
        match node {
            RustExpr::Block(block) => {
                let mut scoped = locals.clone();
                for statement in &block.block.stmts {
                    if flow.returned() {
                        break;
                    }
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
                            checks.push(flow.guard(value.clone(), default_expr(&value.ty)));
                            scoped.insert(id.ident.to_string(), define(value, definitions));
                        }
                        Stmt::Expr(expr, _) => {
                            self.effects(
                                expr,
                                state,
                                &mut scoped,
                                checks,
                                flow,
                                definitions,
                                depth,
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
                self.effects(
                    &yes_expr,
                    &mut yes,
                    &mut locals.clone(),
                    &mut yc,
                    &mut yf,
                    definitions,
                    depth,
                )?;
                if let Some((_, other)) = &condition.else_branch {
                    self.effects(
                        other,
                        &mut no,
                        &mut locals.clone(),
                        &mut nc,
                        &mut nf,
                        definitions,
                        depth,
                    )?;
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
                )?;
                flow.running = running;
            }
            _ => return self.err(node, format!("unsupported effect `{}`", node.to_token_stream())),
        }
        for value in state.values_mut() {
            *value = define(value.clone(), definitions);
        }
        flow.running = define(flow.running.clone(), definitions);
        Ok(())
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
        Type::Int { .. } => Expr::int(0, ty.clone()),
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
        ExprKind::Unary(..)
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
