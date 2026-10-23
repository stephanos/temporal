use crate::install::manifest;
use std::{
    collections::BTreeMap,
    fs,
    path::{Component, Path, PathBuf},
};

#[cfg(test)]
#[path = "repo_tests.rs"]
mod tests;

#[derive(Clone, Copy, Debug, PartialEq, Eq, serde::Serialize)]
#[serde(rename_all = "lowercase")]
pub(crate) enum Derivation {
    Temporal,
    Default,
}

#[derive(Debug)]
pub(crate) struct Repo {
    pub module_root: PathBuf,
    pub runtime_root: PathBuf,
    pub derive: Derivation,
    pub guest: Vec<String>,
}

impl Repo {
    pub(crate) fn discover(cwd: &Path) -> Result<Self, String> {
        Self::optional(cwd)?.ok_or_else(|| {
            "guest auto-installation requires gomad.toml at the Go module root".into()
        })
    }

    pub(crate) fn optional(cwd: &Path) -> Result<Option<Self>, String> {
        let cwd = fs::canonicalize(cwd).map_err(|e| e.to_string())?;
        let mut module_root = None;
        for root in cwd.ancestors() {
            match fs::symlink_metadata(root.join("go.mod")) {
                Ok(_) => {
                    module_root = Some(root.to_path_buf());
                    break;
                }
                Err(error) if error.kind() == std::io::ErrorKind::NotFound => {}
                Err(error) => return Err(error.to_string()),
            }
        }
        let Some(module_root) = module_root else {
            return Ok(None);
        };
        manifest::read(&module_root.join("go.mod"), 16 << 20)?;
        match fs::symlink_metadata(module_root.join("gomad.toml")) {
            Err(error) if error.kind() == std::io::ErrorKind::NotFound => return Ok(None),
            Err(error) => return Err(error.to_string()),
            Ok(_) => {}
        }
        let raw = manifest::read(&module_root.join("gomad.toml"), 64 << 10)?;
        let values = Parser::new(std::str::from_utf8(&raw).map_err(|_| "gomad.toml is not UTF-8")?)
            .document()?;
        let Some(Value::String(runtime)) = values.get("runtime_root") else {
            return Err("gomad.toml requires a runtime_root string".into());
        };
        let derive = match values.get("derive") {
            None => Derivation::Default,
            Some(Value::String(value)) if value == "default" => Derivation::Default,
            Some(Value::String(value)) if value == "temporal" => Derivation::Temporal,
            _ => return Err("gomad.toml derive must be temporal or default".into()),
        };
        let runtime_root = contained(&module_root, runtime)?;
        let guest = match values.get("guest") {
            None => Vec::new(),
            Some(Value::Array(values)) => values
                .iter()
                .map(|value| {
                    contained(&module_root, value)?;
                    relative(value)
                })
                .collect::<Result<Vec<_>, String>>()?,
            _ => return Err("gomad.toml guest must be a string array".into()),
        };
        Ok(Some(Self {
            module_root,
            runtime_root,
            derive,
            guest,
        }))
    }
}

pub(crate) fn relative(value: &str) -> Result<String, String> {
    if value.is_empty()
        || value.contains(['\\', '\0'])
        || value.contains("...")
        || Path::new(value).is_absolute()
    {
        return Err("gomad.toml paths must be module-relative package or runtime paths".into());
    }
    let mut parts = Vec::new();
    for component in Path::new(value).components() {
        match component {
            Component::Normal(part) => parts.push(part.to_str().ok_or("non-UTF-8 path")?),
            Component::CurDir => {}
            Component::ParentDir => {
                parts.pop().ok_or("gomad.toml path escapes module")?;
            }
            _ => return Err("gomad.toml path escapes module".into()),
        }
    }
    Ok(if parts.is_empty() {
        ".".into()
    } else {
        parts.join("/")
    })
}

pub(crate) fn contained(root: &Path, value: &str) -> Result<PathBuf, String> {
    let relative = relative(value)?;
    let mut path = root.to_path_buf();
    // Check each prefix before normalizing away a symlink followed by '..'.
    for part in Path::new(value).components() {
        path.push(part.as_os_str());
        let resolved = fs::canonicalize(&path).map_err(|e| e.to_string())?;
        if !resolved.starts_with(root) {
            return Err("gomad.toml path resolves outside module".into());
        }
    }
    let path = fs::canonicalize(root.join(relative)).map_err(|e| e.to_string())?;
    if !path.starts_with(root) || !path.is_dir() {
        return Err("gomad.toml path must be a directory within the module".into());
    }
    Ok(path)
}

#[derive(Debug, PartialEq, Eq)]
enum Value {
    String(String),
    Array(Vec<String>),
}

