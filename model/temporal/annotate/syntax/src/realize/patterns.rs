//! Match-coverage paths and patterns, independent of reference provenance.

use proc_macro2::Span;
use syn::{Path, spanned::Spanned};

use super::{
    Position, Positions,
    source::{EnumKind, Scope},
};
use crate::realization::Schema;

type Elements = syn::punctuated::Punctuated<syn::Pat, syn::Token![,]>;

pub(super) fn match_positions(positions: Positions) -> Positions {
    let mut result = Positions::new();
    for (path, kind) in positions {
        let path: Vec<_> = path.into_iter().filter(|step| *step != Position::Reference).collect();
        merge_position(&mut result, &path, kind);
    }
    result
}

pub(super) fn positions(
    pattern: &syn::Pat,
    schema: &Schema,
    scope: &Scope,
    prefix: &[Position],
    result: &mut Positions,
) {
    match pattern {
        syn::Pat::Path(path) => {
            merge_position(result, prefix, scope.kind(&path.path, schema, true));
        }
        syn::Pat::Ident(binding)
            if scope.named_imported(&binding.ident.to_string())
                || scope.kind(&Path::from(binding.ident.clone()), schema, true)
                    == EnumKind::Generated =>
        {
            merge_position(
                result,
                prefix,
                scope.kind(&Path::from(binding.ident.clone()), schema, true),
            );
        }
        syn::Pat::Tuple(tuple) => {
            sequence_positions(&tuple.elems, None, schema, scope, prefix, result);
        }
        syn::Pat::TupleStruct(pattern) => {
            sequence_positions(
                &pattern.elems,
                Some(&scope.constructor(&pattern.path)),
                schema,
                scope,
                prefix,
                result,
            );
        }
        syn::Pat::Or(pattern) => {
            for case in &pattern.cases {
                positions(case, schema, scope, prefix, result);
            }
        }
        syn::Pat::Paren(pattern) => positions(&pattern.pat, schema, scope, prefix, result),
        syn::Pat::Reference(pattern) => positions(&pattern.pat, schema, scope, prefix, result),
        _ => {}
    }
}

fn sequence_positions(
    elements: &Elements,
    constructor: Option<&str>,
    schema: &Schema,
    scope: &Scope,
    prefix: &[Position],
    result: &mut Positions,
) {
    let rest = elements.iter().position(|part| matches!(part, syn::Pat::Rest(_)));
    let arity = result
        .keys()
        .filter(|path| path.starts_with(prefix))
        .find_map(|path| match (path.get(prefix.len()), constructor) {
            (Some(Position::Tuple(_, arity)), None) => Some(*arity),
            (Some(Position::Variant(name, _, arity)), Some(constructor)) if name == constructor => {
                Some(*arity)
            }
            _ => None,
        })
        .or_else(|| rest.is_none().then_some(elements.len()));
    let Some(arity) = arity else {
        let mut hints = Positions::new();
        for element in elements {
            positions(element, schema, scope, &[], &mut hints);
        }
        if hints.values().any(|kind| *kind != EnumKind::Other) {
            merge_position(result, prefix, EnumKind::Unknown);
        }
        return;
    };
    for (index, element) in elements.iter().enumerate() {
        if rest == Some(index) {
            continue;
        }
        let index = if rest.is_some_and(|rest| index > rest) {
            let Some(index) = arity.checked_sub(elements.len() - index) else {
                merge_position(result, prefix, EnumKind::Unknown);
                return;
            };
            index
        } else {
            index
        };
        let mut path = prefix.to_vec();
        path.push(constructor.map_or_else(
            || Position::Tuple(index, arity),
            |name| Position::Variant(name.into(), index, arity),
        ));
        positions(element, schema, scope, &path, result);
    }
}

fn merge_position(result: &mut Positions, prefix: &[Position], kind: EnumKind) {
    let current = result.entry(prefix.to_vec()).or_insert(kind);
    if kind == EnumKind::Generated || (*current == EnumKind::Unknown && kind == EnumKind::Other) {
        *current = kind;
    }
}

pub(super) fn wildcard(
    pattern: &syn::Pat,
    position: &[Position],
    scope: &Scope,
    schema: &Schema,
) -> Option<Span> {
    match pattern {
        syn::Pat::Ident(binding)
            if scope.imported(&binding.ident.to_string())
                && scope.kind(&Path::from(binding.ident.clone()), schema, true)
                    == EnumKind::Generated =>
        {
            None
        }
        syn::Pat::Ident(binding)
            if scope.framework(&Path::from(binding.ident.clone()), "None", schema) =>
        {
            None
        }
        syn::Pat::Wild(_) | syn::Pat::Ident(_) | syn::Pat::Rest(_) => Some(pattern.span()),
        syn::Pat::Or(pattern) => {
            pattern.cases.iter().find_map(|case| wildcard(case, position, scope, schema))
        }
        syn::Pat::Paren(pattern) => wildcard(&pattern.pat, position, scope, schema),
        syn::Pat::Reference(pattern) => wildcard(&pattern.pat, position, scope, schema),
        syn::Pat::Tuple(pattern) => match position.first() {
            None => pattern.elems.iter().find_map(|part| wildcard(part, position, scope, schema)),
            Some(Position::Tuple(index, arity)) => {
                sequence_wildcard(&pattern.elems, *index, *arity, &position[1..], scope, schema)
            }
            _ => None,
        },
        syn::Pat::TupleStruct(pattern) => match position.first() {
            None => pattern.elems.iter().find_map(|part| wildcard(part, position, scope, schema)),
            Some(Position::Variant(name, index, arity))
                if *name == scope.constructor(&pattern.path) =>
            {
                sequence_wildcard(&pattern.elems, *index, *arity, &position[1..], scope, schema)
            }
            _ => None,
        },
        syn::Pat::Struct(pattern) if position.is_empty() => {
            pattern.rest.as_ref().map(Spanned::span).or_else(|| {
                pattern
                    .fields
                    .iter()
                    .find_map(|field| wildcard(&field.pat, position, scope, schema))
            })
        }
        syn::Pat::Slice(pattern) if position.is_empty() => {
            pattern.elems.iter().find_map(|part| wildcard(part, position, scope, schema))
        }
        _ => None,
    }
}

fn sequence_wildcard(
    elements: &Elements,
    index: usize,
    arity: usize,
    tail: &[Position],
    scope: &Scope,
    schema: &Schema,
) -> Option<Span> {
    let target =
        if let Some(rest) = elements.iter().position(|part| matches!(part, syn::Pat::Rest(_))) {
            let suffix = elements.len() - rest - 1;
            if index < rest {
                index
            } else if index >= arity.saturating_sub(suffix) {
                elements.len().checked_sub(arity.checked_sub(index)?)?
            } else {
                rest
            }
        } else {
            index
        };
    elements.iter().nth(target).and_then(|part| wildcard(part, tail, scope, schema))
}
