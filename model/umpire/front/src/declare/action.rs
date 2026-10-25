use super::fields::integer;
use crate::Errors;
use syn::{Expr, Field};

/// Where an action input field's values come from.
#[derive(Clone, Debug)]
pub enum FieldDomain {
    /// `#[inputs(..)]`: a bounded range or a finite list.
    Explicit(Expr),
    /// The field type's `Finite` domain.
    Finite,
    /// `#[domain(function)]`: values the declared World yields.
    World(syn::Path),
}

/// A `#[derive(Action)]` enum's markers and input domains.
#[derive(Clone, Debug)]
pub struct ActionEnum {
    /// `#[internal]`.
    pub internal: bool,
    /// `#[fault]`.
    pub fault: bool,
    /// Each variant's field domains, in declaration order.
    pub variants: Vec<Vec<FieldDomain>>,
}

/// Read an action enum; `world` says whether it belongs to a module that declares its World.
/// Every malformed declaration is reported, in source order.
pub fn action_enum(input: &syn::DeriveInput, world: bool) -> syn::Result<ActionEnum> {
    let syn::Data::Enum(data) = &input.data else {
        bail!(input, "Action requires an enum");
    };
    let name = &input.ident;
    let mut errors = Errors::default();
    let mut flags = [false; 2];
    for (flag, enabled) in ["internal", "fault"].into_iter().zip(&mut flags) {
        for attr in input.attrs.iter().filter(|attr| attr.path().is_ident(flag)) {
            if let Err(error) = attr.meta.require_path_only() {
                errors.push(error);
            }
            *enabled = true;
        }
    }
    for attr in input.attrs.iter().filter(|attr| attr.path().is_ident("domain")) {
        errors.push(syn::Error::new_spanned(
            attr,
            "#[domain] belongs on one variant field, not the action enum",
        ));
    }
    let [internal, fault] = flags;
    let mut variants = Vec::new();
    for variant in &data.variants {
        let ident = &variant.ident;
        for attr in &variant.attrs {
            for flag in ["internal", "fault"] {
                if attr.path().is_ident(flag) {
                    errors.push(syn::Error::new_spanned(
                        attr,
                        format!("#[{flag}] belongs on the action enum, not a variant"),
                    ));
                }
            }
            if attr.path().is_ident("domain") {
                errors.push(syn::Error::new_spanned(
                    attr,
                    "#[domain] belongs on one variant field, not the variant",
                ));
            }
        }
        let mut domains = Vec::new();
        for (index, field) in variant.fields.iter().enumerate() {
            match domain(field, &format!("{name}::{ident}"), index, world) {
                Ok(domain) => domains.push(domain),
                Err(error) => errors.push(error),
            }
        }
        variants.push(domains);
    }
    errors.finish(ActionEnum { internal, fault, variants })
}

fn domain(field: &Field, action: &str, index: usize, world: bool) -> syn::Result<FieldDomain> {
    let field_name = field.ident.as_ref().map_or_else(|| index.to_string(), ToString::to_string);
    let context = format!("{action} field `{field_name}`");
    let mut domains = field.attrs.iter().filter(|attr| attr.path().is_ident("domain"));
    if let Some(domain) = domains.next() {
        if let Some(duplicate) = domains.next() {
            bail!(duplicate, "{context} has duplicate #[domain] declarations");
        }
        if let Some(inputs) = field.attrs.iter().find(|attr| attr.path().is_ident("inputs")) {
            bail!(inputs, "{context} declares both #[inputs] and #[domain]; declare one domain");
        }
        let path = domain.parse_args::<syn::Path>().map_err(|_| {
            syn::Error::new_spanned(
                domain,
                format!("{context} #[domain] names a function, such as #[domain(sizes)]"),
            )
        })?;
        if !world {
            bail!(
                domain,
                "{context} #[domain] requires the World of a #[machine] module that declares #[derive(World)]"
            );
        }
        return Ok(FieldDomain::World(path));
    }
    let mut attrs = field.attrs.iter().filter(|attr| attr.path().is_ident("inputs"));
    let Some(attr) = attrs.next() else {
        if integer(&field.ty).is_some() {
            bail!(field, "{context} requires #[inputs] with a bounded range or finite list");
        }
        return Ok(FieldDomain::Finite);
    };
    if let Some(duplicate) = attrs.next() {
        bail!(duplicate, "{context} has duplicate #[inputs] domains");
    }
    let value = attr.parse_args::<Expr>()?;
    match &value {
        Expr::Range(range) if range.start.is_some() && range.end.is_some() => {
            Ok(FieldDomain::Explicit(value))
        }
        Expr::Array(_) => Ok(FieldDomain::Explicit(value)),
        _ => Err(syn::Error::new_spanned(
            attr,
            format!(
                "{context} #[inputs] requires a bounded range or finite list, such as 0..=3 or [false, true]"
            ),
        )),
    }
}
