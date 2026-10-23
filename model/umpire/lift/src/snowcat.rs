use crate::{Lifted, Type};
use std::{collections::BTreeMap, fmt::Write};

pub(super) fn ty(value: &Type) -> String {
    match value {
        Type::Bool => "Bool".into(),
        Type::Int { .. } => "Int".into(),
        Type::Text | Type::Enum { .. } => "Str".into(),
        Type::Unit => "Seq(Int)".into(),
        Type::Seq(element) => format!("Seq({})", ty(element)),
        Type::Tuple(elements) => tuple(elements.iter().map(ty)),
    }
}
pub(super) fn tuple(types: impl IntoIterator<Item = String>) -> String {
    format!("<<{}>>", types.into_iter().collect::<Vec<_>>().join(", "))
}

/// Type the legacy relation without changing any executable TLA+ expression.
pub(super) fn annotate(model: &Lifted, source: &str) -> String {
    let mut types = BTreeMap::new();
    let mut variables: Vec<_> = model.fields.iter().map(|f| (f.name.clone(), ty(&f.ty))).collect();
    variables.extend([
        ("__umpire_edge_tick".into(), "Bool".into()),
        ("__umpire_pending".into(), "Int".into()),
        ("__umpire_steps".into(), "Int".into()),
        ("__umpire_action".into(), "Str".into()),
        ("__umpire_choice".into(), "Int".into()),
        ("__umpire_outcome".into(), ty(&model.outcome_type)),
        ("__umpire_facts".into(), "Seq(Seq(Int))".into()),
    ]);
    if model.has_inputs() {
        variables.push(("__umpire_input".into(), "Int".into()));
    }
    types.insert("vars".into(), tuple(variables.iter().map(|(_, t)| t.clone())));
    types.insert("ModelVars".into(), tuple(model.fields.iter().map(|f| ty(&f.ty))));
    for (name, value) in &model.constants {
        types.insert(model.names.constants.get(name).unwrap_or(name).clone(), ty(&value.ty));
    }
    for action in &model.actions {
        if !action.fields.is_empty() {
            let name = crate::names::operator(&action.name);
            types.insert(
                format!("{name}_Inputs"),
                format!(
                    "Seq({})",
                    tuple(
                        std::iter::once("Int".into())
                            .chain(action.fields.iter().map(|f| ty(&f.ty)))
                    )
                ),
            );
        }
    }
    let world: BTreeMap<_, _> = model
        .names
        .world
        .iter()
        .zip(&model.world_fields)
        .map(|(n, f)| (n.as_str(), ty(&f.ty)))
        .collect();
    let mut result = String::new();
    for line in source.lines() {
        if let Some(names) = line.strip_prefix("VARIABLES ") {
            for name in names.split(", ") {
                let kind =
                    variables.iter().find(|(n, _)| n == name).map_or("Bool", |(_, t)| t.as_str());
                let _ = writeln!(result, "VARIABLE \\* @type: {kind};\n{name}");
            }
        } else if let Some(names) = line.strip_prefix("CONSTANTS ") {
            for name in names.split(", ") {
                let _ = writeln!(result, "CONSTANT \\* @type: {};\n{name}", world[name]);
            }
        } else {
            if let Some((head, _)) = line.split_once(" ==")
                && !head.starts_with(' ')
                && !head.starts_with("\\*")
            {
                let name = head.split('(').next().unwrap_or(head);
                let kind = if name == "__umpire_Div" {
                    "(Int, Int) => Int".into()
                } else if head.contains('(') {
                    "Int => Bool".into()
                } else {
                    types.get(name).cloned().unwrap_or_else(|| "Bool".into())
                };
                let _ = writeln!(result, "\\* @type: {kind};");
            }
            result.push_str(line);
            result.push('\n');
        }
    }
    result
}
