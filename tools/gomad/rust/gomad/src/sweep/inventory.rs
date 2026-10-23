use std::collections::{BTreeMap, BTreeSet};
use std::path::PathBuf;

// Native counterpart of internal/backends/wasm/internal/sweep/methods.go.
// Unsupported registrations fail before any sweep unit can run.
#[cfg(test)]
fn discover_methods(source: &str) -> Result<BTreeMap<String, Vec<String>>, String> {
    inventory(&[("source_test.go".into(), source.into())])
}

#[derive(Clone, Debug, PartialEq)]
pub struct SuiteMethod {
    pub suite: String,
    pub name: String,
    pub file: String,
    pub units: Vec<String>,
}
#[derive(Clone, Debug, PartialEq)]
pub struct SuiteRegistration {
    pub wrapper: String,
    pub suite: String,
    pub path: String,
    pub file: String,
    pub methods: Vec<String>,
}

pub struct SourceInventory {
    pub tests: BTreeMap<String, Vec<String>>,
    pub unit_files: BTreeMap<String, Vec<PathBuf>>,
    pub methods: Vec<SuiteMethod>,
    pub registrations: Vec<SuiteRegistration>,
}

pub fn source_inventory_with_loader(
    files: &[PathBuf],
    mut loader: impl FnMut(&str) -> Result<Vec<PathBuf>, String>,
) -> Result<SourceInventory, String> {
    let sources = read_sources(files)?;
    inventory_with_loader(&sources, &mut |path| read_sources(&loader(path)?))
}

fn read_sources(files: &[PathBuf]) -> Result<Vec<(String, String)>, String> {
    files
        .iter()
        .map(|file| {
            std::fs::read_to_string(file)
                .map(|source| (file.display().to_string(), source))
                .map_err(|error| format!("read {}: {error}", file.display()))
        })
        .collect()
}

type SourceLoader<'a> = dyn FnMut(&str) -> Result<Vec<(String, String)>, String> + 'a;

fn lex(source: &str) -> Result<Vec<String>, String> {
    let chars: Vec<char> = source.chars().collect();
    let mut out = Vec::new();
    let mut i = 0;
    while i < chars.len() {
        let c = chars[i];
        if c == '\n' {
            out.push(";".into());
            i += 1;
        } else if c.is_whitespace() {
            i += 1;
        } else if c == '/' && chars.get(i + 1) == Some(&'/') {
            while i < chars.len() && chars[i] != '\n' {
                i += 1;
            }
        } else if c == '/' && chars.get(i + 1) == Some(&'*') {
            i += 2;
            while i + 1 < chars.len() && !(chars[i] == '*' && chars[i + 1] == '/') {
                i += 1;
            }
            if i + 1 >= chars.len() {
                return Err("unterminated Go comment".into());
            }
            i += 2;
        } else if c == '"' || c == '`' || c == '\'' {
            let start = i;
            i += 1;
            while i < chars.len() && chars[i] != c {
                if chars[i] == '\\' && c != '`' {
                    i += 1;
                }
                i += 1;
            }
            if i >= chars.len() {
                return Err("unterminated Go literal".into());
            }
            i += 1;
            out.push(chars[start..i].iter().collect());
        } else if c == '_' || c.is_alphanumeric() {
            let start = i;
            while i < chars.len() && (chars[i] == '_' || chars[i].is_alphanumeric()) {
                i += 1;
            }
            out.push(chars[start..i].iter().collect());
        } else if c == ':' && chars.get(i + 1) == Some(&'=') {
            out.push(":=".into());
            i += 2;
        } else {
            out.push(c.to_string());
            i += 1;
        }
    }
    let mut stack = Vec::new();
    for token in &out {
        match token.as_str() {
            "(" => stack.push(")"),
            "[" => stack.push("]"),
            "{" => stack.push("}"),
            ")" | "]" | "}" if stack.pop() != Some(token.as_str()) => {
                return Err("unbalanced Go delimiters".into());
            }
            _ => {}
        }
    }
    if !stack.is_empty() {
        return Err("unbalanced Go delimiters".into());
    }
    Ok(out)
}

fn end(tokens: &[String], start: usize) -> Result<usize, String> {
    let close = match tokens.get(start).map(String::as_str) {
        Some("(") => ")",
        Some("[") => "]",
        Some("{") => "}",
        _ => return Err("expected Go delimiter".into()),
    };
    let mut i = start + 1;
    while i < tokens.len() {
        if tokens[i] == close {
            return Ok(i);
        }
        if matches!(tokens[i].as_str(), "(" | "[" | "{") {
            i = end(tokens, i)?;
        }
        i += 1;
    }
    Err("unclosed Go delimiter".into())
}

fn parts(tokens: &[String], separators: &[&str]) -> Result<Vec<Vec<String>>, String> {
    let mut out = Vec::new();
    let mut start = 0;
    let mut i = 0;
    while i < tokens.len() {
        if matches!(tokens[i].as_str(), "(" | "[" | "{") {
            i = end(tokens, i)?;
        } else if separators.contains(&tokens[i].as_str()) {
            if start < i {
                push_part(&mut out, &tokens[start..i]);
            }
            start = i + 1;
        }
        i += 1;
    }
    if start < tokens.len() {
        push_part(&mut out, &tokens[start..]);
    }
    Ok(out)
}

