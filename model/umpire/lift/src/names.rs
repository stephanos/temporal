use crate::{Diagnostic, tree::Lifted};
use proc_macro2::Span;
use quote::ToTokens;
use std::collections::BTreeSet;

const RESERVED_NAMES: &[&str] = &[
    "vars",
    "ModelVars",
    "Init",
    "ModelNext",
    "ModelEdge",
    "Edge",
    "Complete",
    "Next",
    "Spec",
    "__umpire_Div",
    "__umpire_a",
    "__umpire_b",
    "__umpire_d",
    "__umpire_n",
    "__umpire_q",
    "__umpire_r",
    "__umpire_negative",
    "__umpire_edge_tick",
    "__umpire_pending",
    "__umpire_steps",
    "__umpire_action",
    "__umpire_choice",
    "__umpire_outcome",
    "__umpire_facts",
    "__umpire_input",
    "TRUE",
    "FALSE",
    "BOOLEAN",
    "Int",
    "Nat",
    "Assert",
    "Seq",
    "Len",
    "Append",
    "Head",
    "Tail",
    "SubSeq",
    "SelectSeq",
    "DOMAIN",
    "IF",
    "THEN",
    "ELSE",
    "CASE",
    "OTHER",
    "LET",
    "IN",
    "UNCHANGED",
    "ENABLED",
    "SUBSET",
    "UNION",
    "CHOOSE",
    "ASSUME",
    "THEOREM",
    "VARIABLE",
    "VARIABLES",
    "CONSTANT",
    "CONSTANTS",
    "MODULE",
    "INSTANCE",
    "LOCAL",
    "EXTENDS",
];
pub(crate) fn allocate(ast: &syn::File, base: &str, generated: &mut BTreeSet<String>) -> String {
    struct Occupied<'a>(&'a mut BTreeSet<String>);
    impl<'ast> syn::visit::Visit<'ast> for Occupied<'_> {
        fn visit_ident(&mut self, ident: &'ast syn::Ident) {
            self.0.insert(ident.to_string());
        }
        fn visit_item_enum(&mut self, value: &'ast syn::ItemEnum) {
            for variant in &value.variants {
                self.0.insert(format!("{}_{}", value.ident, variant.ident));
            }
            syn::visit::visit_item_enum(self, value);
        }
        fn visit_impl_item_fn(&mut self, value: &'ast syn::ImplItemFn) {
            for prefix in [
                "Invariant",
                "Post",
                "Negative",
                "Assumption",
                "Scoped",
                "Activate",
                "Monitor",
                "ActiveEdge",
                "PrefixEdge",
                "Complete",
                "ProgressNext",
                "ProgressSpec",
                "Maximal",
                "Progress",
            ] {
                self.0.insert(format!("{prefix}_{}", value.sig.ident));
            }
            syn::visit::visit_impl_item_fn(self, value);
        }
    }
    let mut occupied: BTreeSet<_> = RESERVED_NAMES.iter().map(|name| (*name).to_owned()).collect();
    syn::visit::Visit::visit_file(&mut Occupied(&mut occupied), ast);
    occupied.extend(generated.iter().cloned());
    let mut name = base.to_owned();
    let mut suffix = 0;
    while occupied.contains(&name) {
        suffix += 1;
        name = format!("{base}_{suffix}");
    }
    generated.insert(name.clone());
    name
}
pub(crate) fn validate(file: &str, ast: &syn::File, model: &Lifted) -> Result<(), Diagnostic> {
    validate_text(file, ast.to_token_stream())?;
    let mut names: BTreeSet<String> = RESERVED_NAMES.iter().map(|name| (*name).into()).collect();
    // Input symbols are scoped to one variant's relation and may repeat across variants.
    let mut symbols = BTreeSet::new();
    for action in &model.actions {
        let variant = action.name.rsplit("::").next().unwrap_or(&action.name);
        register(file, ast, &mut names, &action.operator, variant)?;
        if !action.fields.is_empty() {
            for suffix in ["Inputs", "At"] {
                register(file, ast, &mut names, &format!("{}_{suffix}", action.operator), variant)?;
            }
            for field in &action.fields {
                if !symbols.contains(&field.symbol) {
                    register(file, ast, &mut names, &field.symbol, &field.name)?;
                    symbols.insert(field.symbol.clone());
                }
            }
        }
    }
    if model.has_inputs() {
        register(file, ast, &mut names, &model.index_symbol, "__umpire_i")?;
    }
    for claim in &model.invariants {
        register(file, ast, &mut names, &format!("Invariant_{}", claim.name), &claim.name)?;
    }
    for claim in &model.postconditions {
        register(file, ast, &mut names, &format!("Post_{}", claim.name), &claim.name)?;
    }
    for claim in &model.negative {
        register(file, ast, &mut names, &format!("Negative_{}", claim.name), &claim.name)?;
    }
    for assumption in &model.assumptions {
        if model.progress.iter().any(|claim| claim.assumption.same_scope(assumption)) {
            register(
                file,
                ast,
                &mut names,
                &format!("Assumption_{}", assumption.name),
                &assumption.name,
            )?;
        }
    }
    for claim in &model.progress {
        for prefix in [
            "Scoped",
            "Activate",
            "Monitor",
            "ActiveEdge",
            "PrefixEdge",
            "Complete",
            "ProgressNext",
            "ProgressSpec",
            "Maximal",
            "Progress",
        ] {
            register(file, ast, &mut names, &format!("{prefix}_{}", claim.name), &claim.name)?;
        }
        for action in &model.actions {
            if !claim.assumption.fair.iter().any(|fair| fair.name == action.name) {
                continue;
            }
            let fair = format!("Fair_{}_{}", claim.name, action.operator);
            if action.fields.is_empty() {
                register(file, ast, &mut names, &fair, &claim.name)?;
            } else {
                // Exact payload obligations append a catalog position once the World is bound.
                register_prefix(file, ast, &format!("{fair}_"), &claim.name)?;
            }
        }
    }
    for field in &model.fields {
        register(file, ast, &mut names, &field.name, &field.name)?;
    }
    for name in model.constants.keys() {
        register(file, ast, &mut names, model.constant_symbols.get(name).unwrap_or(name), name)?;
    }
    for field in &model.world_fields {
        register(file, ast, &mut names, &field.symbol, &field.name)?;
    }
    Ok(())
}

