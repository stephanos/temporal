package network

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// VerifyRecordIdentity recomputes a Record's transition digest and snapshot
// identity with the generated codec, for Records rebuilt outside this owner
// (the cluster helper's network evidence). Unlike replay admission it accepts
// any Record shape the owner can produce, including live connections.
func VerifyRecordIdentity(record Record) error {
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
