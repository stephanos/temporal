package network

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// VerifyRecordIdentity recomputes a Record's transition digest and snapshot
// identity with the generated codec, for Records rebuilt outside this owner
// (the cluster helper's network evidence). It is generated
// validateSimulationReplay without the lane-ordering rule and the snapshot
// replay rules, which refuse Records with live connections that the owner
// itself produces. The generated code projects the native gomadio overlay, so
// the shared part is not factored there; TestModelReplayPreflightRejectsUnboundSnapshot
// pins this copy to replay admission.
func VerifyRecordIdentity(record Record) error {
	for index, transition := range record.Transitions {
		if transition.Ordinal != uint64(index) || !validSimulationTransition(transition) {
			return errors.New("simulation network record transition is invalid")
		}
	}
	encoded, err := encodeSimulationTransitions(record.Transitions)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(append([]byte("gomad3-simulation-network-transitions/v3\x00"), encoded...))
	if record.Snapshot.TransitionSHA256 != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("simulation network record transition digest does not match")
	}
	snapshot := record.Snapshot
	snapshot.Identity = ""
	encoded, err = encodeSimulationSnapshotIdentity(snapshot)
	if err != nil {
		return err
	}
	digest = sha256.Sum256(append([]byte("gomad3-simulation-network-snapshot/v3\x00"), encoded...))
	if record.Snapshot.Identity != "sha256:"+hex.EncodeToString(digest[:]) {
		return errors.New("simulation network record snapshot identity does not match")
	}
	return nil
}
