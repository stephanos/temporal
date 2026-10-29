use super::{Env, Environment, Result, tail};
use crate::{Diagnostic, Locate, Type, expr::Expr};
use quote::ToTokens;
use syn::{Expr as RustExpr, Pat};

impl Environment {
    pub(super) fn bounded_descriptor(
        &self,
        path: &syn::Path,
        expected: Option<&Type>,
    ) -> Result<Option<umpire_ir::Bounded>> {
        if let Some(name) = path.get_ident()
            && let Some(descriptor) = self.bounded.get(&name.to_string())
        {
            return Ok(Some(descriptor.clone()));
        }
        let mut path = path.clone();
        if path
            .segments
            .last()
            .is_some_and(|segment| matches!(segment.arguments, syn::PathArguments::None))
            && let Some(Type::Bounded(descriptor)) = expected
            && let Ok(syn::Type::Path(ty)) = syn::parse_str::<syn::Type>(&descriptor.name)
            && self.dsl_path(&path) == self.dsl_path(&ty.path)
            && let (Some(target), Some(source)) =
                (path.segments.last_mut(), ty.path.segments.last())
        {
            target.arguments = source.arguments.clone();
        }
        self.library_bounded(&path)
    }
    pub(super) fn library_bounded(&self, path: &syn::Path) -> Result<Option<umpire_ir::Bounded>> {
        let Some(canonical) = self.dsl_path(path) else { return Ok(None) };
        let Some((module, name)) = canonical.split_once("::") else { return Ok(None) };
        let Some((structure, range)) = crate::library::bounded(module, name) else {
            return Ok(None);
        };
        let [parameter] = structure.generics.params.iter().collect::<Vec<_>>()[..] else {
            return self.err(path, "library bounded type requires one const parameter");
        };
        let syn::GenericParam::Const(parameter) = parameter else {
            return self.err(path, "library bounded type requires a const parameter");
        };
        let Some(segment) = path.segments.last() else {
            return self.err(path, "empty library type path");
        };
        if path
            .segments
            .iter()
            .take(path.segments.len() - 1)
            .any(|segment| !matches!(segment.arguments, syn::PathArguments::None))
        {
            return self.err(path, "unsupported generic library type path");
        }
        let syn::PathArguments::AngleBracketed(arguments) = &segment.arguments else {
            return self.err(path, "library bounded type requires one literal const argument");
        };
        let [argument] = arguments.args.iter().collect::<Vec<_>>()[..] else {
            return self.err(path, "library bounded type requires one literal const argument");
        };
        let bound = match argument {
            syn::GenericArgument::Const(expr) => self.const_value(expr),
            syn::GenericArgument::Type(syn::Type::Path(path)) => path
                .path
                .get_ident()
                .and_then(|name| self.const_parameters.get(&name.to_string()).copied()),
            _ => None,
        }
        .ok_or_else(|| {
            Diagnostic::at(argument, "library bounded type requires a literal const argument")
        })?;
        if !(1..=255).contains(&bound)
            || !matches!(&parameter.ty, syn::Type::Path(path) if path.path.is_ident("u8"))
        {
            return self.err(argument, "library bounded const argument must be in 1..=255");
        }
        let mut scope = self.clone();
        scope.const_parameters.insert(parameter.ident.to_string(), bound);
        let low = range.start.as_deref().and_then(|expr| scope.const_value(expr));
        let high = range.end.as_deref().and_then(|expr| scope.const_value(expr));
        let (Some(low), Some(high)) = (low, high) else {
            return self.err(path, "unsupported authored library finite range");
        };
        if !matches!(range.limits, syn::RangeLimits::Closed(_)) || structure.fields.len() != 1 {
            return self.err(path, "unsupported authored library finite range");
        }
        let Some(field) = structure.fields.iter().next() else {
            return self.err(path, "library bounded type requires a payload");
        };
        let Type::Int { min, max } = self.ty(&field.ty)? else {
            return self.err(path, "library bounded payload must be an integer");
        };
        let descriptor = umpire_ir::Bounded {
            name: format!("umpire_dsl::{canonical}<{bound}>"),
            field: field.ident.as_ref().map(ToString::to_string),
            backing: field.ty.to_token_stream().to_string(),
            min,
            max,
            low,
            high,
        };
        if !descriptor.valid() {
            return self.err(path, "unsupported library bounded descriptor");
        }
        Ok(Some(descriptor))
    }

