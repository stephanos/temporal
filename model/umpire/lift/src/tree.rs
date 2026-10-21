use crate::{
    Binding, Diagnostic, Field, Literal, Type,
    expr::{Expr, ExprKind},
};
use std::collections::BTreeMap;

/// An admitted Model. Its typed expression tree has no public stability promise.
#[derive(Clone, Debug)]
pub struct Lifted {
    pub(crate) file: String,
    pub(crate) name: String,
    pub(crate) machine_module: String,
    pub(crate) state_name: String,
    pub(crate) outcome_type: Type,
    pub(crate) world_path: String,
    pub(crate) declared_world: bool,
    pub(crate) world_fields: Vec<WorldField>,
    pub(crate) world_values: Option<Vec<Expr>>,
    pub(crate) complete_starts: Option<Vec<Vec<Expr>>>,
    pub(crate) location: Diagnostic,
    pub(crate) fields: Vec<FieldDef>,
    pub(crate) constants: BTreeMap<String, Expr>,
    pub(crate) constant_symbols: BTreeMap<String, String>,
    pub(crate) actions: Vec<ActionDef>,
    pub(crate) invariants: Vec<ClaimDef>,
    pub(crate) postconditions: Vec<PostDef>,
    pub(crate) progress: Vec<ProgressDef>,
    pub(crate) assumptions: Vec<AssumptionDef>,
    pub(crate) negative: Vec<ClaimDef>,
}
#[derive(Clone, Debug)]
pub(crate) struct FieldDef {
    pub name: String,
    pub ty: Type,
    pub start: Start,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
pub(crate) struct WorldField {
    pub name: String,
    pub ty: Type,
    pub symbol: String,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
pub(crate) enum Start {
    Computed,
    Value(Expr),
    Range(Expr, Expr),
    ExclusiveRange(Expr, Expr),
}

impl Lifted {
    /// Whether this source requires an explicitly selected World and complete starts.
    #[must_use]
    pub const fn requires_binding(&self) -> bool {
        self.declared_world
    }

    /// The original inline machine module's name.
    #[must_use]
    pub fn machine_module(&self) -> &str {
        &self.machine_module
    }

    /// Independently admitted World fields in source declaration order.
    #[must_use]
    pub fn world_fields(&self) -> Vec<Field> {
        self.world_fields
            .iter()
            .map(|field| Field { name: field.name.clone(), ty: field.ty.clone() })
            .collect()
    }

    /// Independently admitted state fields in source declaration order.
    #[must_use]
    pub fn state_fields(&self) -> Vec<Field> {
        self.fields
            .iter()
            .map(|field| Field { name: field.name.clone(), ty: field.ty.clone() })
            .collect()
    }

    /// Validate projected literals against original-source schemas and bind complete starts.
    ///
    /// # Errors
    /// Refuses absent, duplicate, extra or incorrectly typed fields, and World integers outside
    /// TLC's `.cfg` range, by source location.
    pub fn bind(&self, binding: &Binding) -> Result<Self, Diagnostic> {
        if !self.requires_binding() {
            return Err(message(&self.location, "this source has no declared World to bind"));
        }
        let world = bind_fields(
            self.world_fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
            &binding.world,
            &self.location,
            "World",
        )?;
        // TLC's .cfg parser accepts only non-negative integers up to i32::MAX as CONSTANT values.
        for (field, value) in self.world_fields.iter().zip(&world) {
            if !cfg_representable(value) {
                return Err(message(
                    &field.location,
                    format!(
                        "World field `{}` value is not representable as a TLC .cfg constant (0..={})",
                        field.name,
                        i32::MAX
                    ),
                ));
            }
        }
        let mut starts = Vec::new();
        for row in &binding.starts {
            let state = bind_fields(
                self.fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
                row,
                &self.location,
                "complete start",
            )?;
            if !starts.contains(&state) {
                starts.push(state);
            }
        }
        let mut bound = self.clone();
        bound.world_values = Some(world);
        bound.complete_starts = Some(starts);
        Ok(bound)
    }
}

fn message(location: &Diagnostic, text: impl Into<String>) -> Diagnostic {
    Diagnostic { message: text.into(), ..location.clone() }
}

fn bind_fields<'a>(
    schema: impl Iterator<Item = (&'a String, &'a Type, &'a Diagnostic)>,
    values: &[(String, Literal)],
    location: &Diagnostic,
    kind: &str,
) -> Result<Vec<Expr>, Diagnostic> {
    let schema: Vec<_> = schema.collect();
    let mut supplied = BTreeMap::new();
    for (name, value) in values {
        let Some((_, _, field_location)) = schema.iter().find(|(field, _, _)| *field == name)
        else {
            return Err(message(location, format!("extra {kind} field `{name}`")));
        };
        if supplied.insert(name, value).is_some() {
            return Err(message(field_location, format!("duplicate {kind} field `{name}`")));
        }
    }
    schema
        .into_iter()
        .map(|(name, ty, location)| {
            let value = supplied
                .get(name)
                .ok_or_else(|| message(location, format!("missing {kind} field `{name}`")))?;
            literal(value, ty).ok_or_else(|| {
                message(
                    location,
                    format!("{kind} field `{name}` value does not match admitted type {ty:?}"),
                )
            })
        })
        .collect()
}

fn cfg_representable(value: &Expr) -> bool {
    match &value.kind {
        ExprKind::Int(value) => (0..=i128::from(i32::MAX)).contains(value),
        ExprKind::Sequence(values) => values.iter().all(cfg_representable),
        _ => true,
    }
}

fn literal(value: &Literal, ty: &Type) -> Option<Expr> {
    let kind = match (value, ty) {
        (Literal::Bool(value), Type::Bool) => ExprKind::Bool(*value),
        (Literal::Int(value), Type::Int { min, max }) if value >= min && value <= max => {
            ExprKind::Int(*value)
        }
        (Literal::Enum(value), Type::Enum { variants, .. }) if variants.contains(value) => {
            ExprKind::Enum(value.clone())
        }
        (Literal::Sequence(values), Type::Seq(inner)) => ExprKind::Sequence(
            values.iter().map(|value| literal(value, inner)).collect::<Option<_>>()?,
        ),
        _ => return None,
    };
    Some(Expr { kind, ty: ty.clone() })
}
#[derive(Clone, Debug)]
pub(crate) struct ActionDef {
    pub name: String,
    pub operator: String,
    pub internal: bool,
    pub branches: Vec<Branch>,
}
#[derive(Clone, Debug)]
pub(crate) struct Branch {
    pub guard: Expr,
    pub updates: BTreeMap<String, Expr>,
    pub checks: Vec<Expr>,
    pub outcome: Expr,
    pub preserve_self_loop: bool,
}
#[derive(Clone, Debug)]
pub(crate) struct ClaimDef {
    pub name: String,
    pub expr: Expr,
}
#[derive(Clone, Debug)]
pub(crate) struct PostDef {
    pub name: String,
    pub expr: Expr,
    pub after: Option<String>,
}
#[derive(Clone, Debug)]
pub(crate) struct ProgressDef {
    pub name: String,
    pub trigger: Expr,
    pub target: Expr,
    pub assumption: AssumptionDef,
    pub bound: Option<u64>,
}
#[derive(Clone, Debug, Default)]
pub(crate) struct AssumptionDef {
    pub name: String,
    pub only: Option<Vec<String>>,
    pub without: Vec<String>,
    pub fair: Vec<String>,
}
impl AssumptionDef {
    pub(crate) fn same_scope(&self, other: &Self) -> bool {
        self.only == other.only && self.without == other.without && self.fair == other.fair
    }
}
