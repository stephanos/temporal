//! Match-coverage paths and patterns, independent of reference provenance.

use proc_macro2::Span;
use syn::{Path, spanned::Spanned};

use super::{
    Position, Positions,
    source::{EnumKind, Scope},
};
use crate::realization::Schema;

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
        syn::Pat::Ident(binding) if scope.imported(&binding.ident.to_string()) => {
            merge_position(
                result,
                prefix,
                scope.kind(&Path::from(binding.ident.clone()), schema, true),
            );
        }
        syn::Pat::Tuple(tuple) => {
            for (index, element) in tuple.elems.iter().enumerate() {
                let mut position = prefix.to_vec();
                position.push(Position::Tuple(index));
                positions(element, schema, scope, &position, result);
            }
        }
        syn::Pat::TupleStruct(pattern) => {
            for (index, element) in pattern.elems.iter().enumerate() {
                let mut position = prefix.to_vec();
                position.push(Position::Variant(scope.constructor(&pattern.path), index));
                positions(element, schema, scope, &position, result);
            }
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
        syn::Pat::Wild(_) | syn::Pat::Ident(_) => Some(pattern.span()),
        syn::Pat::Or(pattern) => {
            pattern.cases.iter().find_map(|case| wildcard(case, position, scope, schema))
        }
        syn::Pat::Paren(pattern) => wildcard(&pattern.pat, position, scope, schema),
        syn::Pat::Reference(pattern) => wildcard(&pattern.pat, position, scope, schema),
        syn::Pat::Tuple(pattern) => match position.first() {
            None => pattern.elems.iter().find_map(|part| wildcard(part, position, scope, schema)),
            Some(Position::Tuple(index)) => pattern
                .elems
                .iter()
                .nth(*index)
                .and_then(|part| wildcard(part, &position[1..], scope, schema)),
            _ => None,
        },
        syn::Pat::TupleStruct(pattern) => match position.first() {
            None => pattern.elems.iter().find_map(|part| wildcard(part, position, scope, schema)),
            Some(Position::Variant(name, index)) if *name == scope.constructor(&pattern.path) => {
                pattern
                    .elems
                    .iter()
                    .nth(*index)
                    .and_then(|part| wildcard(part, &position[1..], scope, schema))
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
