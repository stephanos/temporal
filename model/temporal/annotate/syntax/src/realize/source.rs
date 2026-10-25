//! Bounded source-name resolution for generated-enum admission.

use std::collections::{BTreeMap, BTreeSet};

use proc_macro2::LineColumn;
use syn::{Item, Path, Type, UseTree, ext::IdentExt};

use crate::realization::Schema;

#[derive(Clone, Copy, PartialEq, Eq)]
pub(super) enum EnumKind {
    Generated,
    Other,
    Unknown,
}

#[derive(Clone, Default)]
struct Names {
    bindings: BTreeMap<String, Option<Vec<String>>>,
    globs: Vec<Vec<String>>,
    external: BTreeSet<String>,
    uncertain: BTreeSet<String>,
}

#[derive(Clone, Default)]
pub(super) struct Scope {
    modules: BTreeMap<Vec<String>, Names>,
    module: Vec<String>,
    frames: Vec<Names>,
    model: Vec<String>,
}

enum Resolved {
    Path(Vec<String>),
    Local,
    Unknown,
}

fn path(path: &Path) -> Vec<String> {
    let mut result: Vec<_> =
        path.segments.iter().map(|part| part.ident.unraw().to_string()).collect();
    if path.leading_colon.is_some() {
        result.insert(0, "::".into());
    }
    result
}

fn imports(tree: &UseTree, prefix: &[String], names: &mut Names) {
    let mut target = prefix.to_vec();
    match tree {
        UseTree::Path(part) => {
            target.push(part.ident.unraw().to_string());
            imports(&part.tree, &target, names);
        }
        UseTree::Name(part) => {
            if part.ident != "self" {
                target.push(part.ident.unraw().to_string());
            }
            if target.len() > 1
                && let Some(name) = target.last()
            {
                names.bindings.insert(name.clone(), Some(target.clone()));
            }
        }
        UseTree::Rename(part) => {
            if part.ident != "self" {
                target.push(part.ident.unraw().to_string());
            }
            names.bindings.insert(part.rename.unraw().to_string(), Some(target));
        }
        UseTree::Group(group) => {
            for item in &group.items {
                imports(item, prefix, names);
            }
        }
        UseTree::Glob(_) => names.globs.push(target),
    }
}

fn names<'a>(items: impl IntoIterator<Item = &'a Item>) -> Names {
    let mut result = Names::default();
    for item in items {
        let mut result_item = Names::default();
        let target = &mut result_item;
        match item {
            Item::Use(item) => {
                let prefix = if item.leading_colon.is_some() { vec!["::".into()] } else { vec![] };
                imports(&item.tree, &prefix, target);
            }
            Item::Type(item) => {
                let target = match item.ty.as_ref() {
                    Type::Path(ty)
                        if ty.qself.is_none()
                            && ty.path.segments.iter().all(|segment| {
                                matches!(segment.arguments, syn::PathArguments::None)
                            }) =>
                    {
                        Some(path(&ty.path))
                    }
                    _ => None,
                };
                if target.is_none() {
                    result_item.uncertain.insert(item.ident.unraw().to_string());
                }
                result_item.bindings.insert(item.ident.unraw().to_string(), target);
            }
            Item::Enum(item) => {
                target.bindings.insert(item.ident.unraw().to_string(), None);
            }
            Item::Struct(item) => {
                target.bindings.insert(item.ident.unraw().to_string(), None);
            }
            Item::Union(item) => {
                target.bindings.insert(item.ident.unraw().to_string(), None);
            }
            Item::Trait(item) => {
                target.bindings.insert(item.ident.unraw().to_string(), None);
            }
            Item::Mod(item) => {
                let name = item.ident.unraw().to_string();
                target.bindings.insert(name.clone(), None);
                if item.content.is_none() {
                    target.external.insert(name);
                }
            }
            _ => {}
        }
        let attrs = match item {
            Item::Use(item) => &item.attrs,
            Item::Type(item) => &item.attrs,
            Item::Enum(item) => &item.attrs,
            Item::Struct(item) => &item.attrs,
            Item::Union(item) => &item.attrs,
            Item::Trait(item) => &item.attrs,
            Item::Mod(item) => &item.attrs,
            _ => &[][..],
        };
        if attrs.iter().any(|attr| attr.path().is_ident("cfg") || attr.path().is_ident("cfg_attr"))
        {
            result_item.uncertain.extend(result_item.bindings.keys().cloned());
        }
        result.bindings.extend(result_item.bindings);
        result.globs.extend(result_item.globs);
        result.external.extend(result_item.external);
        result.uncertain.extend(result_item.uncertain);
    }
    result
}

fn modules(items: &[Item], parent: &[String], result: &mut BTreeMap<Vec<String>, Names>) {
    result.insert(parent.to_vec(), names(items));
    for item in items {
        if let Item::Mod(module) = item
            && let Some((_, items)) = &module.content
        {
            let mut child = parent.to_vec();
            child.push(module.ident.unraw().to_string());
            modules(items, &child, result);
        }
    }
}

