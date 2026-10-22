use crate::{
    Diagnostic, Type,
    expr::{Expr, ExprKind},
    tree::{FieldDef, WorldField},
};
use quote::ToTokens;
use std::collections::{BTreeMap, BTreeSet};
use syn::{Expr as RustExpr, Stmt};

mod claim;
mod effects;
mod expr;
mod inputs;
mod progress;
mod typer;
mod vector;

type Result<T> = std::result::Result<T, Diagnostic>;
#[derive(Clone, Default)]
pub(crate) struct Env {
    pub(crate) values: BTreeMap<String, Expr>,
    world: Option<String>,
    state: Option<String>,
    outcome: Option<String>,
}
impl Env {
    pub(crate) fn new() -> Self {
        Self::default()
    }
    pub(crate) fn insert(&mut self, name: String, value: Expr) -> Option<Expr> {
        if !name.contains('.') {
            self.values.retain(|key, _| !key.starts_with(&format!("{name}.")));
            if self.world.as_ref() == Some(&name) {
                self.world = None;
            }
            if self.state.as_ref() == Some(&name) {
                self.state = None;
            }
            if self.outcome.as_ref() == Some(&name) {
                self.outcome = None;
            }
        }
        self.values.insert(name, value)
    }
}
impl std::ops::Deref for Env {
    type Target = BTreeMap<String, Expr>;
    fn deref(&self) -> &Self::Target {
        &self.values
    }
}
impl std::ops::DerefMut for Env {
    fn deref_mut(&mut self) -> &mut Self::Target {
        &mut self.values
    }
}
impl FromIterator<(String, Expr)> for Env {
    fn from_iter<T: IntoIterator<Item = (String, Expr)>>(values: T) -> Self {
        Self { values: values.into_iter().collect(), ..Self::default() }
    }
}

pub(super) struct Parameters<'a> {
    pub(super) world: Option<&'a syn::PatType>,
    pub(super) positional: Vec<&'a syn::PatType>,
}

pub(crate) struct Environment {
    /// The State struct's name, which `&Self` also spells.
    pub(crate) state: String,
    pub(crate) fields: Vec<FieldDef>,
    pub(crate) declared_world: bool,
    pub(crate) world_fields: Vec<WorldField>,
    pub(crate) constants: Env,
    pub(crate) enums: BTreeMap<String, Type>,
    pub(crate) defaults: BTreeMap<String, String>,
    pub(crate) helpers: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) methods: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) vector_imported: bool,
    /// Local fieldless enums deriving `Finite`.
    pub(crate) finite: BTreeSet<String>,
    /// Free functions admitted as World-dependent `#[domain]` sources; bodies are not read.
    pub(crate) domain_functions: BTreeMap<String, syn::ItemFn>,
}
impl Environment {
    #[expect(clippy::unused_self, reason = "refusals read as methods of the admitting environment")]
    fn err<T>(&self, node: &impl ToTokens, message: impl Into<String>) -> Result<T> {
        Err(Diagnostic::at(node, message))
    }
    pub(crate) fn field_env(&self, primed: bool) -> Env {
        self.fields
            .iter()
            .map(|f| (f.name.clone(), Expr::var(&f.name, f.ty.clone(), primed)))
            .collect()
    }
    pub(super) fn parameters<'a>(&self, method: &'a syn::ImplItemFn) -> Result<Parameters<'a>> {
        let declared = crate::declare::world_parameter(&method.sig).map_err(Diagnostic::from)?;
        let mut world = None;
        let mut positional = Vec::new();
        for (index, input) in method.sig.inputs.iter().enumerate() {
            let syn::FnArg::Typed(input) = input else { continue };
            if declared != Some(index) {
                positional.push(input);
                continue;
            }
            if !self.declared_world {
                return self.err(input, "a World parameter requires a declared World");
            }
            if !matches!(input.ty.as_ref(), syn::Type::Reference(reference) if matches!(reference.elem.as_ref(), syn::Type::Path(path) if path.path.is_ident("World")))
            {
                return self
                    .err(&input.ty, "qualified or generic World source types are unsupported");
            }
            world = Some(input);
        }
        Ok(Parameters { world, positional })
    }
    pub(crate) fn method_env(&self, method: &syn::ImplItemFn) -> Result<Env> {
        let parameters = self.parameters(method)?;
        let mut env = Env::new();
        if let Some(parameter) = parameters.world {
            self.bind_world(&parameter.pat, &mut env)?;
        }
        Ok(env)
    }
    pub(super) fn bind_world(&self, pattern: &syn::Pat, env: &mut Env) -> Result<()> {
        match pattern {
            syn::Pat::Ident(id) => {
                let name = id.ident.to_string();
                for field in &self.world_fields {
                    env.insert(
                        format!("{name}.{}", field.name),
                        Expr { kind: ExprKind::World(field.name.clone()), ty: field.ty.clone() },
                    );
                }
                env.world = Some(name);
                Ok(())
            }
            syn::Pat::Wild(_) => Ok(()),
            syn::Pat::Type(pattern) => self.bind_world(&pattern.pat, env),
            _ => self.err(pattern, "World parameters require a name or wildcard"),
        }
    }
    pub(super) fn helper_env(
        &self,
        method: &syn::ImplItemFn,
        args: &syn::punctuated::Punctuated<RustExpr, syn::Token![,]>,
        caller: &Env,
    ) -> Result<Env> {
        let parameters = self.parameters(method)?;
        if !parameters.positional.is_empty()
            || args.len() != usize::from(parameters.world.is_some())
        {
            return self.err(
                method,
                "only pure helper calls with an optional trailing World are admitted",
            );
        }
        if let Some(argument) = args.first()
            && !matches!(argument, RustExpr::Path(path) if caller.world.as_ref().is_some_and(|name| path.path.is_ident(name)))
        {
            return self
                .err(argument, "helper World argument must be the selected World parameter");
        }
        self.method_env(method)
    }
}
fn tail(block: &syn::Block) -> Result<&RustExpr> {
    match block.stmts.as_slice() {
        [Stmt::Expr(expr, None)] => Ok(expr),
        _ => Err(Diagnostic::at(block, "expected a single expression body")),
    }
}
