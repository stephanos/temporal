//! Workflow service classification shared by descriptor admission and code generation.

use std::collections::BTreeMap;

use proc_macro2::Span;
use quote::{ToTokens, quote};
use syn::{
    Expr, ExprCall, FnArg, ItemFn, Pat, PatIdent, Stmt, Type, parse::Parser, spanned::Spanned,
    visit::Visit,
};

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
    /// Original nested Model action constructor, including representative input values.
    pub action: Expr,
    /// Exhaustive reverse binding, independent of representative payload values.
    pub pattern: Pat,
    /// Nested carrier and actor variant identity, independent of payloads.
    pub identity: String,
}

/// Every RPC of a service and its resource family, in match-arm order.
#[derive(Clone, Debug)]
pub struct ServiceTable {
    /// Original protobuf method names and their family reasons.
    pub families: Vec<(String, String)>,
}

/// Service declarations read from catalog sources.
#[derive(Clone, Debug, Default)]
pub struct Declarations {
    /// Exhaustive `classify` matches that classify the whole service by themselves.
    pub classifications: Vec<Classification>,
    /// Exhaustive `family` matches giving every RPC its resource family.
    pub tables: Vec<ServiceTable>,
    /// `classify` matches that claim some RPCs and defer the rest to `family`.
    pub claims: Vec<Classification>,
}

impl Declarations {
    /// Add the declarations of another source.
    pub fn extend(&mut self, other: Self) {
        self.classifications.extend(other.classifications);
        self.tables.extend(other.tables);
        self.claims.extend(other.claims);
    }

    /// Combine the service table with every feature's claims, in table order.
    ///
    /// # Errors
    /// Refuses claims without a table, a second table or complete classification beside a
    /// table, a duplicate feature claim and a claimed RPC the table does not classify.
    pub fn resolve(self, service: &str) -> Result<Vec<Classification>, String> {
        let Self { mut classifications, tables, claims } = self;
        let table = match tables.as_slice() {
            [] if claims.is_empty() => return Ok(classifications),
            [] => return Err(format!("feature claims without a service table: {service}")),
            [table] if classifications.is_empty() => table,
            [_] => {
                return Err(format!(
                    "complete classification duplicates the service table: {service}"
                ));
            }
            _ => return Err(format!("duplicate service table: {service}")),
        };
        let mut claimed = BTreeMap::new();
        for claim in &claims {
            for (index, method) in claim.coverage.methods.iter().enumerate() {
                if claimed.insert(method.name.as_str(), (claim, index)).is_some() {
                    return Err(format!("duplicate feature claim: {service}.{}", method.name));
                }
                if !table.families.iter().any(|(name, _)| *name == method.name) {
                    return Err(format!(
                        "unclassified RPC: {service}.{} is claimed but absent from the service \
                         table",
                        method.name
                    ));
                }
            }
        }
        let mut result = Classification {
            coverage: ServiceCoverage { name: service.into(), methods: Vec::new() },
            actions: Vec::new(),
            bindings: Vec::new(),
        };
        for (method, family) in &table.families {
            if let Some((claim, index)) = claimed.get(method.as_str()) {
                result.coverage.methods.push(claim.coverage.methods[*index].clone());
                result.bindings.push(claim.bindings[*index].clone());
                result.actions.extend(
                    claim.actions.iter().filter(|action| action.method == *method).cloned(),
                );
            } else {
                result.coverage.methods.push(MethodCoverage {
                    name: method.clone(),
                    disposition: Disposition::OutOfScope(family.clone()),
                });
                result.bindings.push((
                    format!("{service}.{method}"),
                    "OutOfScope".into(),
                    family.clone(),
                ));
            }
        }
        classifications.push(result);
        Ok(classifications)
    }
}

/// Read and resolve the service declarations of one source.
///
/// # Errors
/// Refuses malformed declarations and any [`Declarations::resolve`] refusal.
pub fn classifications(
    source: &str,
    service: &str,
    method_enum: &str,
) -> syn::Result<Vec<Classification>> {
    declarations(source, service, method_enum)?
        .resolve(service)
        .map_err(|error| syn::Error::new(Span::call_site(), error))
}