fn enclosing(items: &[Item], location: LineColumn, parent: &[String]) -> Option<Vec<String>> {
    for item in items {
        match item {
            Item::Impl(item) if item.impl_token.span.start() == location => {
                return Some(parent.to_vec());
            }
            Item::Macro(item)
                if item
                    .mac
                    .path
                    .segments
                    .first()
                    .is_some_and(|segment| segment.ident.span().start() <= location)
                    && location <= item.mac.delimiter.span().close().end() =>
            {
                return Some(parent.to_vec());
            }
            Item::Mod(module) if module.content.is_some() => {
                let (_, items) = module.content.as_ref()?;
                let mut child = parent.to_vec();
                child.push(module.ident.unraw().to_string());
                if let Some(found) = enclosing(items, location, &child) {
                    return Some(found);
                }
            }
            _ => {}
        }
    }
    None
}

impl Scope {
    pub(super) fn in_source(text: &str, location: LineColumn) -> syn::Result<Self> {
        struct RestoreCompiler;
        impl Drop for RestoreCompiler {
            fn drop(&mut self) {
                proc_macro2::fallback::unforce();
            }
        }
        proc_macro2::fallback::force();
        let file = {
            let _restore = RestoreCompiler;
            syn::parse_file(text)
        }?;
        let module = enclosing(&file.items, location, &[]).ok_or_else(|| {
            syn::Error::new(
                proc_macro2::Span::call_site(),
                "cannot locate realization impl source scope",
            )
        })?;
        let mut index = BTreeMap::new();
        modules(&file.items, &[], &mut index);
        Ok(Self { modules: index, module, frames: vec![], model: vec![] })
    }

    pub(super) fn model(&mut self, model: &Path) {
        self.model = path(model);
    }

    pub(super) fn block(&mut self, block: &syn::Block) {
        self.frames.push(names(block.stmts.iter().filter_map(|statement| match statement {
            syn::Stmt::Item(item) => Some(item),
            _ => None,
        })));
    }

    pub(super) fn generics(&mut self, generics: &syn::Generics) {
        self.frames.push(Names {
            bindings: generics
                .type_params()
                .map(|parameter| (parameter.ident.unraw().to_string(), None))
                .collect(),
            ..Names::default()
        });
    }

    fn resolve(
        &self,
        mut module: Vec<String>,
        mut parts: Vec<String>,
        mut depth: usize,
        seen: &mut BTreeSet<(Vec<String>, Vec<String>, usize)>,
    ) -> Resolved {
        if !seen.insert((module.clone(), parts.clone(), depth)) {
            return Resolved::Unknown;
        }
        if !self.model.is_empty() && parts.starts_with(&self.model) {
            return Resolved::Local;
        }
        if parts.first().is_some_and(|part| part == "::") {
            return Resolved::Path(parts[1..].to_vec());
        }
        if parts.first().is_some_and(|part| part == "crate" || part == "self" || part == "super") {
            let first = parts.remove(0);
            depth = 0;
            if first == "crate" {
                module.clear();
            }
            if first == "super" && module.pop().is_none() {
                return Resolved::Unknown;
            }
            return self.resolve(module, parts, depth, seen);
        }
        let Some(head) = parts.first() else {
            return Resolved::Unknown;
        };
        if module == self.module {
            for (index, frame) in self.frames[..depth].iter().enumerate().rev() {
                if frame.uncertain.contains(head) {
                    return Resolved::Unknown;
                }
                if let Some(target) = frame.bindings.get(head) {
                    let Some(mut target) = target.clone() else {
                        return Resolved::Local;
                    };
                    target.extend_from_slice(&parts[1..]);
                    return self.resolve(module, target, index + 1, seen);
                }
            }
        }
        let mut child = module.clone();
        child.push(head.clone());
        if self.modules.get(&module).is_some_and(|names| names.uncertain.contains(head)) {
            return Resolved::Unknown;
        }
        if self.modules.contains_key(&child) && parts.len() > 1 {
            return self.resolve(child, parts[1..].to_vec(), 0, seen);
        }
        if let Some(names) = self.modules.get(&module) {
            if names.external.contains(head) || names.uncertain.contains(head) {
                return Resolved::Unknown;
            }
            if let Some(target) = names.bindings.get(head) {
                let Some(mut target) = target.clone() else {
                    return Resolved::Local;
                };
                target.extend_from_slice(&parts[1..]);
                return self.resolve(module, target, 0, seen);
            }
        }
        Resolved::Path(parts)
    }

