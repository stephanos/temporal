use crate::{Diagnostic, Type, expr::Expr, tree::FieldDef};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{Expr as RustExpr, Stmt};

mod claim;
mod effects;
mod expr;
mod progress;
mod typer;
mod vector;

type Result<T> = std::result::Result<T, Diagnostic>;
pub(crate) type Env = BTreeMap<String, Expr>;

pub(crate) struct Environment<'a> {
    pub(crate) file: &'a str,
    pub(crate) fields: Vec<FieldDef>,
    pub(crate) constants: Env,
    pub(crate) enums: BTreeMap<String, Type>,
    pub(crate) defaults: BTreeMap<String, String>,
    pub(crate) helpers: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) methods: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) vector_imported: bool,
}
impl Environment<'_> {
    fn err<T>(&self, node: &impl ToTokens, message: impl Into<String>) -> Result<T> {
        Err(Diagnostic::at(self.file, node, message))
    }
    pub(crate) fn field_env(&self, primed: bool) -> Env {
        self.fields
            .iter()
            .map(|f| (f.name.clone(), Expr::var(&f.name, f.ty.clone(), primed)))
            .collect()
    }
}
fn tail<'a>(file: &str, block: &'a syn::Block) -> Result<&'a RustExpr> {
    match block.stmts.as_slice() {
        [Stmt::Expr(expr, None)] => Ok(expr),
        _ => Err(Diagnostic::at(file, block, "expected a single expression body")),
    }
}