fn push_part(parts: &mut Vec<Vec<String>>, tokens: &[String]) {
    if let Some(start) = tokens.iter().position(|s| s != ";") {
        let end = tokens.iter().rposition(|s| s != ";").unwrap_or(start);
        parts.push(tokens[start..=end].to_vec());
    }
}

fn ident(s: &str) -> bool {
    let mut chars = s.chars();
    chars.next().is_some_and(|c| c == '_' || c.is_alphabetic())
        && chars.all(|c| c == '_' || c.is_alphanumeric())
}
fn test_name(s: &str) -> bool {
    s != "TestMain"
        && s.strip_prefix("Test")
            .is_some_and(|tail| !tail.chars().next().is_some_and(char::is_lowercase))
}
fn clean(tokens: &[String]) -> Vec<String> {
    tokens
        .iter()
        .filter(|s| s.as_str() != ";")
        .cloned()
        .collect()
}
fn base(tokens: &[String]) -> Option<String> {
    let t = clean(tokens);
    let mut i = 0;
    while t.get(i).is_some_and(|s| s == "*" || s == "(") {
        i += 1;
    }
    if !t.get(i).is_some_and(|s| ident(s)) {
        return None;
    }
    let mut name = t[i].clone();
    if t.get(i + 1).is_some_and(|s| s == ".") {
        name.push('.');
        name.push_str(t.get(i + 2)?);
    }
    Some(name)
}

#[derive(Clone)]
struct Function {
    file: PathBuf,
    name: String,
    owner: Option<String>,
    package: String,
    imports: BTreeMap<String, String>,
    params: Vec<String>,
    body: Vec<String>,
    test_file: bool,
    wrapper: bool,
}
#[derive(Default)]
struct Source {
    loaded: BTreeSet<String>,
    types: BTreeMap<String, Vec<String>>,
    functions: Vec<Function>,
}

impl Source {
    fn parse(&mut self, filename: &str, source: &str) -> Result<(), String> {
        self.parse_as(filename, source, None)
    }

