use super::{Env, Environment, Result};
use crate::{
    Diagnostic, Locate, Type,
    expr::{Expr, ExprKind, Op},
};
use quote::ToTokens;
use syn::Expr as RustExpr;

impl Environment {
    pub(crate) fn admit_bounded(&mut self, value: &syn::ItemStruct) -> Result<()> {
        if !value.generics.params.is_empty() || value.generics.where_clause.is_some() {
            return self.err(value, "bounded newtypes must be nongeneric");
        }
        let mut canonical = false;
        for attribute in value.attrs.iter().filter(|a| a.path().is_ident("derive")) {
            let paths = attribute
                .parse_args_with(
                    syn::punctuated::Punctuated::<syn::Path, syn::Token![,]>::parse_terminated,
                )
                .map_err(Diagnostic::syn)?;
            for path in paths {
                if path.segments.last().is_some_and(|s| s.ident == "Finite") {
                    canonical |= self.dsl_path(&path).as_deref() == Some("Finite")
                        && path
                            .segments
                            .iter()
                            .all(|s| matches!(s.arguments, syn::PathArguments::None));
                }
            }
        }
        if !canonical {
            return self.err(value, "bounded newtype requires canonical derive(Finite)");
        }
        let input = syn::DeriveInput::from(value.clone());
        let crate::declare::Finite::Newtype { field, ty, low, high } =
            crate::declare::finite(&input).map_err(Diagnostic::syn)?
        else {
            unreachable!()
        };
        let Type::Int { min, max } = self.ty(&ty)? else { unreachable!() };
        let descriptor = umpire_ir::Bounded {
            name: value.ident.to_string(),
            field: field.map(|f| f.to_string()),
            backing: ty.to_token_stream().to_string(),
            min,
            max,
            low: crate::declare::literal(&low)
                .ok_or_else(|| Diagnostic::at(&low, "expected finite lower bound"))?,
            high: crate::declare::literal(&high)
                .ok_or_else(|| Diagnostic::at(&high, "expected finite upper bound"))?,
        };
        if !descriptor.valid() {
            return self.err(value, "unsupported bounded newtype descriptor");
        }
        if crate::declare::derives(&value.attrs, "Default").map_err(Diagnostic::syn)? {
            self.bounded_defaults.push(descriptor.name.clone());
        }
        self.bounded.insert(descriptor.name.clone(), descriptor);
        Ok(())
    }

    pub(super) fn optional_type(&self, ty: &syn::Type) -> Result<Option<Type>> {
        if self.bounded.is_empty() {
            return Ok(None);
        }
        let syn::Type::Path(path) = ty else { return Ok(None) };
        let names = path.path.segments.iter().map(|s| s.ident.to_string()).collect::<Vec<_>>();
        if !matches!(
            names.join("::").as_str(),
            "Option" | "core::option::Option" | "std::option::Option"
        ) {
            return Ok(None);
        }
        if path.qself.is_some() || !self.canonical_optional(&path.path, &Env::new()) {
            return self.err(ty, "optional value requires an unshadowed canonical Option path");
        }
        let syn::PathArguments::AngleBracketed(args) =
            &path.path.segments[path.path.segments.len() - 1].arguments
        else {
            return self.err(ty, "Option requires exactly one bounded newtype argument");
        };
        let [syn::GenericArgument::Type(inner)] = args.args.iter().collect::<Vec<_>>()[..] else {
            return self.err(ty, "Option requires exactly one bounded newtype argument");
        };
        let Type::Bounded(descriptor) = self.ty(inner)? else {
            return self.err(ty, "Option requires a bounded newtype payload");
        };
        if !descriptor.optional_valid() {
            return self
                .err(ty, "bounded optional range cannot be encoded in signed 32-bit integers");
        }
        Ok(Some(Type::Optional(descriptor)))
    }

    fn canonical_optional(&self, path: &syn::Path, locals: &Env) -> bool {
        let root = path.segments[0].ident.to_string();
        let bare = path.segments.len() <= 2;
        (!bare || path.leading_colon.is_none())
            && (!bare && path.leading_colon.is_some()
                || (!self.fact_shadows.contains(&root) && !locals.contains_key(&root)))
            && path
                .segments
                .iter()
                .take(path.segments.len() - 1)
                .all(|s| matches!(s.arguments, syn::PathArguments::None))
    }