    fn classify(mut resolved: Vec<String>, schema: &Schema, variant: bool) -> EnumKind {
        let variant_name = variant.then(|| resolved.pop()).flatten();
        if schema.enums.iter().any(|(proto, enumeration)| {
            if let Some(name) = &variant_name
                && !enumeration.variants.iter().any(|variant| {
                    syn::parse_str::<Path>(variant).is_ok_and(|variant| {
                        variant.segments.last().is_some_and(|part| part.ident == name)
                    })
                })
            {
                return false;
            }
            let declared = syn::parse_str::<Path>(&enumeration.rust_type).ok().map(|parsed| {
                path(&parsed).into_iter().filter(|part| part != "::").collect::<Vec<_>>()
            });
            let generated = schema
                .enum_type(proto)
                .and_then(|ty| syn::parse_str::<Path>(ty).ok())
                .map(|parsed| {
                    path(&parsed).into_iter().filter(|part| part != "::").collect::<Vec<_>>()
                });
            generated.as_ref() == Some(&resolved)
                || declared.as_ref() == Some(&resolved)
                || (resolved.len() == 1
                    && resolved.last().map(String::as_str) == proto.rsplit('.').next())
        }) {
            return EnumKind::Generated;
        }
        if resolved.len() == 1
            && matches!(
                resolved[0].as_str(),
                "bool"
                    | "char"
                    | "str"
                    | "String"
                    | "Vec"
                    | "Option"
                    | "Result"
                    | "u8"
                    | "u16"
                    | "u32"
                    | "u64"
                    | "u128"
                    | "usize"
                    | "i8"
                    | "i16"
                    | "i32"
                    | "i64"
                    | "i128"
                    | "isize"
                    | "f32"
                    | "f64"
            )
        {
            return EnumKind::Other;
        }
        if schema.messages().any(|message| {
            syn::parse_str::<Path>(&message.rust_type).is_ok_and(|parsed| {
                path(&parsed).into_iter().filter(|part| part != "::").eq(resolved.iter().cloned())
            })
        }) {
            return EnumKind::Other;
        }
        EnumKind::Unknown
    }

    pub(super) fn kind(&self, original: &Path, schema: &Schema, variant: bool) -> EnumKind {
        let original = path(original);
        let resolved = self.resolve(
            self.module.clone(),
            original.clone(),
            self.frames.len(),
            &mut BTreeSet::new(),
        );
        let resolved = match resolved {
            Resolved::Local => return EnumKind::Other,
            Resolved::Unknown => return EnumKind::Unknown,
            Resolved::Path(path) => path,
        };
        let kind = Self::classify(resolved.clone(), schema, variant);
        if kind == EnumKind::Generated {
            return kind;
        }
        let mut unknown_glob = self.unknown_glob(schema);
        for glob in self
            .frames
            .iter()
            .rev()
            .flat_map(|frame| &frame.globs)
            .chain(self.modules.get(&self.module).into_iter().flat_map(|module| &module.globs))
        {
            let mut candidate = glob.clone();
            candidate.extend_from_slice(&resolved);
            match self.resolve(
                self.module.clone(),
                candidate,
                self.frames.len(),
                &mut BTreeSet::new(),
            ) {
                Resolved::Path(candidate)
                    if Self::classify(candidate.clone(), schema, variant)
                        == EnumKind::Generated =>
                {
                    return EnumKind::Generated;
                }
                Resolved::Unknown => unknown_glob = true,
                _ => {}
            }
        }
        if unknown_glob
            && original
                .first()
                .is_some_and(|name| matches!(name.as_str(), "String" | "Vec" | "Option" | "Result"))
        {
            EnumKind::Unknown
        } else {
            kind
        }
    }

    pub(super) fn named_imported(&self, name: &str) -> bool {
        self.frames
            .iter()
            .rev()
            .chain(self.modules.get(&self.module))
            .any(|scope| scope.bindings.get(name).is_some_and(Option::is_some))
    }

    pub(super) fn imported(&self, name: &str) -> bool {
        self.frames.iter().rev().chain(self.modules.get(&self.module)).any(|scope| {
            scope.bindings.get(name).is_some_and(Option::is_some) || !scope.globs.is_empty()
        })
    }

    fn unknown_glob(&self, schema: &Schema) -> bool {
        self.frames.iter().chain(self.modules.get(&self.module)).flat_map(|scope| &scope.globs).any(|glob| {
            !matches!(self.resolve(self.module.clone(), glob.clone(), self.frames.len(), &mut BTreeSet::new()), Resolved::Path(path) if Self::classify(path.clone(), schema, false) == EnumKind::Generated)
        })
    }

    pub(super) fn framework(&self, original: &Path, name: &str, schema: &Schema) -> bool {
        if original.segments.len() == 1 && self.unknown_glob(schema) {
            return false;
        }
        matches!(self.resolve(self.module.clone(), path(original), self.frames.len(), &mut BTreeSet::new()),
            Resolved::Path(path) if path.last().is_some_and(|last| last == name)
                && (path.len() == 1 || path.first().is_some_and(|first| first == "std" || first == "core")))
    }

    pub(super) fn constructor(&self, original: &Path) -> String {
        match self.resolve(
            self.module.clone(),
            path(original),
            self.frames.len(),
            &mut BTreeSet::new(),
        ) {
            Resolved::Path(path) => path.last().cloned().unwrap_or_default(),
            Resolved::Local | Resolved::Unknown => {
                original.segments.last().map(|part| part.ident.to_string()).unwrap_or_default()
            }
        }
    }
}