    #[expect(
        clippy::too_many_lines,
        reason = "The declaration scanner keeps token advancement adjacent to each Go declaration form"
    )]
    fn parse_as(
        &mut self,
        filename: &str,
        source: &str,
        namespace: Option<&str>,
    ) -> Result<(), String> {
        let t = lex(source)?;
        let package_at = t
            .iter()
            .position(|s| s == "package")
            .ok_or("missing Go package declaration")?;
        let package = t
            .get(package_at + 1)
            .ok_or("missing Go package name")?
            .clone();
        let package = namespace.unwrap_or(&package).to_string();
        let mut imports = BTreeMap::new();
        let mut i = 0;
        while i < t.len() {
            if t[i] == "import" {
                i += 1;
                let entries;
                if t.get(i).is_some_and(|s| s == "(") {
                    let e = end(&t, i)?;
                    entries = parts(&t[i + 1..e], &[";"])?;
                    i = e + 1;
                } else {
                    let start = i;
                    while i < t.len() && t[i] != ";" {
                        i += 1;
                    }
                    entries = vec![t[start..i].to_vec()];
                }
                for entry in entries {
                    let Some(path) = entry
                        .last()
                        .filter(|s| s.starts_with('"') || s.starts_with('`'))
                    else {
                        return Err("unsupported Go import".into());
                    };
                    let path = path[1..path.len() - 1].to_string();
                    if path.contains('\\') {
                        return Err("unsupported escaped Go import path".into());
                    }
                    let alias = if entry.len() == 2 {
                        entry[0].clone()
                    } else {
                        path.rsplit('/').next().unwrap_or("").to_string()
                    };
                    if alias == "."
                        && (path.ends_with("/suite") || path.ends_with("/parallelsuite"))
                    {
                        return Err("unsupported dot-imported suite".into());
                    }
                    imports.insert(alias, path);
                }
            } else if matches!(t[i].as_str(), "{" | "(") {
                i = end(&t, i)? + 1;
            } else {
                i += 1;
            }
        }
        i = 0;
        while i < t.len() {
            if t[i] == "type" {
                i += 1;
                if t.get(i).is_some_and(|s| s == "(") {
                    let e = end(&t, i)?;
                    for declaration in parts(&t[i + 1..e], &[";"])? {
                        self.parse_type(&package, &imports, &declaration)?;
                    }
                    i = e + 1;
                } else {
                    let start = i;
                    while i < t.len() && t[i] != ";" {
                        if matches!(t[i].as_str(), "{" | "(" | "[") {
                            i = end(&t, i)?;
                        }
                        i += 1;
                    }
                    self.parse_type(&package, &imports, &t[start..i])?;
                }
            } else if t[i] == "func" {
                i += 1;
                let mut receiver_name = None;
                let owner = if t.get(i).is_some_and(|s| s == "(") {
                    let e = end(&t, i)?;
                    let receiver = &t[i + 1..e];
                    let receiver = if receiver.len() > 1
                        && ident(&receiver[0])
                        && (ident(&receiver[1]) || matches!(receiver[1].as_str(), "*" | "("))
                    {
                        receiver_name = Some(receiver[0].clone());
                        &receiver[1..]
                    } else {
                        receiver
                    };
                    let name = base(receiver).ok_or("unresolved method receiver")?;
                    i = e + 1;
                    Some(format!("{package}.{name}"))
                } else {
                    None
                };
                let name = t.get(i).ok_or("missing function name")?.clone();
                i += 1;
                let generic = t.get(i).is_some_and(|s| s == "[");
                if generic {
                    i = end(&t, i)? + 1;
                }
                let pe = end(&t, i)?;
                let mut params = t[i + 1..pe].to_vec();
                i = pe + 1;
                while i < t.len() && t[i] != "{" && t[i] != ";" {
                    i += 1;
                }
                if t.get(i).is_some_and(|s| s == "{") {
                    let e = end(&t, i)?;
                    let wrapper = owner.is_none()
                        && test_name(&name)
                        && !generic
                        && parts(&params, &[","])?.len() == 1;
                    if let Some(receiver) = receiver_name {
                        params.extend([",".into(), receiver]);
                    }
                    self.functions.push(Function {
                        file: PathBuf::from(filename),
                        name,
                        owner,
                        package: package.clone(),
                        imports: imports.clone(),
                        params,
                        body: t[i + 1..e].to_vec(),
                        test_file: filename.ends_with("_test.go"),
                        wrapper,
                    });
                    i = e + 1;
                }
            } else if matches!(t[i].as_str(), "{" | "(") {
                i = end(&t, i)? + 1;
            } else {
                i += 1;
            }
        }
        Ok(())
    }

    fn parse_type(
        &mut self,
        package: &str,
        imports: &BTreeMap<String, String>,
        tokens: &[String],
    ) -> Result<(), String> {
        if tokens.len() < 2 {
            return Err("unsupported Go type declaration".into());
        }
        if tokens[1] == "[" {
            let e = end(tokens, 1)?;
            let mut declaration = vec![tokens[0].clone()];
            declaration.extend_from_slice(&tokens[e + 1..]);
            return self.parse_type(package, imports, &declaration);
        }
        let name = format!("{package}.{}", tokens[0]);
        let mut embedded = Vec::new();
        if tokens[1] == "=" {
            embedded.push(Self::type_key(package, imports, &tokens[2..])?);
        } else if tokens[1] == "struct" && tokens.get(2).is_some_and(|s| s == "{") {
            let e = end(tokens, 2)?;
            for field in parts(&tokens[3..e], &[";"])? {
                let f = clean(&field);
                if f.is_empty() {
                    continue;
                }
                let offset = usize::from(f[0] == "*");
                if f.get(offset).is_some_and(|s| ident(s))
                    && (f.len() == offset + 1
                        || f.get(offset + 1).is_some_and(|s| {
                            s == "." || s == "[" || s.starts_with('`') || s.starts_with('"')
                        }))
                {
                    embedded.push(Self::type_key(package, imports, &f)?);
                }
            }
        }
        if self.types.insert(name.clone(), embedded).is_some() {
            return Err(format!("duplicate suite type {name}"));
        }
        Ok(())
    }

    fn type_key(
        package: &str,
        imports: &BTreeMap<String, String>,
        tokens: &[String],
    ) -> Result<String, String> {
        let name = base(tokens).ok_or("unresolved embedded suite type")?;
        if let Some((alias, typ)) = name.split_once('.') {
            let path = imports
                .get(alias)
                .ok_or("unresolved embedded suite import")?;
            if typ == "Suite"
                && matches!(
                    path.as_str(),
                    "github.com/stretchr/testify/suite"
                        | "go.temporal.io/server/common/testing/parallelsuite"
                )
            {
                return Ok("@suite-base".into());
            }
            return Ok(format!("@external:{path}.{typ}"));
        }
        Ok(format!("{package}.{name}"))
    }

    fn load_types(
        &mut self,
        key: &str,
        loader: &mut SourceLoader<'_>,
        active: &mut BTreeSet<String>,
    ) -> Result<(), String> {
        if key == "@suite-base" {
            return Ok(());
        }
        if !active.insert(key.into()) {
            return Err(format!("cyclic suite embedding {key}"));
        }
        if !self.types.contains_key(key) {
            let external = key
                .strip_prefix("@external:")
                .ok_or_else(|| format!("unresolved suite type {key}"))?;
            let (path, _) = external
                .rsplit_once('.')
                .ok_or("unresolved external embedded suite type")?;
            if !self.loaded.insert(path.into()) {
                return Err(format!("unresolved suite type {key}"));
            }
            let files = loader(path)?;
            if files.is_empty() {
                return Err(format!("unresolved embedded suite package {path}"));
            }
            for (name, source) in files {
                self.parse_as(&name, &source, Some(&format!("@external:{path}")))?;
            }
        }
        let parents = self
            .types
            .get(key)
            .ok_or_else(|| format!("unresolved suite type {key}"))?
            .clone();
        for parent in parents {
            self.load_types(&parent, loader, active)?;
        }
        active.remove(key);
        Ok(())
    }

    fn methods(
        &self,
        key: &str,
        active: &mut BTreeSet<String>,
    ) -> Result<BTreeMap<String, (usize, usize, bool)>, String> {
        if key == "@suite-base" {
            return Ok(BTreeMap::new());
        }
        if !active.insert(key.into()) {
            return Err(format!("cyclic suite embedding {key}"));
        }
        let embedded = self
            .types
            .get(key)
            .ok_or_else(|| format!("unresolved suite type {key}"))?;
        let mut result = BTreeMap::new();
        for (id, f) in self
            .functions
            .iter()
            .enumerate()
            .filter(|(_, f)| f.owner.as_deref() == Some(key))
        {
            result.insert(f.name.clone(), (id, 0, false));
        }
        let direct: BTreeSet<_> = result.keys().cloned().collect();
        for parent in embedded {
            for (name, (id, depth, ambiguous)) in self.methods(parent, active)? {
                if parent.starts_with("@external:")
                    && !key.starts_with("@external:")
                    && name.starts_with("Test")
                {
                    return Err(format!(
                        "cross-package promoted test method {parent}.{name} requires explicit registration coverage"
                    ));
                }
                if direct.contains(&name) {
                    continue;
                }
                let candidate = (id, depth + 1, ambiguous);
                match result.get(&name) {
                    Some((_, old_depth, _)) if *old_depth < depth + 1 => {}
                    Some((_, old_depth, _)) if *old_depth == depth + 1 => {
                        result.insert(name, (id, depth + 1, true));
                    }
                    _ => {
                        result.insert(name, candidate);
                    }
                }
            }
        }
        active.remove(key);
        for (name, (_, _, ambiguous)) in &result {
            if *ambiguous && name.starts_with("Test") {
                return Err(format!("ambiguous promoted method {key}.{name}"));
            }
        }
        Ok(result)
    }
}

