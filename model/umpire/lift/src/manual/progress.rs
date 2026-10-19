use super::{Manual, path_name, refusal, single_statement, tail, text};
use crate::tree::{AssumptionDef, Expr, ProgressDef};
use crate::{Diagnostic, Lifted, Type};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{Expr as RustExpr, Stmt};

impl Manual<'_> {
    pub(super) fn progress(
        &self,
        method: &syn::ImplItemFn,
        result: &mut Lifted,
    ) -> Result<(), Diagnostic> {
        let mut locals = BTreeMap::<String, ProgressDef>::new();
        for statement in &method.block.stmts {
            match statement {
                Stmt::Local(local) => {
                    let syn::Pat::Ident(name) = &local.pat else {
                        return Err(refusal(self.file, local, "unsupported progress binding"));
                    };
                    let mut progress = ProgressDef {
                        name: String::new(),
                        trigger: Expr::bool(true),
                        target: Expr::bool(true),
                        assumption: AssumptionDef::default(),
                        bound: None,
                    };
                    self.progress_builder(
                        &local
                            .init
                            .as_ref()
                            .ok_or_else(|| {
                                refusal(self.file, local, "uninitialized progress binding")
                            })?
                            .expr,
                        &mut progress,
                    )?;
                    locals.insert(name.ident.to_string(), progress);
                }
                Stmt::Expr(RustExpr::Assign(assign), _) => {
                    let name = path_name(&assign.left).ok_or_else(|| {
                        refusal(self.file, assign, "unsupported progress assignment")
                    })?;
                    let progress = locals
                        .get_mut(&name)
                        .ok_or_else(|| refusal(self.file, assign, "unknown progress binding"))?;
                    self.progress_builder(self.select(&assign.right)?, progress)?;
                }
                Stmt::Expr(RustExpr::Macro(expression), None)
                    if expression.mac.path.is_ident("vec") => {}
                _ => {
                    return Err(refusal(
                        self.file,
                        statement,
                        "unsupported progress registration statement",
                    ));
                }
            }
        }
        for expression in self.vector(
            tail(&method.block)
                .ok_or_else(|| refusal(self.file, method, "empty progress registration"))?,
        )? {
            if let Some(progress) = path_name(&expression).and_then(|name| locals.get(&name)) {
                result.progress.push(progress.clone());
                continue;
            }
            let mut progress = ProgressDef {
                name: String::new(),
                trigger: Expr::bool(true),
                target: Expr::bool(true),
                assumption: AssumptionDef::default(),
                bound: None,
            };
            self.progress_builder(&expression, &mut progress)?;
            result.progress.push(progress);
        }
        Ok(())
    }

    fn progress_builder(
        &self,
        expression: &RustExpr,
        progress: &mut ProgressDef,
    ) -> Result<(), Diagnostic> {
        match expression {
            RustExpr::Path(_) => Ok(()),
            RustExpr::Block(block) => {
                single_statement(self.file, &block.block)?;
                self.progress_builder(
                    tail(&block.block)
                        .ok_or_else(|| refusal(self.file, block, "empty progress builder"))?,
                    progress,
                )
            }
            RustExpr::Call(expression) => {
                let constructor = expression.func.to_token_stream().to_string().replace(' ', "");
                if !constructor.starts_with("Progress::<") || !constructor.ends_with("::new") {
                    return Err(refusal(self.file, expression, "unsupported progress constructor"));
                }
                let name = expression.args.first().and_then(text).ok_or_else(|| {
                    refusal(self.file, expression, "progress name must be a string")
                })?;
                *progress = ProgressDef {
                    name,
                    trigger: Expr::bool(true),
                    target: Expr::bool(true),
                    assumption: AssumptionDef::default(),
                    bound: None,
                };
                Ok(())
            }
            RustExpr::MethodCall(expression) => {
                self.progress_builder(&expression.receiver, progress)?;
                let name = expression.method.to_string();
                if name == "whenever" || name == "eventually" {
                    let Some(RustExpr::Closure(closure)) = expression.args.first() else {
                        return Err(refusal(
                            self.file,
                            expression,
                            "progress predicate must be a closure",
                        ));
                    };
                    let aliases = closure
                        .inputs
                        .iter()
                        .enumerate()
                        .filter_map(|(index, parameter)| {
                            if let syn::Pat::Ident(parameter) = parameter {
                                Some((
                                    parameter.ident.to_string(),
                                    if index == 0 { "world" } else { "state" }.into(),
                                ))
                            } else {
                                None
                            }
                        })
                        .collect();
                    let value = self.expr(&closure.body, &aliases)?;
                    if value.ty != Type::Bool {
                        return Err(refusal(
                            self.file,
                            closure,
                            "progress predicate must be Boolean",
                        ));
                    }
                    if name == "whenever" {
                        progress.trigger = value;
                    } else {
                        progress.target = value;
                    }
                    Ok(())
                } else if name == "within" || name == "assuming" {
                    progress.assumption = AssumptionDef::default();
                    self.assumption(
                        expression
                            .args
                            .first()
                            .ok_or_else(|| refusal(self.file, expression, "missing assumption"))?,
                        &mut progress.assumption,
                    )
                } else {
                    Err(refusal(self.file, expression, "unsupported manual progress assumption"))
                }
            }
            _ => Err(refusal(self.file, expression, "unsupported progress builder")),
        }
    }

    fn assumption(
        &self,
        expression: &RustExpr,
        assumption: &mut AssumptionDef,
    ) -> Result<(), Diagnostic> {
        let (name, argument) = match expression {
            RustExpr::Call(call) => (
                path_name(&call.func)
                    .unwrap_or_default()
                    .rsplit("::")
                    .next()
                    .unwrap_or_default()
                    .to_string(),
                call.args.first(),
            ),
            RustExpr::MethodCall(call) => {
                self.assumption(&call.receiver, assumption)?;
                (call.method.to_string(), call.args.first())
            }
            _ => return Err(refusal(self.file, expression, "unsupported assumption constructor")),
        };
        let actions = self
            .vector(
                argument
                    .ok_or_else(|| refusal(self.file, expression, "assumption requires actions"))?,
            )?
            .iter()
            .map(|action| {
                path_name(action).ok_or_else(|| {
                    refusal(self.file, action, "assumption action must be a variant")
                })
            })
            .collect::<Result<Vec<_>, _>>()?;
        match name.as_str() {
            "only" => assumption.only = Some(actions),
            "without" => assumption.without.extend(actions),
            "fair" => assumption.fair.extend(actions),
            _ => return Err(refusal(self.file, expression, "unsupported assumption operation")),
        }
        Ok(())
    }
}
