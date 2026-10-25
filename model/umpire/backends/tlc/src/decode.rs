use crate::{Refusal, Schema, State, TraceState, Value};

/// Decode every numbered state and require the independently reported distinct-state count.
pub fn decode_dump(
    text: &str,
    expected: usize,
    schema: &Schema,
) -> Result<Vec<TraceState>, Refusal> {
    let mut states = Vec::new();
    let mut current = String::new();
    let mut started = false;
    for line in text.lines() {
        if let Some(number) = line.strip_prefix("State ").and_then(|line| line.strip_suffix(':')) {
            if started {
                states.push(decode_state(&current, schema)?);
                current.clear();
            }
            if number.parse::<usize>().ok() != Some(states.len() + 1) {
                return Err(Refusal("nonconsecutive dump state numbers".into()));
            }
            started = true;
        } else {
            if line.trim().is_empty() {
                continue;
            }
            if !started {
                return Err(Refusal("dump assignment has no numbered state header".into()));
            }
            current.push_str(line);
            current.push('\n');
        }
    }
    if started {
        states.push(decode_state(&current, schema)?);
    }
    if states.len() != expected {
        return Err(Refusal(format!(
            "incomplete dump: decoded {}, TLC reported {expected}",
            states.len()
        )));
    }
    Ok(states)
}

/// Decode one assignment, retaining declared observers separately for trace replay.
pub fn decode_state(text: &str, schema: &Schema) -> Result<TraceState, Refusal> {
    let mut parser = Parser(text.trim());
    let mut fields = State::new();
    loop {
        parser.eat("/\\");
        if parser.0.is_empty() {
            break;
        }
        let name = parser.identifier()?;
        parser.require("=")?;
        let value = parser.value()?;
        if fields.insert(name.clone(), value).is_some() {
            return Err(Refusal(format!("duplicate dump field {name}")));
        }
        if !parser.0.is_empty() && !parser.0.starts_with("/\\") {
            return Err(Refusal(format!("trailing dump syntax: {}", parser.0)));
        }
    }
    schema.validate()?;
    schema.project_assignment(fields)
}

struct Parser<'a>(&'a str);