/// Read ordinary `classify` and `family` functions over the generated method enum.
///
/// # Errors
/// Refuses unparsable sources and malformed declarations.
pub fn declarations(source: &str, service: &str, method_enum: &str) -> syn::Result<Declarations> {
    struct Visitor<'a> {
        service: &'a str,
        method_enum: &'a str,
        result: syn::Result<Declarations>,
    }
    impl Visitor<'_> {
        fn declaration(&self, function: &ItemFn) -> syn::Result<Declarations> {
            let mut declarations = Declarations::default();
            if function.sig.ident == "family" {
                declarations.tables.push(table(function, self.method_enum)?);
            } else {
                let (classification, claims) =
                    classification(function, self.service, self.method_enum)?;
                if claims {
                    declarations.claims.push(classification);
                } else {
                    declarations.classifications.push(classification);
                }
            }
            Ok(declarations)
        }
    }
    impl<'ast> Visit<'ast> for Visitor<'_> {
        fn visit_item_fn(&mut self, function: &'ast ItemFn) {
            if (function.sig.ident == "classify" || function.sig.ident == "family")
                && function.sig.inputs.iter().any(|argument| matches!(argument,
                    FnArg::Typed(argument) if matches!(argument.ty.as_ref(), Type::Path(ty)
                        if ty.path.segments.last().is_some_and(|segment| segment.ident == self.method_enum))))
            {
                let declaration = self.declaration(function);
                match (&mut self.result, declaration) {
                    (Ok(declarations), Ok(declaration)) => declarations.extend(declaration),
                    (result @ Ok(_), Err(error)) => *result = Err(error),
                    _ => {}
                }
            }
            syn::visit::visit_item_fn(self, function);
        }
    }
    let file = syn::parse_file(source)?;
    let mut visitor = Visitor { service, method_enum, result: Ok(Declarations::default()) };
    visitor.visit_file(&file);
    visitor.result
}

fn method_match(function: &ItemFn) -> syn::Result<(&PatIdent, &syn::ExprMatch)> {
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
    for arm in &expression.arms {
        if !arm.attrs.is_empty() {
            return Err(syn::Error::new(
                arm.span(),
                "classification arms cannot have guards or attributes",
            ));
        }
    }
    Ok((parameter, expression))
}

fn table(function: &ItemFn, method_enum: &str) -> syn::Result<ServiceTable> {
    let (_, expression) = method_match(function)?;
    let mut families = Vec::new();
    for arm in &expression.arms {
        let family = reason(classification_expression(&arm.body)?)?;
        for method in method_names(&arm.pat, method_enum)? {
            families.push((method, family.clone()));
        }
    }
    Ok(ServiceTable { families })
}

/// Whether `body` is `RpcRole::OutOfScope(family(method))`, deferring to the service table.
fn defers_to_table(body: &Expr, parameter: &PatIdent) -> bool {
    let Ok(Expr::Call(role)) = classification_expression(body) else {
        return false;
    };
    let Ok(kind) = constructor(role, "RpcRole") else {
        return false;
    };
    let Expr::Call(family) = &role.args[0] else {
        return false;
    };
    kind == "OutOfScope"
        && matches!(family.func.as_ref(), Expr::Path(path)
            if path.path.segments.last().is_some_and(|segment| segment.ident == "family"))
        && matches!(family.args.iter().collect::<Vec<_>>().as_slice(),
            [Expr::Path(argument)] if argument.path.is_ident(&parameter.ident))
}