#[derive(Clone)]
enum Value {
    Type(String),
    Binding(usize),
    Unknown,
}
#[derive(Default)]
struct Binding {
    values: Vec<Value>,
}
struct Call {
    argument: Value,
    path: String,
}
struct Walker<'a> {
    function: &'a Function,
    scopes: Vec<BTreeMap<String, usize>>,
    bindings: Vec<Binding>,
    calls: Vec<Call>,
}
impl Walker<'_> {
    fn lookup(&self, name: &str) -> Option<usize> {
        self.scopes
            .iter()
            .rev()
            .find_map(|scope| scope.get(name).copied())
    }
    fn bind(&mut self, name: &str, value: Value, define: bool) {
        let existing = if define {
            self.scopes.last().and_then(|s| s.get(name)).copied()
        } else {
            self.lookup(name)
        };
        let id = existing.unwrap_or_else(|| {
            self.bindings.push(Binding::default());
            self.bindings.len() - 1
        });
        if existing.is_none() {
            self.scopes.last_mut().unwrap().insert(name.into(), id);
        }
        self.bindings[id].values.push(value);
    }
    fn value(&self, tokens: &[String]) -> Value {
        let mut t = clean(tokens);
        while t.first().is_some_and(|s| s == "&" || s == "*") {
            t.remove(0);
        }
        while t.first().is_some_and(|s| s == "(") && end(&t, 0).ok() == Some(t.len() - 1) {
            t = t[1..t.len() - 1].to_vec();
        }
        if t.len() == 1 {
            return self.lookup(&t[0]).map_or(Value::Unknown, Value::Binding);
        }
        if t.first().is_some_and(|s| s == "new")
            && t.get(1).is_some_and(|s| s == "(")
            && end(&t, 1).ok() == Some(t.len() - 1)
        {
            return base(&t[2..t.len() - 1])
                .filter(|s| !s.contains('.'))
                .map_or(Value::Unknown, |s| {
                    Value::Type(format!("{}.{s}", self.function.package))
                });
        }
        if let Some(open) = t.iter().position(|s| s == "{")
            && end(&t, open).ok() == Some(t.len() - 1)
        {
            return base(&t[..open])
                .filter(|s| !s.contains('.'))
                .map_or(Value::Unknown, |s| {
                    Value::Type(format!("{}.{s}", self.function.package))
                });
        }
        Value::Unknown
    }
    #[expect(
        clippy::too_many_lines,
        reason = "Lexical bindings and nested registration paths share one token cursor and scope stack"
    )]
    fn walk(&mut self, t: &[String], path: &str) -> Result<(), String> {
        let mut i = 0;
        let mut statement = 0;
        while i < t.len() {
            if t[i] == ";" {
                statement = i + 1;
                i += 1;
                continue;
            }
            if matches!(t[i].as_str(), "for" | "if" | "switch" | "select") {
                let mut brace = i + 1;
                while brace < t.len() && t[brace] != "{" {
                    if matches!(t[brace].as_str(), "(" | "[") {
                        brace = end(t, brace)?;
                    }
                    brace += 1;
                }
                if brace < t.len() {
                    let e = end(t, brace)?;
                    self.scopes.push(BTreeMap::new());
                    self.walk(&t[i + 1..brace], path)?;
                    self.walk(&t[brace + 1..e], path)?;
                    self.scopes.pop();
                    i = e + 1;
                    statement = i;
                    continue;
                }
            }
            if (t[i] == ":=" || t[i] == "=")
                && t.get(i.wrapping_sub(1)).is_some_and(|s| ident(s))
                && t.get(i + 1).is_none_or(|s| s != "=")
                && t.get(i.wrapping_sub(1)).is_none_or(|s| s != "=")
            {
                let mut e = i + 1;
                while e < t.len() && t[e] != ";" {
                    if t[e] == "{" && t[i + 1..e].iter().any(|s| s == "range") {
                        break;
                    }
                    if matches!(t[e].as_str(), "(" | "[" | "{") {
                        e = end(t, e)?;
                    }
                    e += 1;
                }
                let lhs = &t[statement..i];
                let lhs = if lhs
                    .first()
                    .is_some_and(|s| s == "for" || s == "var" || s == "if")
                {
                    &lhs[1..]
                } else {
                    lhs
                };
                let names = parts(lhs, &[","])?;
                let values = parts(&t[i + 1..e], &[","])?;
                let range = t[i + 1..e].iter().any(|s| s == "range");
                for (n, name) in names.iter().enumerate() {
                    if let Some(name) = name.first().filter(|name| ident(name)) {
                        let value = if range {
                            Value::Unknown
                        } else {
                            values.get(n).map_or(Value::Unknown, |v| self.value(v))
                        };
                        self.bind(name, value, t[i] == ":=" || t[statement] == "var");
                    }
                }
            }
            if t[i] == "var" && t.get(i + 1).is_some_and(|s| ident(s)) {
                let mut e = i + 2;
                while e < t.len() && !matches!(t[e].as_str(), ";" | "=") {
                    e += 1;
                }
                if t.get(e).is_none_or(|s| s != "=") {
                    let value = base(&t[i + 2..e]).map_or(Value::Unknown, |s| {
                        Value::Type(format!("{}.{s}", self.function.package))
                    });
                    self.bind(&t[i + 1], value, true);
                }
            }
            if t[i] == "." && t.get(i + 2).is_some_and(|s| s == "(") {
                let e = end(t, i + 2)?;
                let args = parts(&t[i + 3..e], &[","])?;
                let qualifier = i.checked_sub(1).and_then(|n| t.get(n));
                let imported = qualifier
                    .filter(|name| self.lookup(name).is_none())
                    .and_then(|name| self.function.imports.get(name));
                let registration = imported.is_some_and(|p| {
                    (p == "github.com/stretchr/testify/suite" && t[i + 1] == "Run")
                        || (p == "go.temporal.io/server/common/testing/parallelsuite"
                            && matches!(t[i + 1].as_str(), "Run" | "RunLegacySequential"))
                });
                if registration {
                    let arg = args.get(1).ok_or("unresolved suite registration")?;
                    if self.calls.iter().any(|call| call.path == path) {
                        return Err("overlapping suite registrations".into());
                    }
                    self.calls.push(Call {
                        argument: self.value(arg),
                        path: path.into(),
                    });
                    i = e + 1;
                    continue;
                }
                if t[i + 1] == "Run" && args.len() == 2 {
                    let callback = &args[1];
                    let start = callback
                        .iter()
                        .position(|s| s != ";")
                        .unwrap_or(callback.len());
                    let callback = &callback[start..];
                    if callback.first().is_some_and(|s| s == "func")
                        && callback.get(1).is_some_and(|s| s == "(")
                    {
                        let pe = end(callback, 1)?;
                        let parameters = &callback[2..pe];
                        if parameters.windows(2).any(|w| w[0] == "." && w[1] == "T")
                            && callback.get(pe + 1).is_some_and(|s| s == "{")
                        {
                            let be = end(callback, pe + 1)?;
                            self.scopes.push(BTreeMap::new());
                            if parameters.first().is_some_and(|s| ident(s)) {
                                self.bind(&parameters[0], Value::Unknown, true);
                            }
                            self.walk(&callback[pe + 2..be], &format!("{path}/.*"))?;
                            self.scopes.pop();
                            i = e + 1;
                            continue;
                        }
                    }
                }
            }
            if t[i] == "{" {
                let e = end(t, i)?;
                self.scopes.push(BTreeMap::new());
                self.walk(&t[i + 1..e], path)?;
                self.scopes.pop();
                i = e + 1;
            } else {
                i += 1;
            }
        }
        Ok(())
    }
    fn resolve(&self, value: &Value, active: &mut BTreeSet<usize>) -> Option<String> {
        match value {
            Value::Type(name) => Some(name.clone()),
            Value::Unknown => None,
            Value::Binding(id) => {
                if !active.insert(*id) {
                    return None;
                }
                let values = &self.bindings[*id].values;
                let mut name = None;
                for value in values {
                    let found = self.resolve(value, active)?;
                    if name.as_ref().is_some_and(|previous| previous != &found) {
                        return None;
                    }
                    name = Some(found);
                }
                active.remove(id);
                name
            }
        }
    }
}

