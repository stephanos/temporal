package qualification

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.temporal.io/server/tests/umpire3/canary"
	umpire3runtime "go.temporal.io/server/tests/umpire3/execution"
	"go.temporal.io/server/tests/umpire3/protocol"
)

const FormatVersion = "umpire3/qualification-receipt/v1"

type Receipt struct {
	FormatVersion         string `json:"formatVersion"`
	Release               string `json:"release"`
	ReleaseDigest         string `json:"releaseDigest"`
	Profile               string `json:"profile"`
	ExperimentID          string `json:"experimentID"`
	ExperimentDigest      string `json:"experimentDigest"`
	BuildID               string `json:"buildID"`
	ConfigurationIdentity string `json:"configurationIdentity"`
	EvidenceDigest        string `json:"evidenceDigest"`
}

type Request struct {
	ReleaseBytes    []byte
	ExperimentBytes []byte
	ResultBytes     []byte
	Profile         string
}

func Qualify(request Request) (Receipt, error) {
	if len(request.ReleaseBytes) == 0 || len(request.ExperimentBytes) == 0 ||
		len(request.ResultBytes) == 0 || request.Profile == "" {
		return Receipt{}, errors.New("release, experiment, result, and profile are required")
	}
	release, err := protocol.DecodeReleaseManifest(request.ReleaseBytes)
	if err != nil {
		return Receipt{}, fmt.Errorf("decode release: %w", err)
	}
	if err := release.ValidateAgainstCurrent(); err != nil {
		return Receipt{}, fmt.Errorf("validate release against current artifacts: %w", err)
	}
	if !requiredQualification(release, request.Profile) {
		return Receipt{}, fmt.Errorf("profile %q is not an external gate for release %q", request.Profile, release.Release)
	}
	experiment, err := protocol.DecodeExperiment(bytes.NewReader(request.ExperimentBytes), protocol.DefaultDecodeLimit)
	if err != nil {
		return Receipt{}, fmt.Errorf("decode experiment: %w", err)
	}
	semanticHash, released := release.Experiments[experiment.ExperimentID]
	if !released || semanticHash != experiment.Model.SemanticHash {
		return Receipt{}, errors.New("experiment is not bound to the candidate release")
	}
	result, err := DecodeResult(request.ResultBytes)
	if err != nil {
		return Receipt{}, err
	}
	if err := validateResult(request.Profile, experiment, result); err != nil {
		return Receipt{}, err
	}
	evidenceBytes, err := result.Evidence.CanonicalJSON()
	if err != nil {
		return Receipt{}, fmt.Errorf("encode evidence: %w", err)
	}
	evidenceHash := sha256.Sum256(evidenceBytes)
	releaseHash := sha256.Sum256(request.ReleaseBytes)
	return Receipt{
		FormatVersion: FormatVersion, Release: release.Release,
		ReleaseDigest: "sha256:" + hex.EncodeToString(releaseHash[:]), Profile: request.Profile,
		ExperimentID: experiment.ExperimentID, ExperimentDigest: result.ExperimentDigest,
		BuildID: result.Environment.BuildID, ConfigurationIdentity: result.Environment.ConfigurationIdentity,
		EvidenceDigest: "sha256:" + hex.EncodeToString(evidenceHash[:]),
	}, nil
}

func requiredQualification(release protocol.ReleaseManifest, name string) bool {
	return slices.ContainsFunc(release.ExternalQualifications, func(value protocol.ExternalQualification) bool {
		return value.Profile == name && value.Status == "required"
	})
}

func DecodeResult(encoded []byte) (umpire3runtime.Result, error) {
	var envelope struct {
		Runtime  json.RawMessage `json:"runtime"`
		Complete bool            `json:"complete"`
	}
	if err := json.Unmarshal(encoded, &envelope); err != nil {
		return umpire3runtime.Result{}, fmt.Errorf("decode result envelope: %w", err)
	}
	if len(envelope.Runtime) != 0 {
		var canaryResult canary.Result
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&canaryResult); err != nil {
			return umpire3runtime.Result{}, fmt.Errorf("decode canary result: %w", err)
		}
		if !canaryResult.Complete {
			return umpire3runtime.Result{}, errors.New("canary result is incomplete")
		}
		if err := validateDecodedResult(canaryResult.Runtime); err != nil {
			return umpire3runtime.Result{}, err
		}
		return canaryResult.Runtime, nil
	}
	var result umpire3runtime.Result
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&result); err != nil {
		return umpire3runtime.Result{}, fmt.Errorf("decode runtime result: %w", err)
	}
	if err := validateDecodedResult(result); err != nil {
		return umpire3runtime.Result{}, err
	}
	return result, nil
}

func validateDecodedResult(result umpire3runtime.Result) error {
	if result.FormatVersion != umpire3runtime.ResultFormatVersion {
		return fmt.Errorf("unsupported runtime result format %q", result.FormatVersion)
	}
	if err := result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate result assurance: %w", err)
	}
	return nil
}

func validateResult(profile string, experiment protocol.Experiment, result umpire3runtime.Result) error {
	digest, err := experiment.Digest()
	if err != nil {
		return err
	}
	if result.FormatVersion != umpire3runtime.ResultFormatVersion || result.ExperimentDigest != digest {
		return errors.New("result is not bound to the executed experiment")
	}
	if err := result.ValidateAssurance(); err != nil {
		return fmt.Errorf("validate result assurance: %w", err)
	}
	if result.Environment.Name != profile || result.Environment.BuildID == "" ||
		result.Environment.ConfigurationIdentity == "" {
		return errors.New("result lacks deployment profile attestation")
	}
	expectedEvidenceProfile := "public-grpc-history"
	if profile == "grpc-only-black-box" {
		expectedEvidenceProfile = "public-grpc"
	}
	if result.Environment.EvidenceProfile != expectedEvidenceProfile {
		return fmt.Errorf("profile %q requires %q evidence", profile, expectedEvidenceProfile)
	}
	if result.Claim.Kind != umpire3runtime.ClaimConforming || result.Claim.Property != experiment.Property.Identifier {
		return errors.New("qualification requires a conforming claim for the experiment property")
	}
	if len(result.Omissions) != 0 || !result.Cleanup.Complete {
		return errors.New("qualification requires complete evidence and cleanup")
	}
	if len(result.Faults) != len(experiment.Faults) {
		return errors.New("qualification requires realization evidence for every declared fault")
	}
	for index, fault := range result.Faults {
		if fault.Identifier != experiment.Faults[index].Identifier || !fault.Realized ||
			!fault.Released || !fault.CleanupComplete || fault.Error != "" ||
			fault.SourceIdentity == "" || fault.Reference == "" || fault.EntityIdentity == "" {
			return fmt.Errorf("qualification fault %q lacks complete realization and cleanup evidence", fault.Identifier)
		}
	}
	if len(result.Evidence.Facts) == 0 || len(result.Evidence.Claims) == 0 {
		return errors.New("qualification result has no supporting evidence claim")
	}
	if err := result.Evidence.Validate(); err != nil {
		return fmt.Errorf("validate qualification evidence: %w", err)
	}
	return nil
}