struct Parser<'a> {
    text: &'a str,
    index: usize,
}
impl<'a> Parser<'a> {
    fn new(text: &'a str) -> Self {
        Self { text, index: 0 }
    }
    fn peek(&self) -> Option<char> {
        self.text[self.index..].chars().next()
    }
    fn next(&mut self) -> Option<char> {
        let value = self.peek()?;
        self.index += value.len_utf8();
        Some(value)
    }
    fn spaces(&mut self) {
        while matches!(self.peek(), Some(' ' | '\t')) {
            self.next();
        }
    }
    fn newline(&mut self) -> Result<(), String> {
        if self.next() == Some('\r') && self.next() != Some('\n') {
            return Err("gomad.toml has a bare carriage return".into());
        }
        Ok(())
    }
    fn comment(&mut self) -> Result<(), String> {
        if self.peek() == Some('#') {
            while !matches!(self.peek(), None | Some('\r' | '\n')) {
                if self
                    .next()
                    .is_some_and(|c| c <= '\u{1f}' && c != '\t' || c == '\u{7f}')
                {
                    return Err("gomad.toml has a control character in a comment".into());
                }
            }
        }
        Ok(())
    }
    fn whitespace(&mut self) -> Result<(), String> {
        loop {
            self.spaces();
            self.comment()?;
            if matches!(self.peek(), Some('\r' | '\n')) {
                self.newline()?;
            } else {
                return Ok(());
            }
        }
    }
    fn string(&mut self) -> Result<String, String> {
        let quote = self
            .next()
            .filter(|c| matches!(c, '\'' | '"'))
            .ok_or("gomad.toml requires a quoted string")?;
        let multiline = self.text[self.index..].starts_with(&quote.to_string().repeat(2));
        if multiline {
            self.next();
            self.next();
            if matches!(self.peek(), Some('\r' | '\n')) {
                self.newline()?;
            }
        }
        let mut output = String::new();
        loop {
            let c = self.next().ok_or("gomad.toml has an unterminated string")?;
            if c == quote {
                if !multiline {
                    return Ok(output);
                }
                let mut count = 1;
                while self.peek() == Some(quote) {
                    self.next();
                    count += 1;
                }
                if count >= 3 {
                    if count > 5 {
                        return Err("gomad.toml has too many closing quotes".into());
                    }
                    output.extend(std::iter::repeat_n(quote, count - 3));
                    return Ok(output);
                }
                output.extend(std::iter::repeat_n(quote, count));
            } else if c == '\\' && quote == '"' {
                if multiline {
                    let start = self.index;
                    self.spaces();
                    if matches!(self.peek(), Some('\r' | '\n')) {
                        self.newline()?;
                        while matches!(self.peek(), Some(' ' | '\t' | '\r' | '\n')) {
                            if matches!(self.peek(), Some('\r' | '\n')) {
                                self.newline()?;
                            } else {
                                self.next();
                            }
                        }
                        continue;
                    }
                    self.index = start;
                }
                let escape = self.next().ok_or("gomad.toml has an incomplete escape")?;
                output.push(match escape {
                    'b' => '\u{8}',
                    't' => '\t',
                    'n' => '\n',
                    'f' => '\u{c}',
                    'r' => '\r',
                    '"' => '"',
                    '\\' => '\\',
                    'u' | 'U' => {
                        let mut code = 0;
                        for _ in 0..if escape == 'u' { 4 } else { 8 } {
                            code = (code << 4)
                                | self
                                    .next()
                                    .and_then(|c| c.to_digit(16))
                                    .ok_or("gomad.toml has an invalid Unicode escape")?;
                        }
                        char::from_u32(code).ok_or("gomad.toml has an invalid Unicode scalar")?
                    }
                    _ => return Err("gomad.toml has an invalid string escape".into()),
                });
            } else if c == '\n' || c == '\r' {
                if !multiline {
                    return Err("gomad.toml single-line string contains a newline".into());
                }
                if c == '\r' && self.next() != Some('\n') {
                    return Err("gomad.toml has a bare carriage return".into());
                }
                output.push('\n');
            } else if c <= '\u{1f}' && c != '\t' || c == '\u{7f}' {
                return Err("gomad.toml string contains a control character".into());
            } else {
                output.push(c);
            }
        }
    }
    fn array(&mut self) -> Result<Vec<String>, String> {
        self.next();
        let mut values = Vec::new();
        self.whitespace()?;
        while self.peek() != Some(']') {
            values.push(self.string()?);
            if values.len() > 1024 {
                return Err("gomad.toml guest array exceeds capacity".into());
            }
            self.whitespace()?;
            match self.peek() {
                Some(',') => {
                    self.next();
                    self.whitespace()?;
                }
                Some(']') => {}
                _ => return Err("gomad.toml array requires commas".into()),
            }
        }
        self.next();
        Ok(values)
    }
    fn document(mut self) -> Result<BTreeMap<String, Value>, String> {
        let mut values = BTreeMap::new();
        self.whitespace()?;
        while self.peek().is_some() {
            let name = if matches!(self.peek(), Some('\'' | '"')) {
                if self.text[self.index..].starts_with("\"\"\"")
                    || self.text[self.index..].starts_with("'''")
                {
                    return Err("gomad.toml keys cannot be multiline strings".into());
                }
                self.string()?
            } else {
                let start = self.index;
                while self
                    .peek()
                    .is_some_and(|c| c.is_ascii_alphanumeric() || matches!(c, '_' | '-'))
                {
                    self.next();
                }
                self.text[start..self.index].to_owned()
            };
            if !matches!(name.as_str(), "runtime_root" | "derive" | "guest") {
                return Err(format!("unknown gomad.toml key: {name}"));
            }
            self.spaces();
            if self.next() != Some('=') {
                return Err("gomad.toml requires key = value".into());
            }
            self.spaces();
            let value = if self.peek() == Some('[') {
                Value::Array(self.array()?)
            } else {
                Value::String(self.string()?)
            };
            if values.insert(name.clone(), value).is_some() {
                return Err(format!("duplicate gomad.toml key: {name}"));
            }
            self.spaces();
            self.comment()?;
            if self.peek().is_some() && !matches!(self.peek(), Some('\r' | '\n')) {
                return Err("gomad.toml requires a newline after a value".into());
            }
            self.whitespace()?;
        }
        Ok(values)
    }
}
