# First formal review triage

Three Codex gpt-6.1-sol/high read-only draws reviewed851c0e92..870ee1de under RID38ff9359bd0d40a5b421d42b0edc1960. Same-family routing was explicitly disclosed. All draws delivered NEEDS_WORK; none was a transport failure.

Keep integration:1: required Number generation can choose absent zero despite satisfiable nonzero values. A new real fill regression with seed3 fails with FillError::Invalid(required), as predicted. Log: .flow/tmp/fn2176-required-number-red.log. The regression also covers nonzero fractional intervals and explicit refusal when zero is the only allowed value. This is a generic annotation generation defect, not a Model or server contract change.

Drop duplicate correctness:1 and contracts:1: their proposed arm.guard check assumes the wrong Syn AST. Cargo tree for umpire-temporal-annotate-syntax selects syn3.0.9; model/Cargo.lock agrees. Its source expr.rs1218-1224 defines Arm with attrs, pat, fat_arrow_token, body and comma, but no guard member. Guards are Pat::Guard (pat.rs33), which method_names in services.rs184-202 rejects because it admits only Pat::Or and Pat::Path. An installed syn3.0.6 source was initially inspected; actual dependency selection and the same evidence were then explicitly verified against3.0.9.

Fresh command: mise exec -- cargo test -p umpire-temporal-annotate-syntax --lib services::tests::classification_rejects_non_declarative_or_unexplained_arms -- --exact --nocapture. It exits0 with exactly1 passing test, including the exact guarded-arm string cited by both draws. Log: .flow/tmp/fn2176-formal-guard-finding-actual-test.log. An earlier unqualified exact selector ran zero tests and is inconclusive, not green evidence.

No correct parser code was changed to satisfy the refuted finding. The coordinator keeps one evidenced actionable finding; the finalizer, not the writer, owns the verdict. The preserved draw coverage concern about R1 follows the refuted guard premise and is recorded rather than silently treated as a new defect.
