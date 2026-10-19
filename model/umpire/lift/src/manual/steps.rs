use super::{Manual, path_name, pattern_names, refusal, same_scalar_type};
use crate::tree::{Branch, Expr, Op};
use crate::{Diagnostic, Type};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{Expr as RustExpr, Item, Stmt};

impl Manual<'_> {
    pub(super) fn steps(&mut self, block: &syn::Block) -> Result<(), Diagnostic> {
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
            let mut checks = Vec::new();
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
                if let Some(previous) = updates.insert(name, value) {
                    checks.push(previous);
                }
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
                        checks: checks.clone(),
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
}
