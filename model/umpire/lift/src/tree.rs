use crate::{
    Binding, CATALOG_LIMIT, Concrete, Diagnostic, Field, Literal, Type,
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
    /// The bound concrete catalog, in native catalog order.
    pub(crate) catalog: Option<Vec<Entry>>,
    /// The generated TLA+ name of a catalog position inside input relations.
    pub(crate) index_symbol: String,
}
/// One concrete catalog action: an index into `Lifted::actions` and its payload values.
#[derive(Clone, Debug)]
pub(crate) struct Entry {
    pub action: usize,
    pub values: Vec<Expr>,
    /// The first catalog ordinal with the same identity, so positional duplicates share one
    /// observable edge as they share one native catalog equality class.
    pub ordinal: usize,
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
        let catalog = self.bind_catalog(binding)?;
        let mut bound = self.clone();
        bound.world_values = Some(world);
        bound.complete_starts = Some(starts);
        bound.catalog = Some(catalog);
        Ok(bound)
    }

    /// Whether any declared action variant carries payload fields.
    pub(crate) fn has_inputs(&self) -> bool {
        self.actions.iter().any(|action| !action.fields.is_empty())
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
                return Err(message(
                    &self.location,
                    format!("extra action domain `{}` field `{}`", domain.action, domain.field),
                ));
            };
            if supplied.insert((domain.action.as_str(), domain.field.as_str()), domain).is_some() {
                return Err(message(
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
                                message(
                                    &field.location,
                                    format!(
                                        "missing action domain `{}` field `{}`",
                                        action.name, field.name
                                    ),
                                )
                            })?;
                        if domain.values.len() > CATALOG_LIMIT {
                            return Err(message(
                                &field.location,
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
                                    message(
                                        &field.location,
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
            None => return Err(message(&self.location, "action catalog count overflows")),
            Some(count) if count > CATALOG_LIMIT => {
                return Err(message(
                    &self.location,
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
            return Err(message(
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
            compare(ordinal, native, action, &entry.values)?;
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
                    return Err(message(
                        &self.location,
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
                return Err(message(
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
    ordinal: usize,
    native: &Concrete,
    action: &ActionDef,
    values: &[Expr],
) -> Result<(), Diagnostic> {
    if native.action != action.name {
        return Err(message(
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
                &action.location,
                format!("extra input field `{name}` at native catalog ordinal {ordinal}"),
            ));
        };
        if !seen.insert(name) {
            return Err(message(
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
            &missing.location,
            format!("missing input field `{}` at native catalog ordinal {ordinal}", missing.name),
        ));
    }
    for (((name, value), field), expected) in native.fields.iter().zip(&action.fields).zip(values) {
        if name != &field.name {
            return Err(message(
                &field.location,
                format!(
                    "native catalog ordinal {ordinal} orders input field `{name}` where the source declares `{}`",
                    field.name
                ),
            ));
        }
        if literal(value, &field.ty).as_ref() != Some(expected) {
            return Err(message(
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
    pub shape: Shape,
    pub fields: Vec<InputField>,
    pub location: Diagnostic,
}
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub(crate) enum Shape {
    Unit,
    Tuple,
    Named,
}
/// One payload field of an action variant, with its independently admitted domain.
#[derive(Clone, Debug)]
pub(crate) struct InputField {
    /// Field name, or its position in a tuple variant.
    pub name: String,
    pub ty: Type,
    pub domain: InputDomain,
    /// The TLA+ name of the chosen value inside the variant's relation.
    pub symbol: String,
    pub location: Diagnostic,
}
#[derive(Clone, Debug)]
pub(crate) enum InputDomain {
    /// Values the source declares, in order, including positional duplicates.
    Values(Vec<Expr>),
    /// A World-dependent `#[domain]` function, whose values the selected binding supplies.
    World,
}
/// A concrete action named in an assumption: a variant and its exact payload values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub(crate) struct ActionRef {
    pub name: String,
    pub values: Vec<Expr>,
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
    pub only: Option<Vec<ActionRef>>,
    pub without: Vec<ActionRef>,
    pub fair: Vec<ActionRef>,
}
impl AssumptionDef {
    pub(crate) fn same_scope(&self, other: &Self) -> bool {
        self.only == other.only && self.without == other.without && self.fair == other.fair
    }
    /// Whether a concrete action is admitted, comparing exact values as native scopes do.
    pub(crate) fn allows(&self, action: &ActionRef) -> bool {
        self.only.as_ref().is_none_or(|only| only.contains(action))
            && !self.without.contains(action)
    }
    pub(crate) fn is_fair(&self, action: &ActionRef) -> bool {
        self.fair.contains(action)
    }
}
