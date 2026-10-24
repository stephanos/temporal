use crate::{BytesRules, EnumRules, GenerationError, Kind, NumberRules, StringRules};

pub(crate) fn negative_kind(kind: &Kind, constraint: &str) -> Result<Kind, GenerationError> {
    let mut negative = kind.clone();
    match &mut negative {
        Kind::DefinedEnum(rule) if constraint == "enum.defined_only" => {
            let value = (i32::MAX - 4096..=i32::MAX)
                .find(|n| !(rule.is_valid)(*n))
                .ok_or(GenerationError::SizeLimit)?;
            negative = Kind::Enum(EnumRules { defined_only: true, values: vec![value] });
        }
        Kind::Uuid if constraint == "string.uuid" => {
            negative = Kind::Text(StringRules {
                min_len: Some(1),
                max_len: Some(1),
                ..Default::default()
            });
        }
        Kind::Identifier(rules) | Kind::Text(rules) => {
            let identifier = matches!(kind, Kind::Identifier(_));
            match constraint {
                "string.min_len" => {
                    let min = rules
                        .min_len
                        .unwrap_or(usize::from(identifier))
                        .max(usize::from(identifier));
                    rules.min_len = None;
                    rules.max_len =
                        Some(min.checked_sub(1).ok_or(GenerationError::ImpossibleConstraints)?);
                    negative = Kind::Text(rules.clone());
                }
                "string.max_len" => {
                    rules.min_len = Some(successor(rules.max_len)?);
                    rules.max_len = None;
                }
                "string.min_bytes" => {
                    rules.max_bytes = Some(predecessor(rules.min_bytes)?);
                    rules.min_bytes = None;
                }
                "string.max_bytes" => {
                    rules.min_bytes = Some(successor(rules.max_bytes)?);
                    rules.max_bytes = None;
                }
                "string.prefix" => {
                    rules.prefix = None;
                }
                "string.suffix" => {
                    rules.suffix = None;
                }
                "string.contains" => {
                    rules.contains = None;
                }
                _ => return Err(GenerationError::UnsupportedType),
            }
        }
        Kind::Bytes(rules) | Kind::Random(rules) => invert_bytes(rules, constraint)?,
        Kind::Counter(rules)
        | Kind::Number(rules)
        | Kind::Duration(rules)
        | Kind::Timestamp(rules) => {
            invert_number(rules, constraint)?;
        }
        Kind::Enum(rules) if constraint == "enum.defined_only" => {
            let mut value = i32::MIN;
            while rules.values.contains(&value) {
                value = value.checked_add(1).ok_or(GenerationError::ImpossibleConstraints)?;
            }
            *rules = EnumRules { defined_only: true, values: vec![value] };
        }
        _ => return Err(GenerationError::UnsupportedType),
    }
    Ok(negative)
}

fn successor(value: Option<usize>) -> Result<usize, GenerationError> {
    value.and_then(|value| value.checked_add(1)).ok_or(GenerationError::ImpossibleConstraints)
}

fn predecessor(value: Option<usize>) -> Result<usize, GenerationError> {
    value.and_then(|value| value.checked_sub(1)).ok_or(GenerationError::ImpossibleConstraints)
}

fn invert_bytes(rules: &mut BytesRules, constraint: &str) -> Result<(), GenerationError> {
    match constraint {
        "bytes.min_len" => {
            rules.max_len = Some(predecessor(rules.min_len)?);
            rules.min_len = None;
        }
        "bytes.max_len" => {
            rules.min_len = Some(successor(rules.max_len)?);
            rules.max_len = None;
        }
        _ => return Err(GenerationError::UnsupportedType),
    }
    Ok(())
}

fn invert_number(rules: &mut NumberRules, constraint: &str) -> Result<(), GenerationError> {
    let bound = constraint.rsplit('.').next().ok_or(GenerationError::UnsupportedType)?;
    let value = match bound {
        "gt" => rules.gt,
        "gte" => rules.gte.and_then(|value| value.checked_sub(1)),
        "lt" => rules.lt,
        "lte" => rules.lte.and_then(|value| value.checked_add(1)),
        _ => return Err(GenerationError::UnsupportedType),
    }
    .ok_or(GenerationError::ImpossibleConstraints)?;
    *rules = NumberRules { gte: Some(value), lte: Some(value), ..Default::default() };
    Ok(())
}
