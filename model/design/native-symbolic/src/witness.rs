//! Decode a BTOR2 witness and replay it through the native Model.
use crate::btor2::{Kind, Layout, Property, Shape};
use std::collections::BTreeMap;
use std::fmt::{Display, Write as _};
use umpire_dsl::{Machine, Value, Values, Verdict};
use umpire_ir::Type;

type Id = u64;

#[derive(Default, Debug, Clone)]
pub struct Frame {
    pub states: BTreeMap<Id, String>,
    pub inputs: BTreeMap<Id, String>,
}

/// Frames of a `sat` witness: full state assignments per frame and the inputs taken from it.
pub fn parse(text: &str) -> Result<Vec<Frame>, String> {
    let mut frames: Vec<Frame> = Vec::new();
    let mut section = None; // Some(true) = states, Some(false) = inputs
    let mut seen_sat = false;
    for line in text.lines() {
        let line = line.trim();
        if line == "sat" {
            seen_sat = true;
            continue;
        }
        if !seen_sat
            || line.is_empty()
            || line.starts_with('b')
            || line.starts_with('j')
            || line.starts_with(';')
        {
            continue;
        }
        if line == "." {
            break;
        }
        if let Some(k) = line.strip_prefix('#') {
            let k: usize = k.parse().map_err(|_| format!("bad frame marker {line}"))?;
            while frames.len() <= k {
                frames.push(Frame::default());
            }
            section = Some(true);
            continue;
        }
        if let Some(k) = line.strip_prefix('@') {
            let k: usize = k.parse().map_err(|_| format!("bad frame marker {line}"))?;
            while frames.len() <= k {
                frames.push(Frame::default());
            }
            section = Some(false);
            continue;
        }
        let mut parts = line.split_whitespace();
        let (Some(id), Some(bits)) = (parts.next(), parts.next()) else { continue };
        if bits.starts_with('[') {
            continue; // array assignment: not used by this encoding
        }
        let id: Id = id.parse().map_err(|_| format!("bad assignment line {line}"))?;
        let frame = frames.last_mut().ok_or("assignment before any frame")?;
        match section {
            Some(true) => frame.states.insert(id, bits.to_owned()),
            Some(false) => frame.inputs.insert(id, bits.to_owned()),
            None => return Err(format!("assignment outside a section: {line}")),
        };
    }
    if !seen_sat {
        return Err("witness has no `sat` line".into());
    }
    Ok(frames)
}

#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Lit {
    Bool(bool),
    Int(i128),
    Enum(String),
}

impl Display for Lit {
    fn fmt(&self, out: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Bool(b) => write!(out, "{b}"),
            Self::Int(i) => write!(out, "{i}"),
            Self::Enum(e) => write!(out, "{e}"),
        }
    }
}

fn decode_bits(bits: &str, shape: Shape) -> Result<i128, String> {
    let raw = u128::from_str_radix(bits, 2).map_err(|_| format!("bad bits {bits}"))?;
    if shape.signed && shape.width < 128 && raw & (1u128 << (shape.width - 1)) != 0 {
        let mask = (1u128 << shape.width) - 1;
        let magnitude = ((!raw) & mask) + 1;
        return Ok(-(magnitude as i128));
    }
    Ok(raw as i128)
}

fn decode(bits: &str, shape: Shape, ty: &Type) -> Result<Lit, String> {
    let value = decode_bits(bits, shape)?;
    Ok(match ty {
        Type::Bool => Lit::Bool(value != 0),
        Type::Int { .. } => Lit::Int(value),
        Type::Enum { variants, .. } => Lit::Enum(
            variants
                .get(usize::try_from(value).map_err(|_| "negative enum index")?)
                .ok_or("enum index out of range")?
                .clone(),
        ),
        other => return Err(format!("undecodable type {other:?}")),
    })
}

