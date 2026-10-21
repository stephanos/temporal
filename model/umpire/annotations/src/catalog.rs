//! Typed field access and admission.

#[path = "probe.rs"]
mod probe;
pub use probe::*;

use crate::{Consequence, FieldValue, GenerationError, Kind, RepeatedRules, Rng, Tag};
use std::{
    collections::{BTreeMap, BTreeSet},
    fmt,
};

/// Non-success gRPC protocol status names and wire discriminants.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
#[repr(i32)]
pub enum Code {
    /// Caller cancelled the operation.
    Cancelled = 1,
    /// Unknown failure.
    Unknown = 2,
    /// Invalid input.
    InvalidArgument = 3,
    /// Operation deadline elapsed.
    DeadlineExceeded = 4,
    /// Requested resource was absent.
    NotFound = 5,
    /// Resource already exists.
    AlreadyExists = 6,
    /// Caller lacks permission.
    PermissionDenied = 7,
    /// Resource limit reached.
    ResourceExhausted = 8,
    /// Required state was absent.
    FailedPrecondition = 9,
    /// Operation aborted.
    Aborted = 10,
    /// Value outside the valid range.
    OutOfRange = 11,
    /// Operation is unsupported.
    Unimplemented = 12,
    /// Internal failure.
    Internal = 13,
    /// Service unavailable.
    Unavailable = 14,
    /// Unrecoverable data loss.
    DataLoss = 15,
    /// Caller is unauthenticated.
    Unauthenticated = 16,
}

/// Expected server behavior for one violated constraint or presence rule.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Expectation {
    /// Any message with exactly this rejection status.
    RejectsCode(Code),
    /// Any non-success protocol status, irrespective of message.
    RejectsAnyStatus,
    /// The server rejects with these diagnostics.
    Rejects {
        /// Protocol status code.
        code: i32,
        /// Pattern interpreted by the system's probe.
        message_pattern: String,
        /// Optional structured error detail type.
        details_type: Option<String>,
    },
    /// The server intentionally accepts this value.
    Accepts,
    /// The rule cannot be checked against a request.
    Unobservable,
}

/// Expected observable behavior for a declared rule.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Expected {
    /// Server behavior.
    pub expectation: Expectation,
}

/// Presence algebra over catalog field paths, relative to the message.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum Presence {
    /// Every named field is present.
    Required(Vec<String>),
    /// At least one named field is present.
    AnyOf(Vec<String>),
    /// Exactly one named field is present.
    OneOf(Vec<String>),
    /// Presence of the first field requires the second.
    Requires(String, String),
    /// The two fields cannot both be present.
    Excludes(String, String),
}

impl Presence {
    /// JSON Schema operator name.
    pub fn name(&self) -> &'static str {
        match self {
            Self::Required(_) => "required",
            Self::AnyOf(_) => "any_of",
            Self::OneOf(_) => "one_of",
            Self::Requires(_, _) => "requires",
            Self::Excludes(_, _) => "excludes",
        }
    }
    /// Fields referenced by this rule.
    pub fn fields(&self) -> Vec<&str> {
        match self {
            Self::Required(fields) | Self::AnyOf(fields) | Self::OneOf(fields) => {
                fields.iter().map(String::as_str).collect()
            }
            Self::Requires(a, b) | Self::Excludes(a, b) => vec![a, b],
        }
    }
    fn holds(&self, present: impl Fn(&str) -> bool) -> bool {
        match self {
            Self::Required(fields) => fields.iter().all(|field| present(field)),
            Self::AnyOf(fields) => fields.iter().any(|field| present(field)),
            Self::OneOf(fields) => fields.iter().filter(|field| present(field)).count() == 1,
            Self::Requires(a, b) => !present(a) || present(b),
            Self::Excludes(a, b) => !present(a) || !present(b),
        }
    }
}

/// A message rule and its optional request expectation.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Rule {
    /// Presence operation.
    pub presence: Presence,
    /// Expected server behavior.
    pub expected: Option<Expected>,
}

