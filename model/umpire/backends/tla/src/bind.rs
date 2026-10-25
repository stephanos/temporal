use crate::{Binding, Concrete, Diagnostic, Lifted, Literal, Type};
use umpire_front::{
    Diagnostic as FrontDiagnostic,
    expr::{Expr, ExprKind},
    tree::ActionRef,
};

/// Named values in schema order, as typed constants and as the literals they were admitted from.
type Typed = (Vec<Expr>, Vec<(String, Literal)>);

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
        let (world, world_literals) = typed(
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
        let selected = self
            .world_fields
            .iter()
            .zip(&world)
            .map(|(field, value)| (field.name.clone(), value.clone()))
            .collect();
        let mut bound = self.clone();
        for assumption in &mut bound.model.assumptions {
            *assumption = assumption
                .select(&selected)
                .map_err(|error| self.refuse(&error, error.message.clone()))?;
        }
        for claim in &mut bound.model.progress {
            claim.assumption = claim
                .assumption
                .select(&selected)
                .map_err(|error| self.refuse(&error, error.message.clone()))?;
        }
        let mut starts = Vec::new();
        let mut start_literals = Vec::new();
        for row in &binding.starts {
            let (state, literals) = typed(
                &self.file,
                self.fields.iter().map(|field| (&field.name, &field.ty, &field.location)),
                row,
                "complete start",
            )?;
            if !starts.contains(&state) {
                starts.push(state);
                start_literals.push(literals);
            }
        }
        let (catalog, actions) = bound.bind_catalog(binding)?;
        bound.world_values = Some(world);
        bound.complete_starts = Some(starts);
        bound.catalog = Some(catalog);
        bound.binding = Some(Binding { world: world_literals, starts: start_literals, actions });
        Ok(bound)
    }

    /// The native catalog as typed entries, whose positional duplicates share their first
    /// ordinal, and as admitted concrete identities in catalog order.
    fn bind_catalog(&self, binding: &Binding) -> Result<(Vec<Entry>, Vec<Concrete>), Diagnostic> {
        let mut catalog: Vec<Entry> = Vec::new();
        let mut actions = Vec::new();
        for native in &binding.actions {
            let Some(index) = self.actions.iter().position(|action| action.name == native.action)
            else {
                return Err(self.refuse(
                    &self.location,
                    format!("native catalog action `{}` is not declared", native.action),
                ));
            };
            let (values, fields) = typed(
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
            actions.push(Concrete { action: native.action.clone(), fields });
        }
        self.scoped_catalog(&catalog)?;
        Ok((catalog, actions))
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
        for (kind, name, assumption) in self
            .assumptions
            .iter()
            .map(|assumption| ("assumption", &assumption.name, assumption))
            .chain(self.progress.iter().map(|claim| ("progress", &claim.name, &claim.assumption)))
        {
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
                            "{kind} `{name}` names `{spelled}`, which the selected World's catalog lacks"
                        ),
                    ));
                }
            }
            if (assumption.only.is_some() || !assumption.without.is_empty())
                && !references.iter().any(|reference| assumption.allows(reference))
            {
                return Err(self.refuse(
                    &self.location,
                    format!("{kind} `{name}` assumption admits no catalog action"),
                ));
            }
        }
        Ok(())
    }

    /// The bound catalog as typed concrete identities.
    pub(crate) fn concrete(&self) -> Vec<Concrete> {
        self.binding.iter().flat_map(|binding| binding.actions.clone()).collect()
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
) -> Result<Typed, Diagnostic> {
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
            let admitted = value.1.admit(ty).map_err(|_| {
                message(
                    file,
                    location,
                    format!("{kind} field `{name}` value does not match admitted type {ty:?}"),
                )
            })?;
            Ok((Expr::constant(ty, &admitted), value.clone()))
        })
        .collect::<Result<Vec<_>, _>>()
        .map(|typed| typed.into_iter().unzip())
}

fn cfg_representable(value: &Expr) -> bool {
    match &value.kind {
        ExprKind::Int(value) => (0..=i128::from(i32::MAX)).contains(value),
        ExprKind::Sequence(values) => values.iter().all(cfg_representable),
        _ => true,
    }
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