fn native(values: &[(&'static str, Value)]) -> Result<Vec<(String, Lit)>, String> {
    values
        .iter()
        .map(|(name, value)| {
            let lit = match value {
                Value::Bool(b) => Lit::Bool(*b),
                Value::Int(i) => Lit::Int(*i),
                Value::Enum(e) => Lit::Enum((*e).to_owned()),
                other => {
                    return Err(format!("unsupported native witness value for {name}: {other:?}"));
                }
            };
            Ok(((*name).to_owned(), lit))
        })
        .collect()
}

/// The decoded state fields of one frame, when the witness lists every field.
fn frame_state(frame: &Frame, layout: &Layout) -> Result<Option<Vec<(String, Lit)>>, String> {
    let mut out = Vec::new();
    for (name, id, shape, ty) in &layout.fields {
        let Some(bits) = frame.states.get(id) else { return Ok(None) };
        out.push((name.clone(), decode(bits, *shape, ty)?));
    }
    Ok(Some(out))
}

pub struct Confirmed {
    pub path: String,
    pub steps: usize,
    pub confirmed: Result<(), String>,
}

/// Replays `frames` through `checked_step` from a declared start and judges `property` natively
/// on the final state or edge. A frame that is not a Model path is a rejection.
pub fn replay<M: Machine>(
    world: &M,
    layout: &Layout,
    frames: &[Frame],
    property: &Property,
    negatives: &[(&str, &dyn Fn(&M::State) -> bool)],
) -> Result<Confirmed, String>
where
    M::State: Values,
{
    let catalog = world.actions();
    for (index, action) in catalog.iter().enumerate() {
        if layout.actions.get(index).map(String::as_str) != Some(action.to_string().as_str()) {
            return Err(format!("catalog order differs at {index}: native {action}"));
        }
    }
    let first = frames.first().ok_or("empty witness")?;
    let initial = frame_state(first, layout)?.ok_or("frame 0 does not assign every state field")?;
    let mut state = world
        .starts()
        .into_iter()
        .map(|start| native(&start.values()).map(|values| (start, values)))
        .collect::<Result<Vec<_>, _>>()?
        .into_iter()
        .find(|(_, values)| *values == initial)
        .map(|(start, _)| start)
        .ok_or_else(|| format!("initial state {initial:?} is not a declared start"))?;
    let mut path = format!("start: {state}\n");
    let mut steps = 0usize;
    let mut last_edge = None;
    let act_shape =
        Shape { width: (usize::BITS - catalog.len().leading_zeros()).max(1), signed: false };
    for k in 1..frames.len() {
        let bits = frames[k - 1]
            .inputs
            .get(&layout.act)
            .ok_or_else(|| format!("frame {} has no act input", k - 1))?;
        let act = decode_bits(bits, act_shape)? as usize;
        let decoded = frame_state(&frames[k], layout)?;
        if act >= catalog.len() {
            if let Some(decoded) = &decoded {
                if native(&state.values())? != *decoded {
                    return Err(format!("frame {k}: no action chosen but the state changed"));
                }
            }
            continue;
        }
        let action = catalog[act].clone();
        let edges = world
            .checked_step(&state, action.clone())
            .map_err(|e| format!("frame {k}: {action}: {e}"))?;
        if edges.is_empty() {
            if let Some(decoded) = &decoded {
                if native(&state.values())? != *decoded {
                    return Err(format!(
                        "frame {k}: {action} is disabled natively but the witness steps"
                    ));
                }
            }
            continue;
        }
        let choice = match &decoded {
            Some(decoded) => edges
                .iter()
                .map(|(_, _, next)| native(&next.values()))
                .collect::<Result<Vec<_>, _>>()?
                .iter()
                .position(|values| values == decoded)
                .ok_or_else(|| format!("frame {k}: {action} has no native edge to {decoded:?}"))?,
            None => 0,
        };
        let before = state.clone();
        state = edges[choice].2.clone();
        steps += 1;
        let _ = writeln!(path, "{steps}: {action} choice={choice} -> {state}");
        last_edge = Some((before, action, choice, edges));
    }
    let confirmed = match property.kind {
        Kind::Invariant => {
            let claim = world
                .invariants()
                .into_iter()
                .find(|c| c.name == property.name)
                .ok_or("no such invariant")?;
            match (claim.predicate)(world, &state) {
                Verdict::Fails => Ok(()),
                other => Err(format!("native invariant verdict {other:?} on the final state")),
            }
        }
        Kind::Negative => {
            let (_, judge) = negatives
                .iter()
                .find(|(name, _)| *name == property.name)
                .ok_or("no such negative claim")?;
            if judge(&state) {
                Err("native negative claim holds on the final state".into())
            } else {
                Ok(())
            }
        }
        Kind::Postcondition => {
            let claim = world
                .postconditions()
                .into_iter()
                .find(|c| c.name == property.name)
                .ok_or("no such postcondition")?;
            let (before, action, choice, edges) =
                last_edge.ok_or("postcondition violation without a step")?;
            match claim.holds(world, &before, &action, &edges[choice]) {
                Verdict::Fails => Ok(()),
                other => Err(format!("native postcondition verdict {other:?} on the final edge")),
            }
        }
        Kind::Arithmetic => Err("arithmetic fault replay is outside the spike".into()),
    };
    Ok(Confirmed { path, steps, confirmed })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn unsupported_native_values_refuse_witness_comparison() {
        for value in [
            Value::Bounded { name: "Count", raw: 1 },
            Value::Optional(None),
            Value::Sequence(vec![]),
        ] {
            assert!(
                native(&[("field", value)])
                    .unwrap_err()
                    .contains("unsupported native witness value for field")
            );
        }
    }
}
