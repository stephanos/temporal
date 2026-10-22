//! The admitted Model, printed as data the compiled Model embeds for the lifted back ends.
use proc_macro2::{Literal, TokenStream};
use quote::{format_ident, quote};
use std::collections::BTreeMap;
use umpire_front::{
    Embedded, Type,
    expr::ExprKind,
    tree::{Shape, Start},
};

/// `impl Lifts for World` in the expanded module, off the wasm guest.
pub(crate) fn embed(declared: &umpire_front::declare::Machine, input: &syn::ItemMod) -> syn::Item {
    let span = input.ident.span();
    // rustc's path for the file; the runner resolves it, since the macro reads no files.
    let file = span.local_file().map_or_else(|| span.file(), |file| file.display().to_string());
    let embedded = Embedded {
        file,
        model: umpire_front::admit(declared, input),
        source: umpire_front::source(input),
    }
    .print();
    syn::parse_quote! {
        #[cfg(not(target_arch = "wasm32"))]
        #[doc(hidden)]
        impl ::umpire_dsl::__lifted::Lifts for World {
            fn __umpire_lifted() -> ::umpire_dsl::__lifted::Embedded { #embedded }
        }
    }
}

/// Constructor tokens for a tree value.
trait Print {
    fn print(&self) -> TokenStream;
}

/// A struct of the tree, printed field by field at its `umpire_dsl::__lifted` path.
macro_rules! record {
    ($($path:ident)::+ { $($field:ident),* $(,)? }) => {
        impl Print for umpire_front::$($path)::+ {
            fn print(&self) -> TokenStream {
                $(let $field = self.$field.print();)*
                quote!(::umpire_dsl::__lifted::$($path)::+ { $($field: #$field),* })
            }
        }
    };
}
record!(Embedded { file, model, source });
record!(Source { idents, variants, methods, first, control });
record!(Diagnostic { line, column, message });
record!(Lifted {
    name,
    state_name,
    outcome_type,
    world_path,
    declared_world,
    world_fields,
    location,
    fields,
    constants,
    actions,
    invariants,
    postconditions,
    progress,
    assumptions,
    negative,
});
record!(expr::Expr { kind, ty });
record!(tree::FieldDef { name, ty, start, location });
record!(tree::WorldField { name, ty, location });
record!(tree::ActionDef { name, internal, branches, shape, fields, location });
record!(tree::InputField { name, ty, location });
record!(tree::Branch { guard, updates, checks, outcome, preserve_self_loop });
record!(tree::ClaimDef { name, expr });
record!(tree::PostDef { name, expr, after });
record!(tree::ProgressDef { name, trigger, target, assumption, bound });
record!(tree::AssumptionDef { name, only, without, fair });
record!(tree::ActionRef { name, values });

impl Print for String {
    fn print(&self) -> TokenStream {
        quote!(::std::string::String::from(#self))
    }
}
impl Print for bool {
    fn print(&self) -> TokenStream {
        quote!(#self)
    }
}
impl Print for usize {
    fn print(&self) -> TokenStream {
        let value = Literal::usize_suffixed(*self);
        quote!(#value)
    }
}
impl Print for u64 {
    fn print(&self) -> TokenStream {
        let value = Literal::u64_suffixed(*self);
        quote!(#value)
    }
}
impl Print for i128 {
    fn print(&self) -> TokenStream {
        // A negative literal may split into `-` and a positive one, which i128::MIN overflows.
        match self.checked_neg() {
            None => quote!(::core::primitive::i128::MIN),
            Some(positive) if *self < 0 => {
                let value = Literal::i128_suffixed(positive);
                quote!(-#value)
            }
            Some(_) => {
                let value = Literal::i128_suffixed(*self);
                quote!(#value)
            }
        }
    }
}
impl<T: Print> Print for Vec<T> {
    fn print(&self) -> TokenStream {
        let values = self.iter().map(Print::print);
        quote!(::std::vec::Vec::from([#(#values),*]))
    }
}
impl<T: Print> Print for Box<T> {
    fn print(&self) -> TokenStream {
        let value = self.as_ref().print();
        quote!(::std::boxed::Box::new(#value))
    }
}
impl<T: Print> Print for Option<T> {
    fn print(&self) -> TokenStream {
        self.as_ref().map_or_else(
            || quote!(::core::option::Option::None),
            |value| {
                let value = value.print();
                quote!(::core::option::Option::Some(#value))
            },
        )
    }
}
impl<T: Print, E: Print> Print for Result<T, E> {
    fn print(&self) -> TokenStream {
        match self {
            Ok(value) => {
                let value = value.print();
                quote!(::core::result::Result::Ok(#value))
            }
            Err(error) => {
                let error = error.print();
                quote!(::core::result::Result::Err(#error))
            }
        }
    }
}
impl<A: Print, B: Print> Print for (A, B) {
    fn print(&self) -> TokenStream {
        let (first, second) = (self.0.print(), self.1.print());
        quote!((#first, #second))
    }
}
impl<V: Print> Print for BTreeMap<String, V> {
    fn print(&self) -> TokenStream {
        let entries = self.iter().map(|(key, value)| (key.clone(), value).print());
        quote!(::std::collections::BTreeMap::from([#(#entries),*]))
    }
}
impl<T: Print> Print for &T {
    fn print(&self) -> TokenStream {
        (*self).print()
    }
}

/// A fieldless enum value, by its `Debug` name.
fn unit(path: &TokenStream, value: &impl std::fmt::Debug) -> TokenStream {
    let variant = format_ident!("{value:?}");
    quote!(#path::#variant)
}

impl Print for Shape {
    fn print(&self) -> TokenStream {
        unit(&quote!(::umpire_dsl::__lifted::tree::Shape), self)
    }
}
impl Print for Type {
    fn print(&self) -> TokenStream {
        let ty = quote!(::umpire_dsl::__lifted::Type);
        match self {
            Self::Bool => quote!(#ty::Bool),
            Self::Int { min, max } => {
                let (min, max) = (min.print(), max.print());
                quote!(#ty::Int { min: #min, max: #max })
            }
            Self::Enum { name, variants } => {
                let (name, variants) = (name.print(), variants.print());
                quote!(#ty::Enum { name: #name, variants: #variants })
            }
            Self::Text => quote!(#ty::Text),
            Self::Unit => quote!(#ty::Unit),
            Self::Tuple(types) => {
                let types = types.print();
                quote!(#ty::Tuple(#types))
            }
            Self::Seq(inner) => {
                let inner = inner.print();
                quote!(#ty::Seq(#inner))
            }
        }
    }
}
impl Print for Start {
    fn print(&self) -> TokenStream {
        let start = quote!(::umpire_dsl::__lifted::tree::Start);
        match self {
            Self::Computed => quote!(#start::Computed),
            Self::Value(value) => {
                let value = value.print();
                quote!(#start::Value(#value))
            }
            Self::Range(low, high) | Self::ExclusiveRange(low, high) => {
                let variant = format_ident!(
                    "{}",
                    if matches!(self, Self::Range(..)) { "Range" } else { "ExclusiveRange" }
                );
                let (low, high) = (low.print(), high.print());
                quote!(#start::#variant(#low, #high))
            }
        }
    }
}
impl Print for ExprKind {
    fn print(&self) -> TokenStream {
        let kind = quote!(::umpire_dsl::__lifted::expr::ExprKind);
        let (unary, op) =
            (quote!(::umpire_dsl::__lifted::expr::Unary), quote!(::umpire_dsl::__lifted::expr::Op));
        let one = |variant: &str, value: TokenStream| {
            let variant = format_ident!("{variant}");
            quote!(#kind::#variant(#value))
        };
        match self {
            Self::Bool(value) => quote!(#kind::Bool(#value)),
            Self::Int(value) => one("Int", value.print()),
            Self::Text(value) => one("Text", value.print()),
            Self::Unit => quote!(#kind::Unit),
            Self::Var(name, primed) => {
                let name = name.print();
                quote!(#kind::Var(#name, #primed))
            }
            Self::World(name) => one("World", name.print()),
            Self::Input(name) => one("Input", name.print()),
            Self::Outcome => quote!(#kind::Outcome),
            Self::Enum(value) => one("Enum", value.print()),
            Self::Unary(operator, value) => {
                let (operator, value) = (unit(&unary, operator), value.print());
                quote!(#kind::Unary(#operator, #value))
            }
            Self::Binary(operator, left, right) => {
                let (operator, left, right) = (unit(&op, operator), left.print(), right.print());
                quote!(#kind::Binary(#operator, #left, #right))
            }
            Self::If(first, second, third) => {
                let (first, second, third) = (first.print(), second.print(), third.print());
                quote!(#kind::If(#first, #second, #third))
            }
            Self::GetOr(first, second, third) => {
                let (first, second, third) = (first.print(), second.print(), third.print());
                quote!(#kind::GetOr(#first, #second, #third))
            }
            Self::Tuple(values) => one("Tuple", values.print()),
            Self::Sequence(values) => one("Sequence", values.print()),
            Self::Checks(values) => one("Checks", values.print()),
            Self::Length(value) => one("Length", value.print()),
            Self::Append(left, right) => {
                let (left, right) = (left.print(), right.print());
                quote!(#kind::Append(#left, #right))
            }
        }
    }
}
