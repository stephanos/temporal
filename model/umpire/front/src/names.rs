use std::collections::BTreeSet;

/// TLA+ keywords, standard-module operators and fixed emitted names that no generated or
/// authored symbol may take.
pub const RESERVED_NAMES: &[&str] = &[
    "vars",
    "ModelVars",
    "Init",
    "ModelNext",
    "ModelEdge",
    "Edge",
    "Complete",
    "Next",
    "Spec",
    "__umpire_Div",
    "__umpire_a",
    "__umpire_b",
    "__umpire_d",
    "__umpire_n",
    "__umpire_q",
    "__umpire_r",
    "__umpire_negative",
    "__umpire_edge_tick",
    "__umpire_pending",
    "__umpire_steps",
    "__umpire_action",
    "__umpire_choice",
    "__umpire_outcome",
    "__umpire_facts",
    "__umpire_input",
    "TRUE",
    "FALSE",
    "BOOLEAN",
    "Int",
    "Nat",
    "Assert",
    "Seq",
    "Len",
    "Append",
    "Head",
    "Tail",
    "SubSeq",
    "SelectSeq",
    "DOMAIN",
    "IF",
    "THEN",
    "ELSE",
    "CASE",
    "OTHER",
    "LET",
    "IN",
    "UNCHANGED",
    "ENABLED",
    "SUBSET",
    "UNION",
    "CHOOSE",
    "ASSUME",
    "THEOREM",
    "VARIABLE",
    "VARIABLES",
    "CONSTANT",
    "CONSTANTS",
    "MODULE",
    "INSTANCE",
    "LOCAL",
    "EXTENDS",
];
pub(crate) fn allocate(ast: &syn::File, base: &str, generated: &mut BTreeSet<String>) -> String {
    struct Occupied<'a>(&'a mut BTreeSet<String>);
    impl<'ast> syn::visit::Visit<'ast> for Occupied<'_> {
        fn visit_ident(&mut self, ident: &'ast syn::Ident) {
            self.0.insert(ident.to_string());
        }
        fn visit_item_enum(&mut self, value: &'ast syn::ItemEnum) {
            for variant in &value.variants {
                self.0.insert(format!("{}_{}", value.ident, variant.ident));
            }
            syn::visit::visit_item_enum(self, value);
        }
        fn visit_impl_item_fn(&mut self, value: &'ast syn::ImplItemFn) {
            for prefix in [
                "Invariant",
                "Post",
                "Negative",
                "Assumption",
                "Scoped",
                "Activate",
                "Monitor",
                "ActiveEdge",
                "PrefixEdge",
                "Complete",
                "ProgressNext",
                "ProgressSpec",
                "Maximal",
                "Progress",
            ] {
                self.0.insert(format!("{prefix}_{}", value.sig.ident));
            }
            syn::visit::visit_impl_item_fn(self, value);
        }
    }
    let mut occupied: BTreeSet<_> = RESERVED_NAMES.iter().map(|name| (*name).to_owned()).collect();
    syn::visit::Visit::visit_file(&mut Occupied(&mut occupied), ast);
    occupied.extend(generated.iter().cloned());
    let mut name = base.to_owned();
    let mut suffix = 0;
    while occupied.contains(&name) {
        suffix += 1;
        name = format!("{base}_{suffix}");
    }
    generated.insert(name.clone());
    name
}
