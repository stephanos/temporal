use crate::{Binding, Concrete, Diagnostic, Lifted, Literal, Type};
use umpire_front::{
    Diagnostic as FrontDiagnostic,
    expr::{Expr, ExprKind},
    tree::ActionRef,
};

impl Lifted {
    /// A refusal at a front-end location in this Model's file.
    fn refuse(&self, location: &FrontDiagnostic, text: impl Into<String>) -> Diagnostic {
        message(&self.file, location, text)
    }

    /// Bind the selected World's values, complete starts and checked catalog, which the same
    /// compiled Model exports, as the admitted types.
    ///
    /// # Errors
    /// Refuses a value outside its admitted type, and World integers outside TLC's `.cfg` range,
    /// by source location.
    pub fn bind(&self, binding: &Binding) -> Result<Self, Diagnostic> {
        if !self.requires_binding() {
            return Err(self.refuse(&self.location, "this source has no declared World to bind"));
        }
        let world = typed(
            &self.file,
            self.world_fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
            &binding.world,
            "World",
        )?;
        // TLC's .cfg parser accepts only non-negative integers up to i32::MAX as CONSTANT values.
        for (field, value) in self.world_fields.iter().zip(&world) {
            if !cfg_representable(value) {
                return Err(self.refuse(&field.location,
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
            let state = typed(
                &self.file,
                self.fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
                row,
                "complete start",
            )?;
            if !starts.contains(&state) {
                starts.push(state);
            }
        }
        let catalog = self.bind_catalog(binding)?;
        let mut bound = self.clone();
        bound.world_values = Some(world);
        bound.complete_starts = Some(starts);
        bound.catalog = Some(catalog);
        Ok(bound)
    }

    /// The native catalog as typed entries; positional duplicates share their first ordinal.
    fn bind_catalog(&self, binding: &Binding) -> Result<Vec<Entry>, Diagnostic> {
        let mut catalog: Vec<Entry> = Vec::new();
        for native in &binding.actions {
            let Some(index) = self.actions.iter().position(|action| action.name == native.action)
            else {
                return Err(self.refuse(
                    &self.location,
                    format!("native catalog action `{}` is not declared", native.action),
                ));
            };
            let values = typed(
                &self.file,
                self.actions[index]
                    .fields
                    .iter()
                    .map(|field| (&field.name, &field.ty, &field.location)),
                &native.fields,
                "input",
            )?;
            let ordinal = catalog
                .iter()
                .find(|entry| entry.action == index && entry.values == values)
                .map_or(catalog.len(), |entry| entry.ordinal);
            catalog.push(Entry { action: index, values, ordinal });
        }
        self.scoped_catalog(&catalog)?;
        Ok(catalog)
    }

    /// As native progress admission: every listed action is in the selected catalog, and an
    /// explicit restriction keeps at least one catalog action.
    fn scoped_catalog(&self, catalog: &[Entry]) -> Result<(), Diagnostic> {
        let references: Vec<_> = catalog
            .iter()
            .map(|entry| ActionRef {
                name: self.actions[entry.action].name.clone(),
                values: entry.values.clone(),
            })
            .collect();
        for claim in &self.progress {
            let assumption = &claim.assumption;
            for listed in
                assumption.only.iter().flatten().chain(&assumption.without).chain(&assumption.fair)
            {
                if !references.contains(listed) {
                    let values: Vec<_> = listed.values.iter().map(crate::emit::rendered).collect();
                    let spelled = if values.is_empty() {
                        listed.name.clone()
                    } else {
                        format!("{}({})", listed.name, values.join(", "))
                    };
                    return Err(self.refuse(&self.location,
                        format!(
                            "progress `{}` names `{spelled}`, which the selected World's catalog lacks",
                            claim.name
                        ),
                    ));
                }
            }
            if (assumption.only.is_some() || !assumption.without.is_empty())
                && !references.iter().any(|reference| assumption.allows(reference))
            {
                return Err(self.refuse(
                    &self.location,
                    format!("progress `{}` assumption admits no catalog action", claim.name),
                ));
            }
        }
        Ok(())
    }

    /// The bound catalog as typed concrete identities.
    pub(crate) fn concrete(&self) -> Vec<Concrete> {
        self.catalog
            .iter()
            .flatten()
            .map(|entry| {
                let action = &self.actions[entry.action];
                Concrete {
                    action: action.name.clone(),
                    fields: action
                        .fields
                        .iter()
                        .zip(&entry.values)
                        .map(|(field, value)| (field.name.clone(), to_literal(value)))
                        .collect(),
                }
            })
            .collect()
    }
}

pub(crate) fn to_literal(value: &Expr) -> Literal {
    if let ExprKind::Int(raw) = &value.kind {
        match &value.ty {
            Type::Bounded(bound) => {
                return Literal::Bounded { name: bound.name.clone(), raw: *raw };
            }
            Type::Optional(bound) => {
                return match bound.decode(*raw) {
                    Some(value) => {
                        Literal::Optional(value.map(|raw| {
                            Box::new(Literal::Bounded { name: bound.name.clone(), raw })
                        }))
                    }
                    None => Literal::Int(*raw),
                };
            }
            _ => {}
        }
    }
    match &value.kind {
        ExprKind::Bool(value) => Literal::Bool(*value),
        ExprKind::Int(value) => Literal::Int(*value),
        ExprKind::Sequence(values) => Literal::Sequence(values.iter().map(to_literal).collect()),
        ExprKind::Enum(value) | ExprKind::Text(value) => Literal::Enum(value.clone()),
        _ => Literal::Sequence(Vec::new()),
    }
}

fn message(file: &str, location: &FrontDiagnostic, text: impl Into<String>) -> Diagnostic {
    Diagnostic::of(file, location, text)
}

/// Named values in schema order, each as its field's admitted type.
fn typed<'a>(
    file: &str,
    schema: impl Iterator<Item = (&'a String, &'a Type, &'a FrontDiagnostic)>,
    values: &[(String, Literal)],
    kind: &str,
) -> Result<Vec<Expr>, Diagnostic> {
    let schema: Vec<_> = schema.collect();
    if let Some((_, _, location)) = schema.first() {
        let mut seen = std::collections::BTreeSet::new();
        for (name, _) in values {
            if !seen.insert(name) {
                return Err(message(file, location, format!("duplicate {kind} field `{name}`")));
            }
            if !schema.iter().any(|(field, _, _)| *field == name) {
                return Err(message(file, location, format!("undeclared {kind} field `{name}`")));
            }
        }
    } else if !values.is_empty() {
        return Err(message(
            file,
            &FrontDiagnostic { line: 1, column: 1, message: String::new() },
            format!("undeclared {kind} field `{}`", values[0].0),
        ));
    }
    schema
        .into_iter()
        .map(|(name, ty, location)| {
            let value = values
                .iter()
                .find(|(field, _)| field == name)
                .ok_or_else(|| message(file, location, format!("missing {kind} field `{name}`")))?;
            literal(&value.1, ty).ok_or_else(|| {
                message(
                    file,
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
        (Literal::Bounded { name, raw }, Type::Bounded(bound))
            if name == &bound.name && bound.contains(*raw) =>
        {
            ExprKind::Int(*raw)
        }
        (Literal::Optional(None), Type::Optional(bound)) if bound.optional_valid() => {
            ExprKind::Int(0)
        }
        (Literal::Optional(Some(value)), Type::Optional(bound)) => {
            let Literal::Bounded { name, raw } = value.as_ref() else {
                return None;
            };
            if name != &bound.name {
                return None;
            }
            ExprKind::Int(bound.encode(*raw)?)
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

/// One concrete catalog action: an index into `Lifted::actions` and its payload values.
#[derive(Clone, Debug)]
pub(crate) struct Entry {
    pub(crate) action: usize,
    pub(crate) values: Vec<Expr>,
    /// The first catalog ordinal with the same identity, so positional duplicates share one
    /// observable edge as they share one native catalog equality class.
    pub(crate) ordinal: usize,
}