#[derive(Default)]
struct Coverage {
    functions: BTreeSet<usize>,
    method_units: BTreeMap<usize, Vec<String>>,
    registrations: Vec<SuiteRegistration>,
    wrapper: String,
}

#[expect(
    clippy::too_many_arguments,
    reason = "Recursive expansion carries branch ancestry separately from shared coverage, cycle detection, source loading and provenance"
)]
fn expand(
    source: &mut Source,
    id: usize,
    path: &str,
    coverage: &mut Coverage,
    active: &mut BTreeSet<usize>,
    parents: &[PathBuf],
    unit_files: &mut BTreeMap<String, Vec<PathBuf>>,
    loader: &mut SourceLoader<'_>,
) -> Result<Vec<String>, String> {
    if !active.insert(id) {
        return Err(format!("cyclic suite registration {path}"));
    }
    coverage.functions.insert(id);
    let f = source.functions[id].clone();
    let mut files = parents.to_vec();
    files.push(f.file.clone());
    files.sort();
    files.dedup();
    let mut walker = Walker {
        function: &f,
        scopes: vec![BTreeMap::new()],
        bindings: Vec::new(),
        calls: Vec::new(),
    };
    for param in parts(&f.params, &[","])? {
        if param.first().is_some_and(|s| ident(s)) {
            walker.bind(&param[0], Value::Unknown, true);
        }
    }
    walker.walk(&f.body, "")?;
    let mut units = Vec::new();
    if walker.calls.is_empty() {
        units.push(path.into());
        unit_files.insert(path.into(), files.clone());
    }
    for call in &walker.calls {
        let suite = walker
            .resolve(&call.argument, &mut BTreeSet::new())
            .ok_or_else(|| format!("unresolved suite registration in {}", f.name))?;
        source.load_types(&suite, loader, &mut BTreeSet::new())?;
        let methods = source.methods(&suite, &mut BTreeSet::new())?;
        let tests: Vec<_> = methods
            .iter()
            .filter(|(name, _)| name.starts_with("Test"))
            .collect();
        if tests.is_empty() {
            return Err(format!("registered suite {suite} has no test methods"));
        }
        coverage.registrations.push(SuiteRegistration {
            wrapper: coverage.wrapper.clone(),
            suite: suite.clone(),
            path: format!("{path}{}", call.path),
            file: f.file.display().to_string(),
            methods: tests.iter().map(|(name, _)| (*name).clone()).collect(),
        });
        for (name, (method, _, _)) in tests {
            units.extend(expand(
                source,
                *method,
                &format!("{path}{}/{name}", call.path),
                coverage,
                active,
                &files,
                unit_files,
                loader,
            )?);
        }
    }
    active.remove(&id);
    units.sort();
    units.dedup();
    if f.owner.is_some() && f.name.starts_with("Test") {
        coverage
            .method_units
            .entry(id)
            .or_default()
            .extend(units.clone());
    }
    Ok(units)
}