/// Public metadata retained independently of erased field operations.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct FieldDescriptor {
    /// Author-written documentation, never a runtime field value.
    pub description: String,
    kind_constraints: Vec<&'static str>,
    /// Fully qualified field path.
    pub path: String,
    /// Actual Rust field type, retained for macro diagnostics.
    pub rust_type: &'static str,
    /// Semantic kind.
    pub kind: Kind,
    /// Tag names paired with their concrete consequences.
    pub tags: Vec<(String, Consequence)>,
    /// Repeated constraints, when declared.
    pub repeated: Option<RepeatedRules>,
    /// Expected server behavior per stable constraint name.
    pub expectations: BTreeMap<String, Expected>,
}

impl FieldDescriptor {
    /// All constraints requiring expectations on requests.
    pub fn constraints(&self) -> Vec<&'static str> {
        let mut names = self.kind_constraints.clone();
        if let Some(rules) = &self.repeated {
            if rules.min_items.is_some() {
                names.push("repeated.min_items");
            }
            if rules.max_items.is_some() {
                names.push("repeated.max_items");
            }
            if rules.unique {
                names.push("repeated.unique");
            }
        }
        names
    }
    fn ignored(&self) -> bool {
        self.tags.iter().any(|(_, tag)| matches!(tag, Consequence::Ignored(_)))
    }
    fn decided(&self) -> bool {
        self.tags.iter().any(|(_, tag)| {
            matches!(tag, Consequence::Decided | Consequence::Observed | Consequence::Ignored(_))
        })
    }
}

type Getter<M, T> = Box<dyn Fn(&M) -> T>;
type Setter<M, T> = Box<dyn Fn(&mut M, T)>;
type FallibleSetter<M, T> = Box<dyn Fn(&mut M, T) -> Result<(), GenerationError>>;

/// A field whose accessors are checked against the concrete Rust message type.
pub struct Field<M, T> {
    descriptor: FieldDescriptor,
    get: Getter<M, T>,
    set: FallibleSetter<M, T>,
    errors: Vec<AdmissionError>,
}

impl<M: 'static, T: FieldValue> Field<M, T> {
    /// Borrows the actual Rust field, without reflection or string lookup.
    pub fn new(
        path: impl Into<String>,
        kind: Kind,
        get: fn(&M) -> &T,
        set: fn(&mut M) -> &mut T,
    ) -> Self {
        Self::projected(
            path,
            kind,
            move |message| get(message).clone(),
            move |message, value| *set(message) = value,
        )
    }
    /// Typed projection for generated wrappers, nested fields and timestamp carriers.
    pub fn projected(
        path: impl Into<String>,
        kind: Kind,
        get: impl Fn(&M) -> T + 'static,
        set: impl Fn(&mut M, T) + 'static,
    ) -> Self {
        Self::try_projected(path, kind, get, move |message, value| {
            set(message, value);
            Ok(())
        })
    }
    /// Typed projection whose setter may refuse an unrepresentable value.
    pub fn try_projected(
        path: impl Into<String>,
        kind: Kind,
        get: impl Fn(&M) -> T + 'static,
        set: impl Fn(&mut M, T) -> Result<(), GenerationError> + 'static,
    ) -> Self {
        let path = path.into();
        let errors = if T::supports(&kind) {
            Vec::new()
        } else {
            vec![AdmissionError::UnsupportedKind {
                path: path.clone(),
                kind: kind.clone(),
                rust_type: std::any::type_name::<T>(),
            }]
        };
        Self {
            descriptor: FieldDescriptor {
                description: String::new(),
                kind_constraints: T::constraints(&kind),
                path,
                rust_type: std::any::type_name::<T>(),
                kind,
                tags: Vec::new(),
                repeated: None,
                expectations: BTreeMap::new(),
            },
            get: Box::new(get),
            set: Box::new(set),
            errors,
        }
    }
    /// Retains author-written field documentation.
    #[must_use]
    pub fn description(mut self, description: impl Into<String>) -> Self {
        self.descriptor.description = description.into();
        self
    }
    /// Adds a tag after checking its declared kind applicability.
    #[must_use]
    pub fn tag(mut self, tag: &impl Tag) -> Self {
        if !tag.applies_to(&self.descriptor.kind) {
            self.errors.push(AdmissionError::InapplicableTag {
                path: self.descriptor.path.clone(),
                tag: tag.name().into(),
            });
        }
        self.descriptor.tags.push((tag.name().into(), tag.consequence()));
        self
    }
    /// Adds repeated constraints; scalar fields are refused at admission.
    #[must_use]
    pub fn repeated(mut self, rules: RepeatedRules) -> Self {
        if !T::is_repeated() {
            self.errors.push(AdmissionError::NotRepeated { path: self.descriptor.path.clone() });
        }
        self.descriptor.repeated = Some(rules);
        self
    }
    /// Declares expected observable behavior for all current constraints.
    #[must_use]
    pub fn expects(mut self, expected: &Expected) -> Self {
        for constraint in self.descriptor.constraints() {
            self.descriptor.expectations.insert(constraint.into(), expected.clone());
        }
        self
    }
    /// Declares expected observable behavior for one named constraint.
    #[must_use]
    pub fn expectation(mut self, constraint: impl Into<String>, expected: Expected) -> Self {
        self.descriptor.expectations.insert(constraint.into(), expected);
        self
    }
}