    #[expect(clippy::too_many_arguments, reason = "expression admission threads its typed frame")]
    pub(super) fn optional_constructor(
        &self,
        path: &syn::ExprPath,
        argument: Option<&RustExpr>,
        state: &Env,
        locals: &Env,
        expected: Option<&Type>,
        depth: usize,
    ) -> Result<Option<Expr>> {
        if self.bounded.is_empty() {
            return Ok(None);
        }
        let names = path.path.segments.iter().map(|s| s.ident.to_string()).collect::<Vec<_>>();
        let name = names.join("::");
        if !matches!(
            name.as_str(),
            "Some"
                | "None"
                | "core::option::Some"
                | "core::option::None"
                | "std::option::Some"
                | "std::option::None"
                | "Option::Some"
                | "Option::None"
                | "core::option::Option::Some"
                | "core::option::Option::None"
                | "std::option::Option::Some"
                | "std::option::Option::None"
        ) {
            return Ok(None);
        }
        let mut canonical = path.path.clone();
        for segment in &mut canonical.segments {
            segment.arguments = syn::PathArguments::None;
        }
        if path.qself.is_some() || !self.canonical_optional(&canonical, locals) {
            return self.err(path, "optional value requires an unshadowed canonical constructor");
        }
        let explicit = self.optional_constructor_type(&path.path)?;
        let context_type = explicit.as_ref().or(expected);
        if names.last().is_some_and(|s| s == "None") {
            let Some(ty @ Type::Optional(_)) = context_type else {
                return self.err(path, "None requires a bounded optional type context");
            };
            if argument.is_some() {
                return self.err(path, "None takes no arguments");
            }
            return Ok(Some(Expr::int(0, ty.clone())));
        }
        let Some(argument) = argument else {
            return self.err(path, "Some requires one bounded payload");
        };
        let context = match context_type {
            Some(Type::Optional(d)) => Some(Type::Bounded(d.clone())),
            _ => None,
        };
        let payload = self.expr(argument, state, locals, context.as_ref(), depth)?;
        let Type::Bounded(descriptor) = &payload.ty else {
            return self.err(argument, "Some requires a bounded newtype payload");
        };
        if !descriptor.optional_valid() {
            return self
                .err(path, "bounded optional range cannot be encoded in signed 32-bit integers");
        }
        Ok(Some(Expr {
            ty: Type::Optional(descriptor.clone()),
            kind: ExprKind::Some(Box::new(payload)),
        }))
    }

    fn optional_constructor_type(&self, path: &syn::Path) -> Result<Option<Type>> {
        let mut explicit = None;
        for (index, segment) in path.segments.iter().enumerate() {
            if matches!(segment.arguments, syn::PathArguments::None) {
                continue;
            }
            let allowed = index + 1 == path.segments.len()
                || (index + 2 == path.segments.len() && segment.ident == "Option");
            if !allowed || explicit.is_some() {
                return self.err(path, "optional constructor has unsupported generic arguments");
            }
            let syn::PathArguments::AngleBracketed(arguments) = &segment.arguments else {
                return self
                    .err(path, "optional constructor requires exactly one bounded type argument");
            };
            let [syn::GenericArgument::Type(inner)] = arguments.args.iter().collect::<Vec<_>>()[..]
            else {
                return self
                    .err(path, "optional constructor requires exactly one bounded type argument");
            };
            let Type::Bounded(descriptor) = self.ty(inner)? else {
                return self.err(path, "optional constructor requires a bounded newtype argument");
            };
            if !descriptor.optional_valid() {
                return self.err(
                    path,
                    "bounded optional range cannot be encoded in signed 32-bit integers",
                );
            }
            explicit = Some(Type::Optional(descriptor));
        }
        Ok(explicit)
    }

    pub(super) fn bounded_call(
        &self,
        call: &syn::ExprCall,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Option<Expr>> {
        let RustExpr::Path(path) = call.func.as_ref() else { return Ok(None) };
        let names = path.path.segments.iter().map(|s| s.ident.to_string()).collect::<Vec<_>>();
        let Some(descriptor) = names.first().and_then(|name| self.bounded.get(name)) else {
            return Ok(None);
        };
        if path.qself.is_some()
            || path.path.leading_colon.is_some()
            || path.path.segments.iter().any(|s| !matches!(s.arguments, syn::PathArguments::None))
            || locals.contains_key(&names[0])
        {
            return self.err(call, "bounded constructor requires the local nominal type");
        }
        let [argument] = call.args.iter().collect::<Vec<_>>()[..] else {
            return self.err(call, "bounded construction and contains require one backing value");
        };
        if names.len() == 1 && descriptor.field.is_none() {
            let value =
                self.expr(argument, state, locals, Some(&descriptor.backing_type()), depth)?;
            return Ok(Some(Expr {
                ty: Type::Bounded(descriptor.clone()),
                kind: ExprKind::Bounded(Box::new(value)),
            }));
        }
        if names.len() == 2 && names[1] == "contains" {
            if self.helpers.contains_key(&format!("{}::contains", descriptor.name)) {
                return self.err(call, "contains must be the generated bounded range predicate");
            }
            let value =
                self.expr(argument, state, locals, Some(&descriptor.backing_type()), depth)?;
            return Ok(Some(
                Expr::binary(Op::Le, Expr::int(descriptor.low, value.ty.clone()), value.clone())
                    .and(Expr::binary(
                        Op::Le,
                        value.clone(),
                        Expr::int(descriptor.high, value.ty.clone()),
                    )),
            ));
        }
        self.err(call, "unsupported bounded associated call")
    }

    pub(super) fn optional_method(
        &self,
        call: &syn::ExprMethodCall,
        receiver: Expr,
        state: &Env,
        locals: &Env,
        depth: usize,
    ) -> Result<Expr> {
        let Type::Optional(descriptor) = &receiver.ty else { unreachable!() };
        if call.turbofish.is_some() {
            return self.err(call, "generic optional methods are unsupported");
        }
        match call.method.to_string().as_str() {
            "is_some" | "is_none" if call.args.is_empty() => {
                let absent =
                    Expr::binary(Op::Eq, receiver.clone(), Expr::int(0, receiver.ty.clone()));
                Ok(if call.method == "is_none" { absent } else { absent.not() })
            }
            "unwrap_or" if call.args.len() == 1 => {
                let fallback = self.expr(
                    &call.args[0],
                    state,
                    locals,
                    Some(&Type::Bounded(descriptor.clone())),
                    depth,
                )?;
                Ok(Expr {
                    ty: fallback.ty.clone(),
                    kind: ExprKind::OptionalOr(Box::new(receiver), Box::new(fallback)),
                })
            }
            _ => self
                .err(call, "unsupported optional method; use is_some, is_none or eager unwrap_or"),
        }
    }
}

#[cfg(test)]
mod tests {
    use crate::{
        Type,
        expr::{Expr, ExprKind},
    };

