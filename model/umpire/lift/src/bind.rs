use crate::{Binding, CATALOG_LIMIT, Concrete, Diagnostic, Lifted, Literal, Type};
use std::collections::BTreeMap;
use umpire_front::{
    Diagnostic as FrontDiagnostic,
    expr::{Expr, ExprKind},
    tree::{ActionDef, ActionRef, Entry, InputDomain},
};

impl Lifted {
    /// A refusal at a front-end location in this Model's file.
    fn refuse(&self, location: &FrontDiagnostic, text: impl Into<String>) -> Diagnostic {
        message(&self.file, location, text)
    }

    /// Validate projected literals against original-source schemas and bind complete starts.
    ///
    /// # Errors
    /// Refuses absent, duplicate, extra or incorrectly typed fields, and World integers outside
    /// TLC's `.cfg` range, by source location.
    pub fn bind(&self, binding: &Binding) -> Result<Self, Diagnostic> {
        if !self.requires_binding() {
            return Err(self.refuse(&self.location, "this source has no declared World to bind"));
        }
        let world = bind_fields(
            &self.file,
            self.world_fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
            &binding.world,
            &self.location,
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
            let state = bind_fields(
                &self.file,
                self.fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
                row,
                &self.location,
                "complete start",
            )?;
            if !starts.contains(&state) {
                starts.push(state);
            }
        }
        let catalog = self.bind_catalog(binding)?;
        let mut bound = self.clone();
        bound.model.world_values = Some(world);
        bound.model.complete_starts = Some(starts);
        bound.model.catalog = Some(catalog);
        Ok(bound)
    }

    /// Each variant's per-field domain values: declared in source, or supplied for the selected
    /// World and checked against the field's admitted type.
    fn bound_domains(&self, binding: &Binding) -> Result<Vec<Vec<Vec<Expr>>>, Diagnostic> {
        let mut supplied = BTreeMap::new();
        for domain in &binding.domains {
            let declared =
                self.actions.iter().find(|action| action.name == domain.action).and_then(
                    |action| {
                        action.fields.iter().find(|field| {
                            field.name == domain.field && matches!(field.domain, InputDomain::World)
                        })
                    },
                );
            let Some(field) = declared else {
                return Err(self.refuse(
                    &self.location,
                    format!("extra action domain `{}` field `{}`", domain.action, domain.field),
                ));
            };
            if supplied.insert((domain.action.as_str(), domain.field.as_str()), domain).is_some() {
                return Err(self.refuse(
                    &field.location,
                    format!("duplicate action domain `{}` field `{}`", domain.action, domain.field),
                ));
            }
        }
        let mut domains = Vec::new();
        for action in &self.actions {
            let mut fields = Vec::new();
            for field in &action.fields {
                let values = match &field.domain {
                    InputDomain::Values(values) => values.clone(),
                    InputDomain::World => {
                        let domain = supplied
                            .get(&(action.name.as_str(), field.name.as_str()))
                            .ok_or_else(|| {
                                self.refuse(
                                    &field.location,
                                    format!(
                                        "missing action domain `{}` field `{}`",
                                        action.name, field.name
                                    ),
                                )
                            })?;
                        if domain.values.len() > CATALOG_LIMIT {
                            return Err(self.refuse(&field.location,
                                format!(
                                    "action domain `{}` field `{}` exceeds the catalog limit {CATALOG_LIMIT}",
                                    action.name, field.name
                                ),
                            ));
                        }
                        domain
                            .values
                            .iter()
                            .map(|value| {
                                literal(value, &field.ty).ok_or_else(|| {
                                    self.refuse(&field.location,
                                        format!(
                                            "action domain `{}` field `{}` value does not match admitted type {:?}",
                                            action.name, field.name, field.ty
                                        ),
                                    )
                                })
                            })
                            .collect::<Result<_, _>>()?
                    }
                };
                fields.push(values);
            }
            domains.push(fields);
        }
        Ok(domains)
    }

