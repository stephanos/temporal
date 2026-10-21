//! Parsed declaration data shared with the compiler and descriptor projection.

use syn::{Expr, Ident, Lit, LitStr, Path, Type, Visibility};

/// One named collection of checked annotations.
#[derive(Debug)]
pub struct Catalog {
    /// Generated module visibility.
    pub vis: Visibility,
    /// Generated module name.
    pub name: Ident,
    /// Message catalogs.
    pub messages: Vec<Message>,
    /// Enum conversion functions.
    pub enums: Vec<Enum>,
    /// Explicit service classifications.
    pub services: Vec<Service>,
    /// Model action bindings.
    pub actions: Vec<Actions>,
    /// Declarative custom tags.
    pub tags: Vec<CustomTag>,
}

/// A typed message catalog factory.
#[derive(Debug)]
pub struct Message {
    /// Request expectation policy.
    pub request: bool,
    /// Factory name.
    pub factory: Ident,
    /// Concrete borrowed struct type.
    pub ty: Path,
    /// Protobuf full name.
    pub proto: LitStr,
    /// Exhaustive fields.
    pub fields: Vec<Field>,
    /// Presence rules.
    pub rules: Vec<Rule>,
}

/// One actual Rust struct field.
#[derive(Debug)]
pub struct Field {
    /// Explicit opaque adapter for an ignored Bytes or Secret field.
    pub opaque: bool,
    /// Actual Rust field name.
    pub name: Ident,
    /// One semantic kind.
    pub kind: Ident,
    /// Literal constraints.
    pub constraints: Vec<(Ident, Lit)>,
    /// Builtin or declarative tags.
    pub tags: Vec<Tag>,
    /// Checked enum declaration factory referenced by Enum kind.
    pub enumeration: Option<Ident>,
    /// Optional typed getter/setter projection.
    pub projection: Option<Projection>,
    /// Oneof source fields and their exhaustive Rust patterns.
    pub variants: Vec<Variant>,
    /// Request expectation shared by field constraints.
    pub expected: Option<Expected>,
}

/// Typed projection for wrapper fields.
#[derive(Debug)]
pub struct Projection {
    /// Projected value type.
    pub ty: Type,
    /// Getter closure.
    pub get: Expr,
    /// Setter closure.
    pub set: Expr,
}

/// A known Rust enum variant and its protobuf spelling.
#[derive(Debug)]
pub struct Variant {
    /// Rust variant pattern (a unit variant for enums, tuple payload for oneofs).
    pub path: Path,
    /// Protobuf descriptor spelling.
    pub proto: LitStr,
}

/// A literal builtin/custom tag application.
#[derive(Debug, Clone)]
pub struct Tag {
    /// Declared tag name.
    pub name: Ident,
    /// Literal consequence bound or ignored reason.
    pub argument: Option<Lit>,
}

/// A tag whose applicability and consequence are statically known.
#[derive(Debug)]
pub struct CustomTag {
    /// Generated type name implementing the generic Tag trait.
    pub name: Ident,
    /// Compatible semantic kind names.
    pub kinds: Vec<Ident>,
    /// Declared generic consequence.
    pub consequence: Tag,
}

/// Expected server behavior with validator provenance.
#[derive(Debug)]
pub struct Expected {
    /// Accepts, Unobservable or Rejects.
    pub outcome: Ident,
    /// Status code, message pattern, optional details type for Rejects.
    pub reject: Option<(syn::LitInt, LitStr, Option<LitStr>)>,
    /// Server validator.
    pub covers: LitStr,
}

/// One presence algebra rule.
#[derive(Debug)]
pub struct Rule {
    /// `required`, `any_of`, `one_of`, `requires` or `excludes`.
    pub operator: Ident,
    /// Referenced source fields, including declared oneof alternatives.
    pub fields: Vec<Ident>,
    /// Expected outcome.
    pub expected: Option<Expected>,
}

/// Exhaustive known enum conversion declaration.
#[derive(Debug)]
pub struct Enum {
    /// Generated conversion function.
    pub factory: Ident,
    /// Rust enum type.
    pub ty: Path,
    /// Protobuf full name.
    pub proto: LitStr,
    /// Explicit known variants.
    pub variants: Vec<Variant>,
}

/// One named protobuf service.
#[derive(Debug)]
pub struct Service {
    /// Protobuf full name.
    pub proto: LitStr,
    /// Exhaustive methods.
    pub methods: Vec<(Ident, Binding)>,
}

/// One model action leaf or service classification.
#[derive(Debug)]
pub struct Binding {
    /// `Rpc`, `Observed`, `Environmental`, `Action` or `OutOfScope`.
    pub kind: Ident,
    /// Fully qualified RPC, model action, observation or reason.
    pub target: LitStr,
}

/// Exhaustive outer and inner model action matches.
#[derive(Debug)]
pub struct Actions {
    /// Callable binding function name.
    pub factory: Ident,
    /// Outer carrier type.
    pub ty: Path,
    /// One outer actor constructor and all its inner action patterns.
    pub actors: Vec<(Path, Vec<(syn::Pat, Binding)>)>,
}