/// Read one `classify` match and whether it claims RPCs for a service table.
fn classification(
    function: &ItemFn,
    service: &str,
    method_enum: &str,
) -> syn::Result<(Classification, bool)> {
    let (parameter, expression) = method_match(function)?;
    let mut claims = false;
    let mut result = Classification {
        coverage: ServiceCoverage { name: service.into(), methods: Vec::new() },
        actions: Vec::new(),
        bindings: Vec::new(),
    };
    for (index, arm) in expression.arms.iter().enumerate() {
        if matches!(arm.pat, Pat::Wild(_)) && index + 1 == expression.arms.len() {
            if !defers_to_table(&arm.body, parameter) {
                return Err(syn::Error::new(
                    arm.span(),
                    "a claim's wildcard arm must be `_ => RpcRole::OutOfScope(family(method))`",
                ));
            }
            claims = true;
            continue;
        }
        let methods = method_names(&arm.pat, method_enum)?;
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
                let (leaf, payload) = actor_variant(&action.args[0])?;
                let carrier = &action.func;
                let pattern: Pat = Pat::parse_single.parse2(quote!(#carrier(#leaf #payload)))?;
                let identity = quote!(#carrier(#leaf)).to_string();
                for method in &methods {
                    result.actions.push(ActionBinding {
                        method: method.clone(),
                        action: value.clone(),
                        pattern: pattern.clone(),
                        identity: identity.clone(),
                    });
                }
                Disposition::Action(leaf.to_token_stream().to_string().replace(' ', ""))
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
    Ok((result, claims))
}

fn actor_variant(expression: &Expr) -> syn::Result<(&syn::Path, proc_macro2::TokenStream)> {
    let (path, payload) = match expression {
        Expr::Path(leaf) => (&leaf.path, quote!()),
        Expr::Call(tuple) => {
            let Expr::Path(leaf) = tuple.func.as_ref() else {
                return Err(syn::Error::new(tuple.span(), "expected actor action variant"));
            };
            (&leaf.path, quote!((..)))
        }
        Expr::Struct(record) => (&record.path, quote!({ .. })),
        other => return Err(syn::Error::new(other.span(), "expected actor action variant")),
    };
    if path.segments.len() < 2 {
        return Err(syn::Error::new(path.span(), "expected actor action variant"));
    }
    Ok((path, payload))
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

fn method_names(pattern: &Pat, method_enum: &str) -> syn::Result<Vec<String>> {
    if let Pat::Or(pattern) = pattern {
        return pattern
            .cases
            .iter()
            .map(|pattern| method_names(pattern, method_enum))
            .collect::<syn::Result<Vec<_>>>()
            .map(|groups| groups.into_iter().flatten().collect());
    }
    if let Pat::Path(path) = pattern {
        let segments: Vec<_> = path.path.segments.iter().collect();
        if segments.len() == 2 && segments[0].ident == method_enum {
            return Ok(vec![segments[1].ident.to_string()]);
        }
    }
    Err(syn::Error::new(
        pattern.span(),
        format!("expected explicit {method_enum} variant; wildcards are prohibited"),
    ))
}

#[cfg(test)]
mod tests {
    fn classifications(source: &str) -> syn::Result<Vec<super::Classification>> {
        super::classifications(source, "example.v1.ExampleService", "ExampleServiceMethod")
    }

    #[test]
    fn ordinary_match_projects_source_order_and_model_action() -> syn::Result<()> {
        let parsed = classifications(
            r#"fn classify(method: ExampleServiceMethod) -> RpcRole {
                match method {
                    ExampleServiceMethod::Poll => RpcRole::Action(ModelAction::Worker(Worker::Poll)),
                    ExampleServiceMethod::Describe => RpcRole::Observed("observation"),
                    ExampleServiceMethod::First | ExampleServiceMethod::Second => RpcRole::OutOfScope("example"),
                }
            }"#,
        )?;
        assert_eq!(parsed.len(), 1);
        assert_eq!(parsed[0].coverage.name, "example.v1.ExampleService");
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
            "ExampleServiceMethod::Poll if true => RpcRole::Observed(\"reason\")",
            "ExampleServiceMethod::Poll => RpcRole::OutOfScope(\"\")",
            "ExampleServiceMethod::Poll => RpcRole::Observed(\"\")",
            "ExampleServiceMethod::Poll => RpcRole::Action(other())",
            "ExampleServiceMethod::Poll => { log(); RpcRole::Observed(\"reason\") }",
            "ExampleServiceMethod::Poll => { let role = RpcRole::Observed(\"reason\"); role }",
        ] {
            let source = format!(
                "fn classify(method: ExampleServiceMethod) -> RpcRole {{ match method {{ {arm} }} }}"
            );
            assert!(classifications(&source).is_err(), "accepted {arm}");
        }
    }

    #[test]
    fn classification_accepts_single_expression_arm_blocks() -> syn::Result<()> {
        let parsed = classifications(
            r#"fn classify(method: ExampleServiceMethod) -> RpcRole {
                match method {
                    ExampleServiceMethod::Poll => { RpcRole::Action(ModelAction::Worker(Worker::Poll)) }
                    ExampleServiceMethod::Describe => { RpcRole::Observed("observation") }
                    ExampleServiceMethod::Other => { RpcRole::OutOfScope("example") }
                }
            }"#,
        )?;
        assert_eq!(parsed[0].coverage.methods.len(), 3);
        assert_eq!(parsed[0].actions[0].method, "Poll");
        assert_eq!(parsed[0].bindings[2].2, "example");
        Ok(())
    }

    const TABLE: &str = r#"fn family(method: ExampleServiceMethod) -> &'static str {
        match method {
            ExampleServiceMethod::Poll | ExampleServiceMethod::Describe => "worker tasks",
            ExampleServiceMethod::Other => { "administration" }
        }
    }"#;

    fn declarations(source: &str) -> syn::Result<super::Declarations> {
        super::declarations(source, "example.v1.ExampleService", "ExampleServiceMethod")
    }

    #[test]
    fn claims_resolve_against_the_service_table_in_table_order() -> syn::Result<()> {
        let claim = r#"fn classify(method: ExampleServiceMethod) -> RpcRole {
            match method {
                ExampleServiceMethod::Describe => RpcRole::Observed("observation"),
                ExampleServiceMethod::Poll => RpcRole::Action(ModelAction::Worker(Worker::Poll)),
                _ => RpcRole::OutOfScope(service::family(method)),
            }
        }"#;
        let parsed = classifications(&format!("{TABLE}{claim}"))?;
        assert_eq!(parsed.len(), 1);
        assert_eq!(
            parsed[0]
                .bindings
                .iter()
                .map(|(rpc, role, target)| format!("{rpc} {role} {target}"))
                .collect::<Vec<_>>(),
            [
                "example.v1.ExampleService.Poll Action Worker::Poll",
                "example.v1.ExampleService.Describe Observed observation",
                "example.v1.ExampleService.Other OutOfScope administration",
            ]
        );
        assert_eq!(parsed[0].actions[0].method, "Poll");
        Ok(())
    }

    #[test]
    fn duplicate_or_unclassified_claims_are_named() -> syn::Result<()> {
        let claim = |method: &str| {
            format!(
                "fn classify(method: ExampleServiceMethod) -> RpcRole {{ match method {{
                    ExampleServiceMethod::{method} => RpcRole::Observed(\"observation\"),
                    _ => RpcRole::OutOfScope(family(method)),
                }} }}"
            )
        };
        let mut duplicate = declarations(TABLE)?;
        duplicate.extend(declarations(&claim("Poll"))?);
        duplicate.extend(declarations(&claim("Poll"))?);
        assert_eq!(
            duplicate.resolve("example.v1.ExampleService").err().as_deref(),
            Some("duplicate feature claim: example.v1.ExampleService.Poll")
        );
        let mut unclassified = declarations(TABLE)?;
        unclassified.extend(declarations(&claim("Added"))?);
        assert!(unclassified.resolve("example.v1.ExampleService").is_err_and(|error| {
            error.starts_with("unclassified RPC: example.v1.ExampleService.Added")
        }));
        assert!(declarations(&claim("Poll"))?.resolve("example.v1.ExampleService").is_err());
        let mut tables = declarations(TABLE)?;
        tables.extend(declarations(TABLE)?);
        assert!(tables.resolve("example.v1.ExampleService").is_err());
        Ok(())
    }

    #[test]
    fn tables_and_claim_fallbacks_are_declarative() {
        for source in [
            "fn family(method: ExampleServiceMethod) -> &'static str { match method { _ => \"all\" } }",
            "fn family(method: ExampleServiceMethod) -> &'static str { match method { ExampleServiceMethod::Poll => \"\" } }",
            "fn family(method: ExampleServiceMethod) -> &'static str { match method { ExampleServiceMethod::Poll => name() } }",
            "fn classify(method: ExampleServiceMethod) -> RpcRole { match method { _ => RpcRole::OutOfScope(\"all\") } }",
            "fn classify(method: ExampleServiceMethod) -> RpcRole { match method { _ => RpcRole::Observed(family(method)) } }",
            "fn classify(method: ExampleServiceMethod) -> RpcRole { match method { _ => RpcRole::OutOfScope(family(other)) } }",
            "fn classify(method: ExampleServiceMethod) -> RpcRole { match method { _ => RpcRole::OutOfScope(family(method)), ExampleServiceMethod::Poll => RpcRole::Observed(\"x\") } }",
        ] {
            assert!(declarations(source).is_err(), "accepted {source}");
        }
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
