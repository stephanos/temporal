//! Lift authored Rust Models, admitted by `umpire-front`, into TLA+.
//!
//! The admitted subset, including the `imbl::Vector<T>` method table, is `umpire-front`'s.
use proc_macro2::Span;
use std::{collections::BTreeMap, fmt, path::Path};
mod bind;
pub mod build;
mod emit;
mod names;
pub use umpire_front::{CATALOG_LIMIT, Type};

/// An admitted Model and the source file it was read from.
#[derive(Clone, Debug)]
pub struct Lifted {
    file: String,
    model: umpire_front::Lifted,
    names: names::Names,
}
impl std::ops::Deref for Lifted {
    type Target = umpire_front::Lifted;
    fn deref(&self) -> &Self::Target {
        &self.model
    }
}
/// A located warning refusing unsupported source behavior.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Diagnostic {
    /// Source filename.
    pub file: String,
    /// One-based line.
    pub line: usize,
    /// One-based column.
    pub column: usize,
    /// Admission failure.
    pub message: String,
}
impl Diagnostic {
    pub(crate) fn span(file: &str, span: Span, message: impl Into<String>) -> Self {
        let location = span.start();
        Self {
            file: file.into(),
            line: location.line,
            column: location.column + 1,
            message: message.into(),
        }
    }
    /// A front-end location in `file`, with its own message.
    pub(crate) fn of(
        file: &str,
        location: &umpire_front::Diagnostic,
        message: impl Into<String>,
    ) -> Self {
        Self {
            file: file.into(),
            line: location.line,
            column: location.column,
            message: message.into(),
        }
    }
}
impl fmt::Display for Diagnostic {
    fn fmt(&self, out: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(
            out,
            "{}:{}:{}: warning: {}; TLA+ emission refused",
            self.file, self.line, self.column, self.message
        )
    }
}
impl std::error::Error for Diagnostic {}
/// An authored state field.
#[derive(Clone, Debug)]
pub struct Field {
    /// Authored name.
    pub name: String,
    /// Declared type.
    pub ty: Type,
}
/// A literal projected from compiler-checked authored fields.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Literal {
    /// Boolean value.
    Bool(bool),
    /// Integer checked against the independently admitted field type.
    Int(i128),
    /// Qualified variant of an independently admitted fieldless enum.
    Enum(String),
    /// Ordered elements of an admitted state sequence.
    Sequence(Vec<Literal>),
}
/// One concrete catalog action: its qualified variant and ordered named payload values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Concrete {
    /// Qualified authored variant name.
    pub action: String,
    /// Payload fields in declaration order; tuple fields are named by position.
    pub fields: Vec<(String, Literal)>,
}
/// The values a World-dependent `#[domain]` function yields for one action field.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Domain {
    /// Qualified authored variant name.
    pub action: String,
    /// Payload field name; tuple fields are named by position.
    pub field: String,
    /// Values in enumeration order, including positional duplicates.
    pub values: Vec<Literal>,
}
/// One selected World and its complete authored start states.
#[derive(Clone, Debug)]
pub struct Binding {
    /// Named World fields, validated against the original source declaration.
    pub world: Vec<(String, Literal)>,
    /// Complete named state fields for each authored start, in enumeration order.
    pub starts: Vec<Vec<(String, Literal)>>,
    /// The checked native catalog of the same World, in catalog order.
    pub actions: Vec<Concrete>,
    /// Values of every World-dependent action field for the same World.
    pub domains: Vec<Domain>,
}
/// An authored action.
#[derive(Clone, Debug)]
pub struct Action {
    /// Qualified authored name.
    pub name: String,
    /// TLA+ source relation operator.
    pub operator: String,
    /// Internal marker, which adds no fairness.
    pub internal: bool,
    /// Typed payload fields in declaration order; empty for a unit variant.
    pub fields: Vec<Field>,
}
/// A property with its isolated TLC configuration.
#[derive(Clone, Debug)]
pub struct Property {
    /// Authored name.
    pub name: String,
    /// TLA+ operator.
    pub operator: String,
    /// Isolated configuration.
    pub cfg: String,
}
/// A progress property with its authored scheduling assumptions.
#[derive(Clone, Debug)]
pub struct Progress {
    /// Authored name.
    pub name: String,
    /// TLA+ property operator.
    pub operator: String,
    /// Scoped observer specification configuration.
    pub cfg: String,
    /// Admitted suffix actions.
    pub allowed: Vec<String>,
    /// Explicit fair actions only.
    pub fair: Vec<String>,
    /// Admitted concrete catalog ordinals; empty without an input catalog.
    pub allowed_inputs: Vec<usize>,
    /// Explicitly fair concrete catalog ordinals; empty without an input catalog.
    pub fair_inputs: Vec<usize>,
    /// Optional Model-transition bound.
    pub bound: Option<u64>,
}
/// Exact decoding and property metadata.
#[derive(Clone, Debug)]
pub struct Metadata {
    /// Authored state struct name.
    pub state_name: String,
    /// Unit World constructor or declared World type path.
    pub world_path: String,
    /// State fields in declaration order.
    pub fields: Vec<Field>,
    /// Actions in catalog order.
    pub actions: Vec<Action>,
    /// The bound concrete catalog that `__umpire_input` ordinals index; empty for Models
    /// without action inputs, whose edges are identified by action name alone.
    pub catalog: Vec<Concrete>,
    /// Registered state claims.
    pub invariants: Vec<Property>,
    /// Registered action claims.
    pub postconditions: Vec<Property>,
    /// Progress claims.
    pub progress: Vec<Progress>,
    /// Supplemental negative fixture claims.
    pub negative: Vec<Property>,
    /// Exactly the fields projected out of state agreement.
    pub observer_fields: Vec<String>,
    /// Exact private field types.
    pub observer_types: BTreeMap<String, Type>,
    /// Outcome type.
    pub outcome_type: Type,
}
/// Generated module and decoding contract.
#[derive(Clone, Debug)]
pub struct Module {
    /// Module name.
    pub name: String,
    /// TLA+ source.
    pub tla: String,
    /// Full-graph specification and safety checks.
    pub cfg: String,
    /// Projection metadata.
    pub metadata: Metadata,
}
/// Authored-source reading through `umpire-front`.
pub struct Lift;
impl Lift {
    /// Read a source root, with the golden hook from a `tests.rs` beside it.
    pub fn source(path: impl AsRef<Path>) -> Result<Lifted, Diagnostic> {
        let path = path.as_ref();
        let file = path.to_string_lossy();
        let read = |path: &Path, file: &str| {
            let source = std::fs::read_to_string(path)
                .map_err(|error| Diagnostic::span(file, Span::call_site(), error.to_string()))?;
            syn::parse_file(&source)
                .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))
        };
        let ast = read(path, &file)?;
        let tests_path = path.with_file_name("tests.rs");
        let tests_file = tests_path.to_string_lossy();
        let tests = if tests_path.is_file() { Some(read(&tests_path, &tests_file)?) } else { None };
        let mut hooks = Hook::all(&file, &ast);
        if let Some(tests) = &tests {
            hooks.extend(Hook::all(&tests_file, tests));
        }
        Self::lift(&file, &ast, &hooks)
    }
    /// Parse a source root without executing its Rust transitions.
    pub fn parse(file: &str, source: &str) -> Result<Lifted, Diagnostic> {
        let ast = syn::parse_file(source)
            .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))?;
        Self::lift(file, &ast, &Hook::all(file, &ast))
    }
    fn lift(file: &str, ast: &syn::File, hooks: &[Hook<'_>]) -> Result<Lifted, Diagnostic> {
        let macros: Vec<_> = hooks.iter().map(|hook| hook.mac).collect();
        let mut model = umpire_front::read(ast, &macros).map_err(|refusal| match refusal {
            umpire_front::Refusal::Model(location) => {
                Diagnostic::of(file, &location, location.message.clone())
            }
            umpire_front::Refusal::Hook(index, location) => {
                Diagnostic::of(hooks[index].file, &location, location.message.clone())
            }
        })?;
        let names = names::assign(ast, &mut model);
        let lifted = Lifted { file: file.into(), model, names };
        names::validate(file, ast, &lifted)?;
        Ok(lifted)
    }
}
/// A golden hook (`fixture!`) and the file it is written in.
pub(crate) struct Hook<'a> {
    file: &'a str,
    mac: &'a syn::ItemMacro,
}
impl<'a> Hook<'a> {
    fn all(file: &'a str, ast: &'a syn::File) -> Vec<Self> {
        ast.items
            .iter()
            .filter_map(|item| match item {
                syn::Item::Macro(mac)
                    if mac.mac.path.segments.last().is_some_and(|s| s.ident == "fixture") =>
                {
                    Some(Self { file, mac })
                }
                _ => None,
            })
            .collect()
    }
}
/// TLA+ emission from the internal typed tree.
pub struct Tla;
impl Tla {
    /// Emit a module with isolated property configurations.
    pub fn emit(lifted: &Lifted) -> Result<Module, Diagnostic> {
        emit::emit(lifted)
    }
}
