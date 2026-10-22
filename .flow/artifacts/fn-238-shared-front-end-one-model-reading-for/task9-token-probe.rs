use proc_macro2::{Literal, TokenStream};
use quote::{format_ident, quote};
use std::collections::BTreeMap;
use umpire_front::{
    Embedded, Type,
    expr::ExprKind,
    tree::{Shape, Start},
};

/// Constructor tokens for a tree value.
trait OldPrint {
    fn print(&self) -> TokenStream;
}

/// A struct of the tree, printed field by field at its `umpire_dsl::__lifted` path.
macro_rules! record {
    ($($path:ident)::+ { $($field:ident),* $(,)? }) => {
        impl OldPrint for umpire_front::$($path)::+ {
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

impl OldPrint for String {
    fn print(&self) -> TokenStream {
        quote!(::std::string::String::from(#self))
    }
}
impl OldPrint for bool {
    fn print(&self) -> TokenStream {
        quote!(#self)
    }
}
impl OldPrint for usize {
    fn print(&self) -> TokenStream {
        let value = Literal::usize_suffixed(*self);
        quote!(#value)
    }
}
impl OldPrint for u64 {
    fn print(&self) -> TokenStream {
        let value = Literal::u64_suffixed(*self);
        quote!(#value)
    }
}
impl OldPrint for i128 {
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
impl<T: OldPrint> OldPrint for Vec<T> {
    fn print(&self) -> TokenStream {
        let values = self.iter().map(OldPrint::print);
        quote!(::std::vec::Vec::from([#(#values),*]))
    }
}
impl<T: OldPrint> OldPrint for Box<T> {
    fn print(&self) -> TokenStream {
        let value = self.as_ref().print();
        quote!(::std::boxed::Box::new(#value))
    }
}
impl<T: OldPrint> OldPrint for Option<T> {
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
impl<T: OldPrint, E: OldPrint> OldPrint for Result<T, E> {
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
impl<A: OldPrint, B: OldPrint> OldPrint for (A, B) {
    fn print(&self) -> TokenStream {
        let (first, second) = (self.0.print(), self.1.print());
        quote!((#first, #second))
    }
}
impl<V: OldPrint> OldPrint for BTreeMap<String, V> {
    fn print(&self) -> TokenStream {
        let entries = self.iter().map(|(key, value)| (key.clone(), value).print());
        quote!(::std::collections::BTreeMap::from([#(#entries),*]))
    }
}
impl<T: OldPrint> OldPrint for &T {
    fn print(&self) -> TokenStream {
        (*self).print()
    }
}

/// A fieldless enum value, by its `Debug` name.
fn unit(path: &TokenStream, value: &impl std::fmt::Debug) -> TokenStream {
    let variant = format_ident!("{value:?}");
    quote!(#path::#variant)
}

impl OldPrint for Shape {
    fn print(&self) -> TokenStream {
        unit(&quote!(::umpire_dsl::__lifted::tree::Shape), self)
    }
}
impl OldPrint for Type {
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
impl OldPrint for Start {
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
impl OldPrint for ExprKind {
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

fn main() {
    let root = std::path::Path::new("/Users/stephan/Workspace/skunkworks/umpire/temporal");
    let list = std::process::Command::new("git").arg("ls-files").current_dir(root).output().unwrap();
    let mut count = 0;
    let mut dump = String::new();
    for file in String::from_utf8(list.stdout).unwrap().lines().filter(|file|
        file.starts_with("model/") && !file.contains("/archived/") && file.ends_with("/model.rs")) {
        let text = std::fs::read_to_string(root.join(file)).unwrap();
        let Ok(ast) = syn::parse_file(&text) else { continue };
        let mut found = false;
        for item in &ast.items {
            if let syn::Item::Mod(module) = item {
                if !module.attrs.iter().any(|a| a.path().segments.last().is_some_and(|s| s.ident == "machine")) {continue;}
                let Ok(declared) = umpire_front::declare::machine(&TokenStream::new(), module) else {continue};
                let value = Embedded { file: file.into(), model: umpire_front::admit(&declared, module), source: umpire_front::source(module) };
                assert_eq!(OldPrint::print(&value).to_string(), umpire_ir::print::Print::print(&value).to_string(), "{file}");
                dump.push_str(&format!("== {file}\n{value:#?}\n"));
                count += 1;
                found = true;
            }
        }
        let _ = found;
    }
    assert!(count > 20, "empty/incomplete fixture selection: {count}");
    std::fs::write(root.join(".flow/artifacts/fn-238-shared-front-end-one-model-reading-for/task9-ir-dump.txt"), dump).unwrap();
    println!("{count} admitted/refused Models: old and derived constructor tokens byte-identical");
}
