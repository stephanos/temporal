//! One mini activity run per guest instance through the reusable ABI hooks.

umpire_runtime::export_guest!(
    umpire_temporal::AnnotationRuntime,
    umpire_temporal::annotation_runtime()
);
