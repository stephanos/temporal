//! Lift authored Rust Models, admitted by `umpire-front`, into TLA+.
//!
//! The admitted subset, including the `imbl::Vector<T>` method table, is `umpire-front`'s.
use proc_macro2::Span;
use std::{collections::BTreeMap, fmt, path::Path};
use syn::spanned::Spanned;
#[doc(hidden)]
pub use umpire_backend_process as process;
pub mod values;
pub use process::{Limits, Refusal};

mod bind;
mod emit;
mod names;
mod safety;
mod snowcat;
use umpire_front::Source;
pub use umpire_front::{CATALOG_LIMIT, Embedded, Type};

/// Admit one plain TLA+ identifier, the only name form either checker is handed.
pub fn identifier(name: &str) -> Result<(), Refusal> {
    let mut bytes = name.bytes();
    if !bytes.next().is_some_and(|byte| byte.is_ascii_alphabetic() || byte == b'_')
        || !bytes.all(|byte| byte.is_ascii_alphanumeric() || byte == b'_')
    {
        return Err(Refusal(format!("invalid TLA+ identifier {name:?}")));
    }
    Ok(())
}

/// An admitted Model, the source file it was read from, and what binding selects.
#[derive(Clone, Debug)]
pub struct Lifted {
    file: String,
    model: umpire_front::Lifted,
    names: names::Names,
    /// Selected World values, after binding.
    world_values: Option<Vec<umpire_front::expr::Expr>>,
    /// Complete start states, after binding.
    complete_starts: Option<Vec<Vec<umpire_front::expr::Expr>>>,
    /// The bound concrete catalog, in native catalog order.
    catalog: Option<Vec<bind::Entry>>,
    /// The file the negative claims are written in, and where each claim's name is.
    claims: (String, Vec<umpire_front::Diagnostic>),
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
    /// Nominal integer with its authored wrapper identity.
    Bounded {
        /// Authored wrapper name.
        name: String,
        /// Raw backing integer.
        raw: i128,
    },
    /// Absent or present nominal payload.
    Optional(Option<Box<Literal>>),
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
/// One selected World and its complete authored start states.
#[derive(Clone, Debug)]
pub struct Binding {
    /// Named World fields.
    pub world: Vec<(String, Literal)>,
    /// Complete named state fields for each authored start, in enumeration order.
    pub starts: Vec<Vec<(String, Literal)>>,
    /// The checked native catalog of the same World, in catalog order.
    pub actions: Vec<Concrete>,
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
/// A typed World field and its collision-free emitted constant name.
#[derive(Clone, Debug)]
pub struct Parameter {
    /// Authored World field.
    pub field: String,
    /// Exact emitted parameter name.
    pub symbol: String,
    /// Independently admitted source type.
    pub ty: Type,
}
/// Exact decoding and property metadata.
#[derive(Clone, Debug)]
pub struct Metadata {
    /// Authored state struct name.
    pub state_name: String,
    /// Unit World constructor or declared World type path.
    pub world_path: String,
    /// Explicit typed World parameters; no naming convention is needed by consumers.
    pub parameters: Vec<Parameter>,
    /// The admitted selected World, complete starts and concrete catalog, when bound.
    pub binding: Option<Binding>,
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
    /// Dedicated safety checks for evaluated arithmetic faults.
    pub arithmetic: Vec<Property>,
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
/// Reading the admitted Model for emission.
pub struct Lift;
impl Lift {
    /// The Model a compiled `#[machine]` World embeds, for the source file the runner names it
    /// by, with the negative claims of the golden hook in the `tests.rs` beside it. A source
    /// that declares no `#[machine]` Model of that World is read only to locate the refusal.
    pub fn embedded(
        embedded: Option<&Embedded>,
        source: impl AsRef<Path>,
    ) -> Result<Lifted, Diagnostic> {
        let path = source.as_ref();
        let file = path.to_string_lossy();
        let Some(embedded) = embedded.filter(|embedded| same_file(&embedded.file, path)) else {
            return Err(unlifted(&file, &read(path, &file)?));
        };
        let mut model = embedded
            .model
            .clone()
            .map_err(|refusal| Diagnostic::of(&file, &refusal, refusal.message.clone()))?;
        let tests = path.with_file_name("tests.rs");
        let mut claims = (file.to_string(), Vec::new());
        if tests.is_file() {
            let tests_file = tests.to_string_lossy();
            let ast = read(&tests, &tests_file)?;
            claims = (
                tests_file.to_string(),
                umpire_front::supplement(&mut model, &hooks(&ast)).map_err(|(_, refusal)| {
                    Diagnostic::of(&tests_file, &refusal, refusal.message.clone())
                })?,
            );
        }
        Self::named(&file, model, &embedded.source, claims)
    }
    /// Read a source root as the `#[machine]` macro reads its module, without executing its Rust
    /// transitions; golden hooks in the same source supply negative claims.
    pub fn parse(file: &str, source: &str) -> Result<Lifted, Diagnostic> {
        let ast = syn::parse_file(source)
            .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))?;
        let (model, names, claims) =
            umpire_front::read(&ast, &hooks(&ast)).map_err(|refusal| match refusal {
                umpire_front::Refusal::Model(location)
                | umpire_front::Refusal::Hook(_, location) => {
                    Diagnostic::of(file, &location, location.message.clone())
                }
            })?;
        Self::named(file, model, &names, (file.into(), claims))
    }
    fn named(
        file: &str,
        mut model: umpire_front::Lifted,
        source: &Source,
        claims: (String, Vec<umpire_front::Diagnostic>),
    ) -> Result<Lifted, Diagnostic> {
        let names = names::assign(source, &mut model);
        let lifted = Lifted {
            file: file.into(),
            model,
            names,
            world_values: None,
            complete_starts: None,
            catalog: None,
            claims,
        };
        names::validate(file, source, &lifted)?;
        Ok(lifted)
    }
}
fn read(path: &Path, file: &str) -> Result<syn::File, Diagnostic> {
    let source = std::fs::read_to_string(path)
        .map_err(|error| Diagnostic::span(file, Span::call_site(), error.to_string()))?;
    syn::parse_file(&source)
        .map_err(|error| Diagnostic::span(file, error.span(), error.to_string()))
}
/// Whether the embedded file, as rustc named it, is the runner's source file.
fn same_file(embedded: &str, source: &Path) -> bool {
    match (Path::new(embedded).canonicalize(), source.canonicalize()) {
        (Ok(embedded), Ok(source)) => embedded == source,
        _ => false,
    }
}
/// The located refusal for a source without the checked World's `#[machine]` module.
fn unlifted(file: &str, ast: &syn::File) -> Diagnostic {
    let module = ast.items.iter().find(|item| {
        matches!(item, syn::Item::Mod(module) if module.attrs.iter().any(|attr| attr.path().is_ident("machine")))
    });
    if let Some(module) = module {
        return Diagnostic::span(
            file,
            module.span(),
            "the checked World is not the #[machine] World this source declares",
        );
    }
    let location = umpire_front::unlifted(ast);
    Diagnostic::of(file, &location, location.message.clone())
}
/// The golden hooks (`fixture!`) of a source.
fn hooks(ast: &syn::File) -> Vec<&syn::ItemMacro> {
    ast.items
        .iter()
        .filter_map(|item| match item {
            syn::Item::Macro(mac)
                if mac.mac.path.segments.last().is_some_and(|s| s.ident == "fixture") =>
            {
                Some(mac)
            }
            _ => None,
        })
        .collect()
}
/// TLA+ emission from the internal typed tree.
pub struct Tla;
impl Tla {
    /// Emit the source-derived bounded safety relation for Apalache.
    pub fn safety(lifted: &Lifted) -> Result<Module, Diagnostic> {
        safety::emit(lifted)
    }
    /// Emit a module with isolated property configurations.
    pub fn emit(lifted: &Lifted) -> Result<Module, Diagnostic> {
        emit::emit(lifted)
    }
}