/// An opaque typed field explicitly excluded from validation and generation.
pub struct IgnoredField<M, T> {
    descriptor: FieldDescriptor,
    set: Setter<M, T>,
    errors: Vec<AdmissionError>,
}

impl<M: 'static, T: Clone + Default + 'static> IgnoredField<M, T> {
    /// Checks the actual field type and retains its kind, including secret redaction.
    pub fn new(
        path: impl Into<String>,
        kind: Kind,
        get: fn(&M) -> &T,
        set: fn(&mut M) -> &mut T,
        reason: impl Into<String>,
    ) -> Self {
        Self::projected(
            path,
            kind,
            move |message| get(message).clone(),
            move |message, value| *set(message) = value,
            reason,
        )
    }
    /// Typed projection for ignored wrappers, nested fields and oneof payloads.
    /// Only secret or unconstrained opaque bytes kinds admit arbitrary Rust types.
    pub fn projected(
        path: impl Into<String>,
        kind: Kind,
        _get: impl Fn(&M) -> T + 'static,
        set: impl Fn(&mut M, T) + 'static,
        reason: impl Into<String>,
    ) -> Self {
        let path = path.into();
        let supported = matches!(&kind, Kind::Secret)
            || matches!(&kind, Kind::Bytes(rules) if rules.min_len.is_none() && rules.max_len.is_none());
        let errors = if supported {
            Vec::new()
        } else {
            vec![AdmissionError::UnsupportedKind {
                path: path.clone(),
                kind: kind.clone(),
                rust_type: std::any::type_name::<T>(),
            }]
        };
        Self {
            descriptor: FieldDescriptor {
                description: String::new(),
                kind_constraints: Vec::new(),
                path,
                rust_type: std::any::type_name::<T>(),
                kind,
                tags: vec![("Ignored".into(), Consequence::Ignored(reason.into()))],
                repeated: None,
                expectations: BTreeMap::new(),
            },
            set: Box::new(set),
            errors,
        }
    }
}

trait Access<M> {
    fn descriptor(&self) -> &FieldDescriptor;
    fn descriptor_mut(&mut self) -> &mut FieldDescriptor;
    fn errors(&self) -> &[AdmissionError];
    fn present(&self, message: &M) -> bool;
    fn violations(&self, message: &M) -> Vec<&'static str>;
    fn fill(&self, message: &mut M, rng: &mut Rng) -> Result<(), GenerationError>;
    fn clear(&self, message: &mut M) -> Result<(), GenerationError>;
    fn negative(
        &self,
        _message: &mut M,
        _constraint: &str,
        _rng: &mut Rng,
    ) -> Result<(), GenerationError> {
        Err(GenerationError::UnsupportedType)
    }
}

impl<M: 'static, T: Clone + Default + 'static> Access<M> for IgnoredField<M, T> {
    fn descriptor(&self) -> &FieldDescriptor {
        &self.descriptor
    }
    fn descriptor_mut(&mut self) -> &mut FieldDescriptor {
        &mut self.descriptor
    }
    fn errors(&self) -> &[AdmissionError] {
        &self.errors
    }
    fn present(&self, _message: &M) -> bool {
        false
    }
    fn violations(&self, _message: &M) -> Vec<&'static str> {
        Vec::new()
    }
    fn fill(&self, _message: &mut M, _rng: &mut Rng) -> Result<(), GenerationError> {
        Ok(())
    }
    fn clear(&self, message: &mut M) -> Result<(), GenerationError> {
        (self.set)(message, T::default());
        Ok(())
    }
}

