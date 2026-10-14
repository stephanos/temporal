package qualification

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"go.temporal.io/server/tools/gomad/internal/evidence/record"
	hostexec "go.temporal.io/server/tools/gomad/internal/host/exec"
	hostfs "go.temporal.io/server/tools/gomad/internal/host/fs"
)

const publicationReceiptLimit = 8 << 10

// PublicationReceipt measures operator self after large Result encoding and
// durable publication. Only the bounded receipt/response emission follows this
// snapshot; it is not a measurement of the process exit tail or child CPU.
type PublicationReceipt struct {
	ResultSHA256      record.SHA256
	OperatorProcessID uint64
	OperatorAfter     *hostexec.ResourceUsage
}

// WriteResultObserved preserves immutable Result bytes and captures the real
// post-publication self high-water/CPU counters, never reconstructed wait4 sums.
func WriteResultObserved(root string, result Result) (string, PublicationReceipt, error) {
	return writeResultObserved(root, result, nil)
}

func writeResultObserved(root string, result Result, observer func(string)) (string, PublicationReceipt, error) {
	path, err := writeResult(root, result, observer)
	if err != nil {
		return path, PublicationReceipt{}, err
	}
	hash, err := publicationResultIdentity(path)
	if err != nil {
		return path, PublicationReceipt{}, err
	}
	observeHostMemory(observer, "result.publication.ready")
	receipt := PublicationReceipt{ResultSHA256: hash, OperatorProcessID: uint64(os.Getpid()), OperatorAfter: hostexec.CurrentProcessResourceUsage()}
	_, observationErr := validatePublicationObservation(result, receipt, hash)
	encoded, err := json.Marshal(receipt)
	if err != nil {
		return path, receipt, err
	}
	_, err = publishQualificationBytes(root, filepath.Base(path)+".publication.json", append(encoded, '\n'), publicationReceiptLimit)
	return path, receipt, errors.Join(err, observationErr)
}

func publicationResultIdentity(path string) (record.SHA256, error) {
	name := filepath.Base(path)
	if !strings.HasPrefix(name, "result-") || !strings.HasSuffix(name, ".json") {
		return "", errors.New("publication requires content-addressed Result path")
	}
	return record.ParseSHA256("sha256:" + strings.TrimSuffix(strings.TrimPrefix(name, "result-"), ".json"))
}

// OpenPublicationReceipt uses the existing strict bounded JSON owner. Original
// source, compiler and helper are not consulted; the caller validates binding
// against its already-opened Result with ValidatePublicationReceipt.
func OpenPublicationReceipt(resultPath string) (PublicationReceipt, error) {
	if _, err := publicationResultIdentity(resultPath); err != nil {
		return PublicationReceipt{}, err
	}
	encoded, err := hostfs.ReadBounded(resultPath+".publication.json", publicationReceiptLimit)
	if err != nil {
		return PublicationReceipt{}, err
	}
	var receipt PublicationReceipt
	if err := strictDecode(encoded, &receipt); err != nil {
		return PublicationReceipt{}, err
	}
	return receipt, nil
}

// ValidatePublicationReceipt is reader-side: it re-encodes the already-opened
// Result to validate exact content binding. Producers must not call it after
// their snapshot; WriteResultObserved already validates without a large copy.
// It returns only the measured self CPU after the sealed
// Result boundary. A corpus adds this once to its existing compiler/helper/self
// costs, without rewriting reports or double-counting children.
func ValidatePublicationReceipt(resultPath string, result Result, receipt PublicationReceipt) (uint64, error) {
	hash, err := publicationResultIdentity(resultPath)
	if err != nil {
		return 0, err
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return 0, err
	}
	if record.HashBytes(append(encoded, '\n')) != hash {
		return 0, errors.New("publication receipt Result content binding changed")
	}
	return validatePublicationObservation(result, receipt, hash)
}

func validatePublicationObservation(result Result, receipt PublicationReceipt, hash record.SHA256) (uint64, error) {
	after := receipt.OperatorAfter
	if hash == "" || receipt.ResultSHA256 != hash || result.OperatorProcessID == 0 || receipt.OperatorProcessID != result.OperatorProcessID || after == nil || result.OperatorPreparationBefore == nil || result.OperatorExecutionAfter == nil {
		return 0, errors.New("missing or mismatched publication self observation")
	}
	if after.PeakRSSBytes == 0 || after.PeakRSSBytes > 2<<30 {
		return 0, errors.New("terminal operator peak exceeds 2GiB budget or is unavailable")
	}
	for _, before := range []*hostexec.ResourceUsage{result.OperatorPreparationBefore, result.OperatorPreparationAfter, result.OperatorExecutionBefore, result.OperatorExecutionAfter} {
		if before != nil && (after.UserNanos < before.UserNanos || after.SystemNanos < before.SystemNanos || after.PeakRSSBytes < before.PeakRSSBytes) {
			return 0, errors.New("publication self observation counters moved backwards")
		}
	}
	user, system := after.UserNanos-result.OperatorExecutionAfter.UserNanos, after.SystemNanos-result.OperatorExecutionAfter.SystemNanos
	if ^uint64(0)-user < system {
		return 0, errors.New("publication self CPU overflow")
	}
	return user + system, nil
}
