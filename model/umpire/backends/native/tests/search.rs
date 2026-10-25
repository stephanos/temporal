//! Shared traversal controls independent of Model state or evidence policy.

#![forbid(unsafe_code)]

use umpire_checker::search::{SearchLimit, Traversal};

#[derive(Debug)]
struct Emitted(&'static str);

#[test]
fn caller_labels_keep_same_state_with_distinct_cursors_and_nonclone_emissions()
-> Result<(), SearchLimit> {
    let mut search = Traversal::new(2, 8);
    let root = search.root((0, 0))?;
    assert_eq!(search.next(), Some(root));
    let first = search.push(root, (0, 1), Some(Emitted("first")))?;
    let second = search.push(root, (0, 2), Some(Emitted("second")))?;
    assert_eq!(search.next(), Some(first));
    assert_eq!(search.next(), Some(second));
    assert_ne!(search.label(first).position, search.label(second).position);
    assert_eq!(
        search.label(first).parent.as_ref().map(|(_, edge)| edge.as_ref().map(|e| e.0)),
        Some(Some("first"))
    );
    assert_eq!(
        search.label(second).parent.as_ref().map(|(_, edge)| edge.as_ref().map(|e| e.0)),
        Some(Some("second"))
    );
    Ok(())
}

#[test]
fn epsilon_labels_precede_edges_without_spending_depth() -> Result<(), SearchLimit> {
    let mut search = Traversal::<_, u8>::new(1, 5);
    let root = search.root(0)?;
    assert_eq!(search.next(), Some(root));
    let edge = search.push(root, 1, Some(7))?;
    let epsilon = search.push(root, 2, None)?;
    assert_eq!(search.next(), Some(epsilon));
    assert_eq!(search.label(epsilon).distance, 0);
    assert_eq!(search.next(), Some(edge));
    assert_eq!(search.label(edge).distance, 1);
    assert_eq!(search.push(edge, 3, Some(8)), Err(SearchLimit::Depth { limit: 1 }));
    assert_eq!(search.len(), 3);
    assert_eq!(search.next(), None);
    Ok(())
}

#[test]
fn work_bound_accounts_labels_and_explicit_evaluations_without_partial_admission()
-> Result<(), SearchLimit> {
    let mut search = Traversal::<_, ()>::new(3, 3);
    let root = search.root(0)?;
    search.charge()?;
    let child = search.push(root, 1, Some(()))?;
    assert_eq!(search.charge(), Err(SearchLimit::Work { limit: 3 }));
    assert_eq!(search.push(root, 2, Some(())), Err(SearchLimit::Work { limit: 3 }));
    assert_eq!(search.len(), 2);
    assert_eq!(search.next(), Some(root));
    assert_eq!(search.next(), Some(child));
    assert_eq!(search.next(), None);
    Ok(())
}