impl<M: 'static, T: FieldValue> Access<M> for Field<M, T> {
    fn negative(
        &self,
        message: &mut M,
        constraint: &str,
        rng: &mut Rng,
    ) -> Result<(), GenerationError> {
        let value = (self.get)(message).negative(
            &self.descriptor.kind,
            self.descriptor.repeated.as_ref(),
            constraint,
            rng,
        )?;
        (self.set)(message, value)
    }
    fn descriptor(&self) -> &FieldDescriptor {
        &self.descriptor
    }
    fn descriptor_mut(&mut self) -> &mut FieldDescriptor {
        &mut self.descriptor
    }
    fn errors(&self) -> &[AdmissionError] {
        &self.errors
    }
    fn present(&self, message: &M) -> bool {
        (self.get)(message).present()
    }
    fn violations(&self, message: &M) -> Vec<&'static str> {
        let value = (self.get)(message);
        let mut names = value.violations(&self.descriptor.kind);
        if let Some(rules) = &self.descriptor.repeated {
            names.extend(value.repeated_violations(rules));
        }
        names.sort_unstable();
        names.dedup();
        names
    }
    fn fill(&self, message: &mut M, rng: &mut Rng) -> Result<(), GenerationError> {
        let mut value = match &self.descriptor.repeated {
            Some(rules) => T::generate_repeated(&self.descriptor.kind, rules, rng)?,
            None => T::generate(&self.descriptor.kind, rng)?,
        };
        if !value.present() {
            let present = match &self.descriptor.repeated {
                Some(rules) => {
                    let mut rules = rules.clone();
                    rules.min_items = Some(rules.min_items.unwrap_or(0).max(1));
                    T::generate_repeated(&self.descriptor.kind, &rules, rng)
                }
                None => T::generate_present(&self.descriptor.kind, rng),
            };
            match present {
                Ok(present) => value = present,
                Err(GenerationError::ImpossibleConstraints) => {}
                Err(reason) => return Err(reason),
            }
        }
        (self.set)(message, value)
    }
    fn clear(&self, message: &mut M) -> Result<(), GenerationError> {
        (self.set)(message, T::default())
    }
}

/// Typed admission failures retained for compile-time macro diagnostics.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum AdmissionError {
    /// Rust type and semantic kind disagree.
    UnsupportedKind {
        /// Field path.
        path: String,
        /// Requested kind.
        kind: Kind,
        /// Actual Rust type.
        rust_type: &'static str,
    },
    /// The tag excludes this kind.
    InapplicableTag {
        /// Field path.
        path: String,
        /// Tag name.
        tag: String,
    },
    /// A repeated constraint was attached to a scalar.
    NotRepeated {
        /// Field path.
        path: String,
    },
    /// Roles or semantic consequences contradict one another.
    ConflictingTags {
        /// Field path.
        path: String,
        /// Conflicting names.
        tags: Vec<String>,
    },
    /// An ignored field has no explanation.
    MissingIgnoreReason {
        /// Field path.
        path: String,
    },
    /// A path is empty, duplicated, or outside its message.
    InvalidPath(String),
    /// A presence rule names a field missing from the catalog.
    UnknownRuleField(String),
    /// Rules cannot depend on ignored fields.
    IgnoredRuleField(String),
    /// An operator has no operands or duplicates an operand.
    InvalidRule(String),
    /// Every request constraint and rule needs an expected outcome.
    MissingExpectation {
        /// Field or message path.
        path: String,
        /// Stable constraint name.
        constraint: String,
    },
    /// Expected outcomes must name an existing constraint.
    InvalidExpectation {
        /// Field or message path.
        path: String,
        /// Constraint name.
        constraint: String,
    },
    /// A consequence has an empty quantitative bound.
    InvalidConsequence {
        /// Field path.
        path: String,
        /// Tag name.
        tag: String,
    },
}