    /// The declaration-order domain product of every variant, checked against the native catalog.
    fn bind_catalog(&self, binding: &Binding) -> Result<Vec<Entry>, Diagnostic> {
        let domains = self.bound_domains(binding)?;
        let count =
            domains.iter().try_fold(0usize, |total, fields| total.checked_add(product(fields)?));
        match count {
            None => {
                return Err(self.refuse(&self.location, "action catalog count overflows"));
            }
            Some(count) if count > CATALOG_LIMIT => {
                return Err(self.refuse(&self.location,
                    format!(
                        "action catalog has {count} concrete actions, above the limit {CATALOG_LIMIT}"
                    ),
                ));
            }
            Some(_) => {}
        }
        let mut expected: Vec<Entry> = Vec::new();
        for (index, fields) in domains.iter().enumerate() {
            for values in expand(fields) {
                let ordinal = expected
                    .iter()
                    .find(|entry| entry.action == index && entry.values == values)
                    .map_or(expected.len(), |entry| entry.ordinal);
                expected.push(Entry { action: index, values, ordinal });
            }
        }
        if binding.actions.len() != expected.len() {
            return Err(self.refuse(
                &self.location,
                format!(
                    "native catalog has {} concrete actions; the source domains admit {}",
                    binding.actions.len(),
                    expected.len()
                ),
            ));
        }
        for (ordinal, (native, entry)) in binding.actions.iter().zip(&expected).enumerate() {
            let action = &self.actions[entry.action];
            compare(&self.file, ordinal, native, action, &entry.values)?;
        }
        self.scoped_catalog(&expected)?;
        Ok(expected)
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

fn compare(
    file: &str,
    ordinal: usize,
    native: &Concrete,
    action: &ActionDef,
    values: &[Expr],
) -> Result<(), Diagnostic> {
    if native.action != action.name {
        return Err(message(
            file,
            &action.location,
            format!(
                "native catalog ordinal {ordinal} is `{}`; the source declares `{}`",
                native.action, action.name
            ),
        ));
    }
    let mut seen = std::collections::BTreeSet::new();
    for (name, _) in &native.fields {
        let Some(field) = action.fields.iter().find(|field| &field.name == name) else {
            return Err(message(
                file,
                &action.location,
                format!("extra input field `{name}` at native catalog ordinal {ordinal}"),
            ));
        };
        if !seen.insert(name) {
            return Err(message(
                file,
                &field.location,
                format!("duplicate input field `{name}` at native catalog ordinal {ordinal}"),
            ));
        }
    }
    if native.fields.len() != action.fields.len() {
        let missing = action
            .fields
            .iter()
            .find(|field| !seen.contains(&field.name))
            .unwrap_or(&action.fields[0]);
        return Err(message(
            file,
            &missing.location,
            format!("missing input field `{}` at native catalog ordinal {ordinal}", missing.name),
        ));
    }
    for (((name, value), field), expected) in native.fields.iter().zip(&action.fields).zip(values) {
        if name != &field.name {
            return Err(message(
                file,
                &field.location,
                format!(
                    "native catalog ordinal {ordinal} orders input field `{name}` where the source declares `{}`",
                    field.name
                ),
            ));
        }
        if literal(value, &field.ty).as_ref() != Some(expected) {
            return Err(message(
                file,
                &field.location,
                format!(
                    "native catalog ordinal {ordinal} field `{name}` is {value:?}; the source domain product has {}",
                    crate::emit::rendered(expected)
                ),
            ));
        }
    }
    Ok(())
}

/// The exact product size: an empty factor empties the variant even beside a huge one.
fn product(fields: &[Vec<Expr>]) -> Option<usize> {
    if fields.iter().any(Vec::is_empty) {
        return Some(0);
    }
    fields.iter().try_fold(1usize, |total, values| total.checked_mul(values.len()))
}

/// Declaration-order product, the last field fastest, as the native catalog enumerates.
fn expand(fields: &[Vec<Expr>]) -> Vec<Vec<Expr>> {
    fields.iter().fold(vec![Vec::new()], |rows, values| {
        rows.into_iter()
            .flat_map(|row| {
                values.iter().map(move |value| {
                    let mut row = row.clone();
                    row.push(value.clone());
                    row
                })
            })
            .collect()
    })
}

fn to_literal(value: &Expr) -> Literal {
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

fn bind_fields<'a>(
    file: &str,
    schema: impl Iterator<Item = (&'a String, &'a Type, &'a FrontDiagnostic)>,
    values: &[(String, Literal)],
    location: &FrontDiagnostic,
    kind: &str,
) -> Result<Vec<Expr>, Diagnostic> {
    let schema: Vec<_> = schema.collect();
    let mut supplied = BTreeMap::new();
    for (name, value) in values {
        let Some((_, _, field_location)) = schema.iter().find(|(field, _, _)| *field == name)
        else {
            return Err(message(file, location, format!("extra {kind} field `{name}`")));
        };
        if supplied.insert(name, value).is_some() {
            return Err(message(file, field_location, format!("duplicate {kind} field `{name}`")));
        }
    }
    schema
        .into_iter()
        .map(|(name, ty, location)| {
            let value = supplied
                .get(name)
                .ok_or_else(|| message(file, location, format!("missing {kind} field `{name}`")))?;
            literal(value, ty).ok_or_else(|| {
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
