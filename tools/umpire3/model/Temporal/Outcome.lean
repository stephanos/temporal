namespace Umpire3.Temporal.Outcome

inductive Claim where
  | conforming
  | violating
  | inconclusive
  deriving BEq, DecidableEq, Inhabited, Repr

inductive TerminalDisposition where
  | success
  | failure
  | untagged
  deriving BEq, DecidableEq, Inhabited, Repr

inductive Verdict where
  | recovered
  | degraded
  | flagged
  | unreached
  deriving BEq, DecidableEq, Inhabited, Repr

def classify : Claim → Option TerminalDisposition → Verdict
  | .violating, _ => .flagged
  | _, some .failure => .degraded
  | _, some .success => .recovered
  | _, some .untagged => .recovered
  | _, none => .unreached

theorem flaggedIffViolation (claim : Claim) (terminal : Option TerminalDisposition) :
    classify claim terminal = .flagged ↔ claim = .violating := by
  cases claim <;> cases terminal <;> simp_all [classify] <;>
    rename_i disposition <;> cases disposition <;> simp

theorem degradedIsConformingFailure :
    classify .conforming (some .failure) = .degraded := by decide

theorem violationWinsOverFailure :
    classify .violating (some .failure) = .flagged := by decide

end Umpire3.Temporal.Outcome
