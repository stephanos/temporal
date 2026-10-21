//! Workflow service classification shared by descriptor admission and code generation.

use quote::ToTokens;
use syn::{Expr, ExprCall, FnArg, ItemFn, Pat, Stmt, Type, spanned::Spanned, visit::Visit};

use crate::descriptor::{Disposition, MethodCoverage, ServiceCoverage};

/// A source-reviewed classification and its executable reverse bindings.
#[derive(Clone, Debug)]
pub struct Classification {
    /// Descriptor obligations, retaining match-arm and or-pattern order.
    pub coverage: ServiceCoverage,
    /// Model actions bound by the classification.
    pub actions: Vec<ActionBinding>,
    /// Canonical RPC names, role names and explanatory targets for reporting.
    pub bindings: Vec<(String, String, String)>,
}

/// One RPC and the Model action expression it classifies.
#[derive(Clone, Debug)]
pub struct ActionBinding {
    /// Original protobuf method name.
    pub method: String,
    /// Nested Model action constructor, reusable as an exhaustive match pattern.
    pub action: Expr,
}

/// Read ordinary `classify` functions over the generated workflow method enum.
pub fn classifications(source: &str) -> syn::Result<Vec<Classification>> {
    struct Classifications {
        result: syn::Result<Vec<Classification>>,
    }
    impl<'ast> Visit<'ast> for Classifications {
        fn visit_item_fn(&mut self, function: &'ast ItemFn) {
            if function.sig.ident == "classify"
                && function.sig.inputs.iter().any(|argument| matches!(argument,
                    FnArg::Typed(argument) if matches!(argument.ty.as_ref(), Type::Path(ty)
                        if ty.path.segments.last().is_some_and(|segment| segment.ident == "WorkflowServiceMethod"))))
            {
                match (&mut self.result, classification(function)) {
                    (Ok(classifications), Ok(classification)) => classifications.push(classification),
                    (result @ Ok(_), Err(error)) => *result = Err(error),
                    _ => {}
                }
            }
            syn::visit::visit_item_fn(self, function);
        }
    }
    let file = syn::parse_file(source)?;
    let mut visitor = Classifications { result: Ok(Vec::new()) };
    visitor.visit_file(&file);
    visitor.result
}

fn classification(function: &ItemFn) -> syn::Result<Classification> {
    let [FnArg::Typed(argument)] = function.sig.inputs.iter().collect::<Vec<_>>().as_slice() else {
        return Err(syn::Error::new(
            function.sig.span(),
            "classification requires one method argument",
        ));
    };
    let Pat::Ident(parameter) = argument.pat.as_ref() else {
        return Err(syn::Error::new(
            argument.span(),
            "classification requires a named method argument",
        ));
    };
    let [Stmt::Expr(Expr::Match(expression), None)] = function.block.stmts.as_slice() else {
        return Err(syn::Error::new(
            function.block.span(),
            "classification must be one exhaustive match",
        ));
    };
    if !matches!(expression.expr.as_ref(), Expr::Path(path) if path.path.is_ident(&parameter.ident))
    {
        return Err(syn::Error::new(
            expression.expr.span(),
            "classification must match its method argument",
        ));
    }
    let mut result = Classification {
        coverage: ServiceCoverage {
            name: "temporal.api.workflowservice.v1.WorkflowService".into(),
            methods: Vec::new(),
        },
        actions: Vec::new(),
        bindings: Vec::new(),
    };
    for arm in &expression.arms {
        if !arm.attrs.is_empty() {
            return Err(syn::Error::new(
                arm.span(),
                "classification arms cannot have guards or attributes",
            ));
        }
        let methods = method_names(&arm.pat)?;
        let Expr::Call(role) = classification_expression(&arm.body)? else {
            return Err(syn::Error::new(arm.body.span(), "expected RpcRole classification"));
        };
        let kind = constructor(role, "RpcRole")?;
        let value = &role.args[0];
        let disposition = match kind.as_str() {
            "Action" => {
                let Expr::Call(action) = value else {
                    return Err(syn::Error::new(
                        value.span(),
                        "expected nested ModelAction constructor",
                    ));
                };
                constructor(action, "ModelAction")?;
                let Expr::Path(leaf) = &action.args[0] else {
                    return Err(syn::Error::new(action.span(), "expected actor action variant"));
                };
                if leaf.path.segments.len() < 2 {
                    return Err(syn::Error::new(leaf.span(), "expected actor action variant"));
                }
                for method in &methods {
                    result
                        .actions
                        .push(ActionBinding { method: method.clone(), action: value.clone() });
                }
                Disposition::Action(leaf.path.to_token_stream().to_string().replace(' ', ""))
            }
            "Observed" => {
                reason(value)?;
                Disposition::Observed
            }
            "OutOfScope" => Disposition::OutOfScope(reason(value)?),
            _ => return Err(syn::Error::new(role.span(), "unknown RpcRole classification")),
        };
        for name in methods {
            let target = match &disposition {
                Disposition::Action(action) => action.clone(),
                Disposition::Observed | Disposition::OutOfScope(_) => reason(value)?,
            };
            result.bindings.push((
                format!("{}.{}", result.coverage.name, name),
                kind.clone(),
                target,
            ));
            result.coverage.methods.push(MethodCoverage { name, disposition: disposition.clone() });
        }
    }
    Ok(result)
}

