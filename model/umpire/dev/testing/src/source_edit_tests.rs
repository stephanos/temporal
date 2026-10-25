use crate::{replace_code, replace_in, replace_once};

#[test]
fn source_edits_preserve_surrounding_text_and_unicode() {
    assert_eq!(replace_once("before target after", "target", "new"), "before new after");
    assert_eq!(replace_in("pre [one] post", "[", "]", "one", "two"), "pre [two] post");
    assert_eq!(replace_code("before α  + β after", "α+β", "new"), "before new after");
}

#[test]
fn source_edits_refuse_missing_duplicate_and_empty_anchors() {
    for source in ["absent", "target target"] {
        assert!(std::panic::catch_unwind(|| replace_once(source, "target", "new")).is_err());
        assert!(std::panic::catch_unwind(|| replace_code(source, "target", "new")).is_err());
    }
    for (source, start, end, from) in [
        ("missing", "[", "]", "x"),
        ("[x][x]", "[", "]", "x"),
        ("[x", "[", "]", "x"),
        ("[x]]", "[", "]", "x"),
        ("[y]", "[", "]", "x"),
        ("[xx]", "[", "]", "x"),
    ] {
        assert!(std::panic::catch_unwind(|| replace_in(source, start, end, from, "new")).is_err());
    }
    assert!(std::panic::catch_unwind(|| replace_code("", "", "new")).is_err());
}
