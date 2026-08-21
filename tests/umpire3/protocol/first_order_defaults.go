package protocol

import (
	"bytes"
	_ "embed"
)

//go:embed generated/nexus-cancellation.first-order.json
var defaultNexusFirstOrderJSON []byte

//go:embed generated/nexus-cancellation-mutated.first-order.json
var defaultNexusMutatedFirstOrderJSON []byte

func DefaultFirstOrderView(target TargetID, variant string) (FirstOrderView, bool, error) {
	var encoded []byte
	switch {
	case target == TargetIDNexusCancellation && variant == "sound":
		encoded = defaultNexusFirstOrderJSON
	case target == TargetIDNexusCancellation && variant == "stale-completion-guard-removed":
		encoded = defaultNexusMutatedFirstOrderJSON
	default:
		return FirstOrderView{}, false, nil
	}
	view, err := DecodeFirstOrderView(bytes.NewReader(encoded), DefaultDecodeLimit)
	if err != nil {
		return FirstOrderView{}, true, err
	}
	return view, true, nil
}

func DefaultFirstOrderViewForFaults(target TargetID, faults []FaultKind) (FirstOrderView, bool, error) {
	faultSet := make(map[FaultKind]struct{}, len(faults))
	for _, fault := range faults {
		faultSet[fault] = struct{}{}
	}
	for _, variant := range []string{"stale-completion-guard-removed", "sound"} {
		view, found, err := DefaultFirstOrderView(target, variant)
		if err != nil {
			return FirstOrderView{}, found, err
		}
		if !found {
			continue
		}
		matches := true
		for _, fault := range view.ActivatingFaults {
			if _, active := faultSet[fault]; !active {
				matches = false
				break
			}
		}
		if matches {
			return view, true, nil
		}
	}
	return FirstOrderView{}, false, nil
}
