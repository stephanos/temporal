//! Stock Kettle guest realization over the reusable ABI library.

mod model;

umpire_runtime::export_guest!(model::Model, model::model());