/// All errors found during one catalog admission.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct AdmissionErrors(pub Vec<AdmissionError>);
impl fmt::Display for AdmissionErrors {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        for error in &self.0 {
            writeln!(f, "{error:?}")?;
        }
        Ok(())
    }
}
impl std::error::Error for AdmissionErrors {}

/// An admitted catalog with typed operations over one message type.
pub struct Catalog<M> {
    description: String,
    inherited_patterns: Vec<Presence>,
    message: String,
    request: bool,
    fields: BTreeMap<String, Box<dyn Access<M>>>,
    rules: Vec<Rule>,
}

/// Unadmitted catalog; callers must handle `build` before use.
pub struct CatalogBuilder<M> {
    catalog: Catalog<M>,
    errors: Vec<AdmissionError>,
}

impl<M: 'static> Catalog<M> {
    /// Starts a request catalog, requiring expectations on all constraints and rules.
    pub fn request(message: impl Into<String>) -> CatalogBuilder<M> {
        Self::builder(message, true)
    }
    /// Starts a response catalog; validation failures are server findings.
    pub fn response(message: impl Into<String>) -> CatalogBuilder<M> {
        Self::builder(message, false)
    }
    fn builder(message: impl Into<String>, request: bool) -> CatalogBuilder<M> {
        CatalogBuilder {
            catalog: Self {
                description: String::new(),
                inherited_patterns: Vec::new(),
                message: message.into(),
                request,
                fields: BTreeMap::new(),
                rules: Vec::new(),
            },
            errors: Vec::new(),
        }
    }
    /// Message's fully qualified name.
    pub fn message(&self) -> &str {
        &self.message
    }
    /// Whether this catalog describes a request.
    pub fn is_request(&self) -> bool {
        self.request
    }
    /// Metadata sorted by fully qualified path.
    pub fn fields(&self) -> impl Iterator<Item = &FieldDescriptor> {
        self.fields.values().map(|field| field.descriptor())
    }
    /// Message rules in declaration order.
    pub fn rules(&self) -> &[Rule] {
        &self.rules
    }
    /// Clears external secrets before a message is written into a Run.
    pub fn redact(&self, message: &mut M) -> Result<(), GenerationError> {
        for field in self.fields.values() {
            if matches!(field.descriptor().kind, Kind::Secret) {
                field.clear(message)?;
            }
        }
        Ok(())
    }
    fn qualified(&self, path: &str) -> String {
        if path.starts_with(&format!("{}.", self.message)) {
            path.into()
        } else {
            format!("{}.{path}", self.message)
        }
    }
    fn qualified_fields(&self, rule: &Rule) -> Vec<String> {
        rule.presence.fields().into_iter().map(|field| self.qualified(field)).collect()
    }
    fn rule_fields(&self, rule: &Rule) -> Vec<String> {
        let mut fields = self.qualified_fields(rule);
        if !matches!(rule.presence, Presence::Requires(_, _)) {
            fields.sort();
        }
        fields
    }
}

