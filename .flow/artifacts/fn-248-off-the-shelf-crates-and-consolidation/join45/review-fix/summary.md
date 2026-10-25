Carried the nine source paths from reviewed task4 fix f40149592b293c958b1b636ff4089299a36a6314 into the existing native-WKT join, excluding its Flow review ledger. Generic scalar paths now retain ExprPath directly, and inference/descriptor/build paths share the reviewed prost identifier sanitizer.

The sole patch conflict was Temporal's collected Rust type path: native well-known primitives still resolve first, followed by naming::upper_camel for ordinary types. The native prost-types namespace, wrapper scalar mutation/presence and strict local Any admission remain. native-adaptation.patch shows every source difference from the upstream fix within its nine paths; they are the retained native-WKT adaptations.

All eight focused commands passed, including the new generic-path/prost-name controls, catalog UI pass fixture, API codegen/WKT/descriptor/services tests, native refusal/consumer checks and affected-package all-targets Clippy with -D warnings. Formatting and diff checks passed. verification.json records commands, exits, source hashes and the exact upstream patch identity. No stderr or snapshot payload changed; the new prost_names.rs pass fixture is carried directly from the source fix.

Original A f9507adf06378fb9d4af80f6e7ebd88088da4673 and original B 3cf2a8464aebb0706b5a2dd42f09210f3e3f6a01 remain unchanged. The corrected join is a new descendant of B; its exact SHA is in the external handover. Original worker refs remain unchanged. No MAIN source/Flow work, guest publication, task7 or other new work ran. Stop after this handover.

stage: impl-review - skipped(policy: conductor owns authentic review and canonical completion)