    fn const_value(&self, expr: &RustExpr) -> Option<i128> {
        if let RustExpr::Lit(syn::ExprLit { lit: syn::Lit::Int(value), .. }) = expr
            && !matches!(value.suffix(), "" | "u8")
        {
            return None;
        }
        crate::declare::literal(expr).or_else(|| match expr {
            RustExpr::Path(path) => {
                self.const_parameters.get(&path.path.get_ident()?.to_string()).copied()
            }
            _ => None,
        })
    }

    pub(super) fn library_name(&self, call: &syn::ExprCall) -> Option<(String, String)> {
        let RustExpr::Path(path) = call.func.as_ref() else { return None };
        let canonical = self.dsl_path(&path.path)?;
        let (module, name) = canonical.split_once("::")?;
        crate::library::source(module)?;
        Some((module.into(), name.into()))
    }

    #[expect(
        clippy::too_many_arguments,
        reason = "library admission threads the typed expression and claim frame"
    )]
    #[expect(
        clippy::too_many_lines,
        reason = "library argument binding preserves whole-state, outcome and scalar contracts"
    )]
    pub(super) fn library_call(
        &self,
        call: &syn::ExprCall,
        state: &Env,
        locals: &Env,
        expected: Option<&Type>,
        depth: usize,
        claim: bool,
    ) -> Result<Option<Expr>> {
        let Some((module, member)) = self.library_name(call) else { return Ok(None) };
        if depth > 32 {
            return self.err(call, "recursive library expansion is unsupported");
        }
        let function = crate::library::function(&module, &member).ok_or_else(|| {
            let kind = if claim { "law" } else { "function" };
            Diagnostic::at(call, format!("unknown `umpire_dsl::{module}` {kind} `{member}`"))
        })?;
        let name = format!("umpire_dsl::{module}::{member}");
        if claim != matches!(function.sig.output, syn::ReturnType::Default) {
            return self.err(
                call,
                "library laws are claim statements; value functions require expression position",
            );
        }
        if let RustExpr::Path(path) = call.func.as_ref()
            && path.path.leading_colon.is_none()
            && locals.contains_key(&path.path.segments[0].ident.to_string())
        {
            return self.err(call, "library call path is shadowed by a local binding");
        }
        if function.sig.inputs.len() != call.args.len() {
            return self
                .err(call, format!("`{name}` takes {} arguments", function.sig.inputs.len()));
        }
        let generic: Vec<_> =
            function.sig.generics.type_params().map(|param| &param.ident).collect();
        let mut scope = self.clone();
        scope.dsl.clear();
        scope.fact_shadows.clear();
        scope.constants = Env::new();
        scope.const_parameters.clear();
        scope.bounded.clear();
        let source = crate::library::source(&module)
            .ok_or_else(|| Diagnostic::at(call, "unknown library module"))?;
        let source = syn::parse_file(source).map_err(Diagnostic::syn)?;
        for item in source.items {
            let ident = match item {
                syn::Item::Struct(item) => item.ident,
                syn::Item::Fn(item) => item.sig.ident,
                _ => continue,
            };
            scope.dsl.insert(ident.to_string(), format!("{module}::{ident}"));
        }
        let mut env = Env::new();
        let RustExpr::Path(path) = call.func.as_ref() else { unreachable!() };
        let generic_index = path
            .path
            .segments
            .len()
            .checked_sub(if member.contains("::") { 2 } else { 1 })
            .ok_or_else(|| {
                Diagnostic::at(call, "unsupported imported library associated function")
            })?;
        if path.qself.is_some()
            || path.path.segments.iter().enumerate().any(|(index, segment)| {
                index != generic_index && !matches!(segment.arguments, syn::PathArguments::None)
            })
        {
            return self.err(call, "unsupported library call path");
        }
        let explicit: Vec<_> = path
            .path
            .segments
            .iter()
            .filter_map(|segment| match &segment.arguments {
                syn::PathArguments::None => None,
                args => Some(args),
            })
            .collect();
        if explicit.len() > 1 {
            return self.err(call, "library call accepts one const argument list");
        }
        if let Some(arguments) = explicit.first() {
            let syn::PathArguments::AngleBracketed(arguments) = arguments else {
                return self.err(call, "library call requires const arguments");
            };
            let parameters: Vec<_> = function.sig.generics.params.iter().collect();
            if arguments.args.len() != parameters.len() {
                let kind = if parameters
                    .iter()
                    .all(|parameter| matches!(parameter, syn::GenericParam::Const(_)))
                {
                    "const"
                } else {
                    "generic"
                };
                return self
                    .err(arguments, format!("library call has the wrong {kind} argument count"));
            }
            for (parameter, argument) in parameters.iter().zip(&arguments.args) {
                match (parameter, argument) {
                    (syn::GenericParam::Const(parameter), syn::GenericArgument::Const(expr)) => {
                        let bound = self.const_value(expr).ok_or_else(|| {
                            Diagnostic::at(
                                argument,
                                "library call requires literal const arguments",
                            )
                        })?;
                        if !(1..=255).contains(&bound) {
                            return self.err(
                                argument,
                                "library bounded const argument must be in 1..=255",
                            );
                        }
                        scope.bind_library_const(&parameter.ident, bound, argument)?;
                    }
                    (
                        syn::GenericParam::Type(_),
                        syn::GenericArgument::Type(syn::Type::Path(ty)),
                    ) if ty.qself.is_none()
                        && (ty.path.is_ident("Self") || ty.path.is_ident(&self.state)) => {}
                    (syn::GenericParam::Type(_), _) => {
                        return self.err(
                            argument,
                            "library call requires a state type argument naming Self or the State",
                        );
                    }
                    _ => {
                        return self.err(argument, "library call requires literal const arguments");
                    }
                }
            }
        }
        if let Some(expected) = expected {
            scope.bind_library_consts(&function.sig, expected, call)?;
        }
        let mut values = Vec::new();
        for (input, argument) in function.sig.inputs.iter().zip(&call.args) {
            let syn::FnArg::Typed(input) = input else {
                return self.err(call, "library calls require a free function");
            };
            let Pat::Ident(parameter) = input.pat.as_ref() else {
                return self.err(call, "library calls require named parameters");
            };
            let parameter = parameter.ident.to_string();
            let borrowed = match input.ty.as_ref() {
                syn::Type::Reference(reference) if reference.mutability.is_none() => {
                    match reference.elem.as_ref() {
                        syn::Type::Path(path) => path.path.get_ident(),
                        _ => None,
                    }
                }
                _ => None,
            };
            if let Some(ty) = borrowed.filter(|ty| generic.contains(ty)) {
                let fields = self.whole(argument, state, locals).ok_or_else(|| {
                    Diagnostic::at(argument, format!("`{name}` reads a whole state `{ty}` here"))
                })?;
                env.bind_state(&parameter, &self.fields, &fields);
            } else if borrowed.is_some_and(|ty| ty == "Outcome") {
                let outcome = match argument {
                    RustExpr::Path(path) => locals
                        .outcome
                        .as_ref()
                        .filter(|outcome| path.path.is_ident(outcome))
                        .and_then(|outcome| locals.get(outcome)),
                    _ => None,
                }
                .ok_or_else(|| {
                    Diagnostic::at(
                        argument,
                        format!("`{name}` reads the judged edge's outcome here"),
                    )
                })?;
                env.insert(parameter.clone(), outcome.clone());
                env.outcome = Some(parameter);
            } else {
                let value = if function.sig.generics.const_params().next().is_some() {
                    let context = scope.ty(&input.ty).ok();
                    if matches!(argument, RustExpr::Path(path) if path.path.segments.last().is_some_and(|segment| segment.ident == "None"))
                        && context.is_none()
                    {
                        values.push((input, argument, parameter, None));
                        continue;
                    }
                    self.expr(argument, state, locals, context.as_ref(), depth)?
                } else {
                    self.expr(argument, state, locals, Some(&scope.ty(&input.ty)?), depth)?
                };
                scope.bind_library_consts(&function.sig, &value.ty, argument)?;
                values.push((input, argument, parameter, Some(value)));
            }
        }
        for (input, argument, parameter, value) in values {
            let ty = scope.ty(&input.ty).map_err(|refusal| {
                Diagnostic::at(argument, format!("`{name}`: {}", refusal.message))
            })?;
            let value = match value {
                Some(value) => value,
                None => self.expr(argument, state, locals, Some(&ty), depth)?,
            };
            env.insert(parameter, self.require(argument, value, Some(&ty))?);
        }
        let result = scope
            .library_body(&function, &env, depth)
            .map_err(|refusal| Diagnostic::at(call, format!("in `{name}`: {}", refusal.message)))?;
        // Each call's checks are part of the enclosing claim's single token index.
        *self.checks.borrow_mut() = scope.checks.into_inner();
        self.require(call, result, expected).map(Some)
    }
    fn bind_library_consts(
        &mut self,
        signature: &syn::Signature,
        ty: &Type,
        node: &impl ToTokens,
    ) -> Result<()> {
        let (Type::Optional(descriptor) | Type::Bounded(descriptor)) = ty else { return Ok(()) };
        let ty: syn::Type = syn::parse_str(&descriptor.name).map_err(Diagnostic::syn)?;
        let syn::Type::Path(path) = ty else { return Ok(()) };
        let Some(segment) = path.path.segments.last() else { return Ok(()) };
        let syn::PathArguments::AngleBracketed(args) = &segment.arguments else { return Ok(()) };
        for (parameter, argument) in signature.generics.const_params().zip(&args.args) {
            let syn::GenericArgument::Const(bound) = argument else {
                return self.err(node, "library const argument is not a literal");
            };
            let bound = self
                .const_value(bound)
                .ok_or_else(|| Diagnostic::at(node, "library const argument is not a literal"))?;
            self.bind_library_const(&parameter.ident, bound, node)?;
        }
        Ok(())
    }

    fn bind_library_const(
        &mut self,
        name: &syn::Ident,
        bound: i128,
        node: &impl ToTokens,
    ) -> Result<()> {
        let name = name.to_string();
        if self.const_parameters.get(&name).is_some_and(|old| *old != bound) {
            return self.err(node, "library const arguments do not agree");
        }
        self.const_parameters.insert(name.clone(), bound);
        self.constants.insert(name, Expr::int(bound, Type::Int { min: 0, max: 255 }));
        Ok(())
    }

    fn library_body(&self, function: &syn::ItemFn, env: &Env, depth: usize) -> Result<Expr> {
        match &function.sig.output {
            syn::ReturnType::Default => self.claim(&function.block, &Env::new(), env, depth + 1),
            syn::ReturnType::Type(_, ty) => {
                let ty = self.ty(ty)?;
                self.expr(tail(&function.block)?, &Env::new(), env, Some(&ty), depth + 1)
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::expr::ExprKind;

    #[test]
    fn authored_body_changes_change_lowered_values_and_checks() -> Result<()> {
        let ast = syn::parse_file("#[machine] mod clock { use umpire_dsl::{State, Action}; #[derive(State)] struct Model { value: bool } #[derive(Action)] enum User { Tick } impl Model { #[on(User)] fn user(&mut self, action: User) { match action { User::Tick => self.value = true } } } } ").map_err(Diagnostic::syn)?;
        let (model, _, _) = crate::read(&ast, &[]).unwrap_or_else(|error| panic!("{error:?}"));
        let environment = Environment::of(&model);
        let descriptor = environment
            .library_bounded(
                &syn::parse_str("umpire_dsl::countdown::Remaining::<3>")
                    .map_err(Diagnostic::syn)?,
            )?
            .unwrap_or_else(|| panic!("missing library type"));
        let env = Env::from_iter([("remaining".into(), Expr::int(1, Type::Optional(descriptor)))]);
        let mut function = crate::library::function("countdown", "armed")
            .unwrap_or_else(|| panic!("missing armed"));
        let original = environment.library_body(&function, &env, 0)?;
        function.block = syn::parse_quote!({ false });
        let changed = environment.library_body(&function, &env, 0)?;
        assert_ne!(original, changed);
        assert_eq!(changed, Expr::bool(false));
        let mut law = crate::library::function("countdown", "within")
            .unwrap_or_else(|| panic!("missing law"));
        law.block = syn::parse_quote!({
            check!(false);
        });
        let changed = environment.library_body(&law, &Env::new(), 0)?;
        assert!(
            matches!(changed.kind, ExprKind::Checks(ref checks) if matches!(&checks[0].kind, ExprKind::Check(_, value) if **value == Expr::bool(false)))
        );
        Ok(())
    }
}
