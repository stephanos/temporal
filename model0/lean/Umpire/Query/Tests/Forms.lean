import Umpire.Query.Tests.Fixtures

/-! Form identity checks for every public Query form. -/

namespace Umpire.QueryTests

open Umpire

def summaryOf
    (result : Except QueryError (CheckedQuery (fun _ => True))) : Option String :=
  result.toOption.map fun query => query.form.name

/-! Every public form carries its own canonical name into the checked Query. -/
example : [
    summaryOf (Query.check exhaustiveContext
      (declaration (.verify Property.checked) exhaustivePolicy)),
    summaryOf (Query.check context (declaration (.find Property.checked))),
    summaryOf (Query.check context (declaration (.findViolation Property.checked))),
    summaryOf (Query.check context (declaration (.pick [Property.checked])))
  ] = [some "verify", some "find", some "find-violation", some "pick"] := by
  native_decide

end Umpire.QueryTests