fn register_prefix(
    file: &str,
    ast: &syn::File,
    prefix: &str,
    source: &str,
) -> Result<(), Diagnostic> {
    struct Prefixed<'a>(&'a str, bool);
    impl<'ast> syn::visit::Visit<'ast> for Prefixed<'_> {
        fn visit_ident(&mut self, ident: &'ast syn::Ident) {
            self.1 |= ident.to_string().starts_with(self.0);
        }
    }
    let mut prefixed = Prefixed(prefix, false);
    syn::visit::Visit::visit_file(&mut prefixed, ast);
    if !prefixed.1 {
        return Ok(());
    }
    let mut found = Find { name: source, span: None };
    syn::visit::Visit::visit_file(&mut found, ast);
    Err(Diagnostic::span(
        file,
        found.span.unwrap_or_else(Span::call_site),
        format!("authored declaration collides with generated TLA+ names `{prefix}*`"),
    ))
}

fn validate_text(file: &str, tokens: proc_macro2::TokenStream) -> Result<(), Diagnostic> {
    for token in tokens {
        match token {
            proc_macro2::TokenTree::Group(group) => validate_text(file, group.stream())?,
            proc_macro2::TokenTree::Literal(literal) => {
                if let Ok(text) = syn::parse_str::<syn::LitStr>(&literal.to_string())
                    && text.value().chars().any(|character| {
                        character.is_control() && !['\n', '\r', '\t'].contains(&character)
                    })
                {
                    return Err(Diagnostic::span(
                        file,
                        literal.span(),
                        "unsupported string control character",
                    ));
                }
            }
            _ => {}
        }
    }
    Ok(())
}

struct Find<'a> {
    name: &'a str,
    span: Option<Span>,
}
impl<'ast> syn::visit::Visit<'ast> for Find<'_> {
    fn visit_ident(&mut self, ident: &'ast syn::Ident) {
        if self.span.is_none() && ident == self.name {
            self.span = Some(ident.span());
        }
    }
    fn visit_lit_str(&mut self, literal: &'ast syn::LitStr) {
        if self.span.is_none() && literal.value() == self.name {
            self.span = Some(literal.span());
        }
    }
}
fn register(
    file: &str,
    ast: &syn::File,
    names: &mut BTreeSet<String>,
    name: &str,
    source: &str,
) -> Result<(), Diagnostic> {
    if names.insert(name.into())
        && !name.starts_with("r#")
        && name.chars().all(|c| c.is_ascii_alphanumeric() || c == '_')
    {
        return Ok(());
    }
    let mut found = Find { name: source, span: None };
    syn::visit::Visit::visit_file(&mut found, ast);
    Err(Diagnostic::span(
        file,
        found.span.unwrap_or_else(Span::call_site),
        format!("authored declaration collides with generated TLA+ name `{name}`"),
    ))
}
