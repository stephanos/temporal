use proc_macro2::TokenStream;
use quote::quote;
use syn::{Expr, Type};

const SHAPE: &str =
    "Finite derives for a fieldless enum or a struct with one #[inputs(lo..=hi)] integer field";

pub(crate) fn expand(input: &syn::DeriveInput) -> manyhow::Result<TokenStream> {
    let name = &input.ident;
    let (implementation, types, constraints) = input.generics.split_for_impl();
    match &input.data {
        syn::Data::Enum(data) => {
            if let Some(variant) = data.variants.iter().find(|variant| !variant.fields.is_empty()) {
                manyhow::bail!(variant, "{SHAPE}; give a variant with fields to derive(Action)");
            }
            let variants = data.variants.iter().map(|variant| &variant.ident);
            let count = data.variants.len();
            Ok(quote! {
                #[automatically_derived]
                impl #implementation ::umpire_dsl::Finite for #name #types #constraints {
                    fn domain() -> impl ::core::iter::Iterator<Item = Self> {
                        ::core::iter::IntoIterator::into_iter([#(Self::#variants),*] as [Self; #count])
                    }
                }
            })
        }
        syn::Data::Struct(data) => {
            let [field] = data.fields.iter().collect::<Vec<_>>()[..] else {
                manyhow::bail!(input, "{SHAPE}");
            };
            let mut ranges = field.attrs.iter().filter(|attr| attr.path().is_ident("inputs"));
            let Some(attr) = ranges.next() else {
                manyhow::bail!(field, "{SHAPE}");
            };
            if let Some(duplicate) = ranges.next() {
                manyhow::bail!(duplicate, "a Finite newtype declares its range once");
            }
            let (low, high) = bounds(attr, &field.ty)?;
            let ty = &field.ty;
            let construct = field
                .ident
                .as_ref()
                .map_or_else(|| quote!(Self(raw)), |field| quote!(Self { #field: raw }));
            Ok(quote! {
                #[automatically_derived]
                impl #implementation ::umpire_dsl::Finite for #name #types #constraints {
                    fn domain() -> impl ::core::iter::Iterator<Item = Self> {
                        ::core::ops::RangeInclusive::<#ty>::new(#low, #high).map(|raw| #construct)
                    }
                    fn __umpire_input_domain() -> ::umpire_dsl::InputDomain<Self>
                    where
                        Self: 'static,
                    {
                        ::umpire_dsl::InputDomain::range(::core::ops::RangeInclusive::<#ty>::new(#low, #high)).map(|raw| #construct)
                    }
                }
                impl #implementation #name #types #constraints {
                    /// Builds a value only when `raw` lies in the declared range.
                    pub fn try_new(raw: #ty) -> ::core::option::Option<Self> {
                        Self::contains(raw).then_some(#construct)
                    }
                    /// Whether `raw` lies in the declared range.
                    pub fn contains(raw: #ty) -> bool {
                        ::core::ops::RangeInclusive::<#ty>::new(#low, #high).contains(&raw)
                    }
                }
            })
        }
        syn::Data::Union(_) => manyhow::bail!(input, "{SHAPE}"),
    }
}

fn bounds(attr: &syn::Attribute, ty: &Type) -> manyhow::Result<(Expr, Expr)> {
    let Some((low_limit, high_limit)) = width(ty) else {
        manyhow::bail!(ty, "a Finite newtype's field must be a primitive integer type");
    };
    let value = attr.parse_args::<Expr>()?;
    let Expr::Range(range) = &value else {
        manyhow::bail!(
            attr,
            "a Finite newtype declares an inclusive range, such as #[inputs(0..=3)]"
        );
    };
    let (Some(low), Some(high), syn::RangeLimits::Closed(_)) =
        (&range.start, &range.end, &range.limits)
    else {
        manyhow::bail!(
            attr,
            "a Finite newtype declares an inclusive range, such as #[inputs(0..=3)]"
        );
    };
    let mut values = [0i128; 2];
    for (bound, value) in [low, high].into_iter().zip(&mut values) {
        *value = literal(bound).ok_or_else(|| {
            syn::Error::new_spanned(bound, "a Finite range bound must be an integer literal")
        })?;
        if *value < low_limit || *value > high_limit {
            manyhow::bail!(bound, "range bound {value} does not fit the field's integer type");
        }
    }
    if values[0] > values[1] {
        manyhow::bail!(attr, "a Finite range must not be empty");
    }
    Ok(((**low).clone(), (**high).clone()))
}

fn literal(bound: &Expr) -> Option<i128> {
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

/// Inclusive representable bounds; 128-bit and pointer-sized types use the widest supported
/// range, leaving target-specific overflow to the compiler's literal checks.
fn width(ty: &Type) -> Option<(i128, i128)> {
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

pub(crate) fn integer(ty: &Type) -> bool {
    width(ty).is_some()
}