impl<M: 'static> CatalogBuilder<M> {
    /// Retains author-written message documentation.
    #[must_use]
    pub fn description(mut self, description: impl Into<String>) -> Self {
        self.catalog.description = description.into();
        self
    }
    /// Marks a presence rule whose expectation came from its field constraint.
    #[must_use]
    pub fn inherited_pattern(mut self, presence: Presence) -> Self {
        self.catalog.inherited_patterns.push(presence);
        self
    }
    /// Adds one typed field.
    #[must_use]
    pub fn field<T: FieldValue>(self, field: Field<M, T>) -> Self {
        self.add(field)
    }
    /// Adds an opaque field with a mandatory ignored role and explanation.
    #[must_use]
    pub fn ignored<T: Clone + Default + 'static>(self, field: IgnoredField<M, T>) -> Self {
        self.add(field)
    }
    fn add(mut self, mut field: impl Access<M> + 'static) -> Self {
        self.errors.extend_from_slice(field.errors());
        let path = self.catalog.qualified(&field.descriptor().path);
        field.descriptor_mut().path.clone_from(&path);
        if self.catalog.fields.insert(path.clone(), Box::new(field)).is_some() {
            self.errors.push(AdmissionError::InvalidPath(path));
        }
        self
    }
    /// Adds a message presence rule.
    #[must_use]
    pub fn rule(mut self, rule: Rule) -> Self {
        self.catalog.rules.push(rule);
        self
    }
    /// Checks every declaration before exposing executable operations.
    pub fn build(mut self) -> Result<Catalog<M>, AdmissionErrors> {
        if self.catalog.message.trim().is_empty() {
            self.errors.push(AdmissionError::InvalidPath(self.catalog.message.clone()));
        }
        for field in self.catalog.fields.values() {
            let d = field.descriptor();
            if d.path.ends_with('.') || d.path.contains("..") {
                self.errors.push(AdmissionError::InvalidPath(d.path.clone()));
            }
            let roles: Vec<_> = d
                .tags
                .iter()
                .filter(|(_, tag)| {
                    matches!(
                        tag,
                        Consequence::Decided | Consequence::Observed | Consequence::Ignored(_)
                    )
                })
                .collect();
            let conflict = roles.len() > 1
                || (matches!(d.kind, Kind::Secret)
                    && d.tags.iter().any(|(_, t)| matches!(t, Consequence::Observed)))
                || (matches!(d.kind, Kind::Random(_))
                    && d.tags.iter().any(|(_, t)| matches!(t, Consequence::Decided)));
            if conflict {
                self.errors.push(AdmissionError::ConflictingTags {
                    path: d.path.clone(),
                    tags: d.tags.iter().map(|(name, _)| name.clone()).collect(),
                });
            }
            for (name, tag) in &d.tags {
                if matches!(tag, Consequence::Ignored(reason) if reason.trim().is_empty()) {
                    self.errors.push(AdmissionError::MissingIgnoreReason { path: d.path.clone() });
                }
                if matches!(tag, Consequence::Cached { ttl } if ttl.is_zero())
                    || matches!(tag, Consequence::EventuallyConsistent { polls: 0 })
                {
                    self.errors.push(AdmissionError::InvalidConsequence {
                        path: d.path.clone(),
                        tag: name.clone(),
                    });
                }
            }
            let constraints = d.constraints();
            if self.catalog.request && !d.ignored() {
                for constraint in &constraints {
                    if !d.expectations.contains_key(*constraint) {
                        self.errors.push(AdmissionError::MissingExpectation {
                            path: d.path.clone(),
                            constraint: (*constraint).into(),
                        });
                    }
                }
            }
            for constraint in d.expectations.keys() {
                if !constraints.contains(&constraint.as_str()) {
                    self.errors.push(AdmissionError::InvalidExpectation {
                        path: d.path.clone(),
                        constraint: constraint.clone(),
                    });
                }
            }
        }
        let mut rule_identities = BTreeSet::new();
        for rule in &self.catalog.rules {
            let fields = self.catalog.qualified_fields(rule);
            if !rule_identities.insert((rule.presence.name(), self.catalog.rule_fields(rule))) {
                self.errors.push(AdmissionError::InvalidRule(rule.presence.name().into()));
            }
            if fields.is_empty()
                || fields.iter().enumerate().any(|(i, field)| fields[..i].contains(field))
            {
                self.errors.push(AdmissionError::InvalidRule(rule.presence.name().into()));
            }
            for qualified in fields {
                match self.catalog.fields.get(&qualified) {
                    None => self.errors.push(AdmissionError::UnknownRuleField(qualified)),
                    Some(field) if field.descriptor().ignored() => {
                        self.errors.push(AdmissionError::IgnoredRuleField(qualified));
                    }
                    Some(_) => {}
                }
            }
            if self.catalog.request && rule.expected.is_none() {
                self.errors.push(AdmissionError::MissingExpectation {
                    path: self.catalog.message.clone(),
                    constraint: rule.presence.name().into(),
                });
            }
        }
        if self.errors.is_empty() { Ok(self.catalog) } else { Err(AdmissionErrors(self.errors)) }
    }
}

/// One validation finding, containing no field values.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct Violation {
    /// Fully qualified field or message path.
    pub path: String,
    /// Stable constraint or presence operator name.
    pub constraint: String,
    /// Related fields for message presence rules.
    pub fields: Vec<String>,
    /// Expected server behavior.
    pub expected: Option<Expected>,
}

