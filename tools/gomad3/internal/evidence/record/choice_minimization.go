package record

import (
	"encoding/binary"
	"errors"
)

const ChoicePrefixMinimizationSchema = "gomad3.minimization/choice-prefix/v1"

func (control ChoicePrefixControl) CandidateSHA256() (SHA256, error) {
	digest, err := control.SHA256.Bytes()
	if err != nil {
		return "", err
	}
	var encoded [40]byte
	binary.BigEndian.PutUint64(encoded[:8], uint64(control.Length))
	copy(encoded[8:], digest[:])
	return DomainHash("gomad3-minimization-choice-prefix/v1", encoded[:]), nil
}

func validateChoiceMinimization(manifest ExecutionRecord) error {
	evidence := manifest.Minimization
	controls := evidence.ChoicePrefix
	if controls == nil || manifest.Target.Backend == nil || manifest.Target.Backend.ReplayMode != ReplayExact || manifest.ChoiceProfile == nil || manifest.ChoiceProfile.Trace.TerminalState != "complete" || manifest.SimulationProfile != nil || manifest.World.Initial.Schema != "gomad3.world.snapshot/none" || manifest.IOProfile.Transcript != nil || manifest.IOProfile.ReadOnlyMounts != nil {
		return errors.New("choice minimization requires an exact standalone backend choice failure")
	}
	if controls.Original.Length != evidence.OriginalForcedDecisions || controls.Final.Length != evidence.FinalForcedDecisions || controls.Original.Length != manifest.ChoiceProfile.Trace.Decisions || controls.Original.SHA256 != manifest.ChoiceProfile.Trace.TapeSHA256 {
		return errors.New("choice minimization control counts or original tape identity changed")
	}
	original, err := controls.Original.CandidateSHA256()
	if err != nil {
		return err
	}
	final, err := controls.Final.CandidateSHA256()
	if err != nil {
		return err
	}
	if original != evidence.OriginalCandidateSHA256 || final != evidence.FinalCandidateSHA256 {
		return errors.New("choice minimization candidate is not bound to its effective prefix")
	}
	remaining := uint64(controls.Original.Length)
	for _, reduction := range evidence.Accepted {
		if reduction.Kind != "schedule_suffix" || uint64(len(reduction.Removed)) > remaining {
			return errors.New("choice minimization requires an effective suffix reduction")
		}
		remaining -= uint64(len(reduction.Removed))
		for index, decision := range reduction.Removed {
			if decision.Dimension != "runtime" || uint64(decision.Ordinal) != remaining+uint64(index) {
				return errors.New("choice minimization removed controls are not a contiguous suffix")
			}
		}
	}
	if remaining != uint64(controls.Final.Length) {
		return errors.New("choice minimization reductions do not reach the final prefix length")
	}
	return nil
}
