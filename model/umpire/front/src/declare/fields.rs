use quote::ToTokens;
use syn::{Attribute, Expr, ExprRange, Field, Type};

/// The named fields of a `State` or, with `world`, a `World` struct.
pub fn named_fields(input: &syn::DeriveInput, world: bool) -> syn::Result<Vec<&Field>> {
    let fields = match &input.data {
        syn::Data::Struct(data) => Some(&data.fields),
        _ => None,
    };
    struct_fields(input, fields, world)
}

/// [`named_fields`] of a declaration located at `node`, which is a struct when it has `fields`.
pub(crate) fn struct_fields<'a>(
    node: &impl ToTokens,
    fields: Option<&'a syn::Fields>,
    world: bool,
) -> syn::Result<Vec<&'a Field>> {
    let label = if world { "World" } else { "State" };
    let Some(syn::Fields::Named(fields)) = fields else {
        bail!(node, "{label} requires a named-field struct");
    };
    Ok(fields.named.iter().collect())
}

/// Each field's `#[starts]` range; every malformed range is reported.
pub fn start_ranges(fields: &[&Field]) -> syn::Result<Vec<Option<ExprRange>>> {
    let mut errors: Option<syn::Error> = None;
    let mut emit = |error: syn::Error| match &mut errors {
        Some(previous) => previous.combine(error),
        None => errors = Some(error),
    };
    let mut ranges = Vec::new();
    for field in fields {
        let mut range = None;
        for attr in field.attrs.iter().filter(|attr| attr.path().is_ident("starts")) {
            match start_range(attr) {
                Ok(value) if range.is_none() => range = Some(value),
                Ok(_) => emit(syn::Error::new_spanned(attr, "duplicate #[starts] domain")),
                Err(error) => emit(error),
            }
        }
        ranges.push(range);
    }
    errors.map_or(Ok(ranges), Err)
}

fn start_range(attr: &Attribute) -> syn::Result<ExprRange> {
    match attr.parse_args::<Expr>()? {
        Expr::Range(range) if range.start.is_some() && range.end.is_some() => Ok(range),
        value => Err(syn::Error::new_spanned(
            value,
            "#[starts] requires a bounded range, such as 0..=40",
        )),
    }
}

const SHAPE: &str =
    "Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field";

/// A `derive(Finite)` declaration.
#[derive(Clone, Debug)]
#[expect(clippy::large_enum_variant, reason = "read once per derive")]
pub enum Finite {
    /// A fieldless enum and its variants.
    Enum(Vec<syn::Ident>),
    /// A struct with one integer field and its inclusive range.
    Newtype {
        /// The field name; `None` for a tuple struct.
        field: Option<syn::Ident>,
        /// The primitive integer type.
        ty: Type,
        /// Inclusive lower bound.
        low: Expr,
        /// Inclusive upper bound.
        high: Expr,
    },
}

/// Read a `derive(Finite)` declaration.
pub fn finite(input: &syn::DeriveInput) -> syn::Result<Finite> {
    match &input.data {
        syn::Data::Enum(data) => {
            if let Some(variant) = data.variants.iter().find(|variant| !variant.fields.is_empty()) {
                bail!(variant, "{SHAPE}; give a variant with fields to derive(Action)");
            }
            Ok(Finite::Enum(data.variants.iter().map(|variant| variant.ident.clone()).collect()))
        }
        syn::Data::Struct(data) => {
            let [field] = data.fields.iter().collect::<Vec<_>>()[..] else {
                return Err(syn::Error::new_spanned(input, SHAPE));
            };
            let mut ranges = field.attrs.iter().filter(|attr| attr.path().is_ident("inputs"));
            let Some(attr) = ranges.next() else {
                return Err(syn::Error::new_spanned(field, SHAPE));
            };
            if let Some(duplicate) = ranges.next() {
                bail!(duplicate, "a Finite newtype declares its range once");
            }
            let (low, high) = bounds(attr, &field.ty)?;
            Ok(Finite::Newtype { field: field.ident.clone(), ty: field.ty.clone(), low, high })
        }
        syn::Data::Union(_) => Err(syn::Error::new_spanned(input, SHAPE)),
    }
}

fn bounds(attr: &Attribute, ty: &Type) -> syn::Result<(Expr, Expr)> {
    let Some((low_limit, high_limit)) = integer(ty) else {
        bail!(ty, "a Finite newtype's field must be a primitive integer type");
    };
    let inclusive = || {
        syn::Error::new_spanned(
            attr,
            "a Finite newtype declares an inclusive range, such as #[inputs(0..=3)]",
        )
    };
    let value = attr.parse_args::<Expr>()?;
    let Expr::Range(range) = &value else { return Err(inclusive()) };
    let (Some(low), Some(high), syn::RangeLimits::Closed(_)) =
        (&range.start, &range.end, &range.limits)
    else {
        return Err(inclusive());
    };
    let mut values = [0i128; 2];
    for (bound, value) in [low, high].into_iter().zip(&mut values) {
        *value = literal(bound).ok_or_else(|| {
            syn::Error::new_spanned(bound, "a Finite range bound must be an integer literal")
        })?;
        if *value < low_limit || *value > high_limit {
            bail!(bound, "range bound {value} does not fit the field's integer type");
        }
    }
    if values[0] > values[1] {
        bail!(attr, "a Finite range must not be empty");
    }
    Ok(((**low).clone(), (**high).clone()))
}

pub(crate) fn literal(bound: &Expr) -> Option<i128> {
    match bound {
        Expr::Lit(syn::ExprLit { lit: syn::Lit::Int(value), .. }) => value.base10_parse().ok(),
        Expr::Unary(syn::ExprUnary { op: syn::UnOp::Neg(_), expr, .. }) => {
            literal(expr).and_then(i128::checked_neg)
        }
        Expr::Group(group) => literal(&group.expr),
        Expr::Paren(paren) => literal(&paren.expr),
        _ => None,
    }
}

/// A primitive integer type's inclusive representable bounds; 128-bit and pointer-sized types
/// use the widest supported range, leaving target-specific overflow to the compiler's literal
/// checks.
#[must_use]
pub fn integer(ty: &Type) -> Option<(i128, i128)> {
    let Type::Path(path) = ty else { return None };
    let ident = path.path.get_ident()?.to_string();
    Some(match ident.as_str() {
        "u8" => (0, u8::MAX.into()),
        "u16" => (0, u16::MAX.into()),
        "u32" => (0, u32::MAX.into()),
        "u64" | "usize" => (0, u64::MAX.into()),
        "u128" => (0, i128::MAX),
        "i8" => (i8::MIN.into(), i8::MAX.into()),
        "i16" => (i16::MIN.into(), i16::MAX.into()),
        "i32" => (i32::MIN.into(), i32::MAX.into()),
        "i64" | "isize" => (i64::MIN.into(), i64::MAX.into()),
        "i128" => (i128::MIN, i128::MAX),
        _ => return None,
    })
}