#[cfg(test)]
fn inventory(files: &[(String, String)]) -> Result<BTreeMap<String, Vec<String>>, String> {
    inventory_with_files(files).map(|inventory| inventory.tests)
}

#[cfg(test)]
fn inventory_with_files(files: &[(String, String)]) -> Result<SourceInventory, String> {
    inventory_with_loader(files, &mut |path| {
        Err(format!("unresolved external embedded suite type {path}"))
    })
}

fn inventory_with_loader(
    files: &[(String, String)],
    loader: &mut SourceLoader<'_>,
) -> Result<SourceInventory, String> {
    let mut source = Source::default();
    for (name, contents) in files {
        source
            .parse(name, contents)
            .map_err(|error| format!("parse {name}: {error}"))?;
    }
    let mut result = BTreeMap::new();
    let mut coverage = Coverage::default();
    let mut unit_files = BTreeMap::new();
    let wrappers: Vec<_> = source
        .functions
        .iter()
        .enumerate()
        .filter(|(_, f)| f.wrapper && f.test_file)
        .map(|(id, f)| (id, f.name.clone()))
        .collect();
    for (id, name) in wrappers {
        coverage.wrapper.clone_from(&name);
        let units = expand(
            &mut source,
            id,
            &name,
            &mut coverage,
            &mut BTreeSet::new(),
            &[],
            &mut unit_files,
            loader,
        )?;
        if result.insert(name.clone(), units).is_some() {
            return Err(format!("duplicate test {name}"));
        }
    }
    for (id, f) in source.functions.iter().enumerate() {
        if f.test_file
            && f.owner.is_some()
            && f.name.starts_with("Test")
            && !coverage.functions.contains(&id)
        {
            return Err(format!(
                "unregistered test method {}.{}",
                f.owner.as_deref().unwrap_or(""),
                f.name
            ));
        }
    }
    let mut methods = Vec::new();
    for (id, mut units) in coverage.method_units {
        let function = &source.functions[id];
        units.sort();
        units.dedup();
        methods.push(SuiteMethod {
            suite: function.owner.clone().ok_or("method receiver missing")?,
            name: function.name.clone(),
            file: function.file.display().to_string(),
            units,
        });
    }
    methods.sort_by(|a, b| (&a.suite, &a.name).cmp(&(&b.suite, &b.name)));
    coverage
        .registrations
        .sort_by(|a, b| (&a.path, &a.suite).cmp(&(&b.path, &b.suite)));
    Ok(SourceInventory {
        tests: result,
        unit_files,
        methods,
        registrations: coverage.registrations,
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    fn parse(body: &str) -> Result<BTreeMap<String, Vec<String>>, String> {
        discover_methods(&format!(
            "package example\nimport \"testing\"\nimport ts \"github.com/stretchr/testify/suite\"\nimport ps \"go.temporal.io/server/common/testing/parallelsuite\"\n{body}"
        ))
    }

    #[test]
    fn declaration_and_registration_metadata_retains_independent_suite_coverage() {
        let source = r#"package example
            import ts "github.com/stretchr/testify/suite"
            type Base struct{}
            func (*Base) TestInherited() {}
            type Outer struct{ Base }
            type Inner struct{}
            func (*Inner) TestLeaf() {}
            func (*Outer) TestNested() { ts.Run(nil, &Inner{}) }
            func TestWrapper(t *testing.T) { ts.Run(t, &Outer{}) }
        "#;
        let result = inventory_with_files(&[("suite_test.go".into(), source.into())]).unwrap();
        assert_eq!(
            result.methods,
            vec![
                SuiteMethod {
                    suite: "example.Base".into(),
                    name: "TestInherited".into(),
                    file: "suite_test.go".into(),
                    units: vec!["TestWrapper/TestInherited".into()]
                },
                SuiteMethod {
                    suite: "example.Inner".into(),
                    name: "TestLeaf".into(),
                    file: "suite_test.go".into(),
                    units: vec!["TestWrapper/TestNested/TestLeaf".into()]
                },
                SuiteMethod {
                    suite: "example.Outer".into(),
                    name: "TestNested".into(),
                    file: "suite_test.go".into(),
                    units: vec!["TestWrapper/TestNested/TestLeaf".into()]
                },
            ]
        );
        assert_eq!(
            result.registrations,
            vec![
                SuiteRegistration {
                    wrapper: "TestWrapper".into(),
                    suite: "example.Outer".into(),
                    path: "TestWrapper".into(),
                    file: "suite_test.go".into(),
                    methods: vec!["TestInherited".into(), "TestNested".into()]
                },
                SuiteRegistration {
                    wrapper: "TestWrapper".into(),
                    suite: "example.Inner".into(),
                    path: "TestWrapper/TestNested".into(),
                    file: "suite_test.go".into(),
                    methods: vec!["TestLeaf".into()]
                },
            ]
        );
    }

    // Original roots: TestSuiteInventoryRegistrations/Arguments/VariableScopes.
    #[test]
    fn registration_aliases_scopes_and_promoted_methods() {
        let result = parse(r#"
            type Base struct{}
            func (*Base) TestInherited() {}
            type S struct{ Base }
            type Alias = S
            func (*S) TestOwn() {}
            func TestFirst(t *testing.T) { ps.Run(t, &S{}, "argument") }
            func TestSecond(t *testing.T) { var s *Alias; s = new(Alias); alias := s; ts.Run(t, alias) }
            func TestNested(t *testing.T) { s := unknown(); t.Run("variant", func(t *testing.T) { s := &S{}; ts.Run(t, s) }); _ = s }
            func TestPlain(t *testing.T) {}
        "#).unwrap();
        assert_eq!(
            result,
            BTreeMap::from([
                (
                    "TestFirst".into(),
                    vec!["TestFirst/TestInherited".into(), "TestFirst/TestOwn".into()]
                ),
                (
                    "TestSecond".into(),
                    vec![
                        "TestSecond/TestInherited".into(),
                        "TestSecond/TestOwn".into()
                    ]
                ),
                (
                    "TestNested".into(),
                    vec![
                        "TestNested/.*/TestInherited".into(),
                        "TestNested/.*/TestOwn".into()
                    ]
                ),
                ("TestPlain".into(), vec!["TestPlain".into()]),
            ])
        );
    }

    // Original root: TestSuiteInventoryNestedSuite.
    #[test]
    fn recursively_expands_nested_suite_registration() {
        let result = parse(r#"
            type Outer struct{}
            type Inner struct{}
            func TestOuter(t *testing.T) { ps.Run(t, &Outer{}) }
            func (s *Outer) TestNested() { s.T().Run("variant", func(t *testing.T) { ps.RunLegacySequential(t, &Inner{}) }) }
            func (*Inner) TestLeaf() {}
        "#).unwrap();
        assert_eq!(result["TestOuter"], ["TestOuter/TestNested/.*/TestLeaf"]);
    }

    // Original root: TestSuiteInventoryRefusesMissingCoverage.
    #[test]
    fn refuses_missing_or_ambiguous_coverage() {
        for (body, reason) in [
            (
                "type S struct{}; func (*S) TestLost() {}",
                "unregistered test method",
            ),
            (
                "func TestWrapper(t *testing.T) { ts.Run(t, makeSuite()) }",
                "unresolved suite registration",
            ),
            (
                "type S struct{}; type Other struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { s := &S{}; s = &Other{}; ts.Run(t, s) }",
                "unresolved suite registration",
            ),
            (
                "type S struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { s := &S{}; for _, s := range unknown { ts.Run(t, s) }; _ = s }",
                "unresolved suite registration",
            ),
            (
                "type S struct{}; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}); ts.Run(t, &S{}) }",
                "overlapping suite registrations",
            ),
            (
                "type A struct{}; type B struct{}; type S struct{ A; B }; func (*A) TestSame() {}; func (*B) TestSame() {}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }",
                "ambiguous promoted method",
            ),
            (
                "type S struct{}; func (*S) TestOne() { ts.Run(nil, &S{}) }; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }",
                "cyclic suite registration",
            ),
        ] {
            assert!(
                parse(body).unwrap_err().contains(reason),
                "{body}: {reason}"
            );
        }
    }

    // Original root: TestSuiteInventoryPromotionUsesShallowestMethod.
    #[test]
    fn promotion_uses_shallowest_method_and_retains_declaration_coverage() {
        let result = parse(
            r"
            type Base struct{}
            func (*Base) TestOne() {}
            type A struct{ Base }
            type B struct{}
            func (*B) TestOne() {}
            type S struct{ A; B }
            func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }
            func TestBase(t *testing.T) { ts.Run(t, &Base{}) }
        ",
        )
        .unwrap();
        assert_eq!(
            result,
            BTreeMap::from([
                ("TestBase".into(), vec!["TestBase/TestOne".into()]),
                ("TestWrapper".into(), vec!["TestWrapper/TestOne".into()]),
            ])
        );
    }

    #[test]
    fn merges_files_without_confusing_external_test_packages() {
        let result = inventory(&[
            ("base.go".into(), "package example; type Base struct{}; func (*Base) TestBase() {}".into()),
            ("suite_test.go".into(), "package example; import ts \"github.com/stretchr/testify/suite\"; type S struct{ Base }; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }".into()),
            ("external_test.go".into(), "package example_test; func TestExternal(t *testing.T) {}".into()),
        ]).unwrap();
        assert_eq!(
            result,
            BTreeMap::from([
                ("TestExternal".into(), vec!["TestExternal".into()]),
                ("TestWrapper".into(), vec!["TestWrapper/TestBase".into()]),
            ])
        );
    }

    #[test]
    fn unknown_external_embedding_fails_closed() {
        assert!(parse("import base \"example.com/base\"; type S struct{ base.Base }; func (*S) TestOne() {}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }").unwrap_err().contains("unresolved external embedded suite type"));
    }

    // Original root: TestSuiteInventoryGuestExclusionsIncludeWrapperAndMethod.
    #[test]
    fn unit_files_include_each_wrapper_and_defining_method() {
        let result = inventory_with_files(&[
            ("wrapper_test.go".into(), "package example; import ts \"github.com/stretchr/testify/suite\"; type S struct{}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }".into()),
            ("method_test.go".into(), "package example; func (*S) TestMethod() {}".into()),
        ]).unwrap();
        assert_eq!(
            result.unit_files,
            BTreeMap::from([(
                "TestWrapper/TestMethod".into(),
                vec![
                    PathBuf::from("method_test.go"),
                    PathBuf::from("wrapper_test.go")
                ]
            ),])
        );
    }

    #[test]
    fn loop_shadow_does_not_escape_and_callbacks_allow_newlines() {
        let result = parse(
            r#"
            type S struct{ ts.Suite }
            func (*S) TestOwn() {}
            func TestWrapper(t *testing.T) {
                s := &S{}
                for _, s := range values { _ = s }
                t.Run("variant",
                    func(t *testing.T) {
                        alias := s
                        ts.Run(t, alias)
                    },
                )
            }
        "#,
        )
        .unwrap();
        assert_eq!(result["TestWrapper"], ["TestWrapper/.*/TestOwn"]);
    }

    // Original root: TestSuiteInventoryExternalEmbedding.
    #[test]
    fn external_embedding_loads_import_metadata_and_refuses_promoted_tests() {
        for promoted in [false, true] {
            let sources = vec![("suite_test.go".into(), "package example; import ts \"github.com/stretchr/testify/suite\"; import base \"example.com/base\"; type S struct{ base.Base }; func (*S) TestOwn() {}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }".into())];
            let mut loaded = Vec::new();
            let result = inventory_with_loader(&sources, &mut |path| {
                loaded.push(path.to_string());
                Ok(vec![(
                    "base.go".into(),
                    format!(
                        "package base; type Base struct{{}}; {}",
                        if promoted {
                            "func (*Base) TestPromoted() {}"
                        } else {
                            "func (*Base) Helper() {}"
                        }
                    ),
                )])
            });
            assert_eq!(loaded, ["example.com/base"]);
            if promoted {
                assert!(
                    result
                        .err()
                        .unwrap()
                        .contains("cross-package promoted test method")
                );
            } else {
                assert_eq!(
                    result.unwrap().tests["TestWrapper"],
                    ["TestWrapper/TestOwn"]
                );
            }
        }
    }

    #[test]
    fn external_packages_with_same_declared_name_have_distinct_types() {
        let sources = vec![("suite_test.go".into(), "package example; import ts \"github.com/stretchr/testify/suite\"; import a \"example.com/a\"; import b \"example.com/b\"; type S struct{ a.Base; b.Base }; func (*S) TestOwn() {}; func TestWrapper(t *testing.T) { ts.Run(t, &S{}) }".into())];
        let result = inventory_with_loader(&sources, &mut |path| {
            Ok(vec![(
                format!("{path}/base.go"),
                "package common; type Base struct{}; func (*Base) Helper() {}".into(),
            )])
        })
        .unwrap();
        assert_eq!(result.tests["TestWrapper"], ["TestWrapper/TestOwn"]);
    }

    #[test]
    fn comments_and_literals_do_not_declare_tests() {
        let result = parse(
            r"
            // func TestFake(t *testing.T) { ts.Run(t, makeSuite()) }
            const text = `func TestFake(t *testing.T) {}`
            func Testlower(t *testing.T) {}
            func TestMain(m *testing.M) {}
            func Test(t *testing.T) {}
        ",
        )
        .unwrap();
        assert_eq!(
            result,
            BTreeMap::from([("Test".into(), vec!["Test".into()])])
        );
    }
}
