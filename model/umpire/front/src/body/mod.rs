use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind},
    tree::{FieldDef, Lifted, WorldField},
};
use quote::ToTokens;
use std::collections::BTreeMap;
use syn::{Expr as RustExpr, Stmt};

mod claim;
mod effects;
mod expr;
mod inputs;
mod library;
mod optional;
mod outcome;
mod progress;
mod typer;
mod vector;

type Result<T> = std::result::Result<T, Diagnostic>;
#[derive(Clone, Default)]
pub(crate) struct Env {
    pub(crate) values: BTreeMap<String, Expr>,
    world: Option<String>,
    /// Names bound to a whole state, whose fields are bound as `name.field`.
    states: Vec<String>,
    outcome: Option<String>,
}
impl Env {
    pub(crate) fn new() -> Self {
        Self::default()
    }
    pub(crate) fn insert(&mut self, name: String, value: Expr) -> Option<Expr> {
        if !name.contains('.') {
            self.shadow(&name);
        }
        self.values.insert(name, value)
    }
    /// Forget what `name` was bound to, as a new binding of it does.
    fn shadow(&mut self, name: &str) {
        self.values.retain(|key, _| !key.starts_with(&format!("{name}.")));
        if self.world.as_deref() == Some(name) {
            self.world = None;
        }
        self.states.retain(|state| state != name);
        if self.outcome.as_deref() == Some(name) {
            self.outcome = None;
        }
    }
    /// Bind `name` to a whole state with these field values.
    pub(crate) fn bind_state(&mut self, name: &str, fields: &[FieldDef], values: &Env) {
        self.shadow(name);
        self.values.remove(name);
        for field in fields {
            if let Some(value) = values.get(&field.name) {
                self.values.insert(format!("{name}.{}", field.name), value.clone());
            }
        }
        self.states.push(name.to_owned());
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

#[derive(Clone)]
pub(crate) struct Environment {
    /// The State struct's name, which `&Self` also spells.
    pub(crate) state: String,
    pub(crate) fields: Vec<FieldDef>,
    pub(crate) declared_world: bool,
    pub(crate) world_fields: Vec<WorldField>,
    pub(crate) constants: Env,
    pub(crate) const_parameters: BTreeMap<String, i128>,
    pub(crate) enums: BTreeMap<String, Type>,
    pub(crate) defaults: BTreeMap<String, String>,
    pub(crate) bounded: BTreeMap<String, umpire_ir::Bounded>,
    pub(crate) bounded_defaults: Vec<String>,
    pub(crate) helpers: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) methods: BTreeMap<String, syn::ImplItemFn>,
    pub(crate) vector_imported: bool,
    /// The step outcome type: unit, or the stock outcome of a rejecting Model.
    pub(crate) outcome_type: Type,
    pub(crate) fact_type: Type,
    pub(crate) fact_shadows: Vec<String>,
    /// Whether the State implements `umpire_dsl::Phased`, whose default methods it then has.
    pub(crate) phased: bool,
    /// The module's `use umpire_dsl::...` bindings: bound name to the path after `umpire_dsl::`.
    pub(crate) dsl: BTreeMap<String, String>,
    /// Each lowered `check!`'s argument tokens, which `ExprKind::Check` indexes.
    pub(crate) checks: std::cell::RefCell<Vec<proc_macro2::TokenStream>>,
}
impl Environment {
    /// The environment golden hook claims read in: the admitted Model's schema and constants.
    /// The tree keeps no helper bodies, so a hook claim reads fields, the World and constants.
    pub(crate) fn of(model: &Lifted) -> Self {
        fn enums(ty: &Type, out: &mut BTreeMap<String, Type>) {
            match ty {
                Type::Enum { name, .. } => {
                    out.insert(name.clone(), ty.clone());
                }
                Type::Tuple(types) => types.iter().for_each(|ty| enums(ty, out)),
                Type::Seq(inner) => enums(inner, out),
                Type::Bool
                | Type::Int { .. }
                | Type::Bounded(_)
                | Type::Optional(_)
                | Type::Text
                | Type::Unit => {}
            }
        }
        let mut known = BTreeMap::new();
        let mut bounded = BTreeMap::new();
        let types = model
            .fields
            .iter()
            .map(|field| &field.ty)
            .chain(model.world_fields.iter().map(|field| &field.ty));
        for ty in types
            .chain(model.actions.iter().flat_map(|action| action.fields.iter().map(|f| &f.ty)))
            .chain(model.constants.values().map(|value| &value.ty))
        {
            enums(ty, &mut known);
            if let Type::Bounded(value) | Type::Optional(value) = ty {
                bounded.insert(value.name.clone(), value.clone());
            }
        }
        enums(&model.fact_type, &mut known);
        Self {
            state: model.state_name.clone(),
            fields: model.fields.clone(),
            declared_world: model.declared_world,
            world_fields: model.world_fields.clone(),
            constants: model.constants.clone().into_iter().collect(),
            const_parameters: BTreeMap::new(),
            enums: known,
            bounded,
            bounded_defaults: Vec::new(),
            defaults: BTreeMap::new(),
            helpers: BTreeMap::new(),
            methods: BTreeMap::new(),
            vector_imported: model.fields.iter().any(|field| matches!(field.ty, Type::Seq(_))),
            outcome_type: model.outcome_type.clone(),
            fact_type: model.fact_type.clone(),
            fact_shadows: Vec::new(),
            phased: false,
            dsl: BTreeMap::new(),
            checks: std::cell::RefCell::default(),
        }
    }
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
        let declared = crate::declare::world_parameter(&method.sig).map_err(Diagnostic::syn)?;
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
impl Environment {
    /// A path's meaning inside `umpire_dsl`, written in full or through the module's imports.
    pub(crate) fn dsl_path(&self, path: &syn::Path) -> Option<String> {
        let mut segments = path.segments.iter().map(|segment| segment.ident.to_string());
        let spelling = segments.clone().collect::<Vec<_>>().join("::");
        if let Some(member) = crate::syntax::dsl_path(&spelling) {
            return Some(member.to_owned());
        }
        let first = segments.next()?;
        let head = Some(self.dsl.get(&first)?.clone());
        let parts: Vec<_> = head.into_iter().chain(segments).collect();
        (!parts.is_empty()).then(|| parts.join("::"))
    }
    /// The field values of a whole-state expression: `self`, or a name bound to a state.
    pub(crate) fn whole(&self, node: &RustExpr, state: &Env, locals: &Env) -> Option<Env> {
        let RustExpr::Path(path) = node else { return None };
        if path.path.is_ident("self") {
            return self
                .fields
                .iter()
                .all(|field| state.contains_key(&field.name))
                .then(|| state.clone());
        }
        let name = path.path.get_ident()?.to_string();
        if !locals.states.contains(&name) {
            return None;
        }
        self.fields
            .iter()
            .map(|field| {
                let value = locals.get(&format!("{name}.{}", field.name))?;
                Some((field.name.clone(), value.clone()))
            })
            .collect()
    }
    /// A pure State method: an inherent or trait-impl method, or a `Phased` default.
    pub(crate) fn state_method(&self, name: &str) -> Option<syn::ImplItemFn> {
        self.methods
            .get(name)
            .cloned()
            .or_else(|| self.phased.then(|| crate::library::phased_default(name)).flatten())
    }
}
fn tail(block: &syn::Block) -> Result<&RustExpr> {
    match block.stmts.as_slice() {
        [Stmt::Expr(expr, None)] => Ok(expr),
        _ => Err(Diagnostic::at(block, "expected a single expression body")),
    }
}