impl fmt::Display for Violation {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}: {}", self.path, self.constraint)?;
        if !self.fields.is_empty() {
            write!(f, " ({})", self.fields.join(", "))?;
        }
        Ok(())
    }
}

/// Validates actual typed values without mutation, sorted by path and constraint.
pub fn validate<M: 'static>(message: &M, catalog: &Catalog<M>) -> Vec<Violation> {
    let mut violations = Vec::new();
    for field in catalog.fields.values() {
        let d = field.descriptor();
        if d.ignored() {
            continue;
        }
        for name in field.violations(message) {
            violations.push(Violation {
                path: d.path.clone(),
                constraint: name.into(),
                fields: Vec::new(),
                expected: d.expectations.get(name).cloned(),
            });
        }
    }
    for rule in &catalog.rules {
        if !rule.presence.holds(|path| catalog.fields[&catalog.qualified(path)].present(message)) {
            violations.push(Violation {
                path: catalog.message.clone(),
                constraint: rule.presence.name().into(),
                fields: rule
                    .presence
                    .fields()
                    .into_iter()
                    .map(|path| catalog.qualified(path))
                    .collect(),
                expected: rule.expected.clone(),
            });
        }
    }
    violations.sort_by(|a, b| {
        (&a.path, &a.constraint, &a.fields).cmp(&(&b.path, &b.constraint, &b.fields))
    });
    violations
}

/// A fill refusal with its field path, or remaining validation findings.
#[derive(Clone, Debug, PartialEq, Eq)]
pub enum FillError {
    /// Generation failed for a named field.
    Field {
        /// Fully qualified path.
        path: String,
        /// Typed refusal.
        reason: GenerationError,
    },
    /// Fixed values or presence constraints prevent a valid message.
    Invalid(Vec<Violation>),
}
impl fmt::Display for FillError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Self::Field { path, reason } => write!(f, "{path}: {reason}"),
            Self::Invalid(violations) => {
                for violation in violations {
                    writeln!(f, "{violation}")?;
                }
                Ok(())
            }
        }
    }
}
impl std::error::Error for FillError {}

/// Transactionally fills absent non-role fields, preserving supplied values and RNG on refusal.
pub fn fill<M: Clone + 'static>(
    message: &mut M,
    catalog: &Catalog<M>,
    rng: &mut Rng,
) -> Result<(), FillError> {
    let mut candidate = message.clone();
    let mut stream = rng.clone();
    let rule_fields: BTreeSet<_> = catalog
        .rules
        .iter()
        .flat_map(|rule| rule.presence.fields())
        .map(|path| catalog.qualified(path))
        .collect();
    let mut filled = Vec::new();
    for field in catalog.fields.values() {
        if field.descriptor().decided() || field.present(&candidate) {
            continue;
        }
        field
            .fill(&mut candidate, &mut stream)
            .map_err(|reason| FillError::Field { path: field.descriptor().path.clone(), reason })?;
        if rule_fields.contains(&field.descriptor().path) {
            filled.push(field.as_ref());
        }
    }
    let mut budget = 65_536;
    if let Some(valid) = select_presence(&candidate, catalog, &filled, 0, &mut budget)? {
        *message = valid;
        *rng = stream;
        Ok(())
    } else {
        Err(FillError::Invalid(validate(&candidate, catalog)))
    }
}

fn select_presence<M: Clone + 'static>(
    candidate: &M,
    catalog: &Catalog<M>,
    filled: &[&dyn Access<M>],
    index: usize,
    budget: &mut usize,
) -> Result<Option<M>, FillError> {
    if *budget == 0 || index > 64 {
        return Err(FillError::Field {
            path: catalog.message.clone(),
            reason: GenerationError::SizeLimit,
        });
    }
    *budget -= 1;
    if validate(candidate, catalog).is_empty() {
        return Ok(Some(candidate.clone()));
    }
    if index == filled.len() {
        return Ok(None);
    }
    if let Some(valid) = select_presence(candidate, catalog, filled, index + 1, budget)? {
        return Ok(Some(valid));
    }
    let mut absent = candidate.clone();
    filled[index].clear(&mut absent).map_err(|reason| FillError::Field {
        path: filled[index].descriptor().path.clone(),
        reason,
    })?;
    select_presence(&absent, catalog, filled, index + 1, budget)
}