    fn source(body: &str) -> String {
        format!(
            r"
#[machine] mod options {{
    use umpire_dsl::{{Finite, State, Action, check}};
    #[derive(Finite, Default)] struct Ticks(#[inputs(1..=3)] u8);
    #[derive(State)] struct Model {{ value: Option<Ticks> }}
    #[derive(Action)] enum User {{ Set }}
    impl Model {{
        #[on(User)] fn user(&mut self, action: User) {{
            match action {{ User::Set => {{ {body} }} }}
        }}
        #[invariant] fn bounded(&self) {{ check!(Ticks::contains(self.value.unwrap_or(Ticks(1)).0)); }}
    }}
}}"
        )
    }

    fn model(body: &str) -> crate::Lifted {
        let ast = syn::parse_file(&source(body)).unwrap_or_else(|e| panic!("{e}"));
        crate::read(&ast, &[]).unwrap_or_else(|e| panic!("{e:?}")).0
    }

    #[test]
    fn defaults_and_eager_ssa_preserve_nominal_types() {
        let model = model(
            "let value: Option<Ticks> = Some(Ticks(2)); self.value = Some(Ticks(value.unwrap_or(Ticks(1 / 0)).0));",
        );
        let branch = &model.actions[0].branches[0];
        let printed = format!("{branch:?}");
        assert!(printed.contains("OptionalOr"), "{printed}");
        assert!(printed.contains("Div"), "{printed}");
        let environment = super::Environment::of(&model);
        let Type::Optional(descriptor) = &model.fields[0].ty else { panic!("optional type lost") };
        assert_eq!(environment.bounded.get("Ticks"), Some(descriptor));
        assert!(environment.default_value(&Type::Bounded(descriptor.clone())).is_err());
        let mut environment = environment;
        environment.bounded_defaults.push("Ticks".into());
        let default = environment
            .default_value(&Type::Bounded(descriptor.clone()))
            .unwrap_or_else(|e| panic!("{e:?}"));
        assert!(matches!(default.kind, ExprKind::Bounded(value) if value.kind == ExprKind::Int(0)));
        let filler = super::super::effects::default_expr(&Type::Bounded(descriptor.clone()));
        assert!(matches!(filler.kind, ExprKind::Bounded(value) if value.kind == ExprKind::Int(1)));
        assert_eq!(
            super::super::effects::default_expr(&model.fields[0].ty),
            Expr::int(0, model.fields[0].ty.clone())
        );
    }
    #[test]
    fn canonical_constructor_types_and_located_refusals() {
        for constructor in [
            "Some::<Ticks>(Ticks(2))",
            "None::<Ticks>",
            "Option::<Ticks>::Some(Ticks(2))",
            "Option::<Ticks>::None",
            "core::option::Some::<Ticks>(Ticks(2))",
            "::std::option::Option::<Ticks>::None",
        ] {
            model(&format!("self.value = {constructor};"));
        }
        for (body, expected) in [
            ("self.value = Some::<u8>(2);", "bounded newtype"),
            ("self.value = Some::<Ticks, Ticks>(Ticks(2));", "exactly one"),
            ("let Some = true; self.value = Some(Ticks(2));", "canonical constructor"),
            ("let None = true; self.value = None;", "canonical constructor"),
            ("let Option = true; self.value = Option::None;", "canonical constructor"),
            ("let core = true; self.value = core::option::Option::None;", "canonical constructor"),
            ("let std = true; self.value = std::option::Option::None;", "canonical constructor"),
            ("let value = None;", "type context"),
            ("self.value = Some(Ticks(self.value.expect(\"present\").0));", "optional method"),
        ] {
            let ast = syn::parse_file(&source(body)).unwrap_or_else(|e| panic!("{e}"));
            let error = match crate::read(&ast, &[]) {
                Ok(_) => panic!("admitted {body}"),
                Err(crate::Refusal::Model(error)) => error,
                Err(error) => panic!("{error:?}"),
            };
            assert!(error.message.contains(expected), "{body}: {error:?}");
            assert!(error.line > 0 && error.column > 0, "{error:?}");
        }
    }
}
