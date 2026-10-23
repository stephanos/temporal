//! Memory-only lexical admission before disabling serde's smaller recursion
//! limit. A value's root is depth zero; an empty container has no child value.

pub(super) fn first(bytes: &[u8], maximum: usize) -> Result<usize, String> {
    let mut parser = Parser {
        bytes,
        offset: 0,
        maximum,
    };
    parser.value(0)?;
    Ok(parser.offset)
}

struct Parser<'a> {
    bytes: &'a [u8],
    offset: usize,
    maximum: usize,
}

impl Parser<'_> {
    fn space(&mut self) {
        while self
            .bytes
            .get(self.offset)
            .is_some_and(|byte| matches!(byte, b' ' | b'\t' | b'\r' | b'\n'))
        {
            self.offset += 1;
        }
    }

    fn take(&mut self, byte: u8) -> bool {
        self.space();
        if self.bytes.get(self.offset) == Some(&byte) {
            self.offset += 1;
            true
        } else {
            false
        }
    }

    fn syntax<T>(&self) -> Result<T, String> {
        Err(format!(
            "qualification JSON: invalid syntax at byte {}",
            self.offset
        ))
    }

    fn string(&mut self) -> Result<String, String> {
        self.space();
        let start = self.offset;
        if !self.take(b'"') {
            return self.syntax();
        }
        while let Some(&byte) = self.bytes.get(self.offset) {
            self.offset += 1;
            if byte == b'\\' {
                self.offset += usize::from(self.offset < self.bytes.len());
            } else if byte == b'"' {
                return serde_json::from_slice::<String>(&self.bytes[start..self.offset])
                    .map_err(|error| format!("qualification JSON: {error}"));
            }
        }
        self.syntax()
    }

    fn literal(&mut self, literal: &[u8]) -> Result<(), String> {
        if self.bytes.get(self.offset..self.offset + literal.len()) != Some(literal) {
            return self.syntax();
        }
        self.offset += literal.len();
        Ok(())
    }

    fn digits(&mut self) -> Result<(), String> {
        let start = self.offset;
        while self.bytes.get(self.offset).is_some_and(u8::is_ascii_digit) {
            self.offset += 1;
        }
        if self.offset == start {
            self.syntax()
        } else {
            Ok(())
        }
    }

    fn number(&mut self) -> Result<(), String> {
        if self.bytes.get(self.offset) == Some(&b'-') {
            self.offset += 1;
        }
        match self.bytes.get(self.offset) {
            Some(b'0') => self.offset += 1,
            Some(b'1'..=b'9') => self.digits()?,
            _ => return self.syntax(),
        }
        if self.bytes.get(self.offset) == Some(&b'.') {
            self.offset += 1;
            self.digits()?;
        }
        if matches!(self.bytes.get(self.offset), Some(b'e' | b'E')) {
            self.offset += 1;
            if matches!(self.bytes.get(self.offset), Some(b'+' | b'-')) {
                self.offset += 1;
            }
            self.digits()?;
        }
        Ok(())
    }

    fn value(&mut self, depth: usize) -> Result<(), String> {
        if depth > self.maximum {
            return Err("qualification JSON nesting limit".into());
        }
        self.space();
        match self.bytes.get(self.offset) {
            Some(b'[') => {
                self.offset += 1;
                if self.take(b']') {
                    return Ok(());
                }
                loop {
                    self.value(depth + 1)?;
                    if self.take(b']') {
                        return Ok(());
                    }
                    if !self.take(b',') {
                        return self.syntax();
                    }
                }
            }
            Some(b'{') => {
                self.offset += 1;
                if self.take(b'}') {
                    return Ok(());
                }
                let mut names = std::collections::BTreeSet::new();
                loop {
                    let key = self.string()?;
                    if !names.insert(key) {
                        return Err("qualification JSON: duplicate object key".into());
                    }
                    if depth == self.maximum {
                        return Err("qualification JSON nesting limit".into());
                    }
                    if !self.take(b':') {
                        return self.syntax();
                    }
                    self.value(depth + 1)?;
                    if self.take(b'}') {
                        return Ok(());
                    }
                    if !self.take(b',') {
                        return self.syntax();
                    }
                }
            }
            Some(b'"') => self.string().map(|_| ()),
            Some(b't') => self.literal(b"true"),
            Some(b'f') => self.literal(b"false"),
            Some(b'n') => self.literal(b"null"),
            Some(b'-' | b'0'..=b'9') => self.number(),
            Some(_) | None => self.syntax(),
        }
    }
}