fn classification_expression(mut expression: &Expr) -> syn::Result<&Expr> {
    while let Expr::Block(block) = expression {
        let [Stmt::Expr(value, None)] = block.block.stmts.as_slice() else {
            return Err(syn::Error::new(
                block.span(),
                "classification block must contain one expression without statements",
            ));
        };
        expression = value;
    }
    Ok(expression)
}

fn constructor(call: &ExprCall, owner: &str) -> syn::Result<String> {
    let Expr::Path(path) = call.func.as_ref() else {
        return Err(syn::Error::new(call.span(), format!("expected {owner} constructor")));
    };
    let segments: Vec<_> = path.path.segments.iter().collect();
    if segments.len() != 2 || segments[0].ident != owner || call.args.len() != 1 {
        return Err(syn::Error::new(
            call.span(),
            format!("expected {owner} constructor with one argument"),
        ));
    }
    Ok(segments[1].ident.to_string())
}

fn reason(expression: &Expr) -> syn::Result<String> {
    if let Expr::Lit(literal) = expression
        && let syn::Lit::Str(reason) = &literal.lit
        && !reason.value().trim().is_empty()
    {
        return Ok(reason.value());
    }
    Err(syn::Error::new(expression.span(), "classification requires a nonempty literal reason"))
}

fn method_names(pattern: &Pat) -> syn::Result<Vec<String>> {
    if let Pat::Or(pattern) = pattern {
        return pattern
            .cases
            .iter()
            .map(method_names)
            .collect::<syn::Result<Vec<_>>>()
            .map(|groups| groups.into_iter().flatten().collect());
    }
    if let Pat::Path(path) = pattern {
        let segments: Vec<_> = path.path.segments.iter().collect();
        if segments.len() == 2 && segments[0].ident == "WorkflowServiceMethod" {
            return Ok(vec![segments[1].ident.to_string()]);
        }
    }
    Err(syn::Error::new(
        pattern.span(),
        "expected explicit WorkflowServiceMethod variant; wildcards are prohibited",
    ))
}

#[cfg(test)]
mod tests {
    use super::classifications;

    #[test]
    fn ordinary_match_projects_source_order_and_model_action() -> syn::Result<()> {
        let parsed = classifications(
            r#"fn classify(method: WorkflowServiceMethod) -> RpcRole {
                match method {
                    WorkflowServiceMethod::Poll => RpcRole::Action(ModelAction::Worker(Worker::Poll)),
                    WorkflowServiceMethod::Describe => RpcRole::Observed("observation"),
                    WorkflowServiceMethod::First | WorkflowServiceMethod::Second => RpcRole::OutOfScope("workflow"),
                }
            }"#,
        )?;
        assert_eq!(parsed.len(), 1);
        assert_eq!(parsed[0].coverage.name, "temporal.api.workflowservice.v1.WorkflowService");
        assert_eq!(
            parsed[0]
                .coverage
                .methods
                .iter()
                .map(|method| method.name.as_str())
                .collect::<Vec<_>>(),
            ["Poll", "Describe", "First", "Second"]
        );
        assert_eq!(parsed[0].actions[0].method, "Poll");
        Ok(())
    }

    #[test]
    fn classification_rejects_non_declarative_or_unexplained_arms() {
        for arm in [
            "_ => RpcRole::Observed(\"reason\")",
            "WorkflowServiceMethod::Poll if true => RpcRole::Observed(\"reason\")",
            "WorkflowServiceMethod::Poll => RpcRole::OutOfScope(\"\")",
            "WorkflowServiceMethod::Poll => RpcRole::Observed(\"\")",
            "WorkflowServiceMethod::Poll => RpcRole::Action(other())",
            "WorkflowServiceMethod::Poll => { log(); RpcRole::Observed(\"reason\") }",
            "WorkflowServiceMethod::Poll => { let role = RpcRole::Observed(\"reason\"); role }",
        ] {
            let source = format!(
                "fn classify(method: WorkflowServiceMethod) -> RpcRole {{ match method {{ {arm} }} }}"
            );
            assert!(classifications(&source).is_err(), "accepted {arm}");
        }
    }

    #[test]
    fn classification_accepts_single_expression_arm_blocks() -> syn::Result<()> {
        let parsed = classifications(
            r#"fn classify(method: WorkflowServiceMethod) -> RpcRole {
                match method {
                    WorkflowServiceMethod::Poll => { RpcRole::Action(ModelAction::Worker(Worker::Poll)) }
                    WorkflowServiceMethod::Describe => { RpcRole::Observed("observation") }
                    WorkflowServiceMethod::Other => { RpcRole::OutOfScope("workflow") }
                }
            }"#,
        )?;
        assert_eq!(parsed[0].coverage.methods.len(), 3);
        assert_eq!(parsed[0].actions[0].method, "Poll");
        assert_eq!(parsed[0].bindings[2].2, "workflow");
        Ok(())
    }

    #[test]
    fn unrelated_matches_are_not_service_declarations() -> syn::Result<()> {
        assert!(
            classifications(
                "fn classify(value: bool) -> bool { match value { true => false, false => true } }"
            )?
            .is_empty()
        );
        Ok(())
    }
}
