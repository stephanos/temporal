package portableevaluation

import umpirespb "go.temporal.io/server/api/umpire/v1"

var canonicalWorkKinds = []umpirespb.WorkUnitKind{
	umpirespb.WORK_UNIT_KIND_EXPRESSION_VISIT,
	umpirespb.WORK_UNIT_KIND_RULE_RECORD_CANDIDATE,
	umpirespb.WORK_UNIT_KIND_EMITTED_COORDINATE,
	umpirespb.WORK_UNIT_KIND_LINK_ENTRY,
	umpirespb.WORK_UNIT_KIND_CLAUSE_STEP_PAIR,
	umpirespb.WORK_UNIT_KIND_PATTERN_VALUE_CANDIDATE,
}

type workTracker struct {
	limit   int64
	total   int64
	charges [7]int64
}

func (w *workTracker) charge(ctxErr error, kind umpirespb.WorkUnitKind, count int64) *evaluationFailure {
	if ctxErr != nil {
		return canceledFailure(ctxErr)
	}
	next := w.total + count
	if next > w.limit {
		return limitFailure("evaluation work exceeds the contract Limit", "evaluation-work", w.limit, next)
	}
	w.total = next
	w.charges[int(kind)] += count
	return nil
}

func (w *workTracker) result() *umpirespb.EvaluationWork {
	result := &umpirespb.EvaluationWork{Total: w.total, Limit: w.limit}
	for _, kind := range canonicalWorkKinds {
		result.Charges = append(result.Charges, &umpirespb.WorkCharge{Kind: kind, Count: w.charges[int(kind)]})
	}
	return result
}