impl Parser<'_> {
    fn eat(&mut self, token: &str) -> bool {
        if let Some(rest) = self.0.strip_prefix(token) {
            self.0 = rest.trim_start();
            true
        } else {
            false
        }
    }

    fn require(&mut self, token: &str) -> Result<(), Refusal> {
        if self.eat(token) {
            Ok(())
        } else {
            Err(Refusal(format!("expected {token:?} in dump: {}", self.0)))
        }
    }

    fn identifier(&mut self) -> Result<String, Refusal> {
        let end =
            self.0.find(|c: char| !c.is_ascii_alphanumeric() && c != '_').unwrap_or(self.0.len());
        if end == 0 {
            return Err(Refusal(format!("expected dump identifier: {}", self.0)));
        }
        let name = self.0[..end].to_owned();
        self.0 = self.0[end..].trim_start();
        Ok(name)
    }

    fn value(&mut self) -> Result<Value, Refusal> {
        if self.eat("TRUE") {
            return Ok(Value::Bool(true));
        }
        if self.eat("FALSE") {
            return Ok(Value::Bool(false));
        }
        if let Some(rest) = self.0.strip_prefix('"') {
            self.0 = rest;
            let mut value = String::new();
            let mut escaped = false;
            for (offset, character) in self.0.char_indices() {
                if escaped {
                    value.push(match character {
                        'n' => '\n',
                        'r' => '\r',
                        't' => '\t',
                        '"' => '"',
                        '\\' => '\\',
                        _ => return Err(Refusal(format!("unsupported string escape {character}"))),
                    });
                    escaped = false;
                } else if character == '\\' {
                    escaped = true;
                } else if character == '"' {
                    self.0 = self.0[offset + 1..].trim_start();
                    return Ok(Value::Text(value));
                } else {
                    value.push(character);
                }
            }
            return Err(Refusal("unterminated dump string".into()));
        }
        if self.eat("<<") {
            return self.list(">>").map(Value::Seq);
        }
        let end = self
            .0
            .char_indices()
            .take_while(|(index, c)| c.is_ascii_digit() || (*index == 0 && *c == '-'))
            .last()
            .map_or(0, |(index, _)| index + 1);
        if end == 0 {
            return Err(Refusal(format!("unsupported dump value: {}", self.0)));
        }
        let value = self.0[..end]
            .parse()
            .map_err(|error| Refusal(format!("invalid dump integer: {error}")))?;
        self.0 = self.0[end..].trim_start();
        Ok(Value::Int(value))
    }

    fn list(&mut self, end: &str) -> Result<Vec<Value>, Refusal> {
        let mut values = Vec::new();
        if !self.eat(end) {
            loop {
                values.push(self.value()?);
                if self.eat(end) {
                    break;
                }
                self.require(",")?;
            }
        }
        Ok(values)
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::collections::BTreeMap;
    use umpire_tla::Type;

    #[test]
    fn observer_only_safety_assignment_keeps_empty_state_refusal() {
        let schema = Schema {
            fields: BTreeMap::new(),
            observers: BTreeMap::from([("flag".into(), Type::Bool)]),
        };
        assert!(decode_state("flag = TRUE", &schema).is_err());
    }

    #[test]
    fn string_and_collection_values_preserve_authored_contents() -> Result<(), Refusal> {
        let schema = Schema {
            fields: BTreeMap::from([(
                "value".into(),
                Type::Tuple(vec![
                    Type::Text,
                    Type::Bool,
                    Type::Seq(Box::new(Type::Int { min: -10, max: 10 })),
                ]),
            )]),
            observers: BTreeMap::new(),
        };
        let state = decode_state(
            "value = <<\"  whitespace\\n\\\"quoted\\\"  \", TRUE, <<-1, 2>>>>",
            &schema,
        )?;
        assert_eq!(
            state.state["value"],
            Value::Seq(vec![
                Value::Text("  whitespace\n\"quoted\"  ".into()),
                Value::Bool(true),
                Value::Seq(vec![Value::Int(-1), Value::Int(2)])
            ])
        );
        Ok(())
    }

    #[test]
    fn typed_dump_rejects_missing_extra_wrong_type_and_truncation() -> Result<(), Refusal> {
        let schema = Schema {
            fields: BTreeMap::from([("x".into(), Type::Int { min: -2, max: 2 })]),
            observers: BTreeMap::from([("tick".into(), Type::Bool)]),
        };
        let valid =
            "State 1:\n/\\ x = -2\n/\\ tick = TRUE\n\nState 2:\n/\\ x = 2\n/\\ tick = FALSE\n";
        let states = decode_dump(valid, 2, &schema)?;
        assert_eq!(states[0].state, BTreeMap::from([("x".into(), Value::Int(-2))]));
        assert_eq!(states[1].observers, BTreeMap::from([("tick".into(), Value::Bool(false))]));
        for invalid in [
            "/\\ x = 1",
            "/\\ x = 1 /\\ tick = TRUE /\\ extra = 0",
            "/\\ x = 3 /\\ tick = TRUE",
            "/\\ x = TRUE /\\ tick = TRUE",
            "/\\ x = 1 /\\ tick =",
            "/\\ x = 1 /\\ x = 2 /\\ tick = TRUE",
        ] {
            assert!(decode_state(invalid, &schema).is_err(), "accepted {invalid}");
        }
        assert!(decode_dump(valid, 3, &schema).is_err());
        assert!(decode_dump("/\\ x = 1 /\\ tick = TRUE", 1, &schema).is_err());
        assert!(decode_dump("State 1:\n", 0, &schema).is_err());
        for retired in ["/\\ x = {1} /\\ tick = TRUE", "/\\ x = [a |-> 1] /\\ tick = TRUE"] {
            assert!(decode_state(retired, &schema).is_err(), "accepted {retired}");
        }
        let collision = Schema { fields: schema.fields.clone(), observers: schema.fields };
        assert!(collision.validate().is_err());
        Ok(())
    }
}
