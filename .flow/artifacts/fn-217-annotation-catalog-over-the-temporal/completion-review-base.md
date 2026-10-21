# Completion-review base

The saved whole-branch run base 951c5516e9e7b3066e7e069adda9565cfd68844c
predates the Rust migration and every annotation task. Its diff exceeds Flow's
64 MiB complete-identity limit. The first completion command exited 2 before
dispatch, with no verdict and completion status still unknown. Its failed output
is retained in `.flow/tmp/fn217-parent-completion-review.out`.

Use `32d63cceeaeba286d6bb843e1fb415b82ab82452`, the parent of the first annotation
implementation commit `8baf7c6ef32bad11bccdda9a53c99f2b1d6ea831`.
Git history confirms the first commit adds the annotations crate itself. This
base therefore includes all fn-217 implementation tasks and their later fixes,
not just the final derive rewrite. The complete diff at the decision checkpoint
6e4f1b4b32 is 2,185,761 bytes, comfortably below the unchanged identity limit.
Interleaved other-spec commits remain included; no path filter or truncation
removes them. No counter reset, force dispatch, fabricated verdict or source
change was used to bypass the evidence guard.
