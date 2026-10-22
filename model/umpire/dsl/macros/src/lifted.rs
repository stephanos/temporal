//! The admitted Model, printed as data the compiled Model embeds for the lifted back ends.
use proc_macro2::{Literal, TokenStream};
use quote::{format_ident, quote};
use std::collections::BTreeMap;
use umpire_front::{
    Diagnostic, Embedded, Lifted, Source, Type,
    expr::{Expr, ExprKind},
    tree::{
        ActionDef, ActionRef, AssumptionDef, Branch, ClaimDef, FieldDef, InputDomain, InputField,
        PostDef, ProgressDef, Start, WorldField,
    },
};

/// `impl Lifts for World` in the expanded module, off the wasm guest.
pub(crate) fn embed(args: &TokenStream, input: &syn::ItemMod) -> syn::Item {
    let span = input.ident.span();
    // rustc's path for the file; the runner resolves it, since the macro reads no files.
    let file = span.local_file().map_or_else(|| span.file(), |file| file.display().to_string());
    let embedded = Embedded {
        file,
        model: umpire_front::admit(args, input),
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

fn root() -> TokenStream {
    quote!(::umpire_dsl::__lifted)
}

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

impl Print for Embedded {
    fn print(&self) -> TokenStream {
        let root = root();
        let file = self.file.print();
        let model = match &self.model {
            Ok(model) => {
                let model = model.print();
                quote!(::core::result::Result::Ok(#model))
            }
            Err(refusal) => {
                let refusal = refusal.print();
                quote!(::core::result::Result::Err(#refusal))
            }
        };
        let source = self.source.print();
        quote!(#root::Embedded { file: #file, model: #model, source: #source })
    }
}
impl Print for Source {
    fn print(&self) -> TokenStream {
        let root = root();
        let (idents, variants, methods) =
            (self.idents.print(), self.variants.print(), self.methods.print());
        let first = self.first.print();
        let control = self.control.print();
        quote!(#root::Source { idents: #idents, variants: #variants, methods: #methods, first: #first, control: #control })
    }
}
impl Print for Diagnostic {
    fn print(&self) -> TokenStream {
        let root = root();
        let (line, column, message) =
            (self.line.print(), self.column.print(), self.message.print());
        quote!(#root::Diagnostic { line: #line, column: #column, message: #message })
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
impl Print for Expr {
    fn print(&self) -> TokenStream {
        let root = root();
        let (kind, ty) = (self.kind.print(), self.ty.print());
        quote!(#root::expr::Expr { kind: #kind, ty: #ty })
    }
}
impl Print for ExprKind {
    fn print(&self) -> TokenStream {
        let kind = quote!(::umpire_dsl::__lifted::expr::ExprKind);
        let (unary, op) =
            (quote!(::umpire_dsl::__lifted::expr::Unary), quote!(::umpire_dsl::__lifted::expr::Op));
        match self {
            Self::Bool(value) => quote!(#kind::Bool(#value)),
            Self::Int(value) => {
                let value = value.print();
                quote!(#kind::Int(#value))
            }
            Self::Unit => quote!(#kind::Unit),
            Self::Outcome => quote!(#kind::Outcome),
            Self::Var(name, primed) => {
                let name = name.print();
                quote!(#kind::Var(#name, #primed))
            }
            Self::Text(value) | Self::World(value) | Self::Input(value) | Self::Enum(value) => {
                let variant = format_ident!("{}", variant(self));
                let value = value.print();
                quote!(#kind::#variant(#value))
            }
            Self::Unary(operator, value) => {
                let (operator, value) = (unit(&unary, operator), value.print());
                quote!(#kind::Unary(#operator, #value))
            }
            Self::Binary(operator, left, right) => {
                let (operator, left, right) = (unit(&op, operator), left.print(), right.print());
                quote!(#kind::Binary(#operator, #left, #right))
            }
            Self::Length(value) => {
                let value = value.print();
                quote!(#kind::Length(#value))
            }
            Self::Append(left, right) => {
                let (left, right) = (left.print(), right.print());
                quote!(#kind::Append(#left, #right))
            }
            Self::If(first, second, third) | Self::GetOr(first, second, third) => {
                let variant = format_ident!("{}", variant(self));
                let (first, second, third) = (first.print(), second.print(), third.print());
                quote!(#kind::#variant(#first, #second, #third))
            }
            Self::Tuple(values) | Self::Sequence(values) | Self::Checks(values) => {
                let variant = format_ident!("{}", variant(self));
                let values = values.print();
                quote!(#kind::#variant(#values))
            }
        }
    }
}
/// The variant name of an expression kind.
fn variant(kind: &ExprKind) -> String {
    let debug = format!("{kind:?}");
    debug.split(['(', ' ']).next().unwrap_or_default().to_owned()
}
impl Print for Lifted {
    fn print(&self) -> TokenStream {
        let root = root();
        let name = self.name.print();
        let state_name = self.state_name.print();
        let outcome_type = self.outcome_type.print();
        let world_path = self.world_path.print();
        let declared_world = self.declared_world;
        let world_fields = self.world_fields.print();
        let location = self.location.print();
        let fields = self.fields.print();
        let constants = self.constants.print();
        let actions = self.actions.print();
        let invariants = self.invariants.print();
        let postconditions = self.postconditions.print();
        let progress = self.progress.print();
        let assumptions = self.assumptions.print();
        let negative = self.negative.print();
        // World values, complete starts and the catalog are bound later, never admitted.
        quote!(#root::Lifted {
            name: #name,
            state_name: #state_name,
            outcome_type: #outcome_type,
            world_path: #world_path,
            declared_world: #declared_world,
            world_fields: #world_fields,
            world_values: ::core::option::Option::None,
            complete_starts: ::core::option::Option::None,
            location: #location,
            fields: #fields,
            constants: #constants,
            actions: #actions,
            invariants: #invariants,
            postconditions: #postconditions,
            progress: #progress,
            assumptions: #assumptions,
            negative: #negative,
            catalog: ::core::option::Option::None,
        })
    }
}
impl Print for FieldDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, ty, start, location) =
            (self.name.print(), self.ty.print(), self.start.print(), self.location.print());
        quote!(#root::tree::FieldDef { name: #name, ty: #ty, start: #start, location: #location })
    }
}
impl Print for WorldField {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, ty, location) = (self.name.print(), self.ty.print(), self.location.print());
        quote!(#root::tree::WorldField { name: #name, ty: #ty, location: #location })
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
            Self::Range(low, high) => {
                let (low, high) = (low.print(), high.print());
                quote!(#start::Range(#low, #high))
            }
            Self::ExclusiveRange(low, high) => {
                let (low, high) = (low.print(), high.print());
                quote!(#start::ExclusiveRange(#low, #high))
            }
        }
    }
}
impl Print for ActionDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, internal, branches) = (self.name.print(), self.internal, self.branches.print());
        let shape = unit(&quote!(#root::tree::Shape), &self.shape);
        let (fields, location) = (self.fields.print(), self.location.print());
        quote!(#root::tree::ActionDef { name: #name, internal: #internal, branches: #branches, shape: #shape, fields: #fields, location: #location })
    }
}
impl Print for InputField {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, ty, location) = (self.name.print(), self.ty.print(), self.location.print());
        let domain = match &self.domain {
            InputDomain::Values(values) => {
                let values = values.print();
                quote!(#root::tree::InputDomain::Values(#values))
            }
            InputDomain::World => quote!(#root::tree::InputDomain::World),
        };
        quote!(#root::tree::InputField { name: #name, ty: #ty, domain: #domain, location: #location })
    }
}
impl Print for Branch {
    fn print(&self) -> TokenStream {
        let root = root();
        let (guard, updates, checks) =
            (self.guard.print(), self.updates.print(), self.checks.print());
        let (outcome, preserve) = (self.outcome.print(), self.preserve_self_loop);
        quote!(#root::tree::Branch { guard: #guard, updates: #updates, checks: #checks, outcome: #outcome, preserve_self_loop: #preserve })
    }
}
impl Print for ClaimDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, expr) = (self.name.print(), self.expr.print());
        quote!(#root::tree::ClaimDef { name: #name, expr: #expr })
    }
}
impl Print for PostDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, expr, after) = (self.name.print(), self.expr.print(), self.after.print());
        quote!(#root::tree::PostDef { name: #name, expr: #expr, after: #after })
    }
}
impl Print for ProgressDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, trigger, target) =
            (self.name.print(), self.trigger.print(), self.target.print());
        let (assumption, bound) = (self.assumption.print(), self.bound.print());
        quote!(#root::tree::ProgressDef { name: #name, trigger: #trigger, target: #target, assumption: #assumption, bound: #bound })
    }
}
impl Print for AssumptionDef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, only, without, fair) =
            (self.name.print(), self.only.print(), self.without.print(), self.fair.print());
        quote!(#root::tree::AssumptionDef { name: #name, only: #only, without: #without, fair: #fair })
    }
}
impl Print for ActionRef {
    fn print(&self) -> TokenStream {
        let root = root();
        let (name, values) = (self.name.print(), self.values.print());
        quote!(#root::tree::ActionRef { name: #name, values: #values })
    }
}
