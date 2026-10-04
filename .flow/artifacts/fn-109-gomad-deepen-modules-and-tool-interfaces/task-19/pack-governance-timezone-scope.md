# Companion timezone effect in pack governance validation

Task-19's read-only effect scout found a second actual pure-root path. The
conductor checked current schema.go:131–159,187–211 and v2_selection.go:187–196:
SelectPacksForPlatform validates every pack, ValidatePack calls
ValidatePackStructure, and structure validation calls validatePackGovernance.
The governance helper calls time.Parse(RFC3339, ReviewedAt) before checking its
trailing Z and UTC location. Numeric-offset input therefore reaches Local
initialization even though it is subsequently rejected.

The scope adds only internal/compatibilitypack/schema*.go for explicit
ParseInLocation(..., time.UTC) and preservation tests, matching the record fix.
No pack bytes, schema, accepted UTC/Z inputs, grammar, classification/precedence,
public surface or capability admission changes. Numeric offsets remain rejected.
Retain actual old-source transitive effect RED and corrected GREEN, plus valid,
invalid, offset, zero-valued/precedence and canonical-pack behavior expectations.
Never mark compatibility validation or all time parsing pure to hide the leak.

This grounded correction is within the milestone architecture effect obligation;
the user requested autonomous recommendations. No source edit by the conductor
or second writer is authorized. The sole task-19 writer implements and verifies
the correction alongside the original scoped repairs.
